package fetcher

import (
	"context"

	"github.com/cpchain-network/flashloan-scanner/scanner"
)

type TxFetcher interface {
	FetchByTxHash(ctx context.Context, chainID uint64, txHash string) (*scanner.ObservedTransaction, error)
	FetchRange(ctx context.Context, chainID uint64, fromBlock, toBlock uint64) error
}

type LatestBlockProvider interface {
	LatestBlockNumber(ctx context.Context) (uint64, error)
}
