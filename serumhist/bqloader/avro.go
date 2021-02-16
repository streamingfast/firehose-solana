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

	t0            time.Time
	count         int
	startSlotNum  *uint64
	latestSlotNum *uint64
	startSlotId   *string
	latestSlotId  *string
	lock          sync.Mutex

	store dstore.Store
}

// NewAvroHandler creates a new Avro event handler. The `scratchSpaceDir` is expected to be a local file system path.
func NewAvroHandler(scratchSpaceDir, scratchSpaceFile string, store dstore.Store, codec *goavro.Codec) *avroHandler {
	if !strings.HasSuffix(scratchSpaceFile, ".ocf") {
		scratchSpaceFile += ".ocf"
	}

	agg := &avroHandler{
		scratchSpaceDir: scratchSpaceDir,
		scratchFilename: filepath.Join(scratchSpaceDir, scratchSpaceFile),
		store:           store,
		codec:           codec,
	}

	return agg
}

func (agg *avroHandler) Shutdown(ctx context.Context) error {
	return agg.flushFile(ctx)
}

func (agg *avroHandler) getOCFWriter(slotNum uint64, slotId string) (*goavro.OCFWriter, error) {
	agg.lock.Lock()
	defer agg.lock.Unlock()

	if agg.ocfWriter != nil {
		return agg.ocfWriter, nil
	}

	if agg.ocfFile == nil {
		agg.t0 = time.Now()
		*agg.startSlotId = slotId
		*agg.startSlotNum = slotNum

		zlog.Info("opening scratch ocf file", zap.String("filename", agg.scratchFilename))

		err := os.MkdirAll(agg.scratchSpaceDir, os.ModePerm)
		if err != nil {
			return nil, err
		}

		agg.ocfFile, err = os.OpenFile(agg.scratchFilename, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			return nil, err
		}

		agg.ocfWriter, err = goavro.NewOCFWriter(goavro.OCFConfig{
			W:               agg.ocfFile,
			Codec:           agg.codec,
			CompressionName: goavro.CompressionSnappyLabel,
		})

		if err != nil {
			return nil, fmt.Errorf("creating ocf writer: %w", err)
		}
	}

	return agg.ocfWriter, nil
}

func (agg *avroHandler) handleEvent(event map[string]interface{}, slotNum uint64, slotId string) error {
	agg.count++
	*agg.startSlotNum = slotNum
	*agg.latestSlotId = slotId

	var err error
	w, err := agg.getOCFWriter(slotNum, slotId)
	if err != nil {
		return fmt.Errorf("failed to get writer: %w", err)
	}

	err = w.Append([]interface{}{event})
	if err != nil {
		return fmt.Errorf("failed writing to scratch file: %w", err)
	}

	return nil
}

func (agg *avroHandler) flushFile(ctx context.Context) error {
	agg.lock.Lock()
	defer agg.lock.Unlock()

	zlog.Info("processed message batch", zap.Int("count", agg.count), zap.Duration("timing_secs", time.Since(agg.t0)/time.Second))

	err := agg.ocfFile.Close()
	derr.Check("failed closing to scratch file", err)

	destPath := fmt.Sprintf("%d-%d-%s-%s-%s.avro",
		*agg.startSlotNum,
		*agg.latestSlotNum,
		*agg.startSlotId,
		*agg.latestSlotId,
		agg.t0.Format("2006-01-02-15-04-05-")+fmt.Sprintf("%010d", rand.Int()),
	)

	zlog.Info("pushing avro file to storage", zap.String("path", destPath))
	err = agg.store.PushLocalFile(ctx, agg.scratchFilename, destPath)
	if err != nil {
		return fmt.Errorf("failed pushing local file to storage: %w", err)
	}
	zlog.Info("done")

	agg.ocfFile = nil
	agg.ocfWriter = nil
	agg.startSlotNum = nil
	agg.startSlotId = nil
	return nil
}
