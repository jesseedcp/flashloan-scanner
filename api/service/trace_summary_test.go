package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	scannertrace "github.com/cpchain-network/flashloan-scanner/scanner/trace"
)

func TestBuildTransactionTraceSummaryBuildsFlowsAndEvidence(t *testing.T) {
	providerAddress := "0x1111111111111111111111111111111111111111"
	receiverAddress := "0x2222222222222222222222222222222222222222"
	tokenAddress := "0x3333333333333333333333333333333333333333"
	repaymentTarget := "0x4444444444444444444444444444444444444444"

	root := &scannertrace.CallFrame{
		Type: "CALL",
		From: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		To:   providerAddress,
		Calls: []scannertrace.CallFrame{
			{
				Type:  "CALL",
				From:  providerAddress,
				To:    receiverAddress,
				Input: "0x12345678",
				Calls: []scannertrace.CallFrame{
					{
						Type:  "CALL",
						From:  receiverAddress,
						To:    tokenAddress,
						Input: "0xa9059cbb00000000000000000000000044444444444444444444444444444444444444440000000000000000000000000000000000000000000000000000000000000064",
					},
				},
			},
		},
	}

	summary := buildTransactionTraceSummary(context.Background(), fakeTraceSource{provider: fakeTraceProvider{root: root}}, 1, "0xtx", []InteractionDetailResult{
		{
			InteractionID:     "interaction-1",
			Protocol:          "aave_v3",
			Entrypoint:        "flashLoanSimple",
			ProviderAddress:   providerAddress,
			ReceiverAddress:   receiverAddress,
			Verified:          true,
			Strict:            true,
			CallbackSeen:      true,
			SettlementSeen:    true,
			RepaymentSeen:     true,
			VerificationNotes: "trace-backed strict verification passed",
			Legs: []InteractionLegDTO{
				{
					AssetAddress:    tokenAddress,
					AmountBorrowed:  "100",
					AmountRepaid:    "100",
					RepaidToAddress: repaymentTarget,
				},
			},
		},
	})

	require.NotNil(t, summary)
	require.Equal(t, traceStatusAvailable, summary.Status)
	require.NotEmpty(t, summary.Frames)
	require.Len(t, summary.AssetFlows, 1)
	require.Equal(t, "transfer", summary.AssetFlows[0].Action)
	require.Equal(t, tokenAddress, summary.AssetFlows[0].AssetAddress)
	require.Equal(t, receiverAddress, summary.AssetFlows[0].Source)
	require.Equal(t, repaymentTarget, summary.AssetFlows[0].Target)
	require.Len(t, summary.InteractionEvidence, 1)
	require.True(t, summary.InteractionEvidence[0].CallbackSeen)
	require.Contains(t, summary.InteractionEvidence[0].CallbackFrameIDs, "trace-root.0")
	require.Contains(t, summary.InteractionEvidence[0].CallbackSubtreeIDs, "trace-root.0.0")
	require.NotEmpty(t, summary.InteractionEvidence[0].RepaymentFrameIDs)
}

func TestBuildTransactionTraceSummaryHandlesProviderErrors(t *testing.T) {
	summary := buildTransactionTraceSummary(context.Background(), fakeTraceSource{}, 1, "0xtx", nil)
	require.Equal(t, traceStatusUnavailable, summary.Status)
	require.NotEmpty(t, summary.Error)
}

type fakeTraceSource struct {
	provider scannertrace.Provider
	err      error
}

func (f fakeTraceSource) TraceProvider(uint64) (scannertrace.Provider, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.provider, nil
}

type fakeTraceProvider struct {
	root *scannertrace.CallFrame
	err  error
}

func (f fakeTraceProvider) TraceTransaction(context.Context, string) (*scannertrace.CallFrame, error) {
	return f.root, f.err
}
