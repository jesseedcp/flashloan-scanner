package scanner

import (
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TransactionTrace struct {
	ChainID   uint64      `gorm:"primaryKey"`
	TxHash    common.Hash `gorm:"primaryKey;serializer:bytes"`
	TraceJSON string      `gorm:"type:jsonb"`
	CreatedAt uint64
	UpdatedAt uint64
}

func (TransactionTrace) TableName() string {
	return "transaction_traces"
}

type TransactionTraceView interface {
	GetTransactionTrace(chainID uint64, txHash common.Hash) (*TransactionTrace, error)
}

type TransactionTraceDB interface {
	TransactionTraceView
	UpsertTransactionTrace(items []TransactionTrace) error
}

type transactionTraceDB struct {
	gorm *gorm.DB
}

func NewTransactionTraceDB(db *gorm.DB) TransactionTraceDB {
	return &transactionTraceDB{gorm: db}
}

func (db *transactionTraceDB) UpsertTransactionTrace(items []TransactionTrace) error {
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
	return db.gorm.Table((TransactionTrace{}).TableName()).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "chain_id"},
			{Name: "tx_hash"},
		},
		DoUpdates: clause.AssignmentColumns([]string{"trace_json", "updated_at"}),
	}).Create(&items).Error
}

func (db *transactionTraceDB) GetTransactionTrace(chainID uint64, txHash common.Hash) (*TransactionTrace, error) {
	var out TransactionTrace
	err := db.gorm.Table((TransactionTrace{}).TableName()).
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
