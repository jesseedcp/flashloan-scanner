package report

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"

	"github.com/cpchain-network/flashloan-scanner/database"
	dbscanner "github.com/cpchain-network/flashloan-scanner/database/scanner"
)

type Options struct {
	ChainID    uint64
	TxHash     string
	Limit      int
	StrictOnly bool
}

type TransactionReport struct {
	Tx           dbscanner.FlashloanTransaction
	Interactions []InteractionReport
}

type InteractionReport struct {
	Interaction dbscanner.ProtocolInteraction
	Legs        []dbscanner.InteractionAssetLeg
}

type jsonTransactionReport struct {
	TxHash                            string                  `json:"tx_hash"`
	ChainID                           uint64                  `json:"chain_id"`
	BlockNumber                       string                  `json:"block_number"`
	ContainsCandidateInteraction      bool                    `json:"contains_candidate_interaction"`
	ContainsVerifiedInteraction       bool                    `json:"contains_verified_interaction"`
	ContainsVerifiedStrictInteraction bool                    `json:"contains_verified_strict_interaction"`
	InteractionCount                  int                     `json:"interaction_count"`
	StrictInteractionCount            int                     `json:"strict_interaction_count"`
	ProtocolCount                     int                     `json:"protocol_count"`
	Protocols                         string                  `json:"protocols"`
	Interactions                      []jsonInteractionReport `json:"interactions"`
}

type jsonInteractionReport struct {
	InteractionID   string          `json:"interaction_id"`
	Protocol        string          `json:"protocol"`
	Entrypoint      string          `json:"entrypoint"`
	ProviderAddress string          `json:"provider_address"`
	ReceiverAddress string          `json:"receiver_address"`
	Verified        bool            `json:"verified"`
	Strict          bool            `json:"strict"`
	CallbackSeen    bool            `json:"callback_seen"`
	RepaymentSeen   bool            `json:"repayment_seen"`
	ExclusionReason string          `json:"exclusion_reason"`
	Legs            []jsonLegReport `json:"legs"`
}

type jsonLegReport struct {
	LegIndex         int    `json:"leg_index"`
	AssetAddress     string `json:"asset_address"`
	AssetRole        string `json:"asset_role"`
	TokenSide        string `json:"token_side"`
	AmountBorrowed   string `json:"amount_borrowed"`
	AmountRepaid     string `json:"amount_repaid"`
	PremiumAmount    string `json:"premium_amount"`
	FeeAmount        string `json:"fee_amount"`
	InterestRateMode string `json:"interest_rate_mode"`
	StrictLeg        bool   `json:"strict_leg"`
	SettlementMode   string `json:"settlement_mode"`
}

type Service struct {
	db *database.DB
}

func NewService(db *database.DB) *Service {
	return &Service{db: db}
}

func (s *Service) LoadReports(opts Options) ([]TransactionReport, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("database is required")
	}
	if opts.ChainID == 0 {
		return nil, fmt.Errorf("chain id is required")
	}

	var txs []dbscanner.FlashloanTransaction
	if opts.TxHash != "" {
		tx, err := s.db.FlashloanTx.GetFlashloanTransaction(opts.ChainID, common.HexToHash(opts.TxHash))
		if err != nil {
			return nil, err
		}
		if tx == nil {
			return nil, nil
		}
		txs = []dbscanner.FlashloanTransaction{*tx}
	} else {
		items, err := s.db.FlashloanTx.ListFlashloanTransactions(opts.ChainID, opts.StrictOnly, opts.Limit)
		if err != nil {
			return nil, err
		}
		txs = items
	}

	out := make([]TransactionReport, 0, len(txs))
	for _, tx := range txs {
		interactions, err := s.db.ProtocolInteraction.ListProtocolInteractionsByTx(opts.ChainID, tx.TxHash)
		if err != nil {
			return nil, err
		}
		report := TransactionReport{Tx: tx}
		for _, interaction := range interactions {
			legs, err := s.db.InteractionAssetLeg.ListInteractionLegs(interaction.InteractionID)
			if err != nil {
				return nil, err
			}
			report.Interactions = append(report.Interactions, InteractionReport{
				Interaction: interaction,
				Legs:        legs,
			})
		}
		out = append(out, report)
	}
	return out, nil
}

func RenderText(reports []TransactionReport) string {
	if len(reports) == 0 {
		return "no flashloan results found"
	}

	var b strings.Builder
	for i, report := range reports {
		if i > 0 {
			b.WriteString("\n\n")
		}
		tx := report.Tx
		fmt.Fprintf(&b, "tx=%s chain=%d block=%s candidate=%t verified=%t strict=%t interactions=%d strict_interactions=%d protocols=%s\n",
			tx.TxHash.Hex(),
			tx.ChainID,
			uintString(tx.BlockNumber),
			tx.ContainsCandidateInteraction,
			tx.ContainsVerifiedInteraction,
			tx.ContainsVerifiedStrictInteraction,
			tx.InteractionCount,
			tx.StrictInteractionCount,
			tx.Protocols,
		)

		sort.Slice(report.Interactions, func(i, j int) bool {
			return report.Interactions[i].Interaction.InteractionOrdinal < report.Interactions[j].Interaction.InteractionOrdinal
		})
		for _, interaction := range report.Interactions {
			item := interaction.Interaction
			fmt.Fprintf(&b, "  interaction=%s protocol=%s entry=%s provider=%s receiver=%s verified=%t strict=%t callback=%t repayment=%t exclusion=%s\n",
				item.InteractionID.String(),
				item.Protocol,
				item.Entrypoint,
				item.ProviderAddress.Hex(),
				addressString(item.ReceiverAddress),
				item.Verified,
				item.Strict,
				item.CallbackSeen,
				item.RepaymentSeen,
				item.ExclusionReason,
			)
			for _, leg := range interaction.Legs {
				fmt.Fprintf(&b, "    leg=%d asset=%s borrowed=%s repaid=%s premium=%s fee=%s mode=%s strict=%t settlement=%s\n",
					leg.LegIndex,
					leg.AssetAddress.Hex(),
					uintString(leg.AmountBorrowed),
					uintString(leg.AmountRepaid),
					uintString(leg.PremiumAmount),
					uintString(leg.FeeAmount),
					modeString(leg.InterestRateMode),
					leg.StrictLeg,
					stringPtrValue(leg.SettlementMode),
				)
			}
		}
	}
	return strings.TrimSpace(b.String())
}

func RenderJSON(reports []TransactionReport) (string, error) {
	payload, err := json.MarshalIndent(toJSONReports(reports), "", "  ")
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func WriteCSV(w io.Writer, reports []TransactionReport) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	header := []string{
		"chain_id",
		"tx_hash",
		"block_number",
		"protocols",
		"interaction_id",
		"protocol",
		"entrypoint",
		"provider_address",
		"receiver_address",
		"verified",
		"strict",
		"callback_seen",
		"repayment_seen",
		"exclusion_reason",
		"leg_index",
		"asset_address",
		"asset_role",
		"token_side",
		"amount_borrowed",
		"amount_repaid",
		"premium_amount",
		"fee_amount",
		"interest_rate_mode",
		"strict_leg",
		"settlement_mode",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	for _, report := range reports {
		for _, interaction := range report.Interactions {
			item := interaction.Interaction
			if len(interaction.Legs) == 0 {
				if err := writer.Write([]string{
					strconv.FormatUint(report.Tx.ChainID, 10),
					report.Tx.TxHash.Hex(),
					uintString(report.Tx.BlockNumber),
					report.Tx.Protocols,
					item.InteractionID.String(),
					item.Protocol,
					item.Entrypoint,
					item.ProviderAddress.Hex(),
					addressString(item.ReceiverAddress),
					strconv.FormatBool(item.Verified),
					strconv.FormatBool(item.Strict),
					strconv.FormatBool(item.CallbackSeen),
					strconv.FormatBool(item.RepaymentSeen),
					item.ExclusionReason,
					"", "", "", "", "", "", "", "", "", "", "",
				}); err != nil {
					return err
				}
				continue
			}
			for _, leg := range interaction.Legs {
				if err := writer.Write([]string{
					strconv.FormatUint(report.Tx.ChainID, 10),
					report.Tx.TxHash.Hex(),
					uintString(report.Tx.BlockNumber),
					report.Tx.Protocols,
					item.InteractionID.String(),
					item.Protocol,
					item.Entrypoint,
					item.ProviderAddress.Hex(),
					addressString(item.ReceiverAddress),
					strconv.FormatBool(item.Verified),
					strconv.FormatBool(item.Strict),
					strconv.FormatBool(item.CallbackSeen),
					strconv.FormatBool(item.RepaymentSeen),
					item.ExclusionReason,
					strconv.Itoa(leg.LegIndex),
					leg.AssetAddress.Hex(),
					leg.AssetRole,
					stringPtrValue(leg.TokenSide),
					uintString(leg.AmountBorrowed),
					uintString(leg.AmountRepaid),
					uintString(leg.PremiumAmount),
					uintString(leg.FeeAmount),
					modeString(leg.InterestRateMode),
					strconv.FormatBool(leg.StrictLeg),
					stringPtrValue(leg.SettlementMode),
				}); err != nil {
					return err
				}
			}
		}
	}

	writer.Flush()
	return writer.Error()
}

func uintString(v interface{ String() string }) string {
	if v == nil {
		return ""
	}
	return v.String()
}

func addressString(v *common.Address) string {
	if v == nil {
		return ""
	}
	return v.Hex()
}

func modeString(v *uint8) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%d", *v)
}

func stringPtrValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func toJSONReports(reports []TransactionReport) []jsonTransactionReport {
	out := make([]jsonTransactionReport, 0, len(reports))
	for _, report := range reports {
		item := jsonTransactionReport{
			TxHash:                            report.Tx.TxHash.Hex(),
			ChainID:                           report.Tx.ChainID,
			BlockNumber:                       uintString(report.Tx.BlockNumber),
			ContainsCandidateInteraction:      report.Tx.ContainsCandidateInteraction,
			ContainsVerifiedInteraction:       report.Tx.ContainsVerifiedInteraction,
			ContainsVerifiedStrictInteraction: report.Tx.ContainsVerifiedStrictInteraction,
			InteractionCount:                  report.Tx.InteractionCount,
			StrictInteractionCount:            report.Tx.StrictInteractionCount,
			ProtocolCount:                     report.Tx.ProtocolCount,
			Protocols:                         report.Tx.Protocols,
			Interactions:                      make([]jsonInteractionReport, 0, len(report.Interactions)),
		}
		for _, interaction := range report.Interactions {
			interactionItem := jsonInteractionReport{
				InteractionID:   interaction.Interaction.InteractionID.String(),
				Protocol:        interaction.Interaction.Protocol,
				Entrypoint:      interaction.Interaction.Entrypoint,
				ProviderAddress: interaction.Interaction.ProviderAddress.Hex(),
				ReceiverAddress: addressString(interaction.Interaction.ReceiverAddress),
				Verified:        interaction.Interaction.Verified,
				Strict:          interaction.Interaction.Strict,
				CallbackSeen:    interaction.Interaction.CallbackSeen,
				RepaymentSeen:   interaction.Interaction.RepaymentSeen,
				ExclusionReason: interaction.Interaction.ExclusionReason,
				Legs:            make([]jsonLegReport, 0, len(interaction.Legs)),
			}
			for _, leg := range interaction.Legs {
				interactionItem.Legs = append(interactionItem.Legs, jsonLegReport{
					LegIndex:         leg.LegIndex,
					AssetAddress:     leg.AssetAddress.Hex(),
					AssetRole:        leg.AssetRole,
					TokenSide:        stringPtrValue(leg.TokenSide),
					AmountBorrowed:   uintString(leg.AmountBorrowed),
					AmountRepaid:     uintString(leg.AmountRepaid),
					PremiumAmount:    uintString(leg.PremiumAmount),
					FeeAmount:        uintString(leg.FeeAmount),
					InterestRateMode: modeString(leg.InterestRateMode),
					StrictLeg:        leg.StrictLeg,
					SettlementMode:   stringPtrValue(leg.SettlementMode),
				})
			}
			item.Interactions = append(item.Interactions, interactionItem)
		}
		out = append(out, item)
	}
	return out
}
