package trace

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
)

type GethProvider struct {
	rpc *rpc.Client
}

func NewGethProvider(rpcClient *rpc.Client) *GethProvider {
	return &GethProvider{rpc: rpcClient}
}

func (p *GethProvider) TraceTransaction(ctx context.Context, txHash string) (*CallFrame, error) {
	if p == nil || p.rpc == nil {
		return nil, fmt.Errorf("trace rpc client is required")
	}
	var result CallFrame
	err := p.rpc.CallContext(ctx, &result, "debug_traceTransaction", common.HexToHash(txHash), map[string]any{
		"tracer": "callTracer",
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}
