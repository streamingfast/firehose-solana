package bqloader

import (
	"fmt"
	"strconv"
	"strings"
)

type FileName struct {
	Prefix             string
	StartSlotNum       uint64
	LatestSlotNum      uint64
	StartSlotId        string
	LatestSlotId       string
	FormattedTimestamp string
}

func NewFileName(prefix string, startSlotNum uint64, latestSlotNum uint64, startSlotId, latestSlotId string, formattedTimeStamp string) *FileName {
	return &FileName{
		Prefix:             prefix,
		StartSlotNum:       startSlotNum,
		LatestSlotNum:      latestSlotNum,
		StartSlotId:        startSlotId,
		LatestSlotId:       latestSlotId,
		FormattedTimestamp: formattedTimeStamp,
	}
}

func (f *FileName) String() string {
	return fmt.Sprintf("%s/%d-%d-%s-%s-%s",
		f.Prefix,
		f.StartSlotNum,
		f.LatestSlotNum,
		f.StartSlotId,
		f.LatestSlotId,
		f.FormattedTimestamp,
	)
}

func parseLatestInfoFromFilename(filepath string) (*FileName, error) {
	pathParts := strings.Split(filepath, "/")
	filename := pathParts[len(pathParts)-1]

	filenameParts := strings.SplitN(filename, "-", 5)
	if len(filenameParts) < 5 {
		return nil, fmt.Errorf("could not parse filename. invalid format")
	}

	fileLatestSlotNum, err := strconv.ParseUint(filenameParts[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("could not parse start slot number: %w", err)
	}

	fileLatestSlotId := filenameParts[3]

	return &FileName{
		LatestSlotNum: fileLatestSlotNum,
		LatestSlotId:  fileLatestSlotId,
	}, nil
}
