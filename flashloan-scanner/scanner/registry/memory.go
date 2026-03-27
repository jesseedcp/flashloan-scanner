package registry

import (
	"strconv"
	"strings"
	"sync"

	"github.com/cpchain-network/flashloan-scanner/scanner"
)

type AddressRecord struct {
	ChainID     uint64
	Protocol    scanner.Protocol
	AddressRole string
	Address     string
}

type MemoryRegistry struct {
	mu             sync.RWMutex
	addressesByKey map[string]struct{}
	pairsByChain   map[uint64]map[string]scanner.UniswapV2Pair
	trackedByChain map[uint64]map[string]struct{}
}

func NewMemoryRegistry() *MemoryRegistry {
	return &MemoryRegistry{
		addressesByKey: make(map[string]struct{}),
		pairsByChain:   make(map[uint64]map[string]scanner.UniswapV2Pair),
		trackedByChain: make(map[uint64]map[string]struct{}),
	}
}

func (r *MemoryRegistry) AddProtocolAddress(record AddressRecord) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := addressKey(record.ChainID, record.Protocol, record.AddressRole, record.Address)
	r.addressesByKey[key] = struct{}{}
	r.addTracked(record.ChainID, record.Address)
}

func (r *MemoryRegistry) AddUniswapV2Pair(pair scanner.UniswapV2Pair) {
	r.mu.Lock()
	defer r.mu.Unlock()
	addr := normalizeAddress(pair.PairAddress)
	if _, ok := r.pairsByChain[pair.ChainID]; !ok {
		r.pairsByChain[pair.ChainID] = make(map[string]scanner.UniswapV2Pair)
	}
	r.pairsByChain[pair.ChainID][addr] = pair
	r.addTracked(pair.ChainID, pair.PairAddress)
	r.addTracked(pair.ChainID, pair.FactoryAddress)
}

func (r *MemoryRegistry) IsOfficialAavePool(chainID uint64, address string) bool {
	return r.hasAddress(chainID, scanner.ProtocolAaveV3, "pool", address)
}

func (r *MemoryRegistry) IsOfficialBalancerVault(chainID uint64, address string) bool {
	return r.hasAddress(chainID, scanner.ProtocolBalancerV2, "vault", address)
}

func (r *MemoryRegistry) IsOfficialUniswapV2Factory(chainID uint64, address string) bool {
	return r.hasAddress(chainID, scanner.ProtocolUniswapV2, "factory", address)
}

func (r *MemoryRegistry) IsOfficialUniswapV2Pair(chainID uint64, address string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	pairs := r.pairsByChain[chainID]
	if len(pairs) == 0 {
		return false
	}
	_, ok := pairs[normalizeAddress(address)]
	return ok
}

func (r *MemoryRegistry) GetUniswapV2Pair(chainID uint64, pair string) (*scanner.UniswapV2Pair, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	pairs := r.pairsByChain[chainID]
	value, ok := pairs[normalizeAddress(pair)]
	if !ok {
		return nil, nil
	}
	out := value
	return &out, nil
}

func (r *MemoryRegistry) ListTrackedAddresses(chainID uint64) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []string
	for addr := range r.trackedByChain[chainID] {
		out = append(out, addr)
	}
	return out
}

func (r *MemoryRegistry) hasAddress(chainID uint64, protocol scanner.Protocol, role, address string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.addressesByKey[addressKey(chainID, protocol, role, address)]
	return ok
}

func (r *MemoryRegistry) addTracked(chainID uint64, address string) {
	if _, ok := r.trackedByChain[chainID]; !ok {
		r.trackedByChain[chainID] = make(map[string]struct{})
	}
	r.trackedByChain[chainID][normalizeAddress(address)] = struct{}{}
}

func addressKey(chainID uint64, protocol scanner.Protocol, role, address string) string {
	return strings.Join([]string{
		strconv.FormatUint(chainID, 10),
		string(protocol),
		role,
		normalizeAddress(address),
	}, "|")
}

func normalizeAddress(address string) string {
	return strings.ToLower(address)
}
