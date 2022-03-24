package tools

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"os"
	"sort"
)

var pathSnapshotCmd = &cobra.Command{
	Use: "path {csv_file} {cvs_file}",
	Short: `Prints to a CSV file the shortest path for full block history
`,
	Args: cobra.ExactArgs(2),
	RunE: pathSnapshotRunE,
}

func init() {
	snapshotCmd.AddCommand(pathSnapshotCmd)
}

func pathSnapshotRunE(cmd *cobra.Command, args []string) error {
	f, err := os.Open(args[0])
	if err != nil {
		return fmt.Errorf("unable to open file %q: %w", args[0], err)
	}
	defer f.Close()

	reader := bufio.NewReader(f)

	snapshots := []*Snapshot{}

	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("unable to read line: %w", err)
		}

		snapshotObj := &Snapshot{}
		err = json.Unmarshal(line, snapshotObj)
		if err != nil {
			return fmt.Errorf("unable unmarshal open object: %w", err)
		}
		snapshots = append(snapshots, snapshotObj)
	}
	sort.SliceStable(snapshots, func(i, j int) bool {
		return snapshots[i].StartSlot < snapshots[j].StartSlot
	})
	out := shortestPath(snapshots)

	file, err := os.Create(args[1])
	if err != nil {
		return fmt.Errorf("failed to open a file: %w", err)
	}
	defer file.Close()

	var csvwriter *csv.Writer

	csvwriter = csv.NewWriter(file)
	csvwriter.Write([]string{"slot", "bucket", "has bounds", "slot count", "start slot", "end slot", "version", "rocksdb path", "rocksdb compression", "snapshot path"})
	defer csvwriter.Flush()

	var lastSnapshot *Snapshot
	versions := map[string]bool{}
	holeCount := 0
	for _, snapshot := range out {
		err = csvwriter.Write([]string{
			fmt.Sprintf("%d", snapshot.Slot),
			snapshot.Bucket,
			fmt.Sprintf("%t", snapshot.HasBound),
			fmt.Sprintf("%d", snapshot.SlotCount),
			fmt.Sprintf("%d", snapshot.StartSlot),
			fmt.Sprintf("%d", snapshot.EndSlot),
			snapshot.Version,
			snapshot.RocksDBPath,
			fmt.Sprintf("%t", snapshot.RocksDBCompressed),
			snapshot.SnapshotPath,
		})
		if err != nil {
			return fmt.Errorf("failed to write to csv: %w", err)
		}

		isContiguous := true
		overlap := 0
		if lastSnapshot != nil {
			if snapshot.StartSlot > lastSnapshot.EndSlot {
				isContiguous = false
				holeCount++
			}
			overlap = (int(lastSnapshot.EndSlot) - int(snapshot.StartSlot))
		}
		prefix := "✅"
		if !isContiguous {
			prefix = "🆘"
		}
		fmt.Printf("%s Snapshot %d - %d (%s | %d count | %d overlap)\n", prefix, snapshot.StartSlot, snapshot.EndSlot, snapshot.Version, snapshot.SlotCount, overlap)
		lastSnapshot = snapshot
		versions[snapshot.Version] = true
	}

	fmt.Printf("Snapshot Count: %d\n", len(out))
	fmt.Printf("Hole Count: %d\n", holeCount)
	fmt.Printf("Unique Version Count: %d\n", len(versions))
	return nil
}

func shortestPath(array []*Snapshot) (out []*Snapshot) {

	var curBestCandidate *Snapshot
	var lastValidSnapshot *Snapshot
	for _, snapshot := range array {
		if snapshot.StartSlot == 0 {
			continue
		}

		if lastValidSnapshot == nil {
			lastValidSnapshot = snapshot
			out = append(out, lastValidSnapshot)
			continue
		}

		if snapshot.StartSlot > lastValidSnapshot.EndSlot {
			if curBestCandidate != nil {
				lastValidSnapshot = curBestCandidate
				out = append(out, lastValidSnapshot)
				curBestCandidate = nil
			} else {
				lastValidSnapshot = snapshot
				out = append(out, lastValidSnapshot)
				curBestCandidate = nil
			}
			//if (snapshot != nil) && (curBestCandidate != nil) && (snapshot.StartSlot > curBestCandidate.EndSlot) {
			//	curBestCandidate = snapshot
			//}
		}

		if snapshot.StartSlot < lastValidSnapshot.EndSlot {
			if (curBestCandidate == nil) || (curBestCandidate != nil && snapshot.EndSlot > curBestCandidate.EndSlot) {
				curBestCandidate = snapshot
			}
		}
	}

	if curBestCandidate != nil {
		out = append(out, curBestCandidate)
	}

	return out
}
