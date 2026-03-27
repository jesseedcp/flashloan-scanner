package service

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/common"

	"github.com/cpchain-network/flashloan-scanner/database"
	dbscanner "github.com/cpchain-network/flashloan-scanner/database/scanner"
	scannerreport "github.com/cpchain-network/flashloan-scanner/scanner/report"
)

type QueryService struct {
	db                  *database.DB
	jobManager          *JobManager
	reporter            *scannerreport.Service
	traceProviderSource TraceProviderSource
}

type SummaryResponse struct {
	ChainID           uint64 `json:"chain_id"`
	TotalTransactions int64  `json:"total_transactions"`
	TotalCandidates   int64  `json:"total_candidates"`
	TotalVerified     int64  `json:"total_verified"`
	TotalStrict       int64  `json:"total_strict"`
}

type TransactionListItem struct {
	TxHash                 string   `json:"tx_hash"`
	ChainID                uint64   `json:"chain_id"`
	BlockNumber            string   `json:"block_number"`
	Candidate              bool     `json:"candidate"`
	Verified               bool     `json:"verified"`
	Strict                 bool     `json:"strict"`
	InteractionCount       int      `json:"interaction_count"`
	StrictInteractionCount int      `json:"strict_interaction_count"`
	ProtocolCount          int      `json:"protocol_count"`
	Protocols              []string `json:"protocols"`
}

type TransactionDetailResponse struct {
	TxHash                 string                    `json:"tx_hash"`
	ChainID                uint64                    `json:"chain_id"`
	BlockNumber            string                    `json:"block_number"`
	Candidate              bool                      `json:"candidate"`
	Verified               bool                      `json:"verified"`
	Strict                 bool                      `json:"strict"`
	InteractionCount       int                       `json:"interaction_count"`
	StrictInteractionCount int                       `json:"strict_interaction_count"`
	ProtocolCount          int                       `json:"protocol_count"`
	Protocols              []string                  `json:"protocols"`
	Summary                TransactionDetailSummary  `json:"summary"`
	TraceSummary           *TransactionTraceSummary  `json:"trace_summary,omitempty"`
	Interactions           []InteractionDetailResult `json:"interactions"`
}

type TransactionDetailSummary struct {
	Addresses  []AddressRoleSummary `json:"addresses"`
	Timeline   []TimelineStep       `json:"timeline"`
	Conclusion DetectionConclusion  `json:"conclusion"`
}

type AddressRoleSummary struct {
	Address string   `json:"address"`
	Roles   []string `json:"roles"`
}

type TimelineStep struct {
	Ordinal        int    `json:"ordinal"`
	Kind           string `json:"kind"`
	Protocol       string `json:"protocol,omitempty"`
	Entrypoint     string `json:"entrypoint,omitempty"`
	AssetAddress   string `json:"asset_address,omitempty"`
	AssetRole      string `json:"asset_role,omitempty"`
	AmountBorrowed string `json:"amount_borrowed,omitempty"`
	AmountRepaid   string `json:"amount_repaid,omitempty"`
	Strict         bool   `json:"strict,omitempty"`
	EventSeen      bool   `json:"event_seen,omitempty"`
	CallbackSeen   bool   `json:"callback_seen,omitempty"`
	SettlementSeen bool   `json:"settlement_seen,omitempty"`
	RepaymentSeen  bool   `json:"repayment_seen,omitempty"`
}

type DetectionConclusion struct {
	Verdict                string   `json:"verdict"`
	Protocols              []string `json:"protocols"`
	InteractionCount       int      `json:"interaction_count"`
	StrictInteractionCount int      `json:"strict_interaction_count"`
	CallbackSeenCount      int      `json:"callback_seen_count"`
	SettlementSeenCount    int      `json:"settlement_seen_count"`
	RepaymentSeenCount     int      `json:"repayment_seen_count"`
	StrictLegCount         int      `json:"strict_leg_count"`
	DebtOpeningCount       int      `json:"debt_opening_count"`
	ExclusionReasons       []string `json:"exclusion_reasons"`
}

type InteractionDetailResult struct {
	InteractionID       string              `json:"interaction_id"`
	Protocol            string              `json:"protocol"`
	Entrypoint          string              `json:"entrypoint"`
	ProviderAddress     string              `json:"provider_address"`
	FactoryAddress      string              `json:"factory_address"`
	PairAddress         string              `json:"pair_address"`
	ReceiverAddress     string              `json:"receiver_address"`
	CallbackTarget      string              `json:"callback_target"`
	Initiator           string              `json:"initiator"`
	OnBehalfOf          string              `json:"on_behalf_of"`
	CandidateLevel      uint8               `json:"candidate_level"`
	Verified            bool                `json:"verified"`
	Strict              bool                `json:"strict"`
	CallbackSeen        bool                `json:"callback_seen"`
	SettlementSeen      bool                `json:"settlement_seen"`
	RepaymentSeen       bool                `json:"repayment_seen"`
	ContainsDebtOpening bool                `json:"contains_debt_opening"`
	ExclusionReason     string              `json:"exclusion_reason"`
	VerificationNotes   string              `json:"verification_notes"`
	RawMethodSelector   string              `json:"raw_method_selector"`
	Legs                []InteractionLegDTO `json:"legs"`
}

type InteractionLegDTO struct {
	LegIndex         int    `json:"leg_index"`
	AssetAddress     string `json:"asset_address"`
	AssetRole        string `json:"asset_role"`
	TokenSide        string `json:"token_side"`
	AmountOut        string `json:"amount_out"`
	AmountIn         string `json:"amount_in"`
	AmountBorrowed   string `json:"amount_borrowed"`
	AmountRepaid     string `json:"amount_repaid"`
	PremiumAmount    string `json:"premium_amount"`
	FeeAmount        string `json:"fee_amount"`
	InterestRateMode string `json:"interest_rate_mode"`
	RepaidToAddress  string `json:"repaid_to_address"`
	OpenedDebt       bool   `json:"opened_debt"`
	StrictLeg        bool   `json:"strict_leg"`
	EventSeen        bool   `json:"event_seen"`
	SettlementMode   string `json:"settlement_mode"`
}

func NewQueryService(db *database.DB, jobManager *JobManager, traceProviderSource TraceProviderSource) *QueryService {
	return &QueryService{
		db:                  db,
		jobManager:          jobManager,
		reporter:            scannerreport.NewService(db),
		traceProviderSource: traceProviderSource,
	}
}

func (s *QueryService) GetSummary(chainID uint64) (*SummaryResponse, error) {
	if chainID == 0 {
		return nil, fmt.Errorf("chain id is required")
	}
	summary, err := s.db.FlashloanTx.GetFlashloanTransactionSummary(chainID)
	if err != nil {
		return nil, err
	}
	return &SummaryResponse{
		ChainID:           chainID,
		TotalTransactions: summary.TotalTransactions,
		TotalCandidates:   summary.TotalCandidate,
		TotalVerified:     summary.TotalVerified,
		TotalStrict:       summary.TotalStrict,
	}, nil
}

func (s *QueryService) ListTransactions(chainID uint64, strictOnly bool, limit int) ([]TransactionListItem, error) {
	if chainID == 0 {
		return nil, fmt.Errorf("chain id is required")
	}
	items, err := s.db.FlashloanTx.ListFlashloanTransactions(chainID, strictOnly, limit)
	if err != nil {
		return nil, err
	}
	out := make([]TransactionListItem, 0, len(items))
	for _, item := range items {
		out = append(out, mapTransactionListItem(item))
	}
	return out, nil
}

func (s *QueryService) GetTransactionDetail(ctx context.Context, chainID uint64, txHash string) (*TransactionDetailResponse, error) {
	if chainID == 0 {
		return nil, fmt.Errorf("chain id is required")
	}
	if !isHexHash(txHash) {
		return nil, fmt.Errorf("invalid tx hash")
	}

	reports, err := s.reporter.LoadReports(scannerreport.Options{
		ChainID: chainID,
		TxHash:  txHash,
		Limit:   1,
	})
	if err != nil {
		return nil, err
	}
	if len(reports) == 0 {
		return nil, nil
	}

	report := reports[0]
	sort.Slice(report.Interactions, func(i, j int) bool {
		return report.Interactions[i].Interaction.InteractionOrdinal < report.Interactions[j].Interaction.InteractionOrdinal
	})

	response := &TransactionDetailResponse{
		TxHash:                 report.Tx.TxHash.Hex(),
		ChainID:                report.Tx.ChainID,
		BlockNumber:            bigIntToString(report.Tx.BlockNumber),
		Candidate:              report.Tx.ContainsCandidateInteraction,
		Verified:               report.Tx.ContainsVerifiedInteraction,
		Strict:                 report.Tx.ContainsVerifiedStrictInteraction,
		InteractionCount:       report.Tx.InteractionCount,
		StrictInteractionCount: report.Tx.StrictInteractionCount,
		ProtocolCount:          report.Tx.ProtocolCount,
		Protocols:              splitProtocols(report.Tx.Protocols),
		Interactions:           make([]InteractionDetailResult, 0, len(reports[0].Interactions)),
	}
	for _, interaction := range report.Interactions {
		response.Interactions = append(response.Interactions, mapInteractionDetail(interaction.Interaction, interaction.Legs))
	}
	response.Summary = buildTransactionDetailSummary(response)
	response.TraceSummary = buildTransactionTraceSummary(ctx, s.traceProviderSource, chainID, txHash, response.Interactions)
	return response, nil
}

func (s *QueryService) GetJobResults(jobID string) ([]JobFinding, error) {
	if s.jobManager == nil {
		return nil, fmt.Errorf("job manager is unavailable")
	}
	findings, ok := s.jobManager.GetJobFindings(jobID)
	if !ok {
		return nil, fmt.Errorf("job %s not found", jobID)
	}
	return findings, nil
}

func mapTransactionListItem(item dbscanner.FlashloanTransaction) TransactionListItem {
	return TransactionListItem{
		TxHash:                 item.TxHash.Hex(),
		ChainID:                item.ChainID,
		BlockNumber:            bigIntToString(item.BlockNumber),
		Candidate:              item.ContainsCandidateInteraction,
		Verified:               item.ContainsVerifiedInteraction,
		Strict:                 item.ContainsVerifiedStrictInteraction,
		InteractionCount:       item.InteractionCount,
		StrictInteractionCount: item.StrictInteractionCount,
		ProtocolCount:          item.ProtocolCount,
		Protocols:              splitProtocols(item.Protocols),
	}
}

func mapInteractionDetail(interaction dbscanner.ProtocolInteraction, legs []dbscanner.InteractionAssetLeg) InteractionDetailResult {
	result := InteractionDetailResult{
		InteractionID:       interaction.InteractionID.String(),
		Protocol:            interaction.Protocol,
		Entrypoint:          interaction.Entrypoint,
		ProviderAddress:     interaction.ProviderAddress.Hex(),
		FactoryAddress:      addressToString(interaction.FactoryAddress),
		PairAddress:         addressToString(interaction.PairAddress),
		ReceiverAddress:     addressToString(interaction.ReceiverAddress),
		CallbackTarget:      addressToString(interaction.CallbackTarget),
		Initiator:           addressToString(interaction.Initiator),
		OnBehalfOf:          addressToString(interaction.OnBehalfOf),
		CandidateLevel:      interaction.CandidateLevel,
		Verified:            interaction.Verified,
		Strict:              interaction.Strict,
		CallbackSeen:        interaction.CallbackSeen,
		SettlementSeen:      interaction.SettlementSeen,
		RepaymentSeen:       interaction.RepaymentSeen,
		ContainsDebtOpening: interaction.ContainsDebtOpening,
		ExclusionReason:     interaction.ExclusionReason,
		VerificationNotes:   interaction.VerificationNotes,
		RawMethodSelector:   interaction.RawMethodSelector,
		Legs:                make([]InteractionLegDTO, 0, len(legs)),
	}
	for _, leg := range legs {
		result.Legs = append(result.Legs, InteractionLegDTO{
			LegIndex:         leg.LegIndex,
			AssetAddress:     leg.AssetAddress.Hex(),
			AssetRole:        leg.AssetRole,
			TokenSide:        stringPtrValue(leg.TokenSide),
			AmountOut:        bigIntToString(leg.AmountOut),
			AmountIn:         bigIntToString(leg.AmountIn),
			AmountBorrowed:   bigIntToString(leg.AmountBorrowed),
			AmountRepaid:     bigIntToString(leg.AmountRepaid),
			PremiumAmount:    bigIntToString(leg.PremiumAmount),
			FeeAmount:        bigIntToString(leg.FeeAmount),
			InterestRateMode: uint8PtrToString(leg.InterestRateMode),
			RepaidToAddress:  addressToString(leg.RepaidToAddress),
			OpenedDebt:       leg.OpenedDebt,
			StrictLeg:        leg.StrictLeg,
			EventSeen:        leg.EventSeen,
			SettlementMode:   stringPtrValue(leg.SettlementMode),
		})
	}
	return result
}

func buildTransactionDetailSummary(response *TransactionDetailResponse) TransactionDetailSummary {
	return TransactionDetailSummary{
		Addresses:  buildAddressRoleSummaries(response.Interactions),
		Timeline:   buildTimeline(response.Interactions),
		Conclusion: buildDetectionConclusion(response),
	}
}

func buildAddressRoleSummaries(interactions []InteractionDetailResult) []AddressRoleSummary {
	roleMap := map[string]map[string]struct{}{}

	appendRole := func(address string, role string) {
		address = strings.TrimSpace(address)
		if address == "" {
			return
		}
		if roleMap[address] == nil {
			roleMap[address] = map[string]struct{}{}
		}
		roleMap[address][role] = struct{}{}
	}

	for _, interaction := range interactions {
		appendRole(interaction.Initiator, "initiator")
		appendRole(interaction.OnBehalfOf, "on_behalf_of")
		appendRole(interaction.ReceiverAddress, "receiver")
		appendRole(interaction.CallbackTarget, "callback_target")
		appendRole(interaction.ProviderAddress, "provider")
		appendRole(interaction.FactoryAddress, "factory")
		appendRole(interaction.PairAddress, "pair")
		for _, leg := range interaction.Legs {
			appendRole(leg.RepaidToAddress, "repayment_target")
		}
	}

	addresses := make([]string, 0, len(roleMap))
	for address := range roleMap {
		addresses = append(addresses, address)
	}
	sort.Strings(addresses)

	out := make([]AddressRoleSummary, 0, len(addresses))
	for _, address := range addresses {
		roles := make([]string, 0, len(roleMap[address]))
		for role := range roleMap[address] {
			roles = append(roles, role)
		}
		sort.Strings(roles)
		out = append(out, AddressRoleSummary{
			Address: address,
			Roles:   roles,
		})
	}
	return out
}

func buildTimeline(interactions []InteractionDetailResult) []TimelineStep {
	out := make([]TimelineStep, 0, len(interactions)*4)
	ordinal := 1
	for _, interaction := range interactions {
		out = append(out, TimelineStep{
			Ordinal:    ordinal,
			Kind:       "entrypoint",
			Protocol:   interaction.Protocol,
			Entrypoint: interaction.Entrypoint,
		})
		ordinal++

		for _, leg := range interaction.Legs {
			out = append(out, TimelineStep{
				Ordinal:        ordinal,
				Kind:           "asset_leg",
				Protocol:       interaction.Protocol,
				AssetAddress:   leg.AssetAddress,
				AssetRole:      leg.AssetRole,
				AmountBorrowed: leg.AmountBorrowed,
				AmountRepaid:   leg.AmountRepaid,
				Strict:         leg.StrictLeg,
				EventSeen:      leg.EventSeen,
			})
			ordinal++
		}

		out = append(out, TimelineStep{
			Ordinal:        ordinal,
			Kind:           "evidence",
			Protocol:       interaction.Protocol,
			Strict:         interaction.Strict,
			CallbackSeen:   interaction.CallbackSeen,
			SettlementSeen: interaction.SettlementSeen,
			RepaymentSeen:  interaction.RepaymentSeen,
		})
		ordinal++
	}
	return out
}

func buildDetectionConclusion(response *TransactionDetailResponse) DetectionConclusion {
	conclusion := DetectionConclusion{
		Verdict:                "candidate",
		Protocols:              append([]string(nil), response.Protocols...),
		InteractionCount:       response.InteractionCount,
		StrictInteractionCount: response.StrictInteractionCount,
		ExclusionReasons:       []string{},
	}
	if response.Verified {
		conclusion.Verdict = "verified"
	}
	if response.Strict {
		conclusion.Verdict = "strict"
	}

	exclusionSeen := map[string]struct{}{}
	for _, interaction := range response.Interactions {
		if interaction.CallbackSeen {
			conclusion.CallbackSeenCount++
		}
		if interaction.SettlementSeen {
			conclusion.SettlementSeenCount++
		}
		if interaction.RepaymentSeen {
			conclusion.RepaymentSeenCount++
		}
		if interaction.ContainsDebtOpening {
			conclusion.DebtOpeningCount++
		}
		reason := strings.TrimSpace(interaction.ExclusionReason)
		if reason != "" {
			if _, ok := exclusionSeen[reason]; !ok {
				exclusionSeen[reason] = struct{}{}
				conclusion.ExclusionReasons = append(conclusion.ExclusionReasons, reason)
			}
		}
		for _, leg := range interaction.Legs {
			if leg.StrictLeg {
				conclusion.StrictLegCount++
			}
		}
	}
	return conclusion
}

func splitProtocols(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func bigIntToString(value *big.Int) string {
	if value == nil {
		return ""
	}
	return value.String()
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func uint8PtrToString(value *uint8) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%d", *value)
}

func addressToString(value *common.Address) string {
	if value == nil {
		return ""
	}
	return value.Hex()
}

func isHexHash(value string) bool {
	if len(value) != 66 || !strings.HasPrefix(value, "0x") {
		return false
	}
	_, err := hex.DecodeString(value[2:])
	return err == nil
}
