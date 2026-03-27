package cursor

import (
	"context"
	"math/big"

	dbscanner "github.com/cpchain-network/flashloan-scanner/database/scanner"
)

type GormManager struct {
	db dbscanner.ScannerCursorDB
}

func NewGormManager(db dbscanner.ScannerCursorDB) *GormManager {
	return &GormManager{db: db}
}

func (m *GormManager) Get(_ context.Context, scannerName string, chainID uint64, cursorType string) (uint64, error) {
	item, err := m.db.GetScannerCursor(scannerName, chainID, cursorType)
	if err != nil {
		return 0, err
	}
	if item == nil || item.BlockNumber == nil {
		return 0, nil
	}
	return item.BlockNumber.Uint64(), nil
}

func (m *GormManager) Save(_ context.Context, scannerName string, chainID uint64, cursorType string, blockNumber uint64) error {
	return m.db.SaveScannerCursor(dbscanner.ScannerCursor{
		ScannerName: scannerName,
		ChainID:     chainID,
		CursorType:  cursorType,
		BlockNumber: new(big.Int).SetUint64(blockNumber),
	})
}
