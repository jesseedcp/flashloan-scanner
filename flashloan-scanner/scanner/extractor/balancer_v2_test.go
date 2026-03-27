package extractor

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	dbevent "github.com/cpchain-network/flashloan-scanner/database/event"
	"github.com/cpchain-network/flashloan-scanner/scanner"
	"github.com/cpchain-network/flashloan-scanner/scanner/balancer"
	scannerregistry "github.com/cpchain-network/flashloan-scanner/scanner/registry"
)

func TestBalancerV2CandidateExtractorFallsBackToVaultEvent(t *testing.T) {
	reg := scannerregistry.NewMemoryRegistry()
	vault := common.HexToAddress("0xba12222222228d8ba445958a75a0704d566bf2c8")
	reg.AddProtocolAddress(scannerregistry.AddressRecord{
		ChainID:     1,
		Protocol:    scanner.ProtocolBalancerV2,
		AddressRole: "vault",
		Address:     vault.Hex(),
	})

	recipient := common.HexToAddress("0x2222222222222222222222222222222222222222")
	token := common.HexToAddress("0x3333333333333333333333333333333333333333")
	txHash := common.HexToHash("0xcfb9ce1213d710127eadd563ae61d58d8a35a789243ae8fee2712c05cd1110d3")
	router := common.HexToAddress("0x2face030001ef666668ba8a423470f2470cce954")

	extractor, err := NewBalancerV2CandidateExtractor(reg, fakeBalancerEventView{
		events: []dbevent.ContractEvent{
			makeBalancerFlashLoanEvent(t, vault, recipient, token, txHash, 1000, 0),
		},
	})
	require.NoError(t, err)

	interactions, legs, err := extractor.Extract(context.Background(), 1, []scanner.ObservedTransaction{{
		ChainID:        1,
		TxHash:         txHash.Hex(),
		BlockNumber:    "22486336",
		ToAddress:      router.Hex(),
		MethodSelector: "0xdeadbeef",
	}})
	require.NoError(t, err)
	require.Len(t, interactions, 1)
	require.Len(t, legs, 1)
	require.Equal(t, "flashLoan:event", interactions[0].Entrypoint)
	require.Equal(t, scanner.CandidateLevelWeak, interactions[0].CandidateLevel)
	require.Equal(t, vault.Hex(), interactions[0].ProviderAddress)
	require.Equal(t, recipient.Hex(), interactions[0].ReceiverAddress)
	require.Equal(t, token.Hex(), legs[0].AssetAddress)
	require.NotNil(t, legs[0].AmountBorrowed)
	require.Equal(t, "1000", *legs[0].AmountBorrowed)
}

type fakeBalancerEventView struct {
	events []dbevent.ContractEvent
}

func (f fakeBalancerEventView) ChainContractEvent(string, uuid.UUID) (*dbevent.ContractEvent, error) {
	return nil, nil
}

func (f fakeBalancerEventView) ChainContractEventWithFilter(string, dbevent.ContractEvent) (*dbevent.ContractEvent, error) {
	return nil, nil
}

func (f fakeBalancerEventView) ChainContractEventsWithFilter(string, dbevent.ContractEvent, *big.Int, *big.Int) ([]dbevent.ContractEvent, error) {
	return nil, nil
}

func (f fakeBalancerEventView) ChainContractEventsByTxHash(string, common.Hash) ([]dbevent.ContractEvent, error) {
	return f.events, nil
}

func (f fakeBalancerEventView) ChainLatestContractEventWithFilter(string, dbevent.ContractEvent) (*dbevent.ContractEvent, error) {
	return nil, nil
}

func (f fakeBalancerEventView) ContractEventsWithFilter(string, dbevent.ContractEvent, *big.Int, *big.Int) ([]dbevent.ContractEvent, error) {
	return nil, nil
}

func makeBalancerFlashLoanEvent(t *testing.T, vault, recipient, token common.Address, txHash common.Hash, amount, feeAmount int64) dbevent.ContractEvent {
	t.Helper()
	vaultABI, err := balancer.VaultABI()
	require.NoError(t, err)

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
		BlockNumber: 22486336,
		Index:       0,
	}
	return dbevent.ContractEventFromLog(log, 1710001000, big.NewInt(22486336))
}
