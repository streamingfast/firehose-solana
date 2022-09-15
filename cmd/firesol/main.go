package main

import (
	"github.com/streamingfast/firehose-solana/cmd/firesol/cli"
)

// Commit sha1 value, injected via go build `ldflags` at build time
var Commit = ""

// Version value, injected via go build `ldflags` at build time
var Version = "dev"

// IsDirty value, injected via go build `ldflags` at build time
var IsDirty = ""

func init() {
	cli.RootCmd.Version = version()
}

func main() {
	cli.Main()
}

func version() string {
	shortCommit := Commit
	if len(shortCommit) >= 7 {
		shortCommit = shortCommit[0:7]
	}

	if len(shortCommit) == 0 {
		shortCommit = "adhoc"
	}

	out := Version + "-" + shortCommit
	if IsDirty != "" {
		out += "-dirty"
	}

	return out
}
