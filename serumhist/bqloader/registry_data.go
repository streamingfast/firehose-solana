package bqloader

import (
	"context"
	"fmt"
	"strings"

	"github.com/dfuse-io/dfuse-solana/registry"
	"google.golang.org/api/googleapi"
)

func (bq *BQLoader) LoadMarkets(ctx context.Context) error {
	marketRows := make([]marketRow, 0, 128)

	for _, market := range bq.registryServer.GetMarkets() {
		marketRows = append(marketRows, marketToRow(market))
	}

	if !(len(marketRows) > 0) {
		return nil
	}

	if err := bq.truncateTable(ctx, fmt.Sprintf("%s.serum.%s", bq.dataset.ProjectID, markets)); err != nil {
		if isStreamingBufferError(err) {
			return nil
		}
		return err
	}

	return bq.dataset.Table(markets).Inserter().Put(ctx, marketRows)
}

type marketRow struct {
	Name         string `bigquery:"name"`
	Address      string `bigquery:"address"`
	Deprecated   bool   `bigquery:"deprecated"`
	ProgramID    string `bigquery:"program_id"`
	BaseToken    string `bigquery:"base_token"`
	QuoteToken   string `bigquery:"quote_token"`
	BaseLotSize  int64  `bigquery:"base_lot_size"`
	QuoteLotSize int64  `bigquery:"quote_lot_size"`
	RequestQueue string `bigquery:"request_queue"`
	EventQueue   string `bigquery:"event_queue"`
}

func marketToRow(market *registry.Market) marketRow {
	return marketRow{
		Name:         market.Name,
		Address:      market.Address.String(),
		Deprecated:   market.Deprecated,
		ProgramID:    market.ProgramID.String(),
		BaseToken:    market.BaseToken.String(),
		QuoteToken:   market.QuoteToken.String(),
		BaseLotSize:  int64(market.BaseLotSize),
		QuoteLotSize: int64(market.QuoteLotSize),
		RequestQueue: market.RequestQueue.String(),
		EventQueue:   market.EventQueue.String(),
	}
}

func (bq *BQLoader) LoadTokens(ctx context.Context) error {
	tokenRows := make([]tokenRow, 0, 128)
	for _, token := range bq.registryServer.GetTokens() {
		tokenRows = append(tokenRows, tokenToRow(token))
	}

	if !(len(tokenRows) > 0) {
		return nil
	}

	if err := bq.truncateTable(ctx, fmt.Sprintf("%s.serum.%s", bq.dataset.ProjectID, tokens)); err != nil {
		if isStreamingBufferError(err) {
			return nil
		}
		return err
	}

	return bq.dataset.Table(tokens).Inserter().Put(ctx, tokenRows)
}

type tokenRow struct {
	Name                  string `bigquery:"name"`
	Symbol                string `bigquery:"symbol"`
	Address               string `bigquery:"address"`
	MintAuthorityOption   int32  `bigquery:"mint_authority_option"`
	MintAuthority         string `bigquery:"mint_authority"`
	Supply                int64  `bigquery:"supply"`
	Decimals              int8   `bigquery:"decimals"`
	IsInitialized         bool   `bigquery:"is_initialized"`
	FreezeAuthorityOption int32  `bigquery:"freeze_authority_option"`
	FreezeAuthority       string `bigquery:"freeze_authority"`
	Verified              bool   `bigquery:"verified"`
}

func tokenToRow(token *registry.Token) tokenRow {
	row := tokenRow{
		Address:               token.Address.String(),
		MintAuthorityOption:   int32(token.MintAuthorityOption),
		MintAuthority:         token.MintAuthority.String(),
		Supply:                int64(token.Supply),
		Decimals:              int8(token.Decimals),
		IsInitialized:         token.IsInitialized,
		FreezeAuthorityOption: int32(token.FreezeAuthorityOption),
		FreezeAuthority:       token.FreezeAuthority.String(),
		Verified:              token.Verified,
	}
	if token.Meta != nil {
		row.Name = token.Meta.Name
		row.Symbol = token.Meta.Symbol
	}
	return row
}

func (bq *BQLoader) truncateTable(ctx context.Context, tableName string) error {
	query := bq.client.Query(fmt.Sprintf("DELETE FROM %s WHERE true", tableName))
	job, err := query.Run(ctx)
	if err != nil {
		return err
	}
	js, err := job.Wait(ctx)
	if err != nil {
		return err
	}
	if js.Err() != nil {
		return js.Err()
	}

	return nil
}

func isStreamingBufferError(err error) bool {
	apiError, ok := err.(*googleapi.Error)
	if !ok {
		return false
	}
	if strings.Contains(apiError.Body, "would affect rows in the streaming buffer") {
		return true
	}

	return false
}
