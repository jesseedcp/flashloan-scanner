package fixture

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestBuildUniswapV2FlashSwapFixture(t *testing.T) {
	fixture := DefaultUniswapV2FlashSwapFixture(
		1,
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
		common.HexToAddress("0x3333333333333333333333333333333333333333"),
		common.HexToAddress("0x4444444444444444444444444444444444444444"),
		19000000,
	)

	header, eventRecord, observedTx, err := BuildUniswapV2FlashSwapFixture(fixture)
	require.NoError(t, err)
	require.NotNil(t, header)
	require.Equal(t, uint64(19000000), header.Number.Uint64())
	require.Equal(t, fixture.TxHash, eventRecord.TransactionHash)
	require.Equal(t, fixture.Pair, eventRecord.ContractAddress)
	require.Equal(t, fixture.TxHash, observedTx.TxHash)
	require.NotEmpty(t, observedTx.InputData)
	require.Equal(t, fixture.Pair.Hex(), observedTx.ToAddress.Hex())
}
