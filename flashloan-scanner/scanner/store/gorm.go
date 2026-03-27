package store

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"

	"github.com/cpchain-network/flashloan-scanner/database"
	dbscanner "github.com/cpchain-network/flashloan-scanner/database/scanner"
	"github.com/cpchain-network/flashloan-scanner/scanner"
)

type GormStore struct {
	db *database.DB
}

func NewGormStore(db *database.DB) *GormStore {
	return &GormStore{db: db}
}

func (s *GormStore) UpsertObservedTransactions(_ context.Context, txs []scanner.ObservedTransaction) error {
	items := make([]dbscanner.ObservedTransaction, 0, len(txs))
	for i := range txs {
		item, err := observedTransactionToDB(txs[i])
		if err != nil {
			return err
		}
		items = append(items, item)
	}
	return s.db.ObservedTx.UpsertObservedTransactions(items)
}

func (s *GormStore) ListObservedTransactionsByBlockRange(_ context.Context, chainID uint64, fromBlock, toBlock uint64) ([]scanner.ObservedTransaction, error) {
	items, err := s.db.ObservedTx.ListObservedTransactionsByBlockRange(chainID, big.NewInt(int64(fromBlock)), big.NewInt(int64(toBlock)))
	if err != nil {
		return nil, err
	}
	out := make([]scanner.ObservedTransaction, 0, len(items))
	for i := range items {
		out = append(out, observedTransactionFromDB(items[i]))
	}
	return out, nil
}

func (s *GormStore) UpsertInteractions(_ context.Context, items []scanner.CandidateInteraction) error {
	dbItems := make([]dbscanner.ProtocolInteraction, 0, len(items))
	for i := range items {
		item, err := candidateInteractionToDB(items[i])
		if err != nil {
			return err
		}
		dbItems = append(dbItems, item)
	}
	return s.db.ProtocolInteraction.UpsertProtocolInteractions(dbItems)
}

func (s *GormStore) UpdateVerificationResult(_ context.Context, result scanner.VerifiedInteraction) error {
	interactionID, err := uuid.Parse(result.InteractionID)
	if err != nil {
		return fmt.Errorf("parse interaction id: %w", err)
	}
	update := dbscanner.ProtocolInteractionVerificationUpdate{
		Verified:            result.Verified,
		Strict:              result.Strict,
		CallbackSeen:        result.CallbackSeen,
		SettlementSeen:      result.SettlementSeen,
		RepaymentSeen:       result.RepaymentSeen,
		ContainsDebtOpening: result.ContainsDebtOpening,
		ExclusionReason:     stringValue(result.ExclusionReason),
		VerificationNotes:   stringValue(result.VerificationNotes),
	}
	return s.db.ProtocolInteraction.UpdateProtocolInteractionVerification(interactionID, update)
}

func (s *GormStore) ListCandidateInteractions(_ context.Context, chainID uint64, protocol scanner.Protocol, fromBlock, toBlock uint64) ([]scanner.CandidateInteraction, error) {
	items, err := s.db.ProtocolInteraction.ListCandidateInteractions(chainID, string(protocol), big.NewInt(int64(fromBlock)), big.NewInt(int64(toBlock)))
	if err != nil {
		return nil, err
	}
	out := make([]scanner.CandidateInteraction, 0, len(items))
	for i := range items {
		out = append(out, candidateInteractionFromDB(items[i]))
	}
	return out, nil
}

func (s *GormStore) ReplaceInteractionLegs(_ context.Context, interactionID string, legs []scanner.InteractionLeg) error {
	uid, err := uuid.Parse(interactionID)
	if err != nil {
		return fmt.Errorf("parse interaction id: %w", err)
	}
	dbItems := make([]dbscanner.InteractionAssetLeg, 0, len(legs))
	for i := range legs {
		item, err := interactionLegToDB(uid, legs[i])
		if err != nil {
			return err
		}
		dbItems = append(dbItems, item)
	}
	return s.db.InteractionAssetLeg.ReplaceInteractionLegs(uid, dbItems)
}

func (s *GormStore) UpsertTxSummary(_ context.Context, summary scanner.TxSummary) error {
	item, err := txSummaryToDB(summary)
	if err != nil {
		return err
	}
	return s.db.FlashloanTx.UpsertFlashloanTransactions([]dbscanner.FlashloanTransaction{item})
}

func observedTransactionToDB(item scanner.ObservedTransaction) (dbscanner.ObservedTransaction, error) {
	blockNumber, err := parseBigInt(item.BlockNumber)
	if err != nil {
		return dbscanner.ObservedTransaction{}, err
	}
	value, err := parseBigIntPtr(item.Value)
	if err != nil {
		return dbscanner.ObservedTransaction{}, err
	}
	gasUsed, err := parseBigIntPtr(item.GasUsed)
	if err != nil {
		return dbscanner.ObservedTransaction{}, err
	}
	effectiveGasPrice, err := parseBigIntPtr(item.EffectiveGasPrice)
	if err != nil {
		return dbscanner.ObservedTransaction{}, err
	}
	var toAddress *common.Address
	if item.ToAddress != "" {
		addr := common.HexToAddress(item.ToAddress)
		toAddress = &addr
	}
	return dbscanner.ObservedTransaction{
		ChainID:           item.ChainID,
		TxHash:            common.HexToHash(item.TxHash),
		BlockNumber:       blockNumber,
		TxIndex:           item.TxIndex,
		FromAddress:       common.HexToAddress(item.FromAddress),
		ToAddress:         toAddress,
		Status:            item.Status,
		Value:             value,
		InputData:         item.InputData,
		MethodSelector:    item.MethodSelector,
		GasUsed:           gasUsed,
		EffectiveGasPrice: effectiveGasPrice,
	}, nil
}

func observedTransactionFromDB(item dbscanner.ObservedTransaction) scanner.ObservedTransaction {
	var toAddress string
	if item.ToAddress != nil {
		toAddress = item.ToAddress.Hex()
	}
	return scanner.ObservedTransaction{
		ChainID:           item.ChainID,
		TxHash:            item.TxHash.Hex(),
		BlockNumber:       bigIntToString(item.BlockNumber),
		TxIndex:           item.TxIndex,
		FromAddress:       item.FromAddress.Hex(),
		ToAddress:         toAddress,
		Status:            item.Status,
		Value:             bigIntToString(item.Value),
		InputData:         item.InputData,
		MethodSelector:    item.MethodSelector,
		GasUsed:           bigIntToString(item.GasUsed),
		EffectiveGasPrice: bigIntToString(item.EffectiveGasPrice),
	}
}

func candidateInteractionToDB(item scanner.CandidateInteraction) (dbscanner.ProtocolInteraction, error) {
	blockNumber, err := parseBigInt(item.BlockNumber)
	if err != nil {
		return dbscanner.ProtocolInteraction{}, err
	}
	var interactionID uuid.UUID
	if item.InteractionID != "" {
		interactionID, err = uuid.Parse(item.InteractionID)
		if err != nil {
			return dbscanner.ProtocolInteraction{}, fmt.Errorf("parse interaction id: %w", err)
		}
	}
	return dbscanner.ProtocolInteraction{
		InteractionID:      interactionID,
		InteractionOrdinal: item.InteractionOrdinal,
		ChainID:            item.ChainID,
		TxHash:             common.HexToHash(item.TxHash),
		BlockNumber:        blockNumber,
		Protocol:           string(item.Protocol),
		Entrypoint:         item.Entrypoint,
		ProviderAddress:    common.HexToAddress(item.ProviderAddress),
		FactoryAddress:     addressPtr(item.FactoryAddress),
		PairAddress:        addressPtr(item.PairAddress),
		ReceiverAddress:    addressPtr(item.ReceiverAddress),
		CallbackTarget:     addressPtr(item.CallbackTarget),
		Initiator:          addressPtr(item.Initiator),
		OnBehalfOf:         addressPtr(item.OnBehalfOf),
		CandidateLevel:     uint8(item.CandidateLevel),
		RawMethodSelector:  item.RawMethodSelector,
		DataNonEmpty:       item.DataNonEmpty,
	}, nil
}

func candidateInteractionFromDB(item dbscanner.ProtocolInteraction) scanner.CandidateInteraction {
	return scanner.CandidateInteraction{
		InteractionID:      normalizeUUID(item.InteractionID),
		InteractionOrdinal: item.InteractionOrdinal,
		ChainID:            item.ChainID,
		TxHash:             item.TxHash.Hex(),
		BlockNumber:        bigIntToString(item.BlockNumber),
		Protocol:           scanner.Protocol(item.Protocol),
		Entrypoint:         item.Entrypoint,
		ProviderAddress:    item.ProviderAddress.Hex(),
		FactoryAddress:     addressToString(item.FactoryAddress),
		PairAddress:        addressToString(item.PairAddress),
		ReceiverAddress:    addressToString(item.ReceiverAddress),
		CallbackTarget:     addressToString(item.CallbackTarget),
		Initiator:          addressToString(item.Initiator),
		OnBehalfOf:         addressToString(item.OnBehalfOf),
		CandidateLevel:     scanner.CandidateLevel(item.CandidateLevel),
		RawMethodSelector:  item.RawMethodSelector,
		DataNonEmpty:       item.DataNonEmpty,
	}
}

func interactionLegToDB(interactionID uuid.UUID, item scanner.InteractionLeg) (dbscanner.InteractionAssetLeg, error) {
	amountOut, err := parseBigIntPtr(valueOrEmpty(item.AmountOut))
	if err != nil {
		return dbscanner.InteractionAssetLeg{}, err
	}
	amountIn, err := parseBigIntPtr(valueOrEmpty(item.AmountIn))
	if err != nil {
		return dbscanner.InteractionAssetLeg{}, err
	}
	amountBorrowed, err := parseBigIntPtr(valueOrEmpty(item.AmountBorrowed))
	if err != nil {
		return dbscanner.InteractionAssetLeg{}, err
	}
	amountRepaid, err := parseBigIntPtr(valueOrEmpty(item.AmountRepaid))
	if err != nil {
		return dbscanner.InteractionAssetLeg{}, err
	}
	premiumAmount, err := parseBigIntPtr(valueOrEmpty(item.PremiumAmount))
	if err != nil {
		return dbscanner.InteractionAssetLeg{}, err
	}
	feeAmount, err := parseBigIntPtr(valueOrEmpty(item.FeeAmount))
	if err != nil {
		return dbscanner.InteractionAssetLeg{}, err
	}
	var interestRateMode *uint8
	if item.InterestRateMode != nil {
		mode := uint8(*item.InterestRateMode)
		interestRateMode = &mode
	}
	return dbscanner.InteractionAssetLeg{
		InteractionID:    interactionID,
		LegIndex:         item.LegIndex,
		AssetAddress:     common.HexToAddress(item.AssetAddress),
		AssetRole:        item.AssetRole,
		TokenSide:        item.TokenSide,
		AmountOut:        amountOut,
		AmountIn:         amountIn,
		AmountBorrowed:   amountBorrowed,
		AmountRepaid:     amountRepaid,
		PremiumAmount:    premiumAmount,
		FeeAmount:        feeAmount,
		InterestRateMode: interestRateMode,
		RepaidToAddress:  addressPtr(valueOrEmpty(item.RepaidToAddress)),
		OpenedDebt:       item.OpenedDebt,
		StrictLeg:        item.StrictLeg,
		EventSeen:        item.EventSeen,
		SettlementMode:   item.SettlementMode,
	}, nil
}

func txSummaryToDB(summary scanner.TxSummary) (dbscanner.FlashloanTransaction, error) {
	blockNumber, err := parseBigInt(summary.BlockNumber)
	if err != nil {
		return dbscanner.FlashloanTransaction{}, err
	}
	protocols := make([]string, 0, len(summary.Protocols))
	for _, protocol := range summary.Protocols {
		protocols = append(protocols, string(protocol))
	}
	return dbscanner.FlashloanTransaction{
		ChainID:                           summary.ChainID,
		TxHash:                            common.HexToHash(summary.TxHash),
		BlockNumber:                       blockNumber,
		ContainsCandidateInteraction:      summary.ContainsCandidateInteraction,
		ContainsVerifiedInteraction:       summary.ContainsVerifiedInteraction,
		ContainsVerifiedStrictInteraction: summary.ContainsVerifiedStrictInteraction,
		InteractionCount:                  summary.InteractionCount,
		StrictInteractionCount:            summary.StrictInteractionCount,
		ProtocolCount:                     summary.ProtocolCount,
		Protocols:                         strings.Join(protocols, ","),
	}, nil
}

func parseBigInt(raw string) (*big.Int, error) {
	if raw == "" {
		return nil, fmt.Errorf("empty big integer")
	}
	out, ok := new(big.Int).SetString(raw, 10)
	if !ok {
		return nil, fmt.Errorf("invalid big integer: %s", raw)
	}
	return out, nil
}

func parseBigIntPtr(raw string) (*big.Int, error) {
	if raw == "" {
		return nil, nil
	}
	return parseBigInt(raw)
}

func bigIntToString(v *big.Int) string {
	if v == nil {
		return ""
	}
	return v.String()
}

func addressPtr(raw string) *common.Address {
	if raw == "" {
		return nil
	}
	addr := common.HexToAddress(raw)
	return &addr
}

func addressToString(v *common.Address) string {
	if v == nil {
		return ""
	}
	return v.Hex()
}

func valueOrEmpty[T any](v *T) string {
	if v == nil {
		return ""
	}
	switch value := any(*v).(type) {
	case string:
		return value
	default:
		return ""
	}
}

func stringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func normalizeUUID(v uuid.UUID) string {
	if v == uuid.Nil {
		return ""
	}
	return v.String()
}
