package orchestrator

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildAaveV3RunnerRequiresDatabase(t *testing.T) {
	runner, err := BuildAaveV3Runner(nil, nil, 1, AaveV3RunnerConfig{})
	require.Error(t, err)
	require.Nil(t, runner)
}
