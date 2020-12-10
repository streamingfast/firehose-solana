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
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/dfuse-io/dstore"
	"go.uber.org/zap"
)

func (s *Superviser) Bootstrap(bootstrapDataURL string) error {
	s.logger.Info("bootstrapping geth chain data from pre-built data", zap.String("bootstrap_data_url", bootstrapDataURL))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	reader, _, _, err := dstore.OpenObject(ctx, bootstrapDataURL, dstore.Compression("zstd"))
	if err != nil {
		return fmt.Errorf("cannot get snapshot from gstore: %w", err)
	}
	defer reader.Close()

	s.createChainData(reader)
	return nil
}

func (s *Superviser) createChainData(reader io.Reader) error {
	err := os.MkdirAll(s.options.DataDirPath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to create blocks log file: %w", err)
	}

	s.logger.Info("extracting bootstrapping data into node data directory", zap.String("data_dir", s.options.DataDirPath))
	tr := tar.NewReader(reader)
	for {
		header, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				return nil
			}

			return err
		}

		path := filepath.Join(s.options.DataDirPath, header.Name)
		s.logger.Debug("about to write content of entry", zap.String("name", header.Name), zap.String("path", path), zap.Bool("is_dir", header.FileInfo().IsDir()))
		if header.FileInfo().IsDir() {
			err = os.MkdirAll(path, os.ModePerm)
			if err != nil {
				return fmt.Errorf("unable to create directory: %w", err)
			}

			continue
		}

		file, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("unable to create file: %w", err)
		}

		if _, err := io.Copy(file, tr); err != nil {
			file.Close()
			return err
		}
		file.Close()
	}
}
