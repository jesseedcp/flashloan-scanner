package verifier

import (
	"context"
	"fmt"
	"strings"

	"github.com/cpchain-network/flashloan-scanner/scanner"
	scannertrace "github.com/cpchain-network/flashloan-scanner/scanner/trace"
)

type BalancerV2TraceVerifier struct {
	base          *BalancerV2Verifier
	traceProvider scannertrace.Provider
}

func NewBalancerV2TraceVerifier(base *BalancerV2Verifier, traceProvider scannertrace.Provider) (*BalancerV2TraceVerifier, error) {
	if base == nil {
		return nil, fmt.Errorf("base verifier is required")
	}
	if traceProvider == nil {
		return nil, fmt.Errorf("trace provider is required")
	}
	return &BalancerV2TraceVerifier{
		base:          base,
		traceProvider: traceProvider,
	}, nil
}

func (v *BalancerV2TraceVerifier) Protocol() scanner.Protocol {
	return scanner.ProtocolBalancerV2
}

func (v *BalancerV2TraceVerifier) Verify(ctx context.Context, chainID uint64, interaction scanner.CandidateInteraction) (*scanner.VerifiedInteraction, []scanner.InteractionLeg, error) {
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

	providerAddress := strings.ToLower(interaction.ProviderAddress)
	receiverAddress := strings.ToLower(interaction.ReceiverAddress)
	callbackSeen := hasSuccessfulDirectCall(traceRoot, providerAddress, receiverAddress)
	repaymentPathSeen := hasBalancerRepaymentPath(traceRoot, providerAddress, legs)

	baseResult.CallbackSeen = callbackSeen
	baseResult.SettlementSeen = callbackSeen
	baseResult.RepaymentSeen = repaymentPathSeen

	if !callbackSeen {
		baseResult.Verified = false
		baseResult.Strict = false
		reason := "missing_trace_callback"
		baseResult.ExclusionReason = &reason
		return baseResult, legs, nil
	}

	if !repaymentPathSeen {
		baseResult.Strict = false
		note := appendNote(baseResult.VerificationNotes, "trace callback seen but repayment path evidence not found; downgraded from strict")
		baseResult.VerificationNotes = &note
		return baseResult, legs, nil
	}

	baseResult.Strict = true
	note := appendNote(baseResult.VerificationNotes, "trace-backed strict verification passed via Balancer callback and repayment-path evidence")
	baseResult.VerificationNotes = &note
	return baseResult, annotateBalancerTraceLegs(legs), nil
}

func hasBalancerRepaymentPath(root *scannertrace.CallFrame, providerAddress string, legs []scanner.InteractionLeg) bool {
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
	return hasRepaymentCall(root, strings.ToLower(providerAddress), assets)
}

func annotateBalancerTraceLegs(legs []scanner.InteractionLeg) []scanner.InteractionLeg {
	out := make([]scanner.InteractionLeg, 0, len(legs))
	for _, leg := range legs {
		leg.EventSeen = true
		leg.StrictLeg = true
		if leg.SettlementMode == nil {
			leg.SettlementMode = stringPtr("full_repayment")
		}
		out = append(out, leg)
	}
	return out
}
