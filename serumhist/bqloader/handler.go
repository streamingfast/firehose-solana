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
	"github.com/dfuse-io/dstore"
	"github.com/linkedin/goavro/v2"
	"go.uber.org/zap"
)

type eventHandler struct {
	scratchSpaceDir string
	scratchFilename string

	codec     *goavro.Codec
	ocfFile   *os.File
	ocfWriter *goavro.OCFWriter

	checkpointSlotNum uint64

	t0            time.Time
	count         uint64
	startSlotNum  uint64
	latestSlotNum uint64
	startSlotId   string
	latestSlotId  string
	lock          sync.Mutex

	Store     dstore.Store
	StoreURL  string
	Dataset   *bigquery.Dataset
	TableName string
}

// newEventHandler creates a new event handler which writes Avro files to its Google Cloud Storage and then loads that file into BigQuery. The `scratchSpaceDir` is expected to be a local file system path.
func newEventHandler(scratchSpaceDir string, dataset *bigquery.Dataset, storeUrl string, store dstore.Store, tableName string, codec *goavro.Codec) *eventHandler {
	return &eventHandler{
		Store:           store,
		StoreURL:        storeUrl,
		Dataset:         dataset,
		TableName:       tableName,
		scratchSpaceDir: filepath.Join(scratchSpaceDir, tableName),
		codec:           codec,
	}
}

func (h *eventHandler) Shutdown(ctx context.Context) error {
	return h.flush(ctx)
}

func (h *eventHandler) SetCheckpoint(slotNum uint64) {
	zlog.Debug("set checkpoint", zap.Uint64("checkpoint", slotNum))
	h.checkpointSlotNum = slotNum
}

func (h *eventHandler) HandleEvent(event map[string]interface{}, slotNum uint64, slotId string) error {
	if slotNum <= h.checkpointSlotNum {
		zlog.Debug("ignoring event from before our checkpoint", zap.Uint64("slot_num", slotNum), zap.Uint64("checkpoint", h.checkpointSlotNum))
		return nil
	}

	h.count++
	h.latestSlotNum = slotNum
	h.latestSlotId = slotId

	var err error
	w, err := h.getOCFWriter(slotNum, slotId)
	if err != nil {
		return fmt.Errorf("failed to get writer: %w", err)
	}

	err = w.Append([]interface{}{event})
	if err != nil {
		return fmt.Errorf("failed writing to scratch file: %w", err)
	}

	return nil
}

func (h *eventHandler) FlushIfNeeded(ctx context.Context) error {
	if time.Since(h.t0).Seconds() > 15*time.Minute.Seconds() {
		return h.flush(ctx)
	}
	return nil
}

func (h *eventHandler) flush(ctx context.Context) error {
	if h.ocfWriter == nil || h.ocfFile == nil {
		//nothing to flush
		return nil
	}

	h.lock.Lock()
	defer h.lock.Unlock()

	zlog.Info("processed message batch", zap.Uint64("count", h.count), zap.Duration("timing_secs", time.Since(h.t0)/time.Second))

	err := h.ocfFile.Close()
	if err != nil {
		return fmt.Errorf("failed to close scratch file: %w", err)
	}

	defer func() {
		h.ocfFile = nil
		h.ocfWriter = nil
		h.count = 0
	}() // cleanup

	destPath := NewFileName(
		h.TableName,
		h.startSlotNum,
		h.latestSlotNum,
		h.startSlotId,
		h.latestSlotId,
		h.t0.Format("2006-01-02-15-04-05-")+fmt.Sprintf("%010d", rand.Int()),
	).String()

	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	zlog.Info("pushing avro file to store", zap.String("path", destPath))
	err = h.Store.PushLocalFile(ctx, h.scratchFilename, destPath)
	if err != nil {
		return fmt.Errorf("failed pushing local file to store: %w", err)
	}
	zlog.Info("done uploading file to store")

	// launch job to import GCS file
	uri := strings.Join([]string{h.StoreURL, fmt.Sprintf("%s.avro", destPath)}, "/")
	ref := bigquery.NewGCSReference(uri)
	ref.SourceFormat = bigquery.Avro

	loader := h.Dataset.Table(h.TableName).LoaderFrom(ref)
	loader.UseAvroLogicalTypes = true

	zlog.Info("loading file into bigquery", zap.String("file", uri))
	var loadError bool
	defer func() {
		if loadError {
			err := h.Store.DeleteObject(ctx, destPath)
			if err != nil {
				zlog.Error("could not delete store object after bq load job failure", zap.Error(err), zap.String("store_object", destPath))
			}
		}
	}()

	job, err := loader.Run(ctx)
	if err != nil {
		loadError = true
		return fmt.Errorf("could not run loader for table %s: %w", h.TableName, err)
	}
	js, err := job.Wait(ctx)
	if err != nil {
		loadError = true
		return fmt.Errorf("could not create loader for table %s: %w", h.TableName, err)
	}
	if js.Err() != nil {
		loadError = true
		return fmt.Errorf("error while running loader for table %s: %w", h.TableName, js.Err())
	}
	zlog.Info("done loading file to into bigquery", zap.String("file", uri))

	return nil
}

func (h *eventHandler) getOCFWriter(slotNum uint64, slotId string) (*goavro.OCFWriter, error) {
	h.lock.Lock()
	defer h.lock.Unlock()

	if h.ocfWriter != nil {
		return h.ocfWriter, nil
	}

	if h.ocfFile == nil {
		h.t0 = time.Now()
		h.startSlotId = slotId
		h.startSlotNum = slotNum

		err := os.MkdirAll(h.scratchSpaceDir, os.ModePerm)
		if err != nil {
			return nil, err
		}

		h.scratchFilename = filepath.Join(h.scratchSpaceDir, fmt.Sprintf("pending-%010d.ocf", rand.Int()))
		zlog.Info("opening scratch ocf file", zap.String("filename", h.scratchFilename))

		h.ocfFile, err = os.OpenFile(h.scratchFilename, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			return nil, err
		}

		h.ocfWriter, err = goavro.NewOCFWriter(goavro.OCFConfig{
			W:               h.ocfFile,
			Codec:           h.codec,
			CompressionName: goavro.CompressionSnappyLabel,
		})

		if err != nil {
			return nil, fmt.Errorf("creating ocf writer: %w", err)
		}
	}

	return h.ocfWriter, nil
}
