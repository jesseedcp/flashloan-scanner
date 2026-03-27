package verifier

import (
	"context"
	"fmt"
	"strings"

	"github.com/cpchain-network/flashloan-scanner/scanner"
	scannertrace "github.com/cpchain-network/flashloan-scanner/scanner/trace"
)

const (
	erc20ApproveSelector      = "0x095ea7b3"
	erc20TransferSelector     = "0xa9059cbb"
	erc20TransferFromSelector = "0x23b872dd"
)

type AaveV3TraceVerifier struct {
	base          *AaveV3Verifier
	traceProvider scannertrace.Provider
}

func NewAaveV3TraceVerifier(base *AaveV3Verifier, traceProvider scannertrace.Provider) (*AaveV3TraceVerifier, error) {
	if base == nil {
		return nil, fmt.Errorf("base verifier is required")
	}
	if traceProvider == nil {
		return nil, fmt.Errorf("trace provider is required")
	}
	return &AaveV3TraceVerifier{
		base:          base,
		traceProvider: traceProvider,
	}, nil
}

func (v *AaveV3TraceVerifier) Protocol() scanner.Protocol {
	return scanner.ProtocolAaveV3
}

func (v *AaveV3TraceVerifier) Verify(ctx context.Context, chainID uint64, interaction scanner.CandidateInteraction) (*scanner.VerifiedInteraction, []scanner.InteractionLeg, error) {
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
	repaymentPathSeen := hasAaveRepaymentPath(traceRoot, providerAddress, legs)

	baseResult.CallbackSeen = callbackSeen
	baseResult.SettlementSeen = callbackSeen
	baseResult.RepaymentSeen = repaymentPathSeen && !baseResult.ContainsDebtOpening

	if baseResult.ContainsDebtOpening {
		baseResult.Strict = false
		note := appendNote(baseResult.VerificationNotes, "trace confirms non-strict path because debt opening is present")
		baseResult.VerificationNotes = &note
		return baseResult, legs, nil
	}

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
	note := appendNote(baseResult.VerificationNotes, "trace-backed strict verification passed via callback and repayment-path evidence")
	baseResult.VerificationNotes = &note
	return baseResult, annotateAaveTraceLegs(legs), nil
}

func hasSuccessfulDirectCall(root *scannertrace.CallFrame, fromAddress, toAddress string) bool {
	if root == nil {
		return false
	}
	for _, child := range root.Calls {
		if strings.ToLower(child.From) == fromAddress && strings.ToLower(child.To) == toAddress && child.Error == "" {
			return true
		}
		if hasSuccessfulDirectCall(&child, fromAddress, toAddress) {
			return true
		}
	}
	return false
}

func hasAaveRepaymentPath(root *scannertrace.CallFrame, providerAddress string, legs []scanner.InteractionLeg) bool {
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

func hasRepaymentCall(frame *scannertrace.CallFrame, providerAddress string, assets map[string]struct{}) bool {
	if frame == nil {
		return false
	}
	if frame.Error == "" {
		from := strings.ToLower(frame.From)
		to := strings.ToLower(frame.To)
		selector := methodSelector(frame.Input)
		_, fromAsset := assets[from]
		_, toAsset := assets[to]
		if (fromAsset || toAsset) && (selector == erc20ApproveSelector || selector == erc20TransferSelector || selector == erc20TransferFromSelector) {
			return true
		}
		if from == providerAddress || to == providerAddress {
			if fromAsset || toAsset {
				return true
			}
		}
	}
	for i := range frame.Calls {
		if hasRepaymentCall(&frame.Calls[i], providerAddress, assets) {
			return true
		}
	}
	return false
}

func annotateAaveTraceLegs(legs []scanner.InteractionLeg) []scanner.InteractionLeg {
	out := make([]scanner.InteractionLeg, 0, len(legs))
	for _, leg := range legs {
		leg.EventSeen = true
		if leg.SettlementMode == nil && !leg.OpenedDebt {
			leg.SettlementMode = stringPtr("full_repayment")
		}
		if !leg.OpenedDebt {
			leg.StrictLeg = true
		}
		out = append(out, leg)
	}
	return out
}

func methodSelector(input string) string {
	if len(input) < 10 || !strings.HasPrefix(input, "0x") {
		return ""
	}
	return strings.ToLower(input[:10])
}

func appendNote(existing *string, extra string) string {
	if existing == nil || *existing == "" {
		return extra
	}
	return *existing + "; " + extra
}
