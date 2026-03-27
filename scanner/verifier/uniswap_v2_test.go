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
	"github.com/cpchain-network/flashloan-scanner/scanner/uniswapv2"
)

func TestUniswapV2VerifierStrict(t *testing.T) {
	pairABI, err := uniswapv2.PairABI()
	require.NoError(t, err)

	pair := common.HexToAddress("0x1111111111111111111111111111111111111111")
	receiver := common.HexToAddress("0x2222222222222222222222222222222222222222")
	txHash := common.HexToHash("0xcafe")
	interactionID := uuid.New()
	dataNonEmpty := true
	tokenSide := "token0"

	verifier, err := NewUniswapV2Verifier(
		fakeEventView{events: []dbevent.ContractEvent{makeUniswapSwapEvent(t, pairABI, pair, receiver, txHash, 1100, 0, 1000, 0)}},
		fakeObservedTxView{tx: &dbscanner.ObservedTransaction{ChainID: 1, TxHash: txHash, Status: 1}},
		fakeLegView{legs: []dbscanner.InteractionAssetLeg{{
			InteractionID:  interactionID,
			LegIndex:       0,
			AssetAddress:   common.HexToAddress("0x3333333333333333333333333333333333333333"),
			AssetRole:      "borrowed",
			TokenSide:      &tokenSide,
			AmountBorrowed: big.NewInt(1000),
		}}},
	)
	require.NoError(t, err)

	result, legs, err := verifier.Verify(context.Background(), 1, scanner.CandidateInteraction{
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
	require.Len(t, legs, 1)
	require.Equal(t, "1100", *legs[0].AmountIn)
	require.Equal(t, "invariant_restored", *legs[0].SettlementMode)
}

func makeUniswapSwapEvent(t *testing.T, pairABI abi.ABI, pair, to common.Address, txHash common.Hash, amount0In, amount1In, amount0Out, amount1Out int64) dbevent.ContractEvent {
	t.Helper()
	eventSpec := pairABI.Events["Swap"]
	data, err := eventSpec.Inputs.NonIndexed().Pack(
		big.NewInt(amount0In),
		big.NewInt(amount1In),
		big.NewInt(amount0Out),
		big.NewInt(amount1Out),
	)
	require.NoError(t, err)
	log := &types.Log{
		Address: pair,
		Topics: []common.Hash{
			eventSpec.ID,
			common.BytesToHash(common.Address{}.Bytes()),
			common.BytesToHash(to.Bytes()),
		},
		Data:        data,
		TxHash:      txHash,
		BlockNumber: 100,
		Index:       0,
	}
	return dbevent.ContractEventFromLog(log, 1000, big.NewInt(100))
}
