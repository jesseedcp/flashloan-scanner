package orchestrator

import "context"

type Runner interface {
	RunOnce(ctx context.Context, chainID uint64, fromBlock, toBlock uint64) error
	RunLoop(ctx context.Context, chainID uint64) error
}
