package trace

import (
	"context"
	"fmt"
)

type PersistenceSink interface {
	PersistTrace(ctx context.Context, chainID uint64, txHash string, root *CallFrame) error
}

type PersistingProvider struct {
	base    Provider
	chainID uint64
	sink    PersistenceSink
}

func NewPersistingProvider(base Provider, chainID uint64, sink PersistenceSink) *PersistingProvider {
	return &PersistingProvider{
		base:    base,
		chainID: chainID,
		sink:    sink,
	}
}

func (p *PersistingProvider) TraceTransaction(ctx context.Context, txHash string) (*CallFrame, error) {
	if p == nil || p.base == nil {
		return nil, fmt.Errorf("trace provider is required")
	}
	root, err := p.base.TraceTransaction(ctx, txHash)
	if err != nil || root == nil {
		return root, err
	}
	if p.sink != nil {
		_ = p.sink.PersistTrace(ctx, p.chainID, txHash, cloneCallFrame(root))
	}
	return root, nil
}
