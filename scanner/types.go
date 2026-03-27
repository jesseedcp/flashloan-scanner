package scanner

type Protocol string

const (
	ProtocolAaveV3     Protocol = "aave_v3"
	ProtocolBalancerV2 Protocol = "balancer_v2"
	ProtocolUniswapV2  Protocol = "uniswap_v2"
)

type CandidateLevel uint8

const (
	CandidateLevelWeak   CandidateLevel = 1
	CandidateLevelNormal CandidateLevel = 2
	CandidateLevelStrict CandidateLevel = 3
)

type CursorType string

const (
	CursorTypeRawLogs          CursorType = "raw_logs"
	CursorTypeTxFetch          CursorType = "tx_fetch"
	CursorTypeCandidateExtract CursorType = "candidate_extract"
	CursorTypeVerification     CursorType = "verification"
)

type ObservedTransaction struct {
	ChainID           uint64
	TxHash            string
	BlockNumber       string
	TxIndex           uint64
	FromAddress       string
	ToAddress         string
	Status            uint8
	Value             string
	InputData         string
	MethodSelector    string
	GasUsed           string
	EffectiveGasPrice string
}

type CandidateInteraction struct {
	InteractionID      string
	InteractionOrdinal int
	ChainID            uint64
	TxHash             string
	BlockNumber        string
	Protocol           Protocol
	Entrypoint         string
	ProviderAddress    string
	FactoryAddress     string
	PairAddress        string
	ReceiverAddress    string
	CallbackTarget     string
	Initiator          string
	OnBehalfOf         string
	CandidateLevel     CandidateLevel
	RawMethodSelector  string
	DataNonEmpty       *bool
}

type VerifiedInteraction struct {
	InteractionID       string
	Verified            bool
	Strict              bool
	CallbackSeen        bool
	SettlementSeen      bool
	RepaymentSeen       bool
	ContainsDebtOpening bool
	ExclusionReason     *string
	VerificationNotes   *string
}

type InteractionLeg struct {
	InteractionID    string
	LegIndex         int
	AssetAddress     string
	AssetRole        string
	TokenSide        *string
	AmountOut        *string
	AmountIn         *string
	AmountBorrowed   *string
	AmountRepaid     *string
	PremiumAmount    *string
	FeeAmount        *string
	InterestRateMode *int
	RepaidToAddress  *string
	OpenedDebt       bool
	StrictLeg        bool
	EventSeen        bool
	SettlementMode   *string
}

type TxSummary struct {
	ChainID                           uint64
	TxHash                            string
	BlockNumber                       string
	ContainsCandidateInteraction      bool
	ContainsVerifiedInteraction       bool
	ContainsVerifiedStrictInteraction bool
	InteractionCount                  int
	StrictInteractionCount            int
	ProtocolCount                     int
	Protocols                         []Protocol
}

type UniswapV2Pair struct {
	ChainID        uint64
	FactoryAddress string
	PairAddress    string
	Token0         string
	Token1         string
	CreatedBlock   string
}

type Header struct {
	ChainID    uint64
	Hash       string
	ParentHash string
	Number     string
	Timestamp  uint64
	RLPEncoded []byte
}

type RawLog struct {
	ChainID         uint64
	BlockNumber     string
	BlockHash       string
	TxHash          string
	LogIndex        uint64
	ContractAddress string
	EventSignature  string
	RLPEncoded      []byte
}
