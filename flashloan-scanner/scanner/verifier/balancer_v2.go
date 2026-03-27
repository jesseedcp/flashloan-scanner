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
	"github.com/cpchain-network/flashloan-scanner/scanner/balancer"
)

type BalancerV2Verifier struct {
	eventView dbevent.ContractEventsView
	txView    dbscanner.ObservedTransactionView
	legView   dbscanner.InteractionAssetLegDB
	vaultABI  abi.ABI
}

func NewBalancerV2Verifier(eventView dbevent.ContractEventsView, txView dbscanner.ObservedTransactionView, legView dbscanner.InteractionAssetLegDB) (*BalancerV2Verifier, error) {
	vaultABI, err := balancer.VaultABI()
	if err != nil {
		return nil, err
	}
	return &BalancerV2Verifier{
		eventView: eventView,
		txView:    txView,
		legView:   legView,
		vaultABI:  vaultABI,
	}, nil
}

func (v *BalancerV2Verifier) Protocol() scanner.Protocol {
	return scanner.ProtocolBalancerV2
}

func (v *BalancerV2Verifier) Verify(_ context.Context, chainID uint64, interaction scanner.CandidateInteraction) (*scanner.VerifiedInteraction, []scanner.InteractionLeg, error) {
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

	verifiedLegs := make([]scanner.InteractionLeg, 0, len(legs))
	for _, leg := range legs {
		outLeg := convertLeg(leg)
		matched, feeAmount := findMatchingBalancerEvent(flashLoanEvents, leg.AssetAddress.Hex(), bigIntString(leg.AmountBorrowed), interaction.ReceiverAddress)
		if !matched {
			reason := "flashloan_event_mismatch"
			return &scanner.VerifiedInteraction{
				InteractionID:   interaction.InteractionID,
				ExclusionReason: &reason,
			}, convertLegs(legs), nil
		}
		if feeAmount != "" {
			outLeg.FeeAmount = stringPtr(feeAmount)
		}
		outLeg.EventSeen = true
		outLeg.StrictLeg = true
		outLeg.SettlementMode = stringPtr("full_repayment")
		verifiedLegs = append(verifiedLegs, outLeg)
	}

	return &scanner.VerifiedInteraction{
		InteractionID:     interaction.InteractionID,
		Verified:          true,
		Strict:            true,
		CallbackSeen:      true,
		SettlementSeen:    true,
		RepaymentSeen:     true,
		VerificationNotes: stringPtr("event-backed Balancer V2 verification; callback and repayment inferred from successful Vault flashLoan settlement"),
	}, verifiedLegs, nil
}

type balancerFlashLoanEvent struct {
	Recipient common.Address
	Token     common.Address
	Amount    *big.Int
	FeeAmount *big.Int
}

func (v *BalancerV2Verifier) matchFlashLoanEvents(events []dbevent.ContractEvent, providerAddress string) []balancerFlashLoanEvent {
	eventSpec := v.vaultABI.Events["FlashLoan"]
	provider := common.HexToAddress(providerAddress)
	out := make([]balancerFlashLoanEvent, 0)
	for _, event := range events {
		if event.ContractAddress != provider {
			continue
		}
		if len(event.RLPLog.Topics) == 0 || event.RLPLog.Topics[0] != eventSpec.ID {
			continue
		}
		var decoded struct {
			Amount    *big.Int
			FeeAmount *big.Int
		}
		if err := v.vaultABI.UnpackIntoInterface(&decoded, "FlashLoan", event.RLPLog.Data); err != nil {
			continue
		}
		if len(event.RLPLog.Topics) < 3 {
			continue
		}
		out = append(out, balancerFlashLoanEvent{
			Recipient: common.BytesToAddress(event.RLPLog.Topics[1].Bytes()),
			Token:     common.BytesToAddress(event.RLPLog.Topics[2].Bytes()),
			Amount:    decoded.Amount,
			FeeAmount: decoded.FeeAmount,
		})
	}
	return out
}

func findMatchingBalancerEvent(events []balancerFlashLoanEvent, assetAddress, amount, recipient string) (bool, string) {
	for _, event := range events {
		if event.Token.Hex() != assetAddress {
			continue
		}
		if event.Recipient.Hex() != recipient {
			continue
		}
		if event.Amount == nil || event.Amount.String() != amount {
			continue
		}
		feeAmount := ""
		if event.FeeAmount != nil {
			feeAmount = event.FeeAmount.String()
		}
		return true, feeAmount
	}
	return false, ""
}
