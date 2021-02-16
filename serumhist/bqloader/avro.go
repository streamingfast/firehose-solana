package bqloader

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dstore"
	"github.com/linkedin/goavro/v2"
	"go.uber.org/zap"
)

type avroHandler struct {
	scratchSpaceDir string
	scratchFilename string

	codec     *goavro.Codec
	ocfFile   *os.File
	ocfWriter *goavro.OCFWriter

	CheckpointSlotNum uint64

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
func NewAvroHandler(scratchSpaceDir, scratchSpaceFile string, store dstore.Store, prefix string, codec *goavro.Codec) *avroHandler {
	if !strings.HasSuffix(scratchSpaceFile, ".ocf") {
		scratchSpaceFile += ".ocf"
	}

	agg := &avroHandler{
		Store:           store,
		Prefix:          prefix,
		scratchSpaceDir: scratchSpaceDir,
		scratchFilename: filepath.Join(scratchSpaceDir, scratchSpaceFile),
		codec:           codec,
	}

	return agg
}

func (h *avroHandler) Shutdown(ctx context.Context) error {
	return h.flush(ctx)
}

func (h *avroHandler) HandleEvent(event map[string]interface{}, slotNum uint64, slotId string) error {
	if slotNum <= h.CheckpointSlotNum {
		zlog.Debug("")
		return nil
	}

	atomic.AddUint64(&h.count, 1)
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
	if time.Since(h.t0).Seconds() > 15 * 60 || atomic.LoadUint64(&h.count) > 1000000 {
		return h.flush(ctx)
	}
	return nil
}

func (h *avroHandler) flush(ctx context.Context) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	if h.ocfWriter != nil || h.ocfFile == nil {
		//nothing to flush
		return nil
	}

	zlog.Info("processed message batch", zap.Uint64("count", atomic.LoadUint64(&h.count)), zap.Duration("timing_secs", time.Since(h.t0)/time.Second))

	err := h.ocfFile.Close()
	derr.Check("failed to close scratch file", err)

	destPath := fmt.Sprintf("%s/%d-%d-%s-%s-%s.avro",
		h.Prefix,
		h.startSlotNum,
		h.latestSlotNum,
		h.startSlotId,
		h.latestSlotId,
		h.t0.Format("2006-01-02-15-04-05-")+fmt.Sprintf("%010d", rand.Int()),
	)

	zlog.Info("pushing avro file to storage", zap.String("path", destPath))
	err = h.Store.PushLocalFile(ctx, h.scratchFilename, destPath)
	if err != nil {
		return fmt.Errorf("failed pushing local file to storage: %w", err)
	}
	zlog.Info("done")

	h.ocfFile = nil
	h.ocfWriter = nil
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
