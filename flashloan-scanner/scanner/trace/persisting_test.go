package trace

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPersistingProviderPersistsSuccessfulTrace(t *testing.T) {
	sink := &fakePersistenceSink{}
	root := &CallFrame{Type: "CALL", From: "0x1", To: "0x2"}
	provider := NewPersistingProvider(fakeProvider{root: root}, 1, sink)

	out, err := provider.TraceTransaction(context.Background(), "0xtx")
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Equal(t, uint64(1), sink.chainID)
	require.Equal(t, "0xtx", sink.txHash)
	require.NotNil(t, sink.root)
	require.Equal(t, "0x1", sink.root.From)
}

func TestCachedProviderAvoidsDuplicateFetches(t *testing.T) {
	base := &countingProvider{root: &CallFrame{Type: "CALL", From: "0x1", To: "0x2"}}
	provider := NewCachedProvider(base)

	_, err := provider.TraceTransaction(context.Background(), "0xtx")
	require.NoError(t, err)
	_, err = provider.TraceTransaction(context.Background(), "0xtx")
	require.NoError(t, err)

	require.Equal(t, 1, base.calls)
}

type fakeProvider struct {
	root *CallFrame
	err  error
}

func (f fakeProvider) TraceTransaction(context.Context, string) (*CallFrame, error) {
	return f.root, f.err
}

type countingProvider struct {
	root  *CallFrame
	err   error
	calls int
}

func (p *countingProvider) TraceTransaction(context.Context, string) (*CallFrame, error) {
	p.calls++
	if p.err != nil {
		return nil, p.err
	}
	return cloneCallFrame(p.root), nil
}

type fakePersistenceSink struct {
	chainID uint64
	txHash  string
	root    *CallFrame
	err     error
}

func (f *fakePersistenceSink) PersistTrace(_ context.Context, chainID uint64, txHash string, root *CallFrame) error {
	f.chainID = chainID
	f.txHash = txHash
	f.root = root
	return f.err
}

func TestPersistingProviderIgnoresSinkFailure(t *testing.T) {
	sink := &fakePersistenceSink{err: errors.New("write failed")}
	provider := NewPersistingProvider(fakeProvider{root: &CallFrame{Type: "CALL"}}, 1, sink)

	out, err := provider.TraceTransaction(context.Background(), "0xtx")
	require.NoError(t, err)
	require.NotNil(t, out)
}
