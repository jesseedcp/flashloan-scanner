package verifier

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	dbevent "github.com/cpchain-network/flashloan-scanner/database/event"
	dbscanner "github.com/cpchain-network/flashloan-scanner/database/scanner"
	"github.com/cpchain-network/flashloan-scanner/scanner"
	"github.com/cpchain-network/flashloan-scanner/scanner/aave"
	scannertrace "github.com/cpchain-network/flashloan-scanner/scanner/trace"
)

func TestAaveV3TraceVerifierStrict(t *testing.T) {
	poolABI, err := aave.PoolABI()
	require.NoError(t, err)

	pool := common.HexToAddress("0x1111111111111111111111111111111111111111")
	receiver := common.HexToAddress("0x2222222222222222222222222222222222222222")
	asset := common.HexToAddress("0x3333333333333333333333333333333333333333")
	txHash := common.HexToHash("0xabc1")
	interactionID := uuid.New()

	base, err := NewAaveV3Verifier(
		fakeEventView{events: []dbevent.ContractEvent{makeFlashLoanEvent(t, poolABI, pool, receiver, asset, txHash, 1000, 0, 5)}},
		fakeObservedTxView{tx: &dbscanner.ObservedTransaction{ChainID: 1, TxHash: txHash, Status: 1}},
		fakeLegView{legs: []dbscanner.InteractionAssetLeg{{
			InteractionID:    interactionID,
			LegIndex:         0,
			AssetAddress:     asset,
			AssetRole:        "borrowed",
			AmountBorrowed:   big.NewInt(1000),
			InterestRateMode: uint8Ptr(0),
		}}},
	)
	require.NoError(t, err)

	traceVerifier, err := NewAaveV3TraceVerifier(base, fakeTraceProvider{
		root: &scannertrace.CallFrame{
			From: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			To:   pool.Hex(),
			Calls: []scannertrace.CallFrame{
				{
					From:  pool.Hex(),
					To:    receiver.Hex(),
					Input: "0x12345678",
					Calls: []scannertrace.CallFrame{
						{
							From:  receiver.Hex(),
							To:    asset.Hex(),
							Input: erc20ApproveSelector + "00",
						},
					},
				},
			},
		},
	})
	require.NoError(t, err)

	result, legs, err := traceVerifier.Verify(context.Background(), 1, scanner.CandidateInteraction{
		InteractionID:   interactionID.String(),
		ChainID:         1,
		TxHash:          txHash.Hex(),
		BlockNumber:     "100",
		Protocol:        scanner.ProtocolAaveV3,
		Entrypoint:      "flashLoanSimple",
		ProviderAddress: pool.Hex(),
		ReceiverAddress: receiver.Hex(),
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Verified)
	require.True(t, result.Strict)
	require.True(t, result.CallbackSeen)
	require.True(t, result.RepaymentSeen)
	require.NotNil(t, result.VerificationNotes)
	require.Contains(t, *result.VerificationNotes, "trace-backed strict verification passed")
	require.Len(t, legs, 1)
	require.True(t, legs[0].StrictLeg)
}

func TestAaveV3TraceVerifierMissingCallback(t *testing.T) {
	poolABI, err := aave.PoolABI()
	require.NoError(t, err)

	pool := common.HexToAddress("0x1111111111111111111111111111111111111111")
	receiver := common.HexToAddress("0x2222222222222222222222222222222222222222")
	asset := common.HexToAddress("0x3333333333333333333333333333333333333333")
	txHash := common.HexToHash("0xabc2")
	interactionID := uuid.New()

	base, err := NewAaveV3Verifier(
		fakeEventView{events: []dbevent.ContractEvent{makeFlashLoanEvent(t, poolABI, pool, receiver, asset, txHash, 1000, 0, 5)}},
		fakeObservedTxView{tx: &dbscanner.ObservedTransaction{ChainID: 1, TxHash: txHash, Status: 1}},
		fakeLegView{legs: []dbscanner.InteractionAssetLeg{{
			InteractionID:    interactionID,
			LegIndex:         0,
			AssetAddress:     asset,
			AssetRole:        "borrowed",
			AmountBorrowed:   big.NewInt(1000),
			InterestRateMode: uint8Ptr(0),
		}}},
	)
	require.NoError(t, err)

	traceVerifier, err := NewAaveV3TraceVerifier(base, fakeTraceProvider{
		root: &scannertrace.CallFrame{
			From: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			To:   pool.Hex(),
			Calls: []scannertrace.CallFrame{
				{
					From:  receiver.Hex(),
					To:    asset.Hex(),
					Input: erc20ApproveSelector + "00",
				},
			},
		},
	})
	require.NoError(t, err)

	result, _, err := traceVerifier.Verify(context.Background(), 1, scanner.CandidateInteraction{
		InteractionID:   interactionID.String(),
		ChainID:         1,
		TxHash:          txHash.Hex(),
		BlockNumber:     "100",
		Protocol:        scanner.ProtocolAaveV3,
		Entrypoint:      "flashLoanSimple",
		ProviderAddress: pool.Hex(),
		ReceiverAddress: receiver.Hex(),
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.Verified)
	require.False(t, result.Strict)
	require.False(t, result.CallbackSeen)
	require.NotNil(t, result.ExclusionReason)
	require.Equal(t, "missing_trace_callback", *result.ExclusionReason)
}

func TestAaveV3TraceVerifierDowngradesWithoutRepaymentPath(t *testing.T) {
	poolABI, err := aave.PoolABI()
	require.NoError(t, err)

	pool := common.HexToAddress("0x1111111111111111111111111111111111111111")
	receiver := common.HexToAddress("0x2222222222222222222222222222222222222222")
	asset := common.HexToAddress("0x3333333333333333333333333333333333333333")
	txHash := common.HexToHash("0xabc3")
	interactionID := uuid.New()

	base, err := NewAaveV3Verifier(
		fakeEventView{events: []dbevent.ContractEvent{makeFlashLoanEvent(t, poolABI, pool, receiver, asset, txHash, 1000, 0, 5)}},
		fakeObservedTxView{tx: &dbscanner.ObservedTransaction{ChainID: 1, TxHash: txHash, Status: 1}},
		fakeLegView{legs: []dbscanner.InteractionAssetLeg{{
			InteractionID:    interactionID,
			LegIndex:         0,
			AssetAddress:     asset,
			AssetRole:        "borrowed",
			AmountBorrowed:   big.NewInt(1000),
			InterestRateMode: uint8Ptr(0),
		}}},
	)
	require.NoError(t, err)

	traceVerifier, err := NewAaveV3TraceVerifier(base, fakeTraceProvider{
		root: &scannertrace.CallFrame{
			From: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			To:   pool.Hex(),
			Calls: []scannertrace.CallFrame{
				{
					From:  pool.Hex(),
					To:    receiver.Hex(),
					Input: "0x12345678",
				},
			},
		},
	})
	require.NoError(t, err)

	result, _, err := traceVerifier.Verify(context.Background(), 1, scanner.CandidateInteraction{
		InteractionID:   interactionID.String(),
		ChainID:         1,
		TxHash:          txHash.Hex(),
		BlockNumber:     "100",
		Protocol:        scanner.ProtocolAaveV3,
		Entrypoint:      "flashLoanSimple",
		ProviderAddress: pool.Hex(),
		ReceiverAddress: receiver.Hex(),
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Verified)
	require.False(t, result.Strict)
	require.True(t, result.CallbackSeen)
	require.False(t, result.RepaymentSeen)
	require.NotNil(t, result.VerificationNotes)
	require.Contains(t, *result.VerificationNotes, "downgraded from strict")
}

type fakeTraceProvider struct {
	root *scannertrace.CallFrame
	err  error
}

func (f fakeTraceProvider) TraceTransaction(context.Context, string) (*scannertrace.CallFrame, error) {
	return f.root, f.err
}
