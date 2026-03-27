package bootstrap

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"

	"github.com/cpchain-network/flashloan-scanner/config"
	dbscanner "github.com/cpchain-network/flashloan-scanner/database/scanner"
	"github.com/cpchain-network/flashloan-scanner/scanner"
)

const (
	DefaultSourceAaveV3     = "manual_seed"
	DefaultSourceBalancerV2 = "manual_seed"
	DefaultSourceUniswapV2  = "manual_seed"
)

func SeedOfficialAaveV3Pools(db dbscanner.ProtocolAddressDB, chainID uint64, pools []string) error {
	return SeedOfficialProtocolAddresses(db, chainID, scanner.ProtocolAaveV3, "pool", pools, DefaultSourceAaveV3)
}

func SeedOfficialBalancerV2Vaults(db dbscanner.ProtocolAddressDB, chainID uint64, vaults []string) error {
	return SeedOfficialProtocolAddresses(db, chainID, scanner.ProtocolBalancerV2, "vault", vaults, DefaultSourceBalancerV2)
}

func SeedOfficialUniswapV2Factories(db dbscanner.ProtocolAddressDB, chainID uint64, factories []string) error {
	return SeedOfficialProtocolAddresses(db, chainID, scanner.ProtocolUniswapV2, "factory", factories, DefaultSourceUniswapV2)
}

func SeedOfficialUniswapV2Pairs(db dbscanner.UniswapV2PairDB, chainID uint64, pairs []config.ScannerUniswapV2Pair) error {
	items := make([]dbscanner.UniswapV2Pair, 0, len(pairs))
	for _, pair := range pairs {
		if strings.TrimSpace(pair.PairAddress) == "" {
			continue
		}
		items = append(items, dbscanner.UniswapV2Pair{
			ChainID:        chainID,
			FactoryAddress: common.HexToAddress(pair.FactoryAddress),
			PairAddress:    common.HexToAddress(pair.PairAddress),
			Token0:         common.HexToAddress(pair.Token0),
			Token1:         common.HexToAddress(pair.Token1),
			CreatedBlock:   new(big.Int).SetUint64(pair.CreatedBlock),
			IsOfficial:     true,
		})
	}
	return db.UpsertUniswapV2Pairs(items)
}

func SeedOfficialProtocolAddresses(
	db dbscanner.ProtocolAddressDB,
	chainID uint64,
	protocol scanner.Protocol,
	role string,
	addresses []string,
	source string,
) error {
	items := make([]dbscanner.ProtocolAddress, 0, len(addresses))
	for _, address := range addresses {
		if strings.TrimSpace(address) == "" {
			continue
		}
		items = append(items, dbscanner.ProtocolAddress{
			ChainID:         chainID,
			Protocol:        string(protocol),
			AddressRole:     role,
			ContractAddress: common.HexToAddress(address),
			IsOfficial:      true,
			Source:          source,
		})
	}
	return db.UpsertProtocolAddresses(items)
}
