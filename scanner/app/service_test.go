package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cpchain-network/flashloan-scanner/config"
)

func TestNormalizeRunMode(t *testing.T) {
	require.Equal(t, RunModeLoop, normalizeRunMode(""))
	require.Equal(t, RunModeLoop, normalizeRunMode(RunModeLoop))
	require.Equal(t, RunModeOnce, normalizeRunMode(RunModeOnce))
	require.Equal(t, "weird", normalizeRunMode("weird"))
}

func TestNewFlashloanScannerServiceRejectsDisabledScanner(t *testing.T) {
	service, err := NewFlashloanScannerService(context.Background(), &config.Config{
		Scanner: config.Scanner{
			Enabled: false,
		},
	}, func(error) {})
	require.Error(t, err)
	require.Nil(t, service)
}
