package report

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	dbscanner "github.com/cpchain-network/flashloan-scanner/database/scanner"
)

func TestRenderText(t *testing.T) {
	reports := sampleReports()
	output := RenderText(reports)

	require.Contains(t, output, "tx=0x0000000000000000000000000000000000000000000000000000000000000abc")
	require.Contains(t, output, "protocol=aave_v3")
	require.Contains(t, output, "settlement=full_repayment")
}

func TestRenderJSON(t *testing.T) {
	output, err := RenderJSON(sampleReports())
	require.NoError(t, err)
	require.Contains(t, output, "\"protocol\": \"aave_v3\"")
	require.Contains(t, output, "\"settlement_mode\": \"full_repayment\"")
}

func TestWriteCSV(t *testing.T) {
	var buf bytes.Buffer
	err := WriteCSV(&buf, sampleReports())
	require.NoError(t, err)
	output := buf.String()
	require.Contains(t, output, "chain_id,tx_hash,block_number")
	require.Contains(t, output, "aave_v3")
	require.Contains(t, output, "full_repayment")
}

func sampleReports() []TransactionReport {
	settlementMode := "full_repayment"
	return []TransactionReport{{
		Tx: dbscanner.FlashloanTransaction{
			ChainID:                           1,
			TxHash:                            common.HexToHash("0xabc"),
			BlockNumber:                       big.NewInt(100),
			ContainsCandidateInteraction:      true,
			ContainsVerifiedInteraction:       true,
			ContainsVerifiedStrictInteraction: true,
			InteractionCount:                  1,
			StrictInteractionCount:            1,
			Protocols:                         "aave_v3",
		},
		Interactions: []InteractionReport{{
			Interaction: dbscanner.ProtocolInteraction{
				InteractionID:   uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
				Protocol:        "aave_v3",
				Entrypoint:      "flashLoanSimple",
				ProviderAddress: common.HexToAddress("0x1111111111111111111111111111111111111111"),
				Verified:        true,
				Strict:          true,
			},
			Legs: []dbscanner.InteractionAssetLeg{{
				LegIndex:       0,
				AssetAddress:   common.HexToAddress("0x3333333333333333333333333333333333333333"),
				AmountBorrowed: big.NewInt(1000),
				PremiumAmount:  big.NewInt(5),
				StrictLeg:      true,
				SettlementMode: &settlementMode,
			}},
		}},
	}}
}
