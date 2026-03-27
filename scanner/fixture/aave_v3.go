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
	"github.com/cpchain-network/flashloan-scanner/scanner/aave"
)

type AaveV3FlashLoanSimpleFixture struct {
	ChainID      uint64
	BlockNumber  uint64
	BlockTime    uint64
	TxHash       common.Hash
	FromAddress  common.Address
	PoolAddress  common.Address
	Receiver     common.Address
	Asset        common.Address
	Amount       *big.Int
	Premium      *big.Int
	ReferralCode uint16
}

func DefaultAaveV3FlashLoanSimpleFixture(chainID uint64, poolAddress common.Address, blockNumber uint64) AaveV3FlashLoanSimpleFixture {
	if blockNumber == 0 {
		blockNumber = 19000000
	}
	return AaveV3FlashLoanSimpleFixture{
		ChainID:      chainID,
		BlockNumber:  blockNumber,
		BlockTime:    1710000000,
		TxHash:       common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		FromAddress:  common.HexToAddress("0x1000000000000000000000000000000000000001"),
		PoolAddress:  poolAddress,
		Receiver:     common.HexToAddress("0x2000000000000000000000000000000000000002"),
		Asset:        common.HexToAddress("0x3000000000000000000000000000000000000003"),
		Amount:       big.NewInt(1000000000000000000),
		Premium:      big.NewInt(900000000000000),
		ReferralCode: 0,
	}
}

func LoadAaveV3FlashLoanSimpleFixture(db *database.DB, fixture AaveV3FlashLoanSimpleFixture) error {
	if db == nil {
		return fmt.Errorf("database is required")
	}

	header, eventRecord, observedTx, err := BuildAaveV3FlashLoanSimpleFixture(fixture)
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

func BuildAaveV3FlashLoanSimpleFixture(fixture AaveV3FlashLoanSimpleFixture) (*dbcommon.BlockHeader, dbevent.ContractEvent, dbscanner.ObservedTransaction, error) {
	poolABI, err := aave.PoolABI()
	if err != nil {
		return nil, dbevent.ContractEvent{}, dbscanner.ObservedTransaction{}, err
	}

	method := poolABI.Methods["flashLoanSimple"]
	input, err := method.Inputs.Pack(
		fixture.Receiver,
		fixture.Asset,
		fixture.Amount,
		[]byte{},
		fixture.ReferralCode,
	)
	if err != nil {
		return nil, dbevent.ContractEvent{}, dbscanner.ObservedTransaction{}, err
	}
	inputData := hexutil.Encode(append(method.ID, input...))

	headerObj := &types.Header{
		ParentHash: common.HexToHash("0x01"),
		Number:     new(big.Int).SetUint64(fixture.BlockNumber),
		Time:       fixture.BlockTime,
		GasLimit:   30000000,
	}
	header := dbcommon.BlockHeaderFromHeader(headerObj)

	eventSpec := poolABI.Events["FlashLoan"]
	eventData, err := eventSpec.Inputs.NonIndexed().Pack(
		fixture.FromAddress,
		fixture.Amount,
		uint8(0),
		fixture.Premium,
	)
	if err != nil {
		return nil, dbevent.ContractEvent{}, dbscanner.ObservedTransaction{}, err
	}
	logRecord := &types.Log{
		Address: fixture.PoolAddress,
		Topics: []common.Hash{
			eventSpec.ID,
			common.BytesToHash(fixture.Receiver.Bytes()),
			common.BytesToHash(fixture.Asset.Bytes()),
			common.BigToHash(new(big.Int).SetUint64(uint64(fixture.ReferralCode))),
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
		ToAddress:      addressPtr(fixture.PoolAddress),
		Status:         1,
		Value:          big.NewInt(0),
		InputData:      inputData,
		MethodSelector: hexutil.Encode(method.ID),
		GasUsed:        big.NewInt(500000),
	}

	return &header, contractEvent, observedTx, nil
}

func addressPtr(address common.Address) *common.Address {
	return &address
}
