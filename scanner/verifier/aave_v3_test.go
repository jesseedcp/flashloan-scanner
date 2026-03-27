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
	"github.com/cpchain-network/flashloan-scanner/scanner/aave"
)

func TestAaveV3VerifierFlashLoanSimpleStrict(t *testing.T) {
	poolABI, err := aave.PoolABI()
	require.NoError(t, err)

	pool := common.HexToAddress("0x1111111111111111111111111111111111111111")
	receiver := common.HexToAddress("0x2222222222222222222222222222222222222222")
	asset := common.HexToAddress("0x3333333333333333333333333333333333333333")
	txHash := common.HexToHash("0xabc")
	interactionID := uuid.New()

	verifier, err := NewAaveV3Verifier(
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

	result, legs, err := verifier.Verify(context.Background(), 1, scanner.CandidateInteraction{
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
	require.True(t, result.SettlementSeen)
	require.True(t, result.RepaymentSeen)
	require.Len(t, legs, 1)
	require.Equal(t, "5", *legs[0].PremiumAmount)
	require.Equal(t, "full_repayment", *legs[0].SettlementMode)
}

func makeFlashLoanEvent(t *testing.T, poolABI abi.ABI, provider, target, asset common.Address, txHash common.Hash, amount int64, mode uint8, premium int64) dbevent.ContractEvent {
	t.Helper()
	eventSpec := poolABI.Events["FlashLoan"]
	data, err := eventSpec.Inputs.NonIndexed().Pack(common.Address{}, big.NewInt(amount), mode, big.NewInt(premium))
	require.NoError(t, err)
	log := &types.Log{
		Address: provider,
		Topics: []common.Hash{
			eventSpec.ID,
			common.BytesToHash(target.Bytes()),
			common.BytesToHash(asset.Bytes()),
			common.BigToHash(big.NewInt(0)),
		},
		Data:        data,
		TxHash:      txHash,
		BlockNumber: 100,
		Index:       0,
	}
	return dbevent.ContractEventFromLog(log, 1000, big.NewInt(100))
}

type fakeEventView struct {
	events []dbevent.ContractEvent
}

func (f fakeEventView) ChainContractEvent(string, uuid.UUID) (*dbevent.ContractEvent, error) {
	return nil, nil
}
func (f fakeEventView) ChainContractEventWithFilter(string, dbevent.ContractEvent) (*dbevent.ContractEvent, error) {
	return nil, nil
}
func (f fakeEventView) ChainContractEventsWithFilter(string, dbevent.ContractEvent, *big.Int, *big.Int) ([]dbevent.ContractEvent, error) {
	return nil, nil
}
func (f fakeEventView) ChainContractEventsByTxHash(string, common.Hash) ([]dbevent.ContractEvent, error) {
	return f.events, nil
}
func (f fakeEventView) ChainLatestContractEventWithFilter(string, dbevent.ContractEvent) (*dbevent.ContractEvent, error) {
	return nil, nil
}
func (f fakeEventView) ContractEventsWithFilter(string, dbevent.ContractEvent, *big.Int, *big.Int) ([]dbevent.ContractEvent, error) {
	return nil, nil
}

type fakeObservedTxView struct {
	tx *dbscanner.ObservedTransaction
}

func (f fakeObservedTxView) GetObservedTransaction(uint64, common.Hash) (*dbscanner.ObservedTransaction, error) {
	return f.tx, nil
}
func (f fakeObservedTxView) ListObservedTransactionsByBlockRange(uint64, *big.Int, *big.Int) ([]dbscanner.ObservedTransaction, error) {
	return nil, nil
}

type fakeLegView struct {
	legs []dbscanner.InteractionAssetLeg
}

func (f fakeLegView) ReplaceInteractionLegs(uuid.UUID, []dbscanner.InteractionAssetLeg) error {
	return nil
}
func (f fakeLegView) ListInteractionLegs(uuid.UUID) ([]dbscanner.InteractionAssetLeg, error) {
	return f.legs, nil
}

func uint8Ptr(v uint8) *uint8 {
	return &v
}
