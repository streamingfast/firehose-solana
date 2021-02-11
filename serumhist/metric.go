package serumhist

import (
	"time"

	"go.uber.org/zap"
)

type slotMetrics struct {
	startTime      time.Time
	slotCount      uint64
	serumFillCount int
}

func (m slotMetrics) dump() (out []zap.Field) {
	out = append(out, []zap.Field{
		zap.Duration("pipeline_start_time", time.Since(m.startTime)),
		zap.Float64("slot_rate", float64(m.slotCount)/(float64(time.Since(m.startTime))/float64(time.Second))),
		zap.Int("serum_fill_count", m.serumFillCount),
	}...)

	return out

}
