package orchestrator

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildUniswapV2RunnerRequiresDatabase(t *testing.T) {
	runner, err := BuildUniswapV2Runner(nil, nil, 1, UniswapV2RunnerConfig{})
	require.Error(t, err)
	require.Nil(t, runner)
}
