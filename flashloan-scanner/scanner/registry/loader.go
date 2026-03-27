package registry

import (
	dbscanner "github.com/cpchain-network/flashloan-scanner/database/scanner"
	"github.com/cpchain-network/flashloan-scanner/scanner"
)

func NewMemoryRegistryFromDB(chainID uint64, addressView dbscanner.ProtocolAddressView, pairView dbscanner.UniswapV2PairView) (*MemoryRegistry, error) {
	out := NewMemoryRegistry()

	addresses, err := addressView.ListOfficialProtocolAddresses(chainID)
	if err != nil {
		return nil, err
	}
	for _, item := range addresses {
		out.AddProtocolAddress(AddressRecord{
			ChainID:     item.ChainID,
			Protocol:    scanner.Protocol(item.Protocol),
			AddressRole: item.AddressRole,
			Address:     item.ContractAddress.Hex(),
		})
	}

	pairs, err := pairView.ListUniswapV2Pairs(chainID)
	if err != nil {
		return nil, err
	}
	for _, item := range pairs {
		if !item.IsOfficial {
			continue
		}
		out.AddUniswapV2Pair(scanner.UniswapV2Pair{
			ChainID:        item.ChainID,
			FactoryAddress: item.FactoryAddress.Hex(),
			PairAddress:    item.PairAddress.Hex(),
			Token0:         item.Token0.Hex(),
			Token1:         item.Token1.Hex(),
			CreatedBlock:   item.CreatedBlock.String(),
		})
	}

	return out, nil
}
