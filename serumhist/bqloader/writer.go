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
	"github.com/dfuse-io/shutter"
	"github.com/linkedin/goavro/v2"
)

type EventHandler struct {
	*shutter.Shutter

	lock sync.Mutex

	startBlockNum uint64

	store    dstore.Store
	storeUrl string
	bqloader *BigQueryLoader
	prefix   string

	bufferFileDir  string
	bufferFile     *os.File
	bufferFileName string
	bufferedWriter *goavro.OCFWriter

	startTime     time.Time
	count         int
	startSlotNum  uint64
	startSlotId   string
	latestSlotNum uint64
	latestSlotId  string
}

func NewEventHandler(startBlockNum uint64, storeUrl string, store dstore.Store, prefix string, bqloader *BigQueryLoader, scratchSpaceDir string) *EventHandler {
	return &EventHandler{
		startBlockNum: startBlockNum,
		store:         store,
		storeUrl:      storeUrl,
		bqloader:      bqloader,
		prefix:        prefix,
		bufferFileDir: scratchSpaceDir,
	}
}

func (h *EventHandler) HandleEvent(event Encoder, slotNum uint64, slotId string) error {
	if slotNum < h.startBlockNum {
		return nil
	}

	h.count++
	h.latestSlotNum = slotNum
	h.latestSlotId = slotId

	var err error
	bw, err := h.getBufferedWriter(event.Codec(), slotNum, slotId)
	if err != nil {
		return err
	}

	err = bw.Append([]interface{}{event.Encode()})
	if err != nil {
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
			h.prefix,
			h.startSlotNum,
			h.latestSlotNum,
			h.startSlotId,
			h.latestSlotId,
			h.startTime.Format("2006-01-02-15-04-05-")+fmt.Sprintf("%010d", rand.Int()),
		).String()

		err := h.store.PushLocalFile(ctx, h.bufferFileName, destPath)
		if err != nil {
			return fmt.Errorf("failed pushing local file to store: %w", err)
		}

		table := h.prefix
		uri := strings.Join([]string{h.storeUrl, fmt.Sprintf("%s.avro", destPath)}, "/")
		h.bqloader.SubmitJob(ctx, table, uri)

		return nil
	}

	err := derr.RetryContext(ctx, 5, uploadFunc)
	if err != nil {
		return fmt.Errorf("could not upload file to storage: %w", err)
	}

	h.bufferFile = nil
	h.bufferedWriter = nil
	h.count = 0

	return nil
}

func (h *EventHandler) getBufferedWriter(codec *goavro.Codec, slotNum uint64, slotId string) (*goavro.OCFWriter, error) {
	h.lock.Lock()
	defer h.lock.Unlock()

	if h.bufferedWriter != nil {
		return h.bufferedWriter, nil
	}

	if h.bufferFile != nil {
		return h.bufferedWriter, nil
	}

	h.startTime = time.Now()
	h.startSlotNum = slotNum
	h.startSlotId = slotId

	err := os.MkdirAll(h.bufferFileDir, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("could not create buffer file directory: %w", err)
	}

	h.bufferFileName = filepath.Join(h.bufferFileDir, fmt.Sprintf("pending-%010d.ocf", rand.Int()))

	h.bufferFile, err = os.OpenFile(h.bufferFileName, os.O_RDWR|os.O_CREATE, 0644)
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
