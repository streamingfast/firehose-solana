package resolvers

import (
	"time"

	"github.com/graph-gophers/graphql-go"
)

type DailyVolume struct {
	date  time.Time
	value float64
}

// Date represents the date at which time the volume as taken
func (v DailyVolume) Date() graphql.Time { return graphql.Time{Time: v.date} }

// Volume represents the value in unspecified unit
func (v DailyVolume) Value() Float64 { return Float64(v.value) }
