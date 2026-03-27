package bootstrap

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/cpchain-network/flashloan-scanner/config"
	dbscanner "github.com/cpchain-network/flashloan-scanner/database/scanner"
	"github.com/cpchain-network/flashloan-scanner/scanner"
)

func TestSeedOfficialAaveV3Pools(t *testing.T) {
	db := &fakeProtocolAddressDB{}
	err := SeedOfficialAaveV3Pools(db, 1, []string{
		"0x1111111111111111111111111111111111111111",
		"",
		"0x2222222222222222222222222222222222222222",
	})
	require.NoError(t, err)
	require.Len(t, db.items, 2)
	require.Equal(t, "aave_v3", db.items[0].Protocol)
	require.Equal(t, "pool", db.items[0].AddressRole)
	require.True(t, db.items[0].IsOfficial)
}

func TestSeedOfficialBalancerV2Vaults(t *testing.T) {
	db := &fakeProtocolAddressDB{}
	err := SeedOfficialBalancerV2Vaults(db, 1, []string{
		"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	})
	require.NoError(t, err)
	require.Len(t, db.items, 1)
	require.Equal(t, "balancer_v2", db.items[0].Protocol)
	require.Equal(t, "vault", db.items[0].AddressRole)
}

func TestSeedOfficialUniswapV2Factories(t *testing.T) {
	db := &fakeProtocolAddressDB{}
	err := SeedOfficialUniswapV2Factories(db, 1, []string{
		"0xfac0000000000000000000000000000000000000",
	})
	require.NoError(t, err)
	require.Len(t, db.items, 1)
	require.Equal(t, "uniswap_v2", db.items[0].Protocol)
	require.Equal(t, "factory", db.items[0].AddressRole)
}

func TestSeedOfficialUniswapV2Pairs(t *testing.T) {
	db := &fakeUniswapPairDB{}
	err := SeedOfficialUniswapV2Pairs(db, 1, []config.ScannerUniswapV2Pair{{
		FactoryAddress: "0xfac0000000000000000000000000000000000000",
		PairAddress:    "0x1111111111111111111111111111111111111111",
		Token0:         "0x0000000000000000000000000000000000000001",
		Token1:         "0x0000000000000000000000000000000000000002",
		CreatedBlock:   123,
	}})
	require.NoError(t, err)
	require.Len(t, db.items, 1)
	require.Equal(t, uint64(1), db.items[0].ChainID)
}

type fakeProtocolAddressDB struct {
	items []dbscanner.ProtocolAddress
}

type fakeUniswapPairDB struct {
	items []dbscanner.UniswapV2Pair
}

func (f *fakeProtocolAddressDB) UpsertProtocolAddresses(items []dbscanner.ProtocolAddress) error {
	f.items = append(f.items, items...)
	return nil
}

func (f *fakeProtocolAddressDB) ListProtocolAddresses(uint64, string) ([]dbscanner.ProtocolAddress, error) {
	return nil, nil
}

func (f *fakeProtocolAddressDB) GetProtocolAddress(uint64, string, string, common.Address) (*dbscanner.ProtocolAddress, error) {
	return nil, nil
}

func (f *fakeProtocolAddressDB) ListOfficialProtocolAddresses(uint64) ([]dbscanner.ProtocolAddress, error) {
	return nil, nil
}

func (f *fakeUniswapPairDB) UpsertUniswapV2Pairs(items []dbscanner.UniswapV2Pair) error {
	f.items = append(f.items, items...)
	return nil
}

func (f *fakeUniswapPairDB) GetUniswapV2Pair(uint64, common.Address) (*dbscanner.UniswapV2Pair, error) {
	return nil, nil
}

func (f *fakeUniswapPairDB) ListUniswapV2Pairs(uint64) ([]dbscanner.UniswapV2Pair, error) {
	return nil, nil
}

var _ dbscanner.ProtocolAddressDB = (*fakeProtocolAddressDB)(nil)
var _ dbscanner.UniswapV2PairDB = (*fakeUniswapPairDB)(nil)
var _ = scanner.ProtocolAaveV3
