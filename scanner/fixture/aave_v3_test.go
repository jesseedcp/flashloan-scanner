package fixture

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestBuildAaveV3FlashLoanSimpleFixture(t *testing.T) {
	fixture := DefaultAaveV3FlashLoanSimpleFixture(
		1,
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		19000000,
	)

	header, eventRecord, observedTx, err := BuildAaveV3FlashLoanSimpleFixture(fixture)
	require.NoError(t, err)
	require.NotNil(t, header)
	require.Equal(t, uint64(19000000), header.Number.Uint64())
	require.Equal(t, fixture.TxHash, eventRecord.TransactionHash)
	require.Equal(t, fixture.PoolAddress, eventRecord.ContractAddress)
	require.Equal(t, fixture.TxHash, observedTx.TxHash)
	require.NotEmpty(t, observedTx.InputData)
	require.Equal(t, fixture.PoolAddress.Hex(), observedTx.ToAddress.Hex())
}
