package pbserumhist

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

func (f *Fill) GetPrice() (uint64, error) {
	d, err := hex.DecodeString(f.OrderId)
	if err != nil {
		return 0, fmt.Errorf("unable to decode order ID: %w", err)
	}

	if len(d) < 8 {
		return 0, fmt.Errorf("order ID too short expecting atleast 8 bytes got %d", len(d))
	}

	return binary.BigEndian.Uint64(d[:8]), nil
}

func (f *Fill) GetSeqNum() (uint64, error) {
	d, err := hex.DecodeString(f.OrderId)
	if err != nil {
		return 0, fmt.Errorf("unable to decode order ID: %w", err)
	}

	if len(d) < 16 {
		return 0, fmt.Errorf("order ID too short expecting atleast 8 bytes got %d", len(d))
	}

	v := binary.BigEndian.Uint64(d[8:])

	if f.Side == Side_BID {
		return ^v, nil
	}

	return v, nil
}
