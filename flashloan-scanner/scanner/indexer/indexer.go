package indexer

import (
	"context"

	"github.com/cpchain-network/flashloan-scanner/scanner"
)

type RawIndexer interface {
	RunOnce(ctx context.Context, chainID uint64, fromBlock, toBlock uint64) error
	RunLoop(ctx context.Context, chainID uint64) error
}

type HeaderSource interface {
	BlockHeadersByRange(ctx context.Context, chainID uint64, fromBlock, toBlock uint64) ([]scanner.Header, error)
}

type LogSource interface {
	FilterLogs(ctx context.Context, chainID uint64, fromBlock, toBlock uint64, addresses []string) ([]scanner.RawLog, error)
}
