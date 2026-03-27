package orchestrator

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cpchain-network/flashloan-scanner/scanner"
)

func TestProtocolRunnerRunOnce(t *testing.T) {
	txStore := &fakeTransactionStore{
		txs: []scanner.ObservedTransaction{{
			ChainID:        1,
			TxHash:         "0xabc",
			BlockNumber:    "100",
			ToAddress:      "0x1111111111111111111111111111111111111111",
			MethodSelector: "0x12345678",
		}},
	}
	interactionStore := &fakeInteractionStore{}
	legStore := &fakeLegStore{}
	aggregator := &fakeAggregator{}

	runner, err := NewProtocolRunner(
		"aave_v3_scanner",
		scanner.ProtocolAaveV3,
		fakeFetcher{},
		txStore,
		interactionStore,
		legStore,
		fakeExtractor{},
		fakeVerifier{},
		aggregator,
	)
	require.NoError(t, err)

	err = runner.RunOnce(context.Background(), 1, 100, 100)
	require.NoError(t, err)
	require.True(t, interactionStore.upsertCalled)
	require.True(t, interactionStore.updateCalled)
	require.Len(t, legStore.calls, 2)
	require.Equal(t, []string{"0xabc"}, aggregator.txHashes)
}

func TestProtocolRunnerRunOnceObserver(t *testing.T) {
	txStore := &fakeTransactionStore{
		txs: []scanner.ObservedTransaction{{
			ChainID:        1,
			TxHash:         "0xabc",
			BlockNumber:    "100",
			ToAddress:      "0x1111111111111111111111111111111111111111",
			MethodSelector: "0x12345678",
		}},
	}
	observer := &fakeObserver{}

	runner, err := NewProtocolRunner(
		"aave_v3_scanner",
		scanner.ProtocolAaveV3,
		fakeFetcher{},
		txStore,
		&fakeInteractionStore{},
		&fakeLegStore{},
		fakeExtractor{},
		fakeVerifier{},
		&fakeAggregator{},
	)
	require.NoError(t, err)

	runner.WithObserver(observer)

	err = runner.RunOnce(context.Background(), 1, 100, 100)
	require.NoError(t, err)
	require.Equal(t, []string{"started", "finding", "progress", "completed"}, observer.events)
	require.Len(t, observer.started, 1)
	require.Len(t, observer.findings, 1)
	require.Len(t, observer.progress, 1)
	require.Len(t, observer.completed, 1)
	require.Empty(t, observer.failed)
	require.Equal(t, uint64(100), observer.progress[0].CurrentBlock)
	require.Equal(t, 1, observer.progress[0].Findings)
	require.Equal(t, 1, observer.progress[0].VerifiedFindings)
	require.Equal(t, 1, observer.progress[0].StrictFindings)
	require.Equal(t, "0xabc", observer.findings[0].TxHash)
	require.True(t, observer.findings[0].Candidate)
	require.True(t, observer.findings[0].Verified)
	require.True(t, observer.findings[0].Strict)
	require.Equal(t, 1, observer.completed[0].VerifiedInteractions)
	require.Equal(t, 1, observer.completed[0].StrictInteractions)
}

func TestProtocolRunnerRunOnceObserverFailure(t *testing.T) {
	txStore := &fakeTransactionStore{
		txs: []scanner.ObservedTransaction{{
			ChainID:        1,
			TxHash:         "0xabc",
			BlockNumber:    "100",
			ToAddress:      "0x1111111111111111111111111111111111111111",
			MethodSelector: "0x12345678",
		}},
	}
	observer := &fakeObserver{}

	runner, err := NewProtocolRunner(
		"aave_v3_scanner",
		scanner.ProtocolAaveV3,
		fakeFetcher{},
		txStore,
		&fakeInteractionStore{},
		&fakeLegStore{},
		fakeExtractor{},
		fakeVerifier{err: errors.New("verify failed")},
		&fakeAggregator{},
	)
	require.NoError(t, err)

	runner.WithObserver(observer)

	err = runner.RunOnce(context.Background(), 1, 100, 100)
	require.EqualError(t, err, "verify failed")
	require.Equal(t, []string{"started", "failed"}, observer.events)
	require.Len(t, observer.failed, 1)
	require.Equal(t, uint64(100), observer.failed[0].CurrentBlock)
	require.Equal(t, "verify failed", observer.failed[0].Error)
	require.Empty(t, observer.findings)
	require.Empty(t, observer.progress)
	require.Empty(t, observer.completed)
}

type fakeFetcher struct{}

func (fakeFetcher) FetchByTxHash(context.Context, uint64, string) (*scanner.ObservedTransaction, error) {
	return nil, nil
}

func (fakeFetcher) FetchRange(context.Context, uint64, uint64, uint64) error {
	return nil
}

type fakeTransactionStore struct {
	txs []scanner.ObservedTransaction
}

func (f *fakeTransactionStore) UpsertObservedTransactions(context.Context, []scanner.ObservedTransaction) error {
	return nil
}

func (f *fakeTransactionStore) ListObservedTransactionsByBlockRange(context.Context, uint64, uint64, uint64) ([]scanner.ObservedTransaction, error) {
	return f.txs, nil
}

type fakeInteractionStore struct {
	upsertCalled bool
	updateCalled bool
}

func (f *fakeInteractionStore) UpsertInteractions(context.Context, []scanner.CandidateInteraction) error {
	f.upsertCalled = true
	return nil
}

func (f *fakeInteractionStore) UpdateVerificationResult(context.Context, scanner.VerifiedInteraction) error {
	f.updateCalled = true
	return nil
}

func (f *fakeInteractionStore) ListCandidateInteractions(context.Context, uint64, scanner.Protocol, uint64, uint64) ([]scanner.CandidateInteraction, error) {
	return nil, nil
}

type fakeLegStore struct {
	calls [][]scanner.InteractionLeg
}

func (f *fakeLegStore) ReplaceInteractionLegs(_ context.Context, _ string, legs []scanner.InteractionLeg) error {
	cloned := make([]scanner.InteractionLeg, len(legs))
	copy(cloned, legs)
	f.calls = append(f.calls, cloned)
	return nil
}

type fakeExtractor struct{}

func (fakeExtractor) Protocol() scanner.Protocol {
	return scanner.ProtocolAaveV3
}

func (fakeExtractor) Extract(context.Context, uint64, []scanner.ObservedTransaction) ([]scanner.CandidateInteraction, []scanner.InteractionLeg, error) {
	mode := 0
	return []scanner.CandidateInteraction{{
			InteractionID:      "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
			InteractionOrdinal: 0,
			ChainID:            1,
			TxHash:             "0xabc",
			BlockNumber:        "100",
			Protocol:           scanner.ProtocolAaveV3,
			Entrypoint:         "flashLoanSimple",
			ProviderAddress:    "0x1111111111111111111111111111111111111111",
			ReceiverAddress:    "0x2222222222222222222222222222222222222222",
			CandidateLevel:     scanner.CandidateLevelNormal,
			RawMethodSelector:  "0x12345678",
		}},
		[]scanner.InteractionLeg{{
			InteractionID:    "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
			LegIndex:         0,
			AssetAddress:     "0x3333333333333333333333333333333333333333",
			AssetRole:        "borrowed",
			AmountBorrowed:   stringPtr("1000"),
			InterestRateMode: &mode,
		}}, nil
}

type fakeVerifier struct {
	err error
}

func (fakeVerifier) Protocol() scanner.Protocol {
	return scanner.ProtocolAaveV3
}

func (f fakeVerifier) Verify(context.Context, uint64, scanner.CandidateInteraction) (*scanner.VerifiedInteraction, []scanner.InteractionLeg, error) {
	if f.err != nil {
		return nil, nil, f.err
	}
	mode := 0
	return &scanner.VerifiedInteraction{
			InteractionID:  "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
			Verified:       true,
			Strict:         true,
			CallbackSeen:   true,
			SettlementSeen: true,
			RepaymentSeen:  true,
		},
		[]scanner.InteractionLeg{{
			InteractionID:    "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
			LegIndex:         0,
			AssetAddress:     "0x3333333333333333333333333333333333333333",
			AssetRole:        "borrowed",
			AmountBorrowed:   stringPtr("1000"),
			AmountRepaid:     stringPtr("1005"),
			PremiumAmount:    stringPtr("5"),
			InterestRateMode: &mode,
			StrictLeg:        true,
			EventSeen:        true,
		}}, nil
}

type fakeAggregator struct {
	txHashes []string
}

func (f *fakeAggregator) AggregateByTx(context.Context, uint64, string) (*scanner.TxSummary, error) {
	f.txHashes = append(f.txHashes, "0xabc")
	return &scanner.TxSummary{
		ChainID:                           1,
		TxHash:                            "0xabc",
		BlockNumber:                       "100",
		ContainsCandidateInteraction:      true,
		ContainsVerifiedInteraction:       true,
		ContainsVerifiedStrictInteraction: true,
		InteractionCount:                  1,
		StrictInteractionCount:            1,
		ProtocolCount:                     1,
		Protocols:                         []scanner.Protocol{scanner.ProtocolAaveV3},
	}, nil
}

func (f *fakeAggregator) AggregateRange(context.Context, uint64, uint64, uint64) error {
	return nil
}

type fakeObserver struct {
	events    []string
	started   []ProtocolRunStarted
	progress  []ProtocolRunProgress
	findings  []ProtocolFinding
	completed []ProtocolRunCompleted
	failed    []ProtocolRunFailed
}

func (f *fakeObserver) OnProtocolStarted(event ProtocolRunStarted) {
	f.events = append(f.events, "started")
	f.started = append(f.started, event)
}

func (f *fakeObserver) OnProtocolProgress(event ProtocolRunProgress) {
	f.events = append(f.events, "progress")
	f.progress = append(f.progress, event)
}

func (f *fakeObserver) OnFinding(event ProtocolFinding) {
	f.events = append(f.events, "finding")
	f.findings = append(f.findings, event)
}

func (f *fakeObserver) OnProtocolCompleted(event ProtocolRunCompleted) {
	f.events = append(f.events, "completed")
	f.completed = append(f.completed, event)
}

func (f *fakeObserver) OnProtocolFailed(event ProtocolRunFailed) {
	f.events = append(f.events, "failed")
	f.failed = append(f.failed, event)
}

func stringPtr(v string) *string {
	return &v
}
