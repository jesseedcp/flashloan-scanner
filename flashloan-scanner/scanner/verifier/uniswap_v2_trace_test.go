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
	scannertrace "github.com/cpchain-network/flashloan-scanner/scanner/trace"
	"github.com/cpchain-network/flashloan-scanner/scanner/uniswapv2"
)

func TestUniswapV2TraceVerifierStrict(t *testing.T) {
	pairABI, err := uniswapv2.PairABI()
	require.NoError(t, err)

	pair := common.HexToAddress("0x1111111111111111111111111111111111111111")
	receiver := common.HexToAddress("0x2222222222222222222222222222222222222222")
	token := common.HexToAddress("0x3333333333333333333333333333333333333333")
	txHash := common.HexToHash("0xca11")
	interactionID := uuid.New()
	dataNonEmpty := true
	tokenSide := "token0"

	base, err := NewUniswapV2Verifier(
		fakeEventView{events: []dbevent.ContractEvent{makeUniswapSwapEvent(t, pairABI, pair, receiver, txHash, 1100, 0, 1000, 0)}},
		fakeObservedTxView{tx: &dbscanner.ObservedTransaction{ChainID: 1, TxHash: txHash, Status: 1}},
		fakeLegView{legs: []dbscanner.InteractionAssetLeg{{
			InteractionID:  interactionID,
			LegIndex:       0,
			AssetAddress:   token,
			AssetRole:      "borrowed",
			TokenSide:      &tokenSide,
			AmountBorrowed: big.NewInt(1000),
		}}},
	)
	require.NoError(t, err)

	traceVerifier, err := NewUniswapV2TraceVerifier(base, fakeTraceProvider{
		root: &scannertrace.CallFrame{
			From: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			To:   pair.Hex(),
			Calls: []scannertrace.CallFrame{
				{
					From:  pair.Hex(),
					To:    receiver.Hex(),
					Input: "0x12345678",
					Calls: []scannertrace.CallFrame{
						{
							From:  receiver.Hex(),
							To:    token.Hex(),
							Input: erc20TransferSelector + "00",
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
		Protocol:        scanner.ProtocolUniswapV2,
		Entrypoint:      "swap",
		ProviderAddress: pair.Hex(),
		PairAddress:     pair.Hex(),
		ReceiverAddress: receiver.Hex(),
		DataNonEmpty:    &dataNonEmpty,
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

func TestUniswapV2TraceVerifierMissingCallback(t *testing.T) {
	pairABI, err := uniswapv2.PairABI()
	require.NoError(t, err)

	pair := common.HexToAddress("0x1111111111111111111111111111111111111111")
	receiver := common.HexToAddress("0x2222222222222222222222222222222222222222")
	token := common.HexToAddress("0x3333333333333333333333333333333333333333")
	txHash := common.HexToHash("0xca12")
	interactionID := uuid.New()
	dataNonEmpty := true
	tokenSide := "token0"

	base, err := NewUniswapV2Verifier(
		fakeEventView{events: []dbevent.ContractEvent{makeUniswapSwapEvent(t, pairABI, pair, receiver, txHash, 1100, 0, 1000, 0)}},
		fakeObservedTxView{tx: &dbscanner.ObservedTransaction{ChainID: 1, TxHash: txHash, Status: 1}},
		fakeLegView{legs: []dbscanner.InteractionAssetLeg{{
			InteractionID:  interactionID,
			LegIndex:       0,
			AssetAddress:   token,
			AssetRole:      "borrowed",
			TokenSide:      &tokenSide,
			AmountBorrowed: big.NewInt(1000),
		}}},
	)
	require.NoError(t, err)

	traceVerifier, err := NewUniswapV2TraceVerifier(base, fakeTraceProvider{
		root: &scannertrace.CallFrame{
			From: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			To:   pair.Hex(),
			Calls: []scannertrace.CallFrame{
				{
					From:  receiver.Hex(),
					To:    token.Hex(),
					Input: erc20TransferSelector + "00",
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
		Protocol:        scanner.ProtocolUniswapV2,
		Entrypoint:      "swap",
		ProviderAddress: pair.Hex(),
		PairAddress:     pair.Hex(),
		ReceiverAddress: receiver.Hex(),
		DataNonEmpty:    &dataNonEmpty,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.Verified)
	require.False(t, result.Strict)
	require.NotNil(t, result.ExclusionReason)
	require.Equal(t, "missing_trace_callback", *result.ExclusionReason)
}

func TestUniswapV2TraceVerifierDowngradesWithoutInvariantPath(t *testing.T) {
	pairABI, err := uniswapv2.PairABI()
	require.NoError(t, err)

	pair := common.HexToAddress("0x1111111111111111111111111111111111111111")
	receiver := common.HexToAddress("0x2222222222222222222222222222222222222222")
	token := common.HexToAddress("0x3333333333333333333333333333333333333333")
	txHash := common.HexToHash("0xca13")
	interactionID := uuid.New()
	dataNonEmpty := true
	tokenSide := "token0"

	base, err := NewUniswapV2Verifier(
		fakeEventView{events: []dbevent.ContractEvent{makeUniswapSwapEvent(t, pairABI, pair, receiver, txHash, 1100, 0, 1000, 0)}},
		fakeObservedTxView{tx: &dbscanner.ObservedTransaction{ChainID: 1, TxHash: txHash, Status: 1}},
		fakeLegView{legs: []dbscanner.InteractionAssetLeg{{
			InteractionID:  interactionID,
			LegIndex:       0,
			AssetAddress:   token,
			AssetRole:      "borrowed",
			TokenSide:      &tokenSide,
			AmountBorrowed: big.NewInt(1000),
		}}},
	)
	require.NoError(t, err)

	traceVerifier, err := NewUniswapV2TraceVerifier(base, fakeTraceProvider{
		root: &scannertrace.CallFrame{
			From: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			To:   pair.Hex(),
			Calls: []scannertrace.CallFrame{
				{
					From:  pair.Hex(),
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
		Protocol:        scanner.ProtocolUniswapV2,
		Entrypoint:      "swap",
		ProviderAddress: pair.Hex(),
		PairAddress:     pair.Hex(),
		ReceiverAddress: receiver.Hex(),
		DataNonEmpty:    &dataNonEmpty,
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
