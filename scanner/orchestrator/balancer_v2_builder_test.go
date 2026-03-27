package orchestrator

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildBalancerV2RunnerRequiresDatabase(t *testing.T) {
	runner, err := BuildBalancerV2Runner(nil, nil, 1, BalancerV2RunnerConfig{})
	require.Error(t, err)
	require.Nil(t, runner)
}
