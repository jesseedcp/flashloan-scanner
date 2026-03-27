package verifier

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	dbevent "github.com/cpchain-network/flashloan-scanner/database/event"
	dbscanner "github.com/cpchain-network/flashloan-scanner/database/scanner"
	"github.com/cpchain-network/flashloan-scanner/scanner"
	"github.com/cpchain-network/flashloan-scanner/scanner/balancer"
)

func TestBalancerV2VerifierStrict(t *testing.T) {
	vaultABI, err := balancer.VaultABI()
	require.NoError(t, err)

	vault := common.HexToAddress("0x1111111111111111111111111111111111111111")
	recipient := common.HexToAddress("0x2222222222222222222222222222222222222222")
	token := common.HexToAddress("0x3333333333333333333333333333333333333333")
	txHash := common.HexToHash("0xbeef")
	interactionID := uuid.New()

	verifier, err := NewBalancerV2Verifier(
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

	result, legs, err := verifier.Verify(context.Background(), 1, scanner.CandidateInteraction{
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
	require.True(t, result.RepaymentSeen)
	require.Len(t, legs, 1)
	require.Equal(t, "7", *legs[0].FeeAmount)
	require.Equal(t, "full_repayment", *legs[0].SettlementMode)
}

func makeBalancerFlashLoanEvent(t *testing.T, vaultABI abi.ABI, vault, recipient, token common.Address, txHash common.Hash, amount, feeAmount int64) dbevent.ContractEvent {
	t.Helper()
	eventSpec := vaultABI.Events["FlashLoan"]
	data, err := eventSpec.Inputs.NonIndexed().Pack(big.NewInt(amount), big.NewInt(feeAmount))
	require.NoError(t, err)
	log := &types.Log{
		Address: vault,
		Topics: []common.Hash{
			eventSpec.ID,
			common.BytesToHash(recipient.Bytes()),
			common.BytesToHash(token.Bytes()),
		},
		Data:        data,
		TxHash:      txHash,
		BlockNumber: 100,
		Index:       0,
	}
	return dbevent.ContractEventFromLog(log, 1000, big.NewInt(100))
}
