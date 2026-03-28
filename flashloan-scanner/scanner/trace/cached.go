package trace

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

type CachedProvider struct {
	base  Provider
	mu    sync.RWMutex
	cache map[string]*CallFrame
}

func NewCachedProvider(base Provider) *CachedProvider {
	return &CachedProvider{
		base:  base,
		cache: make(map[string]*CallFrame),
	}
}

func (p *CachedProvider) TraceTransaction(ctx context.Context, txHash string) (*CallFrame, error) {
	if p == nil || p.base == nil {
		return nil, fmt.Errorf("trace provider is required")
	}
	key := strings.ToLower(strings.TrimSpace(txHash))
	if key == "" {
		return nil, fmt.Errorf("tx hash is required")
	}

	p.mu.RLock()
	cached := p.cache[key]
	p.mu.RUnlock()
	if cached != nil {
		return cloneCallFrame(cached), nil
	}

	root, err := p.base.TraceTransaction(ctx, txHash)
	if err != nil || root == nil {
		return root, err
	}

	cloned := cloneCallFrame(root)
	p.mu.Lock()
	p.cache[key] = cloned
	p.mu.Unlock()
	return cloneCallFrame(cloned), nil
}

func cloneCallFrame(frame *CallFrame) *CallFrame {
	if frame == nil {
		return nil
	}
	cloned := &CallFrame{
		Type:         frame.Type,
		From:         frame.From,
		To:           frame.To,
		Input:        frame.Input,
		Output:       frame.Output,
		Error:        frame.Error,
		RevertReason: frame.RevertReason,
	}
	if len(frame.Calls) == 0 {
		return cloned
	}
	cloned.Calls = make([]CallFrame, len(frame.Calls))
	for i := range frame.Calls {
		child := cloneCallFrame(&frame.Calls[i])
		if child != nil {
			cloned.Calls[i] = *child
		}
	}
	return cloned
}
