package fixture

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestBuildBalancerV2FlashLoanFixture(t *testing.T) {
	fixture := DefaultBalancerV2FlashLoanFixture(
		1,
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		19000000,
	)

	header, eventRecord, observedTx, err := BuildBalancerV2FlashLoanFixture(fixture)
	require.NoError(t, err)
	require.NotNil(t, header)
	require.Equal(t, uint64(19000000), header.Number.Uint64())
	require.Equal(t, fixture.TxHash, eventRecord.TransactionHash)
	require.Equal(t, fixture.Vault, eventRecord.ContractAddress)
	require.Equal(t, fixture.TxHash, observedTx.TxHash)
	require.NotEmpty(t, observedTx.InputData)
	require.Equal(t, fixture.Vault.Hex(), observedTx.ToAddress.Hex())
}
