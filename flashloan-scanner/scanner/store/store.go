package store

import (
	"context"

	"github.com/cpchain-network/flashloan-scanner/scanner"
)

type TransactionStore interface {
	UpsertObservedTransactions(ctx context.Context, txs []scanner.ObservedTransaction) error
	ListObservedTransactionsByBlockRange(ctx context.Context, chainID uint64, fromBlock, toBlock uint64) ([]scanner.ObservedTransaction, error)
}

type InteractionStore interface {
	UpsertInteractions(ctx context.Context, items []scanner.CandidateInteraction) error
	UpdateVerificationResult(ctx context.Context, result scanner.VerifiedInteraction) error
	ListCandidateInteractions(ctx context.Context, chainID uint64, protocol scanner.Protocol, fromBlock, toBlock uint64) ([]scanner.CandidateInteraction, error)
}

type LegStore interface {
	ReplaceInteractionLegs(ctx context.Context, interactionID string, legs []scanner.InteractionLeg) error
}

type TxSummaryStore interface {
	UpsertTxSummary(ctx context.Context, summary scanner.TxSummary) error
}
