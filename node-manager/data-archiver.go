// Copyright 2019 dfuse Platform Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package nodemanager

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/abourget/llerrgroup"
	"github.com/dfuse-io/bstream"
	pbcodec "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/codec/v1"
	"github.com/dfuse-io/dstore"
	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"
)

type BlockMarshaller func(block *bstream.Block) ([]byte, error)

type BlockDataArchiver struct {
	Store  dstore.Store
	suffix string

	uploadMutex sync.Mutex
	workDir     string
	logger      *zap.Logger
}

func NewBlockDataArchiver(
	store dstore.Store,
	workDir string,
	suffix string,
	logger *zap.Logger,
) *BlockDataArchiver {
	return &BlockDataArchiver{
		Store:   store,
		suffix:  suffix,
		workDir: workDir,
		logger:  logger,
	}
}

func (s *BlockDataArchiver) StoreBlockData(bundle *pbcodec.AccountChangesBundle, fileName string) error {

	// Store the actual file using multiple folders instead of a single one.
	// We assume 10 digits block number at start of file name. We take the first 7
	// ones and used them as the sub folder for the file.
	subDirectory := fileName[0:7]

	targetDir := filepath.Join(s.workDir, subDirectory)
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		err := os.MkdirAll(targetDir, 0755)
		if err != nil {
			return fmt.Errorf("mkdir all: %w", err)
		}
	}

	tempFile := filepath.Join(targetDir, fileName+".dat.temp")
	finalFile := filepath.Join(targetDir, fileName+".dat")

	file, err := os.OpenFile(tempFile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	data, err := proto.Marshal(bundle)
	if err != nil {
		return fmt.Errorf("proto marshall: %w", err)
	}

	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	//blockWriter, err := s.blockWriterFactory.New(file)
	//if err != nil {
	//	file.Close()
	//	return fmt.Errorf("write block factory: %w", err)
	//}
	//
	//if err := blockWriter.Write(block); err != nil {
	//	file.Close()
	//	return fmt.Errorf("write block: %w", err)
	//}

	if err := file.Close(); err != nil {
		return fmt.Errorf("close file: %w", err)
	}

	if err := os.Rename(tempFile, finalFile); err != nil {
		return fmt.Errorf("rename %q to %q: %w", tempFile, finalFile, err)
	}

	return nil
}

func (a *BlockDataArchiver) Start() {
	lastUploadFailed := false
	for {
		err := a.uploadFiles()
		if err != nil {
			a.logger.Warn("temporary failure trying to upload mindreader block files, will retry", zap.Error(err))
			lastUploadFailed = true
		} else {
			if lastUploadFailed {
				a.logger.Warn("success uploading previously failed mindreader block files")
				lastUploadFailed = false
			}
		}

		select {
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func (s *BlockDataArchiver) uploadFiles() error {
	s.uploadMutex.Lock()
	defer s.uploadMutex.Unlock()
	filesToUpload, err := findFilesToUpload(s.workDir, s.logger, ".dat")
	if err != nil {
		return fmt.Errorf("unable to find files to upload: %w", err)
	}

	if len(filesToUpload) == 0 {
		return nil
	}

	eg := llerrgroup.New(20)
	for _, file := range filesToUpload {
		if eg.Stop() {
			break
		}

		file := file
		toBaseName := strings.TrimSuffix(filepath.Base(file), ".dat")

		eg.Go(func() error {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()

			if traceEnabled {
				s.logger.Debug("uploading file to storage", zap.String("local_file", file), zap.String("remove_base", toBaseName))
			}

			if err = s.Store.PushLocalFile(ctx, file, toBaseName); err != nil {
				return fmt.Errorf("moving file %q to storage: %w", file, err)
			}
			return nil
		})
	}

	return eg.Wait()
}

// Terminate assumes that no more 'StoreBlock' command is coming
func (s *BlockDataArchiver) Terminate() <-chan interface{} {
	ch := make(chan interface{})
	go func() {
		s.uploadFiles()
		close(ch)
	}()
	return ch
}

func (s *BlockDataArchiver) Init() error {
	if err := os.MkdirAll(s.workDir, 0755); err != nil {
		return fmt.Errorf("mkdir work folder: %w", err)
	}

	return nil
}

func findFilesToUpload(workingDirectory string, logger *zap.Logger, suffix string) (filesToUpload []string, err error) {
	err = filepath.Walk(workingDirectory, func(path string, info os.FileInfo, err error) error {
		if os.IsNotExist(err) {
			logger.Debug("skipping file that disappeared", zap.Error(err))
			return nil
		}
		if err != nil {
			return err
		}

		// clean up empty folders
		if info.IsDir() {
			if path == workingDirectory {
				return nil
			}
			// Prevents deleting folder that JUST got created and causing error on os.Open
			if isDirEmpty(path) && time.Since(info.ModTime()) > 60*time.Second {
				err := os.Remove(path)
				if err != nil {
					logger.Warn("cannot delete empty directory", zap.String("filename", path), zap.Error(err))
				}
			}
			return nil
		}

		if !strings.HasSuffix(path, suffix) {
			return nil
		}
		filesToUpload = append(filesToUpload, path)

		return nil
	})

	sort.Slice(filesToUpload, func(i, j int) bool { return filesToUpload[i] < filesToUpload[j] })
	return
}

func isDirEmpty(name string) bool {
	f, err := os.Open(name)
	if err != nil {
		return false
	}

	defer f.Close()
	_, err = f.Readdir(1)
	if err == io.EOF {
		return true
	}

	return false
}
