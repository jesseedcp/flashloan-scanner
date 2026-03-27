package config

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Server struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type Database struct {
	DbHost     string `yaml:"db_host"`
	DbPort     int    `yaml:"db_port"`
	DbName     string `yaml:"db_name"`
	DbUser     string `yaml:"db_user"`
	DbPassword string `yaml:"db_password"`
}

type Contract struct {
	PoolManagerAddress    string `yaml:"pool_manager_address"`
	MessageManagerAddress string `yaml:"message_manager_address"`
}

type Token struct {
	Eth  TokenCommon `yaml:"eth"`
	USDT TokenCommon `yaml:"usdt"`
	Cp   TokenCommon `yaml:"cp"`
}

type TokenCommon struct {
	Address string `json:"address"`
	Decimal string `json:"decimal"`
	Name    string `json:"name"`
}

type RPC struct {
	RpcUrl           string   `yaml:"rpc_url"`
	ChainId          uint64   `yaml:"chain_id"`
	StartBlock       uint64   `yaml:"start_block"`
	HeaderBufferSize uint64   `yaml:"header_buffer_size"`
	EventUnpackBlock uint64   `yaml:"event_unpack_block"`
	Contracts        Contract `yaml:"contracts"`
	Tokens           Token    `yaml:"tokens"`
}

type ScannerAave struct {
	Pools []string `yaml:"pools"`
}

type ScannerBalancer struct {
	Vaults []string `yaml:"vaults"`
}

type ScannerUniswapV2Pair struct {
	FactoryAddress string `yaml:"factory_address"`
	PairAddress    string `yaml:"pair_address"`
	Token0         string `yaml:"token0"`
	Token1         string `yaml:"token1"`
	CreatedBlock   uint64 `yaml:"created_block"`
}

type ScannerUniswapV2 struct {
	Factories []string               `yaml:"factories"`
	Pairs     []ScannerUniswapV2Pair `yaml:"pairs"`
}

type Scanner struct {
	Enabled             bool             `yaml:"enabled"`
	Protocol            string           `yaml:"protocol"`
	ChainID             uint64           `yaml:"chain_id"`
	RunMode             string           `yaml:"run_mode"`
	StartBlock          uint64           `yaml:"start_block"`
	EndBlock            uint64           `yaml:"end_block"`
	BatchSize           uint64           `yaml:"batch_size"`
	LoopIntervalSeconds uint64           `yaml:"loop_interval_seconds"`
	SkipTxFetch         bool             `yaml:"skip_tx_fetch"`
	TraceEnabled        bool             `yaml:"trace_enabled"`
	Aave                ScannerAave      `yaml:"aave"`
	Balancer            ScannerBalancer  `yaml:"balancer"`
	UniswapV2           ScannerUniswapV2 `yaml:"uniswap_v2"`
}

type Config struct {
	Server                    Server   `yaml:"server"`
	WebSocketServer           Server   `yaml:"websocket_server"`
	GasOracleEndpoint         string   `yaml:"gas_oracle_endpoint"`
	RPCs                      []*RPC   `yaml:"rpcs"`
	Metrics                   Server   `yaml:"metrics"`
	MasterDb                  Database `yaml:"master_db"`
	SlaveDb                   Database `yaml:"slave_db"`
	PrivateKey                string   `yaml:"private_key"`
	NumConfirmations          uint64   `yaml:"num_confirmations"`
	SafeAbortNonceTooLowCount uint64   `yaml:"safe_abort_nonce_too_low_count"`
	CallerAddress             string   `yaml:"caller_address"`
	SlaveDbEnable             bool     `yaml:"slave_db_enable"`
	EnableApiCache            bool     `yaml:"enable_api_cache"`
	Scanner                   Scanner  `yaml:"scanner"`
}

func New(path string) (*Config, error) {
	var config = new(Config)
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func (c *Config) RPCByChainID(chainID uint64) (*RPC, error) {
	for i := range c.RPCs {
		if c.RPCs[i] != nil && c.RPCs[i].ChainId == chainID {
			return c.RPCs[i], nil
		}
	}
	return nil, fmt.Errorf("rpc config not found for chain id %d", chainID)
}
