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
	"github.com/cpchain-network/flashloan-scanner/scanner/balancer"
	scannertrace "github.com/cpchain-network/flashloan-scanner/scanner/trace"
)

func TestBalancerV2TraceVerifierStrict(t *testing.T) {
	vaultABI, err := balancer.VaultABI()
	require.NoError(t, err)

	vault := common.HexToAddress("0x1111111111111111111111111111111111111111")
	recipient := common.HexToAddress("0x2222222222222222222222222222222222222222")
	token := common.HexToAddress("0x3333333333333333333333333333333333333333")
	txHash := common.HexToHash("0xbee1")
	interactionID := uuid.New()

	base, err := NewBalancerV2Verifier(
		fakeEventView{events: []dbevent.ContractEvent{makeBalancerFlashLoanEvent(t, vaultABI, vault, recipient, token, txHash, 1000, 7)}},
		fakeObservedTxView{tx: &dbscanner.ObservedTransaction{ChainID: 1, TxHash: txHash, Status: 1}},
		fakeLegView{legs: []dbscanner.InteractionAssetLeg{{
			InteractionID:  interactionID,
			LegIndex:       0,
			AssetAddress:   token,
			AssetRole:      "borrowed",
			AmountBorrowed: big.NewInt(1000),
		}}},
	)
	require.NoError(t, err)

	traceVerifier, err := NewBalancerV2TraceVerifier(base, fakeTraceProvider{
		root: &scannertrace.CallFrame{
			From: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			To:   vault.Hex(),
			Calls: []scannertrace.CallFrame{
				{
					From:  vault.Hex(),
					To:    recipient.Hex(),
					Input: "0x12345678",
					Calls: []scannertrace.CallFrame{
						{
							From:  recipient.Hex(),
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
		Protocol:        scanner.ProtocolBalancerV2,
		Entrypoint:      "flashLoan",
		ProviderAddress: vault.Hex(),
		ReceiverAddress: recipient.Hex(),
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

func TestBalancerV2TraceVerifierMissingCallback(t *testing.T) {
	vaultABI, err := balancer.VaultABI()
	require.NoError(t, err)

	vault := common.HexToAddress("0x1111111111111111111111111111111111111111")
	recipient := common.HexToAddress("0x2222222222222222222222222222222222222222")
	token := common.HexToAddress("0x3333333333333333333333333333333333333333")
	txHash := common.HexToHash("0xbee2")
	interactionID := uuid.New()

	base, err := NewBalancerV2Verifier(
		fakeEventView{events: []dbevent.ContractEvent{makeBalancerFlashLoanEvent(t, vaultABI, vault, recipient, token, txHash, 1000, 7)}},
		fakeObservedTxView{tx: &dbscanner.ObservedTransaction{ChainID: 1, TxHash: txHash, Status: 1}},
		fakeLegView{legs: []dbscanner.InteractionAssetLeg{{
			InteractionID:  interactionID,
			LegIndex:       0,
			AssetAddress:   token,
			AssetRole:      "borrowed",
			AmountBorrowed: big.NewInt(1000),
		}}},
	)
	require.NoError(t, err)

	traceVerifier, err := NewBalancerV2TraceVerifier(base, fakeTraceProvider{
		root: &scannertrace.CallFrame{
			From: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			To:   vault.Hex(),
			Calls: []scannertrace.CallFrame{
				{
					From:  recipient.Hex(),
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
		Protocol:        scanner.ProtocolBalancerV2,
		Entrypoint:      "flashLoan",
		ProviderAddress: vault.Hex(),
		ReceiverAddress: recipient.Hex(),
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.Verified)
	require.False(t, result.Strict)
	require.NotNil(t, result.ExclusionReason)
	require.Equal(t, "missing_trace_callback", *result.ExclusionReason)
}

func TestBalancerV2TraceVerifierDowngradesWithoutRepaymentPath(t *testing.T) {
	vaultABI, err := balancer.VaultABI()
	require.NoError(t, err)

	vault := common.HexToAddress("0x1111111111111111111111111111111111111111")
	recipient := common.HexToAddress("0x2222222222222222222222222222222222222222")
	token := common.HexToAddress("0x3333333333333333333333333333333333333333")
	txHash := common.HexToHash("0xbee3")
	interactionID := uuid.New()

	base, err := NewBalancerV2Verifier(
		fakeEventView{events: []dbevent.ContractEvent{makeBalancerFlashLoanEvent(t, vaultABI, vault, recipient, token, txHash, 1000, 7)}},
		fakeObservedTxView{tx: &dbscanner.ObservedTransaction{ChainID: 1, TxHash: txHash, Status: 1}},
		fakeLegView{legs: []dbscanner.InteractionAssetLeg{{
			InteractionID:  interactionID,
			LegIndex:       0,
			AssetAddress:   token,
			AssetRole:      "borrowed",
			AmountBorrowed: big.NewInt(1000),
		}}},
	)
	require.NoError(t, err)

	traceVerifier, err := NewBalancerV2TraceVerifier(base, fakeTraceProvider{
		root: &scannertrace.CallFrame{
			From: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			To:   vault.Hex(),
			Calls: []scannertrace.CallFrame{
				{
					From:  vault.Hex(),
					To:    recipient.Hex(),
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
		Protocol:        scanner.ProtocolBalancerV2,
		Entrypoint:      "flashLoan",
		ProviderAddress: vault.Hex(),
		ReceiverAddress: recipient.Hex(),
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
