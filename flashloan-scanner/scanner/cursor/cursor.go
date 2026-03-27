package cursor

import "context"

type Manager interface {
	Get(ctx context.Context, scannerName string, chainID uint64, cursorType string) (uint64, error)
	Save(ctx context.Context, scannerName string, chainID uint64, cursorType string, blockNumber uint64) error
}
