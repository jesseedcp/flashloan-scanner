package verifier

import (
	"context"
	"fmt"
	"strings"

	"github.com/cpchain-network/flashloan-scanner/scanner"
	scannertrace "github.com/cpchain-network/flashloan-scanner/scanner/trace"
)

type UniswapV2TraceVerifier struct {
	base          *UniswapV2Verifier
	traceProvider scannertrace.Provider
}

func NewUniswapV2TraceVerifier(base *UniswapV2Verifier, traceProvider scannertrace.Provider) (*UniswapV2TraceVerifier, error) {
	if base == nil {
		return nil, fmt.Errorf("base verifier is required")
	}
	if traceProvider == nil {
		return nil, fmt.Errorf("trace provider is required")
	}
	return &UniswapV2TraceVerifier{
		base:          base,
		traceProvider: traceProvider,
	}, nil
}

func (v *UniswapV2TraceVerifier) Protocol() scanner.Protocol {
	return scanner.ProtocolUniswapV2
}

func (v *UniswapV2TraceVerifier) Verify(ctx context.Context, chainID uint64, interaction scanner.CandidateInteraction) (*scanner.VerifiedInteraction, []scanner.InteractionLeg, error) {
	baseResult, legs, err := v.base.Verify(ctx, chainID, interaction)
	if err != nil {
		return nil, nil, err
	}
	if baseResult == nil || !baseResult.Verified {
		return baseResult, legs, nil
	}

	traceRoot, err := v.traceProvider.TraceTransaction(ctx, interaction.TxHash)
	if err != nil {
		if fallbackResult, fallbackLegs, fallbackErr := fallbackOnTraceUnavailable(baseResult, legs, err); fallbackErr == nil {
			return fallbackResult, fallbackLegs, nil
		}
		return nil, nil, err
	}
	if traceRoot == nil {
		reason := "missing_trace"
		baseResult.Verified = false
		baseResult.Strict = false
		baseResult.CallbackSeen = false
		baseResult.RepaymentSeen = false
		baseResult.ExclusionReason = &reason
		return baseResult, legs, nil
	}

	pairAddress := strings.ToLower(interaction.ProviderAddress)
	receiverAddress := strings.ToLower(interaction.ReceiverAddress)
	callbackSeen := hasSuccessfulDirectCall(traceRoot, pairAddress, receiverAddress)
	invariantPathSeen := hasUniswapInvariantPath(traceRoot, pairAddress, legs)

	baseResult.CallbackSeen = callbackSeen
	baseResult.SettlementSeen = callbackSeen
	baseResult.RepaymentSeen = invariantPathSeen

	if !callbackSeen {
		baseResult.Verified = false
		baseResult.Strict = false
		reason := "missing_trace_callback"
		baseResult.ExclusionReason = &reason
		return baseResult, legs, nil
	}

	if !invariantPathSeen {
		baseResult.Strict = false
		note := appendNote(baseResult.VerificationNotes, "trace callback seen but invariant-restoration path evidence not found; downgraded from strict")
		baseResult.VerificationNotes = &note
		return baseResult, legs, nil
	}

	baseResult.Strict = true
	note := appendNote(baseResult.VerificationNotes, "trace-backed strict verification passed via Uniswap V2 callback and invariant-path evidence")
	baseResult.VerificationNotes = &note
	return baseResult, annotateUniswapTraceLegs(legs), nil
}

func hasUniswapInvariantPath(root *scannertrace.CallFrame, pairAddress string, legs []scanner.InteractionLeg) bool {
	if root == nil {
		return false
	}
	assets := make(map[string]struct{})
	for _, leg := range legs {
		if leg.AssetAddress != "" {
			assets[strings.ToLower(leg.AssetAddress)] = struct{}{}
		}
	}
	if len(assets) == 0 {
		return false
	}
	return hasRepaymentCall(root, strings.ToLower(pairAddress), assets)
}

func annotateUniswapTraceLegs(legs []scanner.InteractionLeg) []scanner.InteractionLeg {
	out := make([]scanner.InteractionLeg, 0, len(legs))
	for _, leg := range legs {
		leg.EventSeen = true
		leg.StrictLeg = true
		if leg.SettlementMode == nil {
			leg.SettlementMode = stringPtr("invariant_restored")
		}
		out = append(out, leg)
	}
	return out
}
