package aggregator

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	dbscanner "github.com/cpchain-network/flashloan-scanner/database/scanner"
)

func TestSimpleTxAggregatorAggregateByTx(t *testing.T) {
	txHash := common.HexToHash("0xabc")
	agg := NewSimpleTxAggregator(
		fakeInteractionView{
			byTx: []dbscanner.ProtocolInteraction{
				{
					InteractionID: uuid.New(),
					ChainID:       1,
					TxHash:        txHash,
					BlockNumber:   big.NewInt(100),
					Protocol:      "aave_v3",
					Verified:      true,
					Strict:        true,
				},
				{
					InteractionID: uuid.New(),
					ChainID:       1,
					TxHash:        txHash,
					BlockNumber:   big.NewInt(100),
					Protocol:      "aave_v3",
					Verified:      false,
					Strict:        false,
				},
			},
		},
		&fakeFlashloanTxStore{},
	)

	summary, err := agg.AggregateByTx(context.Background(), 1, txHash.Hex())
	require.NoError(t, err)
	require.NotNil(t, summary)
	require.True(t, summary.ContainsCandidateInteraction)
	require.True(t, summary.ContainsVerifiedInteraction)
	require.True(t, summary.ContainsVerifiedStrictInteraction)
	require.Equal(t, 2, summary.InteractionCount)
	require.Equal(t, 1, summary.StrictInteractionCount)
	require.Equal(t, 1, summary.ProtocolCount)
	require.Equal(t, "100", summary.BlockNumber)
}

type fakeInteractionView struct {
	byTx       []dbscanner.ProtocolInteraction
	candidates []dbscanner.ProtocolInteraction
}

func (f fakeInteractionView) ListCandidateInteractions(uint64, string, *big.Int, *big.Int) ([]dbscanner.ProtocolInteraction, error) {
	return f.candidates, nil
}

func (f fakeInteractionView) ListProtocolInteractionsByTx(uint64, common.Hash) ([]dbscanner.ProtocolInteraction, error) {
	return f.byTx, nil
}

type fakeFlashloanTxStore struct {
	items []dbscanner.FlashloanTransaction
}

func (f *fakeFlashloanTxStore) UpsertFlashloanTransactions(items []dbscanner.FlashloanTransaction) error {
	f.items = append(f.items, items...)
	return nil
}

func (f *fakeFlashloanTxStore) GetFlashloanTransaction(uint64, common.Hash) (*dbscanner.FlashloanTransaction, error) {
	return nil, nil
}

func (f *fakeFlashloanTxStore) ListFlashloanTransactions(uint64, bool, int) ([]dbscanner.FlashloanTransaction, error) {
	return f.items, nil
}

func (f *fakeFlashloanTxStore) GetFlashloanTransactionSummary(uint64) (*dbscanner.FlashloanTransactionSummary, error) {
	return &dbscanner.FlashloanTransactionSummary{}, nil
}
