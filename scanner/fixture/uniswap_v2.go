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
	"github.com/cpchain-network/flashloan-scanner/scanner/uniswapv2"
)

type UniswapV2FlashSwapFixture struct {
	ChainID      uint64
	BlockNumber  uint64
	BlockTime    uint64
	TxHash       common.Hash
	FromAddress  common.Address
	Factory      common.Address
	Pair         common.Address
	Receiver     common.Address
	Token0       common.Address
	Token1       common.Address
	Amount0In    *big.Int
	Amount1In    *big.Int
	Amount0Out   *big.Int
	Amount1Out   *big.Int
	CallbackData []byte
}

func DefaultUniswapV2FlashSwapFixture(chainID uint64, factory common.Address, pair common.Address, token0 common.Address, token1 common.Address, blockNumber uint64) UniswapV2FlashSwapFixture {
	if blockNumber == 0 {
		blockNumber = 19000000
	}
	return UniswapV2FlashSwapFixture{
		ChainID:      chainID,
		BlockNumber:  blockNumber,
		BlockTime:    1710002000,
		TxHash:       common.HexToHash("0xcccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"),
		FromAddress:  common.HexToAddress("0x7000000000000000000000000000000000000007"),
		Factory:      factory,
		Pair:         pair,
		Receiver:     common.HexToAddress("0x8000000000000000000000000000000000000008"),
		Token0:       token0,
		Token1:       token1,
		Amount0In:    big.NewInt(1010000000000000000),
		Amount1In:    big.NewInt(0),
		Amount0Out:   big.NewInt(1000000000000000000),
		Amount1Out:   big.NewInt(0),
		CallbackData: []byte{0xab, 0xcd, 0xef},
	}
}

func LoadUniswapV2FlashSwapFixture(db *database.DB, fixture UniswapV2FlashSwapFixture) error {
	if db == nil {
		return fmt.Errorf("database is required")
	}

	header, eventRecord, observedTx, err := BuildUniswapV2FlashSwapFixture(fixture)
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

func BuildUniswapV2FlashSwapFixture(fixture UniswapV2FlashSwapFixture) (*dbcommon.BlockHeader, dbevent.ContractEvent, dbscanner.ObservedTransaction, error) {
	pairABI, err := uniswapv2.PairABI()
	if err != nil {
		return nil, dbevent.ContractEvent{}, dbscanner.ObservedTransaction{}, err
	}

	method := pairABI.Methods["swap"]
	input, err := method.Inputs.Pack(
		fixture.Amount0Out,
		fixture.Amount1Out,
		fixture.Receiver,
		fixture.CallbackData,
	)
	if err != nil {
		return nil, dbevent.ContractEvent{}, dbscanner.ObservedTransaction{}, err
	}
	inputData := hexutil.Encode(append(method.ID, input...))

	headerObj := &types.Header{
		ParentHash: common.HexToHash("0x03"),
		Number:     new(big.Int).SetUint64(fixture.BlockNumber),
		Time:       fixture.BlockTime,
		GasLimit:   30000000,
	}
	header := dbcommon.BlockHeaderFromHeader(headerObj)

	eventSpec := pairABI.Events["Swap"]
	eventData, err := eventSpec.Inputs.NonIndexed().Pack(
		fixture.Amount0In,
		fixture.Amount1In,
		fixture.Amount0Out,
		fixture.Amount1Out,
	)
	if err != nil {
		return nil, dbevent.ContractEvent{}, dbscanner.ObservedTransaction{}, err
	}
	logRecord := &types.Log{
		Address: fixture.Pair,
		Topics: []common.Hash{
			eventSpec.ID,
			common.BytesToHash(fixture.FromAddress.Bytes()),
			common.BytesToHash(fixture.Receiver.Bytes()),
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
		ToAddress:      addressPtr(fixture.Pair),
		Status:         1,
		Value:          big.NewInt(0),
		InputData:      inputData,
		MethodSelector: hexutil.Encode(method.ID),
		GasUsed:        big.NewInt(350000),
	}

	return &header, contractEvent, observedTx, nil
}
