package aggregator

import (
	"context"

	"github.com/cpchain-network/flashloan-scanner/scanner"
)

type TxAggregator interface {
	AggregateByTx(ctx context.Context, chainID uint64, txHash string) (*scanner.TxSummary, error)
	AggregateRange(ctx context.Context, chainID uint64, fromBlock, toBlock uint64) error
}
