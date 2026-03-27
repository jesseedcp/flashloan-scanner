package scanner

import (
	"errors"
	"math/big"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ScannerCursor struct {
	ScannerName string   `gorm:"primaryKey"`
	ChainID     uint64   `gorm:"primaryKey"`
	CursorType  string   `gorm:"primaryKey"`
	BlockNumber *big.Int `gorm:"serializer:u256"`
	UpdatedAt   uint64
}

func (ScannerCursor) TableName() string {
	return "scanner_cursors"
}

type ScannerCursorDB interface {
	GetScannerCursor(scannerName string, chainID uint64, cursorType string) (*ScannerCursor, error)
	SaveScannerCursor(cursor ScannerCursor) error
}

type scannerCursorDB struct {
	gorm *gorm.DB
}

func NewScannerCursorDB(db *gorm.DB) ScannerCursorDB {
	return &scannerCursorDB{gorm: db}
}

func (db *scannerCursorDB) GetScannerCursor(scannerName string, chainID uint64, cursorType string) (*ScannerCursor, error) {
	var out ScannerCursor
	err := db.gorm.Table((ScannerCursor{}).TableName()).
		Where("scanner_name = ? AND chain_id = ? AND cursor_type = ?", scannerName, chainID, cursorType).
		Take(&out).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &out, nil
}

func (db *scannerCursorDB) SaveScannerCursor(cursor ScannerCursor) error {
	cursor.UpdatedAt = uint64(time.Now().Unix())
	return db.gorm.Table((ScannerCursor{}).TableName()).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "scanner_name"},
			{Name: "chain_id"},
			{Name: "cursor_type"},
		},
		DoUpdates: clause.AssignmentColumns([]string{"block_number", "updated_at"}),
	}).Create(&cursor).Error
}
