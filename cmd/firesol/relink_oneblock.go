package main

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/streamingfast/bstream"
	pbbstream "github.com/streamingfast/bstream/pb/sf/bstream/v1"
	"github.com/streamingfast/cli"
	"github.com/streamingfast/cli/sflags"
	"github.com/streamingfast/dstore"
	pbsol "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

func NewRelinkOneBlockCmd(logger *zap.Logger, tracer logging.Tracer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "relink-oneblock <one-block-file> <parent-num> <parent-id>",
		Short: "rewrite a one-block-file, changing its parent block number and hash (used to relink a chain of one-block-files)",
		Args:  cobra.ExactArgs(3),
		RunE:  getRelinkOneBlockRunner(logger, tweakBlockParent),
	}

	cmd.Flags().Bool("delete-original", false, "delete the original file when the rewritten file has a different name")

	return cmd
}

func NewRelinkAccountOneBlockCmd(logger *zap.Logger, tracer logging.Tracer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "relink-account-oneblock <one-block-file> <parent-num> <parent-id>",
		Short: "rewrite an account one-block-file (AccountBlock model), changing its parent block number and hash",
		Args:  cobra.ExactArgs(3),
		RunE:  getRelinkOneBlockRunner(logger, tweakAccountBlockParent),
	}

	cmd.Flags().Bool("delete-original", false, "delete the original file when the rewritten file has a different name")

	return cmd
}

// payloadTweaker keeps the solana payload's parent fields in sync with the bstream envelope.
type payloadTweaker func(block *pbbstream.Block, parentNum uint64, parentID string) error

func tweakBlockParent(block *pbbstream.Block, parentNum uint64, parentID string) error {
	b := &pbsol.Block{}
	if err := block.Payload.UnmarshalTo(b); err != nil {
		return fmt.Errorf("unmarshaling payload to pbsol.Block: %w", err)
	}
	b.ParentSlot = parentNum
	b.PreviousBlockhash = parentID
	if err := block.Payload.MarshalFrom(b); err != nil {
		return fmt.Errorf("marshaling pbsol.Block payload: %w", err)
	}
	return nil
}

func tweakAccountBlockParent(block *pbbstream.Block, parentNum uint64, parentID string) error {
	b := &pbsol.AccountBlock{}
	if err := block.Payload.UnmarshalTo(b); err != nil {
		return fmt.Errorf("unmarshaling payload to pbsol.AccountBlock: %w", err)
	}
	b.ParentSlot = parentNum
	b.ParentHash = parentID
	if err := block.Payload.MarshalFrom(b); err != nil {
		return fmt.Errorf("marshaling pbsol.AccountBlock payload: %w", err)
	}
	return nil
}

func getRelinkOneBlockRunner(logger *zap.Logger, tweak payloadTweaker) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		path := args[0]
		parentNum, err := strconv.ParseUint(args[1], 10, 64)
		if err != nil {
			return fmt.Errorf("parsing parent-num %q: %w", args[1], err)
		}
		parentID := args[2]

		// split on the last "/" manually to preserve URL schemes (gs://, s3://, ...)
		// which filepath.Dir would mangle into gs:/ ...
		slash := strings.LastIndex(path, "/")
		if slash < 0 {
			return fmt.Errorf("expected a path with a directory, got %q", path)
		}
		dir := path[:slash]
		base := path[slash+1:]
		name := strings.TrimSuffix(base, ".dbin.zst")
		if name == base {
			return fmt.Errorf("expected a .dbin.zst one-block-file, got %q", base)
		}

		store, err := dstore.NewDBinStore(dir)
		if err != nil {
			return fmt.Errorf("opening store at %q: %w", dir, err)
		}

		rc, err := store.OpenObject(ctx, name)
		if err != nil {
			return fmt.Errorf("opening %q: %w", path, err)
		}
		defer rc.Close()

		br, err := bstream.NewDBinBlockReader(rc)
		if err != nil {
			return fmt.Errorf("creating block reader: %w", err)
		}

		block, err := br.Read()
		if err != nil {
			return fmt.Errorf("reading block from %q: %w", path, err)
		}

		if _, err := br.Read(); !errors.Is(err, io.EOF) {
			return fmt.Errorf("file %q contains more than one block, not a one-block-file", path)
		}

		logger.Info("relinking one-block-file",
			zap.Uint64("block_num", block.Number),
			zap.String("block_id", block.Id),
			zap.Uint64("old_parent_num", block.ParentNum),
			zap.String("old_parent_id", block.ParentId),
			zap.Uint64("new_parent_num", parentNum),
			zap.String("new_parent_id", parentID),
		)

		oldParentNum := block.ParentNum
		oldParentID := block.ParentId

		block.ParentNum = parentNum
		block.ParentId = parentID
		if err := tweak(block, parentNum, parentID); err != nil {
			return err
		}

		newName := bstream.BlockFileName(block)

		targetURL := store.ObjectURL(newName)
		deleteOriginal := newName != name && sflags.MustGetBool(cmd, "delete-original")

		confirmation := fmt.Sprintf(`About to relink one-block-file:
  Writing to:    %s
  Block:         #%d %s
  Parent change: #%d %s -> #%d %s`,
			targetURL,
			block.Number, block.Id,
			oldParentNum, oldParentID,
			parentNum, parentID,
		)
		if deleteOriginal {
			confirmation += fmt.Sprintf("\n  Deleting original: %s", store.ObjectURL(name))
		}
		confirmation += "\n\nProceed?"

		if confirmed, wasAnswered := cli.AskConfirmation(confirmation); !confirmed || !wasAnswered {
			logger.Info("aborted by user")
			return nil
		}

		pr, pw := io.Pipe()
		go func() {
			var err error
			defer func() { pw.CloseWithError(err) }()

			bw, err := bstream.NewDBinBlockWriter(pw)
			if err != nil {
				return
			}
			err = bw.Write(block)
		}()

		if err := store.WriteObject(ctx, newName, pr); err != nil {
			return fmt.Errorf("writing %q: %w", newName, err)
		}

		logger.Info("wrote relinked one-block-file", zap.String("file", store.ObjectURL(newName)))

		if deleteOriginal {
			if err := store.DeleteObject(ctx, name); err != nil {
				return fmt.Errorf("deleting original %q: %w", path, err)
			}
			logger.Info("deleted original file", zap.String("file", path))
		}

		return nil
	}
}
