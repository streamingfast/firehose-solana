package bqloader

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/linkedin/goavro/v2"
	"github.com/streamingfast/derr"
	"github.com/streamingfast/dstore"
	"github.com/streamingfast/shutter"
	"go.uber.org/zap"
)

type EventHandler struct {
	*shutter.Shutter

	lock sync.Mutex

	dataset    *bigquery.Dataset
	store      dstore.Store
	storeUrl   string
	bqinjector *BigQueryInjector
	table      Table

	bufferFileDir  string
	bufferFile     *os.File
	bufferFileName string
	bufferedWriter *goavro.OCFWriter

	startTime      time.Time
	count          int
	startBlockNum  uint64
	startBlockId   []byte
	latestBlockNum uint64
	latestBlockId  []byte
}

func NewEventHandler(startBlockNum uint64, storeUrl string, store dstore.Store, dataset *bigquery.Dataset, table Table, bqloader *BigQueryInjector, scratchSpaceDir string) *EventHandler {
	h := &EventHandler{
		Shutter:       shutter.New(),
		startBlockNum: startBlockNum,
		store:         store,
		storeUrl:      storeUrl,
		bqinjector:    bqloader,
		table:         table,
		bufferFileDir: scratchSpaceDir,
		dataset:       dataset,
	}

	h.OnTerminating(func(err error) {
		h.lock.Lock()
		defer h.lock.Unlock()

		if h.bufferedWriter != nil || h.bufferFile != nil {
			_ = os.Remove(h.bufferFileName)
		}
		h.bufferFile = nil
		h.bufferedWriter = nil
	})

	return h
}

func (h *EventHandler) HandleEvent(event Encoder, blockNum uint64, blockId []byte) error {
	event.Log()

	if blockNum < h.startBlockNum {
		return nil
	}

	h.count++
	h.latestBlockNum = blockNum
	h.latestBlockId = blockId

	var err error
	bw, err := h.getBufferedWriter(blockNum, blockId)
	if err != nil {
		return err
	}

	if err = bw.Append([]interface{}{event.Encode()}); err != nil {
		return fmt.Errorf("failed writing to buffer file: %w", err)
	}

	return nil
}

func (h *EventHandler) Flush(ctx context.Context) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	if time.Since(h.startTime).Seconds() < 15*time.Minute.Seconds() {
		return nil
	}

	if h.bufferedWriter == nil || h.bufferFile == nil {
		return nil
	}

	uploadFunc := func(ctx context.Context) error {
		destPath := NewFileName(
			string(h.table),
			h.startBlockNum,
			h.latestBlockNum,
			h.startBlockId,
			h.latestBlockId,
		).String()

		err := h.store.PushLocalFile(ctx, h.bufferFileName, destPath)
		if err != nil {
			return fmt.Errorf("failed pushing local file to store: %w", err)
		}

		tableName := string(h.table)
		uri := strings.Join([]string{h.storeUrl, fmt.Sprintf("%s.avro", destPath)}, "/")

		format := bigquery.Avro
		h.bqinjector.SubmitJob(uri, tableName, h.dataset, format, func(ctx context.Context) error {
			//checkpoint save callback
			jobStatusRow := ProcessFile{
				Table:         tableName,
				Filename:      uri,
				StartBlockNum: int64(h.startBlockNum),
				StartBlockID:  h.startBlockId,
				EndBlockNum:   int64(h.latestBlockNum),
				EndBlockID:    h.latestBlockId,
				Time:          time.Now(),
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			err = h.dataset.Table(tableProcessedFiles.String()).Inserter().Put(ctx, jobStatusRow)
			cancel()

			if err != nil {
				zlog.Error("could not write checkpoint", zap.Stringer("table", tableProcessedFiles), zap.Error(err))
				return err
			}

			zlog.Info("checkpoint written", zap.Stringer("checkpoints_table", tableProcessedFiles), zap.Stringer("table", h.table))
			return nil
		})

		zlog.Debug("flushed file to store", zap.String("local_file", h.bufferFileName), zap.String("store_file", uri))
		return nil
	}

	err := derr.RetryContext(ctx, 5, uploadFunc)
	if err != nil {
		return fmt.Errorf("could not upload file to storage: %w", err)
	}

	h.bufferFile = nil
	h.bufferFileName = ""
	h.bufferedWriter = nil
	h.count = 0

	return nil
}

func (h *EventHandler) getBufferedWriter(blockNum uint64, blockId []byte) (*goavro.OCFWriter, error) {
	h.lock.Lock()
	defer h.lock.Unlock()

	if h.bufferedWriter != nil {
		return h.bufferedWriter, nil
	}

	if h.bufferFile != nil {
		return h.bufferedWriter, nil
	}

	h.startTime = time.Now()
	h.startBlockNum = blockNum
	h.startBlockId = blockId

	err := os.MkdirAll(h.bufferFileDir, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("could not create buffer file directory: %w", err)
	}

	h.bufferFileName = filepath.Join(h.bufferFileDir, fmt.Sprintf("pending-%s-%010d.ocf", string(h.table), rand.Int()))
	h.bufferFile, err = os.OpenFile(h.bufferFileName, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("unable to open local avro file: %w", err)
	}

	codec, err := h.table.Codec()
	if err != nil {
		return nil, err
	}

	h.bufferedWriter, err = goavro.NewOCFWriter(goavro.OCFConfig{
		W:               h.bufferFile,
		Codec:           codec,
		CompressionName: goavro.CompressionSnappyLabel,
	})

	if err != nil {
		return nil, fmt.Errorf("could not create ocf writer: %w", err)
	}

	return h.bufferedWriter, nil
}
