package verifier

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"

	dbevent "github.com/cpchain-network/flashloan-scanner/database/event"
	dbscanner "github.com/cpchain-network/flashloan-scanner/database/scanner"
	"github.com/cpchain-network/flashloan-scanner/scanner"
	"github.com/cpchain-network/flashloan-scanner/scanner/aave"
)

type AaveV3Verifier struct {
	eventView dbevent.ContractEventsView
	txView    dbscanner.ObservedTransactionView
	legView   dbscanner.InteractionAssetLegDB
	poolABI   abi.ABI
}

func NewAaveV3Verifier(eventView dbevent.ContractEventsView, txView dbscanner.ObservedTransactionView, legView dbscanner.InteractionAssetLegDB) (*AaveV3Verifier, error) {
	poolABI, err := aave.PoolABI()
	if err != nil {
		return nil, err
	}
	return &AaveV3Verifier{
		eventView: eventView,
		txView:    txView,
		legView:   legView,
		poolABI:   poolABI,
	}, nil
}

func (v *AaveV3Verifier) Protocol() scanner.Protocol {
	return scanner.ProtocolAaveV3
}

func (v *AaveV3Verifier) Verify(_ context.Context, chainID uint64, interaction scanner.CandidateInteraction) (*scanner.VerifiedInteraction, []scanner.InteractionLeg, error) {
	txRecord, err := v.txView.GetObservedTransaction(chainID, common.HexToHash(interaction.TxHash))
	if err != nil {
		return nil, nil, err
	}
	if txRecord == nil || txRecord.Status != 1 {
		reason := "transaction_not_successful"
		return &scanner.VerifiedInteraction{
			InteractionID:   interaction.InteractionID,
			ExclusionReason: &reason,
		}, nil, nil
	}

	interactionID, err := uuid.Parse(interaction.InteractionID)
	if err != nil {
		return nil, nil, fmt.Errorf("parse interaction id: %w", err)
	}
	legs, err := v.legView.ListInteractionLegs(interactionID)
	if err != nil {
		return nil, nil, err
	}
	events, err := v.eventView.ChainContractEventsByTxHash(fmt.Sprintf("%d", chainID), common.HexToHash(interaction.TxHash))
	if err != nil {
		return nil, nil, err
	}
	flashLoanEvents := v.matchFlashLoanEvents(events, interaction.ProviderAddress)
	if len(flashLoanEvents) == 0 {
		reason := "missing_flashloan_event"
		return &scanner.VerifiedInteraction{
			InteractionID:   interaction.InteractionID,
			ExclusionReason: &reason,
		}, convertLegs(legs), nil
	}

	strict := true
	containsDebtOpening := false
	verifiedLegs := make([]scanner.InteractionLeg, 0, len(legs))
	for _, leg := range legs {
		outLeg := convertLeg(leg)
		matched, premium, mode := findMatchingAaveEvent(flashLoanEvents, leg.AssetAddress.Hex(), bigIntString(leg.AmountBorrowed), interaction.ReceiverAddress)
		if !matched {
			reason := "flashloan_event_mismatch"
			return &scanner.VerifiedInteraction{
				InteractionID:   interaction.InteractionID,
				ExclusionReason: &reason,
			}, convertLegs(legs), nil
		}
		if premium != "" {
			outLeg.PremiumAmount = stringPtr(premium)
		}
		if mode != 0 {
			containsDebtOpening = true
			strict = false
			outLeg.OpenedDebt = true
			outLeg.StrictLeg = false
			outLeg.SettlementMode = stringPtr("debt_opening")
		} else {
			outLeg.StrictLeg = true
			outLeg.EventSeen = true
			outLeg.SettlementMode = stringPtr("full_repayment")
		}
		modeCopy := mode
		outLeg.InterestRateMode = &modeCopy
		verifiedLegs = append(verifiedLegs, outLeg)
	}

	verified := &scanner.VerifiedInteraction{
		InteractionID:       interaction.InteractionID,
		Verified:            true,
		Strict:              strict,
		CallbackSeen:        true,
		SettlementSeen:      true,
		RepaymentSeen:       strict,
		ContainsDebtOpening: containsDebtOpening,
	}
	if !strict {
		note := "successful flash loan with debt opening or non-zero interest rate mode"
		verified.VerificationNotes = &note
	}
	return verified, verifiedLegs, nil
}

type aaveFlashLoanEvent struct {
	Target           common.Address
	Asset            common.Address
	Initiator        common.Address
	Amount           *big.Int
	InterestRateMode uint8
	Premium          *big.Int
	ReferralCode     uint16
}

func (v *AaveV3Verifier) matchFlashLoanEvents(events []dbevent.ContractEvent, providerAddress string) []aaveFlashLoanEvent {
	eventSpec := v.poolABI.Events["FlashLoan"]
	provider := common.HexToAddress(providerAddress)
	out := make([]aaveFlashLoanEvent, 0)
	for _, event := range events {
		if event.ContractAddress != provider {
			continue
		}
		if len(event.RLPLog.Topics) == 0 || event.RLPLog.Topics[0] != eventSpec.ID {
			continue
		}
		var decoded struct {
			Initiator        common.Address
			Amount           *big.Int
			InterestRateMode uint8
			Premium          *big.Int
		}
		if err := v.poolABI.UnpackIntoInterface(&decoded, "FlashLoan", event.RLPLog.Data); err != nil {
			continue
		}
		if len(event.RLPLog.Topics) < 4 {
			continue
		}
		referralCode := uint16(event.RLPLog.Topics[3].Big().Uint64())
		out = append(out, aaveFlashLoanEvent{
			Target:           common.BytesToAddress(event.RLPLog.Topics[1].Bytes()),
			Asset:            common.BytesToAddress(event.RLPLog.Topics[2].Bytes()),
			Initiator:        decoded.Initiator,
			Amount:           decoded.Amount,
			InterestRateMode: decoded.InterestRateMode,
			Premium:          decoded.Premium,
			ReferralCode:     referralCode,
		})
	}
	return out
}

func findMatchingAaveEvent(events []aaveFlashLoanEvent, assetAddress, amount, receiver string) (bool, string, int) {
	for _, event := range events {
		if event.Asset.Hex() != assetAddress {
			continue
		}
		if event.Target.Hex() != receiver {
			continue
		}
		if event.Amount == nil || event.Amount.String() != amount {
			continue
		}
		premium := ""
		if event.Premium != nil {
			premium = event.Premium.String()
		}
		return true, premium, int(event.InterestRateMode)
	}
	return false, "", 0
}

func convertLegs(items []dbscanner.InteractionAssetLeg) []scanner.InteractionLeg {
	out := make([]scanner.InteractionLeg, 0, len(items))
	for _, item := range items {
		out = append(out, convertLeg(item))
	}
	return out
}

func convertLeg(item dbscanner.InteractionAssetLeg) scanner.InteractionLeg {
	var rateMode *int
	if item.InterestRateMode != nil {
		mode := int(*item.InterestRateMode)
		rateMode = &mode
	}
	return scanner.InteractionLeg{
		InteractionID:    item.InteractionID.String(),
		LegIndex:         item.LegIndex,
		AssetAddress:     item.AssetAddress.Hex(),
		AssetRole:        item.AssetRole,
		TokenSide:        stringPtrValue(item.TokenSide),
		AmountOut:        stringPtr(bigIntString(item.AmountOut)),
		AmountIn:         stringPtr(bigIntString(item.AmountIn)),
		AmountBorrowed:   stringPtr(bigIntString(item.AmountBorrowed)),
		AmountRepaid:     stringPtr(bigIntString(item.AmountRepaid)),
		PremiumAmount:    stringPtr(bigIntString(item.PremiumAmount)),
		FeeAmount:        stringPtr(bigIntString(item.FeeAmount)),
		InterestRateMode: rateMode,
		RepaidToAddress:  stringPtr(addressString(item.RepaidToAddress)),
		OpenedDebt:       item.OpenedDebt,
		StrictLeg:        item.StrictLeg,
		EventSeen:        item.EventSeen,
		SettlementMode:   stringPtrValue(item.SettlementMode),
	}
}

func bigIntString(v *big.Int) string {
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

func stringPtr(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func stringPtrValue(v *string) *string {
	if v == nil || *v == "" {
		return nil
	}
	return v
}
