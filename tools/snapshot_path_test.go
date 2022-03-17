package tools

import (
	"github.com/test-go/testify/assert"
	"testing"
)

func TestSnapshotPath(t *testing.T) {

	tests := []struct {
		name   string
		input  []*Snapshot
		expect []*Snapshot
	}{
		{
			name: "Last Valid",
			input: []*Snapshot{
				{SnapshotPath: "A", StartSlot: 1, EndSlot: 4},
				{SnapshotPath: "B", StartSlot: 2, EndSlot: 6},
				{SnapshotPath: "C", StartSlot: 2, EndSlot: 10},
				{SnapshotPath: "D", StartSlot: 3, EndSlot: 7},
				{SnapshotPath: "E", StartSlot: 6, EndSlot: 15},
			},
			expect: []*Snapshot{
				{SnapshotPath: "A", StartSlot: 1, EndSlot: 4},
				{SnapshotPath: "C", StartSlot: 2, EndSlot: 10},
				{SnapshotPath: "E", StartSlot: 6, EndSlot: 15},
			},
		},
		{
			name: "Last ! Valid",
			input: []*Snapshot{
				{SnapshotPath: "A", StartSlot: 1, EndSlot: 4},
				{SnapshotPath: "B", StartSlot: 2, EndSlot: 6},
				{SnapshotPath: "C", StartSlot: 2, EndSlot: 10},
				{SnapshotPath: "D", StartSlot: 3, EndSlot: 7},
				{SnapshotPath: "E", StartSlot: 6, EndSlot: 15},
				{SnapshotPath: "F", StartSlot: 5, EndSlot: 12},
			},
			expect: []*Snapshot{
				{SnapshotPath: "A", StartSlot: 1, EndSlot: 4},
				{SnapshotPath: "C", StartSlot: 2, EndSlot: 10},
				{SnapshotPath: "E", StartSlot: 6, EndSlot: 15},
			},
		},
		{
			name: "Hole",
			input: []*Snapshot{
				{SnapshotPath: "A", StartSlot: 1, EndSlot: 4},
				{SnapshotPath: "B", StartSlot: 2, EndSlot: 6},
				{SnapshotPath: "C", StartSlot: 2, EndSlot: 10},
				{SnapshotPath: "D", StartSlot: 3, EndSlot: 7},
				{SnapshotPath: "E", StartSlot: 12, EndSlot: 14},
			},
			expect: []*Snapshot{
				{SnapshotPath: "A", StartSlot: 1, EndSlot: 4},
				{SnapshotPath: "C", StartSlot: 2, EndSlot: 10},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			out := shortestPath(test.input)
			assert.Equal(t, test.expect, out)
		})
	}

}
