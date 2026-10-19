package pbsol

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBlock_GetFirehoseBlockTime(t *testing.T) {
	tests := []struct {
		name  string
		block *Block
		want  time.Time
	}{
		{
			name:  "no time",
			block: &Block{},
			want:  time.Unix(0, 0),
		},
		{
			name:  "block time only",
			block: &Block{BlockTime: &UnixTimestamp{Timestamp: 1760000000}},
			want:  time.Unix(1760000000, 0),
		},
		{
			name:  "block producer time takes precedence",
			block: &Block{BlockTime: &UnixTimestamp{Timestamp: 1760000000}, BlockProducerTimeNanos: 1760000000123456789},
			want:  time.Unix(1760000000, 123456789),
		},
		{
			name:  "block producer time without block time",
			block: &Block{BlockProducerTimeNanos: 1760000000123456789},
			want:  time.Unix(1760000000, 123456789),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, tt.want.Equal(tt.block.GetFirehoseBlockTime()), "got %s, want %s", tt.block.GetFirehoseBlockTime(), tt.want)
		})
	}
}
