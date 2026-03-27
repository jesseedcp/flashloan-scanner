package verifier

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cpchain-network/flashloan-scanner/scanner"
)

func TestFallbackOnTraceUnavailable(t *testing.T) {
	result := &scanner.VerifiedInteraction{
		Verified: true,
		Strict:   true,
	}
	legs := []scanner.InteractionLeg{{InteractionID: "abc"}}

	fallbackResult, fallbackLegs, err := fallbackOnTraceUnavailable(result, legs, errors.New("historical state is not available"))
	require.NoError(t, err)
	require.NotNil(t, fallbackResult)
	require.Equal(t, legs, fallbackLegs)
	require.True(t, fallbackResult.Verified)
	require.True(t, fallbackResult.Strict)
	require.NotNil(t, fallbackResult.VerificationNotes)
	require.Contains(t, *fallbackResult.VerificationNotes, "trace unavailable")
}

func TestFallbackOnTraceUnavailableNonTraceError(t *testing.T) {
	result := &scanner.VerifiedInteraction{
		Verified: true,
		Strict:   true,
	}

	fallbackResult, fallbackLegs, err := fallbackOnTraceUnavailable(result, nil, errors.New("rpc dial timeout"))
	require.Nil(t, fallbackResult)
	require.Nil(t, fallbackLegs)
	require.EqualError(t, err, "rpc dial timeout")
}
