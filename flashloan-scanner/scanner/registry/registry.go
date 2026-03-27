package registry

import "github.com/cpchain-network/flashloan-scanner/scanner"

type Registry interface {
	IsOfficialAavePool(chainID uint64, address string) bool
	IsOfficialBalancerVault(chainID uint64, address string) bool
	IsOfficialUniswapV2Factory(chainID uint64, address string) bool
	IsOfficialUniswapV2Pair(chainID uint64, address string) bool
	GetUniswapV2Pair(chainID uint64, pair string) (*scanner.UniswapV2Pair, error)
	ListTrackedAddresses(chainID uint64) []string
}
