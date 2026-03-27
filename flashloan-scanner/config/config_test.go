package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRPCByChainID(t *testing.T) {
	cfg := &Config{
		RPCs: []*RPC{
			{ChainId: 1, RpcUrl: "http://localhost:8545"},
			{ChainId: 10, RpcUrl: "http://localhost:9545"},
		},
	}

	rpcCfg, err := cfg.RPCByChainID(10)
	require.NoError(t, err)
	require.Equal(t, uint64(10), rpcCfg.ChainId)

	_, err = cfg.RPCByChainID(42161)
	require.Error(t, err)
}
