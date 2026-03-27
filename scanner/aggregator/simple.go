package aggregator

import (
	"context"
	"math/big"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/common"

	dbscanner "github.com/cpchain-network/flashloan-scanner/database/scanner"
	"github.com/cpchain-network/flashloan-scanner/scanner"
)

type SimpleTxAggregator struct {
	interactionView dbscanner.ProtocolInteractionView
	txStore         dbscanner.FlashloanTransactionDB
}

func NewSimpleTxAggregator(
	interactionView dbscanner.ProtocolInteractionView,
	txStore dbscanner.FlashloanTransactionDB,
) *SimpleTxAggregator {
	return &SimpleTxAggregator{
		interactionView: interactionView,
		txStore:         txStore,
	}
}

func (a *SimpleTxAggregator) AggregateByTx(_ context.Context, chainID uint64, txHash string) (*scanner.TxSummary, error) {
	interactions, err := a.interactionView.ListProtocolInteractionsByTx(chainID, common.HexToHash(txHash))
	if err != nil {
		return nil, err
	}
	if len(interactions) == 0 {
		return nil, nil
	}

	protocolSet := make(map[scanner.Protocol]struct{})
	summary := &scanner.TxSummary{
		ChainID:                      chainID,
		TxHash:                       txHash,
		BlockNumber:                  interactions[0].BlockNumber.String(),
		ContainsCandidateInteraction: true,
		InteractionCount:             len(interactions),
	}

	for _, interaction := range interactions {
		protocol := scanner.Protocol(interaction.Protocol)
		protocolSet[protocol] = struct{}{}
		if interaction.Verified {
			summary.ContainsVerifiedInteraction = true
		}
		if interaction.Verified && interaction.Strict {
			summary.ContainsVerifiedStrictInteraction = true
			summary.StrictInteractionCount++
		}
	}

	summary.Protocols = make([]scanner.Protocol, 0, len(protocolSet))
	for protocol := range protocolSet {
		summary.Protocols = append(summary.Protocols, protocol)
	}
	sort.Slice(summary.Protocols, func(i, j int) bool {
		return summary.Protocols[i] < summary.Protocols[j]
	})
	summary.ProtocolCount = len(summary.Protocols)

	if err := a.txStore.UpsertFlashloanTransactions([]dbscanner.FlashloanTransaction{txSummaryToDB(*summary)}); err != nil {
		return nil, err
	}
	return summary, nil
}

func (a *SimpleTxAggregator) AggregateRange(ctx context.Context, chainID uint64, fromBlock, toBlock uint64) error {
	items, err := a.interactionView.ListCandidateInteractions(chainID, "", uintToBig(fromBlock), uintToBig(toBlock))
	if err != nil {
		return err
	}
	seen := make(map[string]struct{})
	for _, item := range items {
		txHash := item.TxHash.Hex()
		if _, ok := seen[txHash]; ok {
			continue
		}
		seen[txHash] = struct{}{}
		if _, err := a.AggregateByTx(ctx, chainID, txHash); err != nil {
			return err
		}
	}
	return nil
}

func txSummaryToDB(summary scanner.TxSummary) dbscanner.FlashloanTransaction {
	protocols := make([]string, 0, len(summary.Protocols))
	for _, protocol := range summary.Protocols {
		protocols = append(protocols, string(protocol))
	}
	return dbscanner.FlashloanTransaction{
		ChainID:                           summary.ChainID,
		TxHash:                            common.HexToHash(summary.TxHash),
		BlockNumber:                       uintToBigFromString(summary.BlockNumber),
		ContainsCandidateInteraction:      summary.ContainsCandidateInteraction,
		ContainsVerifiedInteraction:       summary.ContainsVerifiedInteraction,
		ContainsVerifiedStrictInteraction: summary.ContainsVerifiedStrictInteraction,
		InteractionCount:                  summary.InteractionCount,
		StrictInteractionCount:            summary.StrictInteractionCount,
		ProtocolCount:                     summary.ProtocolCount,
		Protocols:                         strings.Join(protocols, ","),
	}
}

func uintToBig(v uint64) *big.Int {
	return new(big.Int).SetUint64(v)
}

func uintToBigFromString(raw string) *big.Int {
	if raw == "" {
		return big.NewInt(0)
	}
	out, ok := new(big.Int).SetString(raw, 10)
	if !ok {
		return big.NewInt(0)
	}
	return out
}
