package service

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"

	scannertrace "github.com/cpchain-network/flashloan-scanner/scanner/trace"
)

const (
	traceStatusAvailable   = "available"
	traceStatusUnavailable = "unavailable"
	traceStatusError       = "error"
)

type TraceProviderSource interface {
	TraceProvider(chainID uint64) (scannertrace.Provider, error)
}

type TransactionTraceSummary struct {
	Status              string                     `json:"status"`
	Error               string                     `json:"error,omitempty"`
	RootFrameID         string                     `json:"root_frame_id,omitempty"`
	Frames              []TraceFrameDTO            `json:"frames,omitempty"`
	Sequence            []TraceSequenceStep        `json:"sequence,omitempty"`
	AssetFlows          []TraceAssetFlow           `json:"asset_flows,omitempty"`
	InteractionEvidence []TraceInteractionEvidence `json:"interaction_evidence,omitempty"`
}

type TraceFrameDTO struct {
	ID             string   `json:"id"`
	ParentID       string   `json:"parent_id,omitempty"`
	Depth          int      `json:"depth"`
	CallIndex      int      `json:"call_index"`
	Type           string   `json:"type"`
	From           string   `json:"from"`
	To             string   `json:"to"`
	MethodSelector string   `json:"method_selector,omitempty"`
	Error          string   `json:"error,omitempty"`
	RevertReason   string   `json:"revert_reason,omitempty"`
	TokenAction    string   `json:"token_action,omitempty"`
	AssetAddress   string   `json:"asset_address,omitempty"`
	TokenAmount    string   `json:"token_amount,omitempty"`
	FlowSource     string   `json:"flow_source,omitempty"`
	FlowTarget     string   `json:"flow_target,omitempty"`
	Tags           []string `json:"tags,omitempty"`
}

type TraceSequenceStep struct {
	Step           int    `json:"step"`
	FrameID        string `json:"frame_id"`
	ParentFrameID  string `json:"parent_frame_id,omitempty"`
	Depth          int    `json:"depth"`
	From           string `json:"from"`
	To             string `json:"to"`
	MethodSelector string `json:"method_selector,omitempty"`
	Label          string `json:"label"`
	Detail         string `json:"detail"`
	TokenAction    string `json:"token_action,omitempty"`
	AssetAddress   string `json:"asset_address,omitempty"`
	TokenAmount    string `json:"token_amount,omitempty"`
	Error          string `json:"error,omitempty"`
}

type TraceAssetFlow struct {
	FrameID      string `json:"frame_id"`
	Action       string `json:"action"`
	AssetAddress string `json:"asset_address"`
	Source       string `json:"source"`
	Target       string `json:"target"`
	Amount       string `json:"amount"`
}

type TraceInteractionEvidence struct {
	InteractionID       string   `json:"interaction_id"`
	Protocol            string   `json:"protocol"`
	Entrypoint          string   `json:"entrypoint"`
	Verdict             string   `json:"verdict"`
	CallbackSeen        bool     `json:"callback_seen"`
	SettlementSeen      bool     `json:"settlement_seen"`
	RepaymentSeen       bool     `json:"repayment_seen"`
	ContainsDebtOpening bool     `json:"contains_debt_opening"`
	CallbackFrameIDs    []string `json:"callback_frame_ids,omitempty"`
	CallbackSubtreeIDs  []string `json:"callback_subtree_ids,omitempty"`
	RepaymentFrameIDs   []string `json:"repayment_frame_ids,omitempty"`
	ExclusionReason     string   `json:"exclusion_reason,omitempty"`
	VerificationNotes   string   `json:"verification_notes,omitempty"`
	ProviderAddress     string   `json:"provider_address,omitempty"`
	ReceiverAddress     string   `json:"receiver_address,omitempty"`
}

type flatTraceFrame struct {
	frame        *scannertrace.CallFrame
	id           string
	parentID     string
	depth        int
	callIndex    int
	method       string
	tokenAction  string
	assetAddress string
	tokenAmount  string
	flowSource   string
	flowTarget   string
	tags         []string
}

func buildTransactionTraceSummary(
	ctx context.Context,
	source TraceProviderSource,
	chainID uint64,
	txHash string,
	interactions []InteractionDetailResult,
) *TransactionTraceSummary {
	if source == nil {
		return &TransactionTraceSummary{
			Status: traceStatusUnavailable,
			Error:  "trace provider is unavailable",
		}
	}

	provider, err := source.TraceProvider(chainID)
	if err != nil {
		return &TransactionTraceSummary{
			Status: traceStatusUnavailable,
			Error:  err.Error(),
		}
	}
	if provider == nil {
		return &TransactionTraceSummary{
			Status: traceStatusUnavailable,
			Error:  "trace provider is unavailable",
		}
	}

	root, err := provider.TraceTransaction(ctx, txHash)
	if err != nil {
		return &TransactionTraceSummary{
			Status: traceStatusError,
			Error:  err.Error(),
		}
	}
	if root == nil {
		return &TransactionTraceSummary{
			Status: traceStatusUnavailable,
			Error:  "trace returned no call frames",
		}
	}

	flattened := flattenTraceFrames(root)
	evidence := buildTraceInteractionEvidence(flattened, interactions)
	tagTraceFrames(flattened, evidence)

	frames := make([]TraceFrameDTO, 0, len(flattened))
	sequence := make([]TraceSequenceStep, 0, len(flattened))
	flows := make([]TraceAssetFlow, 0, len(flattened))

	for index, frame := range flattened {
		frames = append(frames, TraceFrameDTO{
			ID:             frame.id,
			ParentID:       frame.parentID,
			Depth:          frame.depth,
			CallIndex:      frame.callIndex,
			Type:           frame.frame.Type,
			From:           frame.frame.From,
			To:             frame.frame.To,
			MethodSelector: frame.method,
			Error:          frame.frame.Error,
			RevertReason:   frame.frame.RevertReason,
			TokenAction:    frame.tokenAction,
			AssetAddress:   frame.assetAddress,
			TokenAmount:    frame.tokenAmount,
			FlowSource:     frame.flowSource,
			FlowTarget:     frame.flowTarget,
			Tags:           append([]string(nil), frame.tags...),
		})
		sequence = append(sequence, TraceSequenceStep{
			Step:           index + 1,
			FrameID:        frame.id,
			ParentFrameID:  frame.parentID,
			Depth:          frame.depth,
			From:           frame.frame.From,
			To:             frame.frame.To,
			MethodSelector: frame.method,
			Label:          buildTraceStepLabel(frame),
			Detail:         buildTraceStepDetail(frame),
			TokenAction:    frame.tokenAction,
			AssetAddress:   frame.assetAddress,
			TokenAmount:    frame.tokenAmount,
			Error:          frame.frame.Error,
		})
		if frame.tokenAction != "" && frame.flowSource != "" && frame.flowTarget != "" {
			flows = append(flows, TraceAssetFlow{
				FrameID:      frame.id,
				Action:       frame.tokenAction,
				AssetAddress: frame.assetAddress,
				Source:       frame.flowSource,
				Target:       frame.flowTarget,
				Amount:       frame.tokenAmount,
			})
		}
	}

	return &TransactionTraceSummary{
		Status:              traceStatusAvailable,
		RootFrameID:         flattened[0].id,
		Frames:              frames,
		Sequence:            sequence,
		AssetFlows:          flows,
		InteractionEvidence: evidence,
	}
}

func flattenTraceFrames(root *scannertrace.CallFrame) []flatTraceFrame {
	if root == nil {
		return nil
	}
	out := make([]flatTraceFrame, 0, 32)
	var walk func(frame *scannertrace.CallFrame, id string, parentID string, depth int)
	walk = func(frame *scannertrace.CallFrame, id string, parentID string, depth int) {
		if frame == nil {
			return
		}
		method := traceMethodSelector(frame.Input)
		tokenAction, tokenAmount, flowSource, flowTarget := decodeTokenAction(frame.From, frame.To, method, frame.Input)
		out = append(out, flatTraceFrame{
			frame:        frame,
			id:           id,
			parentID:     parentID,
			depth:        depth,
			callIndex:    len(out),
			method:       method,
			tokenAction:  tokenAction,
			assetAddress: frame.To,
			tokenAmount:  tokenAmount,
			flowSource:   flowSource,
			flowTarget:   flowTarget,
		})
		for childIndex := range frame.Calls {
			childID := fmt.Sprintf("%s.%d", id, childIndex)
			walk(&frame.Calls[childIndex], childID, id, depth+1)
		}
	}
	walk(root, "trace-root", "", 0)
	return out
}

func buildTraceInteractionEvidence(frames []flatTraceFrame, interactions []InteractionDetailResult) []TraceInteractionEvidence {
	out := make([]TraceInteractionEvidence, 0, len(interactions))
	for _, interaction := range interactions {
		provider := strings.ToLower(strings.TrimSpace(interaction.ProviderAddress))
		receiver := strings.ToLower(strings.TrimSpace(interaction.ReceiverAddress))
		assets := make(map[string]struct{}, len(interaction.Legs))
		for _, leg := range interaction.Legs {
			if leg.AssetAddress != "" {
				assets[strings.ToLower(leg.AssetAddress)] = struct{}{}
			}
		}

		callbackFrameIDs := make([]string, 0, 2)
		repaymentFrameIDs := make([]string, 0, 4)
		callbackSubtreeIDs := make([]string, 0, 8)

		for _, frame := range frames {
			from := strings.ToLower(frame.frame.From)
			to := strings.ToLower(frame.frame.To)
			if frame.frame.Error == "" && from == provider && to == receiver {
				callbackFrameIDs = append(callbackFrameIDs, frame.id)
			}
			if matchesRepaymentFrame(frame, provider, assets) {
				repaymentFrameIDs = append(repaymentFrameIDs, frame.id)
			}
		}

		for _, callbackID := range callbackFrameIDs {
			callbackSubtreeIDs = append(callbackSubtreeIDs, collectSubtreeFrameIDs(frames, callbackID)...)
		}
		callbackSubtreeIDs = dedupeStrings(callbackSubtreeIDs)
		repaymentFrameIDs = dedupeStrings(repaymentFrameIDs)
		callbackFrameIDs = dedupeStrings(callbackFrameIDs)

		out = append(out, TraceInteractionEvidence{
			InteractionID:       interaction.InteractionID,
			Protocol:            interaction.Protocol,
			Entrypoint:          interaction.Entrypoint,
			Verdict:             verdictForInteraction(interaction),
			CallbackSeen:        interaction.CallbackSeen,
			SettlementSeen:      interaction.SettlementSeen,
			RepaymentSeen:       interaction.RepaymentSeen,
			ContainsDebtOpening: interaction.ContainsDebtOpening,
			CallbackFrameIDs:    callbackFrameIDs,
			CallbackSubtreeIDs:  callbackSubtreeIDs,
			RepaymentFrameIDs:   repaymentFrameIDs,
			ExclusionReason:     interaction.ExclusionReason,
			VerificationNotes:   interaction.VerificationNotes,
			ProviderAddress:     interaction.ProviderAddress,
			ReceiverAddress:     interaction.ReceiverAddress,
		})
	}
	return out
}

func tagTraceFrames(frames []flatTraceFrame, evidence []TraceInteractionEvidence) {
	callbackSubtree := map[string]struct{}{}
	repaymentFrames := map[string]struct{}{}
	for _, item := range evidence {
		for _, frameID := range item.CallbackSubtreeIDs {
			callbackSubtree[frameID] = struct{}{}
		}
		for _, frameID := range item.RepaymentFrameIDs {
			repaymentFrames[frameID] = struct{}{}
		}
	}

	for index := range frames {
		frame := &frames[index]
		tags := make([]string, 0, 4)
		if frame.tokenAction != "" {
			tags = append(tags, "token_action")
		}
		if _, ok := callbackSubtree[frame.id]; ok {
			tags = append(tags, "callback_path")
		}
		if _, ok := repaymentFrames[frame.id]; ok {
			tags = append(tags, "repayment_path")
		}
		if frame.frame.Error != "" {
			tags = append(tags, "error")
		}
		frame.tags = tags
	}
}

func matchesRepaymentFrame(frame flatTraceFrame, providerAddress string, assets map[string]struct{}) bool {
	if frame.frame == nil || frame.frame.Error != "" {
		return false
	}
	from := strings.ToLower(frame.frame.From)
	to := strings.ToLower(frame.frame.To)
	_, fromAsset := assets[from]
	_, toAsset := assets[to]
	if frame.tokenAction != "" && (fromAsset || toAsset || from == providerAddress || to == providerAddress) {
		return true
	}
	return false
}

func collectSubtreeFrameIDs(frames []flatTraceFrame, rootID string) []string {
	out := make([]string, 0, 8)
	prefix := rootID + "."
	for _, frame := range frames {
		if frame.id == rootID || strings.HasPrefix(frame.id, prefix) {
			out = append(out, frame.id)
		}
	}
	return out
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return values
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func buildTraceStepLabel(frame flatTraceFrame) string {
	if frame.tokenAction != "" {
		return frame.tokenAction
	}
	if frame.method != "" {
		return frame.method
	}
	if frame.frame.Error != "" {
		return "reverted call"
	}
	if frame.frame.Type != "" {
		return strings.ToLower(frame.frame.Type)
	}
	return "call"
}

func buildTraceStepDetail(frame flatTraceFrame) string {
	if frame.tokenAction != "" {
		return fmt.Sprintf("%s -> %s %s %s", frame.flowSource, frame.flowTarget, frame.tokenAction, frame.tokenAmount)
	}
	if frame.frame.Error != "" {
		if frame.frame.RevertReason != "" {
			return frame.frame.RevertReason
		}
		return frame.frame.Error
	}
	return fmt.Sprintf("%s -> %s", frame.frame.From, frame.frame.To)
}

func verdictForInteraction(interaction InteractionDetailResult) string {
	if interaction.Strict {
		return "strict"
	}
	if interaction.Verified {
		return "verified"
	}
	return "candidate"
}

func traceMethodSelector(input string) string {
	if len(input) < 10 || !strings.HasPrefix(input, "0x") {
		return ""
	}
	return strings.ToLower(input[:10])
}

func decodeTokenAction(caller string, tokenContract string, selector string, input string) (action string, amount string, source string, target string) {
	switch selector {
	case "0xa9059cbb":
		to := decodeTraceInputAddress(input, 0)
		value := decodeTraceInputUint(input, 1)
		if to == "" || value == "" {
			return "", "", "", ""
		}
		return "transfer", value, caller, to
	case "0x23b872dd":
		from := decodeTraceInputAddress(input, 0)
		to := decodeTraceInputAddress(input, 1)
		value := decodeTraceInputUint(input, 2)
		if from == "" || to == "" || value == "" {
			return "", "", "", ""
		}
		return "transferFrom", value, from, to
	case "0x095ea7b3":
		spender := decodeTraceInputAddress(input, 0)
		value := decodeTraceInputUint(input, 1)
		if spender == "" || value == "" {
			return "", "", "", ""
		}
		return "approve", value, caller, spender
	default:
		_ = tokenContract
		return "", "", "", ""
	}
}

func decodeTraceInputAddress(input string, argIndex int) string {
	slot := traceInputSlot(input, argIndex)
	if slot == "" {
		return ""
	}
	return common.HexToAddress("0x" + slot[24:64]).Hex()
}

func decodeTraceInputUint(input string, argIndex int) string {
	slot := traceInputSlot(input, argIndex)
	if slot == "" {
		return ""
	}
	value, ok := new(big.Int).SetString(slot, 16)
	if !ok {
		return ""
	}
	return value.String()
}

func traceInputSlot(input string, argIndex int) string {
	if !strings.HasPrefix(input, "0x") {
		return ""
	}
	payload := input[10:]
	start := argIndex * 64
	end := start + 64
	if len(payload) < end {
		return ""
	}
	return payload[start:end]
}
