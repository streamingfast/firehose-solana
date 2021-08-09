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

type Bootstrapper struct {
	dataURL     string
	dataDirPath string
	logger      *zap.Logger
}

func NewBootstrapper(dataURL string, dataDirPath string, logger *zap.Logger) *Bootstrapper {
	return &Bootstrapper{
		dataURL:     dataURL,
		dataDirPath: dataDirPath,
		logger:      logger,
	}
}

func (b *Bootstrapper) Bootstrap() error {
	b.logger.Info("bootstrapping solana chain data from pre-built data",
		zap.String("bootstrap_data_url", b.dataURL),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	reader, _, _, err := dstore.OpenObject(ctx, b.dataURL, dstore.Compression("zstd"))
	if err != nil {
		return fmt.Errorf("cannot get snapshot from gstore: %w", err)
	}
	defer reader.Close()

	b.createChainData(reader)
	return nil

}
func (b *Bootstrapper) createChainData(reader io.Reader) error {
	err := os.MkdirAll(b.dataDirPath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to create blocks log file: %w", err)
	}

	b.logger.Info("extracting bootstrapping data into node data directory", zap.String("data_dir", b.dataDirPath))
	tr := tar.NewReader(reader)
	for {
		header, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				return nil
			}

			return err
		}

		path := filepath.Join(b.dataDirPath, header.Name)
		b.logger.Debug("about to write content of entry", zap.String("name", header.Name), zap.String("path", path), zap.Bool("is_dir", header.FileInfo().IsDir()))
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
