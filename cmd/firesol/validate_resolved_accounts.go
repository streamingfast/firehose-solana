package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	solana "github.com/gagliardetto/solana-go"
	addresslookuptable "github.com/gagliardetto/solana-go/programs/address-lookup-table"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/spf13/cobra"
	"github.com/streamingfast/cli/sflags"
	firecore "github.com/streamingfast/firehose-core"
	accountsresolver "github.com/streamingfast/firehose-solana/accountresolver"
	pbsolv1 "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
	kvstore "github.com/streamingfast/kvdb/store"
	_ "github.com/streamingfast/kvdb/store/bigkv"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

func newValidateResolvedAddresses(logger *zap.Logger, tracer logging.Tracer, chain *firecore.Chain[*pbsolv1.Block]) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate-resolve-addresses {key} {kv-dsn}",
		Short: "",
		RunE:  processValidateResolvedAddressesE(chain, logger, tracer),
		Args:  cobra.ExactArgs(2),
	}

	cmd.Flags().String("rpc-endpoint", "", "Pass in your RPC endpoint")

	return cmd
}

func processValidateResolvedAddressesE(chain *firecore.Chain[*pbsolv1.Block], logger *zap.Logger, tracer logging.Tracer) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		db, err := kvstore.New(args[1])
		if err != nil {
			return fmt.Errorf("unable to create sourceStore: %w", err)
		}

		resolver := accountsresolver.NewKVDBAccountsResolver(db, logger)
		if err != nil {
			return fmt.Errorf("creating resolver: %w", err)
		}

		endpoint := rpc.MainNetBeta_RPC
		client := rpc.New(endpoint)

		pubKey := solana.MustPublicKeyFromBase58(args[0])

		state, err := addresslookuptable.GetAddressLookupTable(ctx, client, pubKey)
		if err != nil {
			return fmt.Errorf("getting address lookup table: %w", err)
		}

		resolvedAccounts, atBlock, _, err := resolver.ResolveWithBlock(ctx, state.LastExtendedSlot, pubKey.Bytes())
		if err != nil {
			return fmt.Errorf("resolving accounts: %w", err)
		}

		_, err = Validate(ctx, client, pubKey, resolvedAccounts, atBlock, state)
		if err != nil {
			return fmt.Errorf("validating: %w", err)
		}

		fmt.Println("All done: Goodbye!")
		return nil
	}
}

func newValidateAllResolvedAddresses(logger *zap.Logger, tracer logging.Tracer, chain *firecore.Chain[*pbsolv1.Block]) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate-all-resolve-addresses  {kv-dsn}",
		Short: "",
		RunE:  processValidateAllResolvedAddressesE(chain, logger, tracer),
		Args:  cobra.ExactArgs(1),
	}
	cmd.Flags().String("rpc-endpoint", "", "Pass in your RPC endpoint")
	return cmd
}

type ValidationState struct {
	NotFoundTable map[string]bool
	Validated     map[string]bool
	Invalid       map[string]bool
}

func processValidateAllResolvedAddressesE(chain *firecore.Chain[*pbsolv1.Block], logger *zap.Logger, tracer logging.Tracer) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		vs, err := loadValidationState("validation_state.json")
		if err != nil {
			return fmt.Errorf("loading validation state: %w", err)
		}

		db, err := kvstore.New(args[0])
		if err != nil {
			return fmt.Errorf("unable to create sourceStore: %w", err)
		}

		client := rpc.New(sflags.MustGetString(cmd, "rpc-endpoint"))

		iter := db.Prefix(ctx, []byte{accountsresolver.TableAccountLookup}, kvstore.Unlimited)
		keyCount := 0
		newValidationCount := 0
		notFoundCount := 0
		for iter.Next() {
			keyCount++
			if newValidationCount == 100 {
				newValidationCount = 0
				err = SaveState(vs)
				if err != nil {
					return fmt.Errorf("saving validation state: %w", err)
				}
				fmt.Println("Saved validation state file", keyCount, notFoundCount)
			}

			time.Sleep(time.Second / 10)

			item := iter.Item()
			tableAccount, atBlock := accountsresolver.Keys.UnpackTableLookup(item.Key)
			if _, found := vs.NotFoundTable[tableAccount.Base58()]; found {
				notFoundCount++
				continue
			}
			validationKey := fmt.Sprintf("%s@%d", tableAccount.Base58(), atBlock)
			if _, found := vs.Validated[validationKey]; found {
				continue
			}
			newValidationCount++
			accounts := accountsresolver.DecodeAccounts(item.Value)
			state, err := addresslookuptable.GetAddressLookupTable(ctx, client, solana.PublicKeyFromBytes(tableAccount))
			if err != nil {
				if err.Error() == "not found" {
					fmt.Printf("⚠️ %s Account not found.\n", tableAccount.Base58())
					vs.NotFoundTable[tableAccount.Base58()] = true
					continue
				} else {
					return fmt.Errorf("getting address lookup table %q: %w", tableAccount.Base58(), err)
				}
			}
			valid, err := Validate(ctx, client, solana.PublicKeyFromBytes(tableAccount), accounts, atBlock, state)
			if err != nil {
				return fmt.Errorf("validating: %w", err)
			}

			vs.Validated[validationKey] = true

			if !valid {
				vs.Invalid[validationKey] = true
			}

		}
		if iter.Err() != nil {
			return fmt.Errorf("querying accounts: %w", iter.Err())
		}

		err = SaveState(vs)
		fmt.Println("Saved validation state file", keyCount, notFoundCount)
		if err != nil {
			return fmt.Errorf("saving validation state: %w", err)
		}
		fmt.Println("All done: Goodbye!")
		return nil
	}
}

func loadValidationState(filePath string) (*ValidationState, error) {
	if fileExists(filePath) {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("reading validation file: %w", err)
		}
		state := &ValidationState{}
		err = json.Unmarshal(data, state)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling validation file: %w", err)
		}
		fmt.Println("Loaded validation state file")
		return state, nil
	}

	fmt.Println("Creating new validation state file")
	return &ValidationState{
		NotFoundTable: map[string]bool{},
		Validated:     map[string]bool{},
		Invalid:       map[string]bool{},
	}, nil
}

func SaveState(state *ValidationState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshaling validation file: %w", err)
	}
	err = os.WriteFile("validation_state.json", data, 0644)
	if err != nil {
		return fmt.Errorf("writing validation file: %w", err)
	}

	return nil
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func Validate(ctx context.Context, client *rpc.Client, pubKey solana.PublicKey, resolvedAccounts accountsresolver.Accounts, atBlock uint64, state *addresslookuptable.AddressLookupTableState) (bool, error) {
	if atBlock < state.LastExtendedSlot {
		//fmt.Printf("⚠️ %s Resolved accounts found are before the last extended slot.\n", pubKey.String())
	} else {
		if len(resolvedAccounts) != len(state.Addresses) {
			fmt.Printf("❌ %s Resolved accounts count (%d) does not match the number of accounts in the lookup table (%d).\n", pubKey.String(), len(resolvedAccounts), len(state.Addresses))
			return false, nil
		}
	}

	for i, account := range resolvedAccounts {
		if account.Base58() != state.Addresses[i].String() {
			fmt.Printf("❌ %s Resolved account #%d (%s) does not match the account in the lookup table (%s).\n", pubKey.String(), i, account.Base58(), state.Addresses[i].String())
			return false, nil
		}
	}

	fmt.Println("✅", pubKey.String(), "account count", len(resolvedAccounts))
	return true, nil
}
