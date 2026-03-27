package create_table

import (
	"fmt"

	"github.com/cpchain-network/flashloan-scanner/database"
)

func CreateTableFromTemplate(chainId string, db *database.DB) {
	createBlockHeaders(chainId, db)
	createContractEvents(chainId, db)
}

func createBlockHeaders(chainId string, db *database.DB) {
	tableName := "block_headers"
	tableNameByChainId := fmt.Sprintf("block_headers_%s", chainId)
	db.CreateTable.CreateTable(tableNameByChainId, tableName)
}

func createContractEvents(chainId string, db *database.DB) {
	tableName := "contract_events"
	tableNameByChainId := fmt.Sprintf("contract_events_%s", chainId)
	db.CreateTable.CreateTable(tableNameByChainId, tableName)
}
