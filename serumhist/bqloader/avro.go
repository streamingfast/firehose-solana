package bqloader

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dfuse-io/dstore"
	"github.com/linkedin/goavro/v2"
	"go.uber.org/zap"
)

const (
	flushEventCount = 100000
)

type avroHandler struct {
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

	Store  dstore.Store
	Prefix string
}

// NewAvroHandler creates a new Avro event handler. The `scratchSpaceDir` is expected to be a local file system path.
func NewAvroHandler(scratchSpaceDir string, store dstore.Store, prefix string, codec *goavro.Codec) *avroHandler {
	scratchSpaceDir = filepath.Join(scratchSpaceDir, prefix)
	agg := &avroHandler{
		Store:           store,
		Prefix:          prefix,
		scratchSpaceDir: scratchSpaceDir,
		scratchFilename: filepath.Join(scratchSpaceDir, "pending.ocf"),
		codec:           codec,
	}

	return agg
}

func (h *avroHandler) Shutdown(ctx context.Context) error {
	return h.flush(ctx)
}

func (h *avroHandler) SetCheckpoint(slotNum uint64) {
	zlog.Debug("set checkpoint", zap.Uint64("checkpoint", slotNum))
	h.checkpointSlotNum = slotNum
}

func (h *avroHandler) HandleEvent(event map[string]interface{}, slotNum uint64, slotId string) error {
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

func (h *avroHandler) FlushIfNeeded(ctx context.Context) error {
	if time.Since(h.t0).Seconds() > 15*time.Minute.Seconds() || h.count > flushEventCount {
		return h.flush(ctx)
	}
	return nil
}

func (h *avroHandler) flush(ctx context.Context) error {
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

	destPath := NewFileName(
		h.Prefix,
		h.startSlotNum,
		h.latestSlotNum,
		h.startSlotId,
		h.latestSlotId,
		h.t0.Format("2006-01-02-15-04-05-")+fmt.Sprintf("%010d", rand.Int()),
	).String()

	zlog.Info("pushing avro file to storage", zap.String("path", destPath))

	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	err = h.Store.PushLocalFile(ctx, h.scratchFilename, destPath)
	if err != nil {
		return fmt.Errorf("failed pushing local file to storage: %w", err)
	}
	zlog.Info("done")

	h.ocfFile = nil
	h.ocfWriter = nil
	h.count = 0
	return nil
}

func (h *avroHandler) getOCFWriter(slotNum uint64, slotId string) (*goavro.OCFWriter, error) {
	h.lock.Lock()
	defer h.lock.Unlock()

	if h.ocfWriter != nil {
		return h.ocfWriter, nil
	}

	if h.ocfFile == nil {
		h.t0 = time.Now()
		h.startSlotId = slotId
		h.startSlotNum = slotNum

		zlog.Info("opening scratch ocf file", zap.String("filename", h.scratchFilename))

		err := os.MkdirAll(h.scratchSpaceDir, os.ModePerm)
		if err != nil {
			return nil, err
		}

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
