package extractor

import (
	"context"

	"github.com/cpchain-network/flashloan-scanner/scanner"
)

type CandidateExtractor interface {
	Protocol() scanner.Protocol
	Extract(ctx context.Context, chainID uint64, txs []scanner.ObservedTransaction) ([]scanner.CandidateInteraction, []scanner.InteractionLeg, error)
}
