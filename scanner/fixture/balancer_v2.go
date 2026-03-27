package fixture

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/cpchain-network/flashloan-scanner/database"
	dbcommon "github.com/cpchain-network/flashloan-scanner/database/common"
	dbevent "github.com/cpchain-network/flashloan-scanner/database/event"
	dbscanner "github.com/cpchain-network/flashloan-scanner/database/scanner"
	"github.com/cpchain-network/flashloan-scanner/scanner/balancer"
)

type BalancerV2FlashLoanFixture struct {
	ChainID     uint64
	BlockNumber uint64
	BlockTime   uint64
	TxHash      common.Hash
	FromAddress common.Address
	Vault       common.Address
	Recipient   common.Address
	Token       common.Address
	Amount      *big.Int
	FeeAmount   *big.Int
	UserData    []byte
}

func DefaultBalancerV2FlashLoanFixture(chainID uint64, vault common.Address, blockNumber uint64) BalancerV2FlashLoanFixture {
	if blockNumber == 0 {
		blockNumber = 19000000
	}
	return BalancerV2FlashLoanFixture{
		ChainID:     chainID,
		BlockNumber: blockNumber,
		BlockTime:   1710001000,
		TxHash:      common.HexToHash("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
		FromAddress: common.HexToAddress("0x4000000000000000000000000000000000000004"),
		Vault:       vault,
		Recipient:   common.HexToAddress("0x5000000000000000000000000000000000000005"),
		Token:       common.HexToAddress("0x6000000000000000000000000000000000000006"),
		Amount:      big.NewInt(2500000000000000000),
		FeeAmount:   big.NewInt(1000000000000000),
		UserData:    []byte{0x12, 0x34},
	}
}

func LoadBalancerV2FlashLoanFixture(db *database.DB, fixture BalancerV2FlashLoanFixture) error {
	if db == nil {
		return fmt.Errorf("database is required")
	}

	header, eventRecord, observedTx, err := BuildBalancerV2FlashLoanFixture(fixture)
	if err != nil {
		return err
	}

	chainID := strconv.FormatUint(fixture.ChainID, 10)
	existingHeader, err := db.Blocks.ChainBlockHeaderByNumber(chainID, header.Number)
	if err != nil {
		return err
	}
	if existingHeader == nil {
		if err := db.Blocks.StoreBlockHeaders(chainID, []dbcommon.ChainBlockHeader{{BlockHeader: *header}}); err != nil {
			return err
		}
	}

	existingEvents, err := db.ContractEvents.ChainContractEventsByTxHash(chainID, fixture.TxHash)
	if err != nil {
		return err
	}
	if len(existingEvents) == 0 {
		if err := db.ContractEvents.StoreChainContractEvents(chainID, []dbevent.ChainContractEvent{{ContractEvent: eventRecord}}); err != nil {
			return err
		}
	}

	return db.ObservedTx.UpsertObservedTransactions([]dbscanner.ObservedTransaction{observedTx})
}

func BuildBalancerV2FlashLoanFixture(fixture BalancerV2FlashLoanFixture) (*dbcommon.BlockHeader, dbevent.ContractEvent, dbscanner.ObservedTransaction, error) {
	vaultABI, err := balancer.VaultABI()
	if err != nil {
		return nil, dbevent.ContractEvent{}, dbscanner.ObservedTransaction{}, err
	}

	method := vaultABI.Methods["flashLoan"]
	input, err := method.Inputs.Pack(
		fixture.Recipient,
		[]common.Address{fixture.Token},
		[]*big.Int{fixture.Amount},
		fixture.UserData,
	)
	if err != nil {
		return nil, dbevent.ContractEvent{}, dbscanner.ObservedTransaction{}, err
	}
	inputData := hexutil.Encode(append(method.ID, input...))

	headerObj := &types.Header{
		ParentHash: common.HexToHash("0x02"),
		Number:     new(big.Int).SetUint64(fixture.BlockNumber),
		Time:       fixture.BlockTime,
		GasLimit:   30000000,
	}
	header := dbcommon.BlockHeaderFromHeader(headerObj)

	eventSpec := vaultABI.Events["FlashLoan"]
	eventData, err := eventSpec.Inputs.NonIndexed().Pack(fixture.Amount, fixture.FeeAmount)
	if err != nil {
		return nil, dbevent.ContractEvent{}, dbscanner.ObservedTransaction{}, err
	}
	logRecord := &types.Log{
		Address: fixture.Vault,
		Topics: []common.Hash{
			eventSpec.ID,
			common.BytesToHash(fixture.Recipient.Bytes()),
			common.BytesToHash(fixture.Token.Bytes()),
		},
		Data:        eventData,
		TxHash:      fixture.TxHash,
		BlockHash:   header.Hash,
		BlockNumber: fixture.BlockNumber,
		Index:       0,
	}
	contractEvent := dbevent.ContractEventFromLog(logRecord, fixture.BlockTime, new(big.Int).SetUint64(fixture.BlockNumber))

	observedTx := dbscanner.ObservedTransaction{
		ChainID:        fixture.ChainID,
		TxHash:         fixture.TxHash,
		BlockNumber:    new(big.Int).SetUint64(fixture.BlockNumber),
		TxIndex:        0,
		FromAddress:    fixture.FromAddress,
		ToAddress:      addressPtr(fixture.Vault),
		Status:         1,
		Value:          big.NewInt(0),
		InputData:      inputData,
		MethodSelector: hexutil.Encode(method.ID),
		GasUsed:        big.NewInt(400000),
	}

	return &header, contractEvent, observedTx, nil
}
