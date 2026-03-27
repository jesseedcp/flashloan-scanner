package fetcher

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	dbevent "github.com/cpchain-network/flashloan-scanner/database/event"
	"github.com/cpchain-network/flashloan-scanner/scanner"
	"github.com/cpchain-network/flashloan-scanner/scanner/store"
)

type EthereumTxFetcher struct {
	client    *ethclient.Client
	eventView dbevent.ContractEventsView
	store     *store.GormStore
}

func NewEthereumTxFetcher(rawClient *rpc.Client, eventView dbevent.ContractEventsView, txStore *store.GormStore) *EthereumTxFetcher {
	return &EthereumTxFetcher{
		client:    ethclient.NewClient(rawClient),
		eventView: eventView,
		store:     txStore,
	}
}

func (f *EthereumTxFetcher) FetchByTxHash(ctx context.Context, chainID uint64, txHash string) (*scanner.ObservedTransaction, error) {
	hash := commonHexToHash(txHash)
	tx, isPending, err := f.client.TransactionByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	receipt, err := f.client.TransactionReceipt(ctx, hash)
	if err != nil {
		return nil, err
	}
	if isPending {
		return nil, fmt.Errorf("transaction %s is still pending", txHash)
	}
	from, err := f.client.TransactionSender(ctx, tx, receipt.BlockHash, receipt.TransactionIndex)
	if err != nil {
		return nil, err
	}
	item := txToObserved(chainID, from.Hex(), tx, receipt)
	if err := f.store.UpsertObservedTransactions(ctx, []scanner.ObservedTransaction{item}); err != nil {
		return nil, err
	}
	return &item, nil
}

func (f *EthereumTxFetcher) FetchRange(ctx context.Context, chainID uint64, fromBlock, toBlock uint64) error {
	events, err := f.eventView.ContractEventsWithFilter(fmt.Sprintf("%d", chainID), dbevent.ContractEvent{}, big.NewInt(int64(fromBlock)), big.NewInt(int64(toBlock)))
	if err != nil {
		return err
	}
	seen := make(map[string]struct{})
	for _, event := range events {
		txHash := strings.ToLower(event.TransactionHash.Hex())
		if _, ok := seen[txHash]; ok {
			continue
		}
		seen[txHash] = struct{}{}
		item, err := f.FetchByTxHash(ctx, chainID, txHash)
		if err != nil {
			return err
		}
		_ = item
	}
	return nil
}

func (f *EthereumTxFetcher) LatestBlockNumber(ctx context.Context) (uint64, error) {
	return f.client.BlockNumber(ctx)
}

func txToObserved(chainID uint64, from string, tx *types.Transaction, receipt *types.Receipt) scanner.ObservedTransaction {
	var to string
	if tx.To() != nil {
		to = tx.To().Hex()
	}
	var selector string
	if data := tx.Data(); len(data) >= 4 {
		selector = hexutil.Encode(data[:4])
	}
	var effectiveGasPrice string
	if receipt.EffectiveGasPrice != nil {
		effectiveGasPrice = receipt.EffectiveGasPrice.String()
	}
	return scanner.ObservedTransaction{
		ChainID:           chainID,
		TxHash:            tx.Hash().Hex(),
		BlockNumber:       receipt.BlockNumber.String(),
		TxIndex:           uint64(receipt.TransactionIndex),
		FromAddress:       from,
		ToAddress:         to,
		Status:            uint8(receipt.Status),
		Value:             tx.Value().String(),
		InputData:         hexutil.Encode(tx.Data()),
		MethodSelector:    selector,
		GasUsed:           new(big.Int).SetUint64(receipt.GasUsed).String(),
		EffectiveGasPrice: effectiveGasPrice,
	}
}

func commonHexToHash(raw string) common.Hash {
	return common.HexToHash(raw)
}
