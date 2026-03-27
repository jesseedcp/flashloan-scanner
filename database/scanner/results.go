package scanner

import (
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ProtocolInteraction struct {
	InteractionID       uuid.UUID `gorm:"primaryKey;DEFAULT replace(uuid_generate_v4()::text,'-','');serializer:uuid"`
	ChainID             uint64
	TxHash              common.Hash `gorm:"serializer:bytes"`
	BlockNumber         *big.Int    `gorm:"serializer:u256"`
	TxIndex             uint64
	InteractionOrdinal  int
	Timestamp           uint64
	Protocol            string
	Entrypoint          string
	ProviderAddress     common.Address  `gorm:"serializer:bytes"`
	FactoryAddress      *common.Address `gorm:"serializer:bytes"`
	PairAddress         *common.Address `gorm:"serializer:bytes"`
	ReceiverAddress     *common.Address `gorm:"serializer:bytes"`
	CallbackTarget      *common.Address `gorm:"serializer:bytes"`
	Initiator           *common.Address `gorm:"serializer:bytes"`
	OnBehalfOf          *common.Address `gorm:"serializer:bytes"`
	CandidateLevel      uint8
	Verified            bool
	Strict              bool
	CallbackSeen        bool
	SettlementSeen      bool
	RepaymentSeen       bool
	ContainsDebtOpening bool
	DataNonEmpty        *bool
	ExclusionReason     string
	VerificationNotes   string
	RawMethodSelector   string
	SourceConfidence    *uint8
	CreatedAt           uint64
	UpdatedAt           uint64
}

func (ProtocolInteraction) TableName() string {
	return "protocol_interactions"
}

type InteractionAssetLeg struct {
	LegID            uuid.UUID `gorm:"primaryKey;DEFAULT replace(uuid_generate_v4()::text,'-','');serializer:uuid"`
	InteractionID    uuid.UUID `gorm:"serializer:uuid"`
	LegIndex         int
	AssetAddress     common.Address `gorm:"serializer:bytes"`
	AssetRole        string
	TokenSide        *string
	AmountOut        *big.Int `gorm:"serializer:u256"`
	AmountIn         *big.Int `gorm:"serializer:u256"`
	AmountBorrowed   *big.Int `gorm:"serializer:u256"`
	AmountRepaid     *big.Int `gorm:"serializer:u256"`
	PremiumAmount    *big.Int `gorm:"serializer:u256"`
	FeeAmount        *big.Int `gorm:"serializer:u256"`
	InterestRateMode *uint8
	RepaidToAddress  *common.Address `gorm:"serializer:bytes"`
	OpenedDebt       bool
	StrictLeg        bool
	EventSeen        bool
	SettlementMode   *string
	CreatedAt        uint64
	UpdatedAt        uint64
}

func (InteractionAssetLeg) TableName() string {
	return "interaction_asset_legs"
}

type FlashloanTransaction struct {
	ChainID                           uint64      `gorm:"primaryKey"`
	TxHash                            common.Hash `gorm:"primaryKey;serializer:bytes"`
	BlockNumber                       *big.Int    `gorm:"serializer:u256"`
	Timestamp                         uint64
	ContainsCandidateInteraction      bool
	ContainsVerifiedInteraction       bool
	ContainsVerifiedStrictInteraction bool
	InteractionCount                  int
	StrictInteractionCount            int
	ProtocolCount                     int
	Protocols                         string
	CreatedAt                         uint64
	UpdatedAt                         uint64
}

type FlashloanTransactionSummary struct {
	TotalTransactions int64
	TotalCandidate    int64
	TotalVerified     int64
	TotalStrict       int64
}

func (FlashloanTransaction) TableName() string {
	return "flashloan_transactions"
}

type ProtocolInteractionView interface {
	ListCandidateInteractions(chainID uint64, protocol string, fromBlock, toBlock *big.Int) ([]ProtocolInteraction, error)
	ListProtocolInteractionsByTx(chainID uint64, txHash common.Hash) ([]ProtocolInteraction, error)
}

type ProtocolInteractionDB interface {
	ProtocolInteractionView
	UpsertProtocolInteractions(items []ProtocolInteraction) error
	UpdateProtocolInteractionVerification(interactionID uuid.UUID, update ProtocolInteractionVerificationUpdate) error
}

type InteractionAssetLegDB interface {
	ReplaceInteractionLegs(interactionID uuid.UUID, items []InteractionAssetLeg) error
	ListInteractionLegs(interactionID uuid.UUID) ([]InteractionAssetLeg, error)
}

type FlashloanTransactionDB interface {
	UpsertFlashloanTransactions(items []FlashloanTransaction) error
	GetFlashloanTransaction(chainID uint64, txHash common.Hash) (*FlashloanTransaction, error)
	ListFlashloanTransactions(chainID uint64, onlyStrict bool, limit int) ([]FlashloanTransaction, error)
	GetFlashloanTransactionSummary(chainID uint64) (*FlashloanTransactionSummary, error)
}

type ProtocolInteractionVerificationUpdate struct {
	Verified            bool
	Strict              bool
	CallbackSeen        bool
	SettlementSeen      bool
	RepaymentSeen       bool
	ContainsDebtOpening bool
	ExclusionReason     string
	VerificationNotes   string
	SourceConfidence    *uint8
}

type protocolInteractionDB struct {
	gorm *gorm.DB
}

type interactionAssetLegDB struct {
	gorm *gorm.DB
}

type flashloanTransactionDB struct {
	gorm *gorm.DB
}

func NewProtocolInteractionDB(db *gorm.DB) ProtocolInteractionDB {
	return &protocolInteractionDB{gorm: db}
}

func NewInteractionAssetLegDB(db *gorm.DB) InteractionAssetLegDB {
	return &interactionAssetLegDB{gorm: db}
}

func NewFlashloanTransactionDB(db *gorm.DB) FlashloanTransactionDB {
	return &flashloanTransactionDB{gorm: db}
}

func (db *protocolInteractionDB) UpsertProtocolInteractions(items []ProtocolInteraction) error {
	if len(items) == 0 {
		return nil
	}
	now := uint64(time.Now().Unix())
	for i := range items {
		if items[i].InteractionID == uuid.Nil {
			items[i].InteractionID = uuid.New()
		}
		if items[i].CreatedAt == 0 {
			items[i].CreatedAt = now
		}
		items[i].UpdatedAt = now
	}
	return db.gorm.Table((ProtocolInteraction{}).TableName()).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "chain_id"},
			{Name: "tx_hash"},
			{Name: "interaction_ordinal"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"block_number", "tx_index", "timestamp", "protocol", "entrypoint", "provider_address",
			"factory_address", "pair_address", "receiver_address", "callback_target", "initiator",
			"on_behalf_of", "candidate_level", "verified", "strict", "callback_seen", "settlement_seen",
			"repayment_seen", "contains_debt_opening", "data_non_empty", "exclusion_reason",
			"verification_notes", "raw_method_selector", "source_confidence", "updated_at",
		}),
	}).Create(&items).Error
}

func (db *protocolInteractionDB) UpdateProtocolInteractionVerification(interactionID uuid.UUID, update ProtocolInteractionVerificationUpdate) error {
	values := map[string]interface{}{
		"verified":              update.Verified,
		"strict":                update.Strict,
		"callback_seen":         update.CallbackSeen,
		"settlement_seen":       update.SettlementSeen,
		"repayment_seen":        update.RepaymentSeen,
		"contains_debt_opening": update.ContainsDebtOpening,
		"exclusion_reason":      update.ExclusionReason,
		"verification_notes":    update.VerificationNotes,
		"updated_at":            uint64(time.Now().Unix()),
	}
	if update.SourceConfidence != nil {
		values["source_confidence"] = *update.SourceConfidence
	}
	return db.gorm.Table((ProtocolInteraction{}).TableName()).
		Where("interaction_id = ?", compactUUID(interactionID)).
		Updates(values).Error
}

func (db *protocolInteractionDB) ListCandidateInteractions(chainID uint64, protocol string, fromBlock, toBlock *big.Int) ([]ProtocolInteraction, error) {
	if fromBlock == nil {
		fromBlock = big.NewInt(0)
	}
	if toBlock == nil {
		return nil, errors.New("end block unspecified")
	}
	query := db.gorm.Table((ProtocolInteraction{}).TableName()).
		Where("chain_id = ? AND candidate_level >= 2 AND block_number >= ? AND block_number <= ?", chainID, fromBlock, toBlock)
	if protocol != "" {
		query = query.Where("protocol = ?", protocol)
	}
	var out []ProtocolInteraction
	if err := query.Order("block_number ASC, tx_index ASC, interaction_ordinal ASC").Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (db *protocolInteractionDB) ListProtocolInteractionsByTx(chainID uint64, txHash common.Hash) ([]ProtocolInteraction, error) {
	var out []ProtocolInteraction
	if err := db.gorm.Table((ProtocolInteraction{}).TableName()).
		Where("chain_id = ? AND tx_hash = ?", chainID, txHash.Hex()).
		Order("interaction_ordinal ASC").
		Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (db *interactionAssetLegDB) ReplaceInteractionLegs(interactionID uuid.UUID, items []InteractionAssetLeg) error {
	return db.gorm.Transaction(func(tx *gorm.DB) error {
		if err := tx.Table((InteractionAssetLeg{}).TableName()).
			Where("interaction_id = ?", compactUUID(interactionID)).
			Delete(&InteractionAssetLeg{}).Error; err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		now := uint64(time.Now().Unix())
		for i := range items {
			if items[i].LegID == uuid.Nil {
				items[i].LegID = uuid.New()
			}
			items[i].InteractionID = interactionID
			if items[i].CreatedAt == 0 {
				items[i].CreatedAt = now
			}
			items[i].UpdatedAt = now
		}
		return tx.Table((InteractionAssetLeg{}).TableName()).Create(&items).Error
	})
}

func (db *interactionAssetLegDB) ListInteractionLegs(interactionID uuid.UUID) ([]InteractionAssetLeg, error) {
	var out []InteractionAssetLeg
	if err := db.gorm.Table((InteractionAssetLeg{}).TableName()).
		Where("interaction_id = ?", compactUUID(interactionID)).
		Order("leg_index ASC").
		Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (db *flashloanTransactionDB) UpsertFlashloanTransactions(items []FlashloanTransaction) error {
	if len(items) == 0 {
		return nil
	}
	now := uint64(time.Now().Unix())
	for i := range items {
		if items[i].CreatedAt == 0 {
			items[i].CreatedAt = now
		}
		items[i].UpdatedAt = now
	}
	return db.gorm.Table((FlashloanTransaction{}).TableName()).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "chain_id"},
			{Name: "tx_hash"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"block_number", "timestamp", "contains_candidate_interaction", "contains_verified_interaction",
			"contains_verified_strict_interaction", "interaction_count", "strict_interaction_count",
			"protocol_count", "protocols", "updated_at",
		}),
	}).Create(&items).Error
}

func (db *flashloanTransactionDB) GetFlashloanTransaction(chainID uint64, txHash common.Hash) (*FlashloanTransaction, error) {
	var out FlashloanTransaction
	err := db.gorm.Table((FlashloanTransaction{}).TableName()).
		Where("chain_id = ? AND tx_hash = ?", chainID, txHash.Hex()).
		Take(&out).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &out, nil
}

func (db *flashloanTransactionDB) ListFlashloanTransactions(chainID uint64, onlyStrict bool, limit int) ([]FlashloanTransaction, error) {
	if limit <= 0 {
		limit = 10
	}
	query := db.gorm.Table((FlashloanTransaction{}).TableName()).
		Where("chain_id = ?", chainID)
	if onlyStrict {
		query = query.Where("contains_verified_strict_interaction = ?", true)
	}
	var out []FlashloanTransaction
	if err := query.
		Order("block_number DESC, tx_hash DESC").
		Limit(limit).
		Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (db *flashloanTransactionDB) GetFlashloanTransactionSummary(chainID uint64) (*FlashloanTransactionSummary, error) {
	var out FlashloanTransactionSummary
	err := db.gorm.Table((FlashloanTransaction{}).TableName()).
		Select(`
			COUNT(*) AS total_transactions,
			COALESCE(SUM(CASE WHEN contains_candidate_interaction THEN 1 ELSE 0 END), 0) AS total_candidate,
			COALESCE(SUM(CASE WHEN contains_verified_interaction THEN 1 ELSE 0 END), 0) AS total_verified,
			COALESCE(SUM(CASE WHEN contains_verified_strict_interaction THEN 1 ELSE 0 END), 0) AS total_strict
		`).
		Where("chain_id = ?", chainID).
		Scan(&out).Error
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func compactUUID(id uuid.UUID) string {
	return strings.ReplaceAll(id.String(), "-", "")
}
