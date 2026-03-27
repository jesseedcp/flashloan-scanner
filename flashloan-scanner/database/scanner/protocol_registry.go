package scanner

import (
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ProtocolAddress struct {
	ChainID         uint64         `gorm:"primaryKey"`
	Protocol        string         `gorm:"primaryKey"`
	AddressRole     string         `gorm:"primaryKey"`
	ContractAddress common.Address `gorm:"primaryKey;serializer:bytes"`
	IsOfficial      bool
	Source          string
	CreatedAt       uint64
	UpdatedAt       uint64
}

func (ProtocolAddress) TableName() string {
	return "protocol_addresses"
}

type UniswapV2Pair struct {
	ChainID        uint64         `gorm:"primaryKey"`
	FactoryAddress common.Address `gorm:"serializer:bytes"`
	PairAddress    common.Address `gorm:"primaryKey;serializer:bytes"`
	Token0         common.Address `gorm:"serializer:bytes"`
	Token1         common.Address `gorm:"serializer:bytes"`
	CreatedBlock   *big.Int       `gorm:"serializer:u256"`
	IsOfficial     bool
	CreatedAt      uint64
	UpdatedAt      uint64
}

func (UniswapV2Pair) TableName() string {
	return "uniswap_v2_pairs"
}

type ObservedTransaction struct {
	ChainID           uint64      `gorm:"primaryKey"`
	TxHash            common.Hash `gorm:"primaryKey;serializer:bytes"`
	BlockNumber       *big.Int    `gorm:"serializer:u256"`
	TxIndex           uint64
	FromAddress       common.Address  `gorm:"serializer:bytes"`
	ToAddress         *common.Address `gorm:"serializer:bytes"`
	Status            uint8
	Value             *big.Int `gorm:"serializer:u256"`
	InputData         string
	MethodSelector    string
	GasUsed           *big.Int `gorm:"serializer:u256"`
	EffectiveGasPrice *big.Int `gorm:"serializer:u256"`
	CreatedAt         uint64
	UpdatedAt         uint64
}

func (ObservedTransaction) TableName() string {
	return "observed_transactions"
}

type ProtocolAddressView interface {
	ListProtocolAddresses(chainID uint64, protocol string) ([]ProtocolAddress, error)
	GetProtocolAddress(chainID uint64, protocol, role string, address common.Address) (*ProtocolAddress, error)
	ListOfficialProtocolAddresses(chainID uint64) ([]ProtocolAddress, error)
}

type ProtocolAddressDB interface {
	ProtocolAddressView
	UpsertProtocolAddresses(items []ProtocolAddress) error
}

type UniswapV2PairView interface {
	GetUniswapV2Pair(chainID uint64, pairAddress common.Address) (*UniswapV2Pair, error)
	ListUniswapV2Pairs(chainID uint64) ([]UniswapV2Pair, error)
}

type UniswapV2PairDB interface {
	UniswapV2PairView
	UpsertUniswapV2Pairs(items []UniswapV2Pair) error
}

type ObservedTransactionView interface {
	GetObservedTransaction(chainID uint64, txHash common.Hash) (*ObservedTransaction, error)
	ListObservedTransactionsByBlockRange(chainID uint64, fromBlock, toBlock *big.Int) ([]ObservedTransaction, error)
}

type ObservedTransactionDB interface {
	ObservedTransactionView
	UpsertObservedTransactions(items []ObservedTransaction) error
}

type protocolAddressDB struct {
	gorm *gorm.DB
}

type uniswapV2PairDB struct {
	gorm *gorm.DB
}

type observedTransactionDB struct {
	gorm *gorm.DB
}

func NewProtocolAddressDB(db *gorm.DB) ProtocolAddressDB {
	return &protocolAddressDB{gorm: db}
}

func NewUniswapV2PairDB(db *gorm.DB) UniswapV2PairDB {
	return &uniswapV2PairDB{gorm: db}
}

func NewObservedTransactionDB(db *gorm.DB) ObservedTransactionDB {
	return &observedTransactionDB{gorm: db}
}

func (db *protocolAddressDB) UpsertProtocolAddresses(items []ProtocolAddress) error {
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
	return db.gorm.Table((ProtocolAddress{}).TableName()).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "chain_id"},
			{Name: "protocol"},
			{Name: "address_role"},
			{Name: "contract_address"},
		},
		DoUpdates: clause.AssignmentColumns([]string{"is_official", "source", "updated_at"}),
	}).Create(&items).Error
}

func (db *protocolAddressDB) ListProtocolAddresses(chainID uint64, protocol string) ([]ProtocolAddress, error) {
	query := db.gorm.Table((ProtocolAddress{}).TableName()).Where("chain_id = ?", chainID)
	if protocol != "" {
		query = query.Where("protocol = ?", protocol)
	}
	var out []ProtocolAddress
	if err := query.Order("protocol ASC, address_role ASC").Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (db *protocolAddressDB) GetProtocolAddress(chainID uint64, protocol, role string, address common.Address) (*ProtocolAddress, error) {
	var out ProtocolAddress
	err := db.gorm.Table((ProtocolAddress{}).TableName()).
		Where("chain_id = ? AND protocol = ? AND address_role = ? AND contract_address = ?", chainID, protocol, role, address.Hex()).
		Take(&out).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &out, nil
}

func (db *protocolAddressDB) ListOfficialProtocolAddresses(chainID uint64) ([]ProtocolAddress, error) {
	var out []ProtocolAddress
	if err := db.gorm.Table((ProtocolAddress{}).TableName()).
		Where("chain_id = ? AND is_official = ?", chainID, true).
		Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (db *uniswapV2PairDB) UpsertUniswapV2Pairs(items []UniswapV2Pair) error {
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
	return db.gorm.Table((UniswapV2Pair{}).TableName()).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "chain_id"},
			{Name: "pair_address"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"factory_address", "token0", "token1", "created_block", "is_official", "updated_at",
		}),
	}).Create(&items).Error
}

func (db *uniswapV2PairDB) GetUniswapV2Pair(chainID uint64, pairAddress common.Address) (*UniswapV2Pair, error) {
	var out UniswapV2Pair
	err := db.gorm.Table((UniswapV2Pair{}).TableName()).
		Where("chain_id = ? AND pair_address = ?", chainID, pairAddress.Hex()).
		Take(&out).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &out, nil
}

func (db *uniswapV2PairDB) ListUniswapV2Pairs(chainID uint64) ([]UniswapV2Pair, error) {
	var out []UniswapV2Pair
	if err := db.gorm.Table((UniswapV2Pair{}).TableName()).
		Where("chain_id = ?", chainID).
		Order("created_block ASC").
		Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (db *observedTransactionDB) UpsertObservedTransactions(items []ObservedTransaction) error {
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
	return db.gorm.Table((ObservedTransaction{}).TableName()).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "chain_id"},
			{Name: "tx_hash"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"block_number", "tx_index", "from_address", "to_address", "status", "value",
			"input_data", "method_selector", "gas_used", "effective_gas_price", "updated_at",
		}),
	}).Create(&items).Error
}

func (db *observedTransactionDB) GetObservedTransaction(chainID uint64, txHash common.Hash) (*ObservedTransaction, error) {
	var out ObservedTransaction
	err := db.gorm.Table((ObservedTransaction{}).TableName()).
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

func (db *observedTransactionDB) ListObservedTransactionsByBlockRange(chainID uint64, fromBlock, toBlock *big.Int) ([]ObservedTransaction, error) {
	if fromBlock == nil {
		fromBlock = big.NewInt(0)
	}
	if toBlock == nil {
		return nil, errors.New("end block unspecified")
	}
	var out []ObservedTransaction
	err := db.gorm.Table((ObservedTransaction{}).TableName()).
		Where("chain_id = ? AND block_number >= ? AND block_number <= ?", chainID, fromBlock, toBlock).
		Order("block_number ASC, tx_index ASC").
		Find(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}
