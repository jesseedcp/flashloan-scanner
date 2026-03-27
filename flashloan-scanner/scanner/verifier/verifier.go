package verifier

import (
	"context"

	"github.com/cpchain-network/flashloan-scanner/scanner"
)

type InteractionVerifier interface {
	Protocol() scanner.Protocol
	Verify(ctx context.Context, chainID uint64, interaction scanner.CandidateInteraction) (*scanner.VerifiedInteraction, []scanner.InteractionLeg, error)
}
