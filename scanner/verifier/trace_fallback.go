package verifier

import (
	"strings"

	"github.com/cpchain-network/flashloan-scanner/scanner"
)

func fallbackOnTraceUnavailable(
	result *scanner.VerifiedInteraction,
	legs []scanner.InteractionLeg,
	err error,
) (*scanner.VerifiedInteraction, []scanner.InteractionLeg, error) {
	if err == nil || !isTraceUnavailableError(err) {
		return nil, nil, err
	}
	if result == nil {
		return nil, legs, nil
	}
	note := appendNote(result.VerificationNotes, "trace unavailable; fell back to event-based verification")
	result.VerificationNotes = &note
	return result, legs, nil
}

func isTraceUnavailableError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "historical state is not available") ||
		strings.Contains(message, "debug_tracetransaction") ||
		strings.Contains(message, "the method debug_tracetransaction does not exist") ||
		strings.Contains(message, "method not found") ||
		strings.Contains(message, "unsupported")
}
