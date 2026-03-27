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
	"github.com/cpchain-network/flashloan-scanner/scanner/uniswapv2"
)

type UniswapV2Verifier struct {
	eventView dbevent.ContractEventsView
	txView    dbscanner.ObservedTransactionView
	legView   dbscanner.InteractionAssetLegDB
	pairABI   abi.ABI
}

func NewUniswapV2Verifier(eventView dbevent.ContractEventsView, txView dbscanner.ObservedTransactionView, legView dbscanner.InteractionAssetLegDB) (*UniswapV2Verifier, error) {
	pairABI, err := uniswapv2.PairABI()
	if err != nil {
		return nil, err
	}
	return &UniswapV2Verifier{
		eventView: eventView,
		txView:    txView,
		legView:   legView,
		pairABI:   pairABI,
	}, nil
}

func (v *UniswapV2Verifier) Protocol() scanner.Protocol {
	return scanner.ProtocolUniswapV2
}

func (v *UniswapV2Verifier) Verify(_ context.Context, chainID uint64, interaction scanner.CandidateInteraction) (*scanner.VerifiedInteraction, []scanner.InteractionLeg, error) {
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
	if interaction.DataNonEmpty == nil || !*interaction.DataNonEmpty {
		reason := "empty_swap_data"
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
	swapEvents := v.matchSwapEvents(events, interaction.ProviderAddress)
	if len(swapEvents) == 0 {
		reason := "missing_swap_event"
		return &scanner.VerifiedInteraction{
			InteractionID:   interaction.InteractionID,
			ExclusionReason: &reason,
		}, convertLegs(legs), nil
	}

	matched, swapEvent := matchUniswapSwapEvent(swapEvents, interaction.ReceiverAddress, legs)
	if !matched {
		reason := "swap_event_mismatch"
		return &scanner.VerifiedInteraction{
			InteractionID:   interaction.InteractionID,
			ExclusionReason: &reason,
		}, convertLegs(legs), nil
	}

	verifiedLegs := make([]scanner.InteractionLeg, 0, len(legs))
	for _, leg := range legs {
		outLeg := convertLeg(leg)
		outLeg.EventSeen = true
		outLeg.StrictLeg = true
		outLeg.SettlementMode = stringPtr("invariant_restored")
		if outLeg.TokenSide != nil {
			switch *outLeg.TokenSide {
			case "token0":
				if swapEvent.Amount0In != nil && swapEvent.Amount0In.Sign() > 0 {
					outLeg.AmountIn = stringPtr(swapEvent.Amount0In.String())
				}
			case "token1":
				if swapEvent.Amount1In != nil && swapEvent.Amount1In.Sign() > 0 {
					outLeg.AmountIn = stringPtr(swapEvent.Amount1In.String())
				}
			}
		}
		verifiedLegs = append(verifiedLegs, outLeg)
	}

	note := "event-backed Uniswap V2 verification; callback and invariant restoration inferred from successful official pair swap with non-empty data and matching Swap event"
	return &scanner.VerifiedInteraction{
		InteractionID:     interaction.InteractionID,
		Verified:          true,
		Strict:            true,
		CallbackSeen:      true,
		SettlementSeen:    true,
		RepaymentSeen:     true,
		VerificationNotes: &note,
	}, verifiedLegs, nil
}

type uniswapV2SwapEvent struct {
	To         common.Address
	Amount0In  *big.Int
	Amount1In  *big.Int
	Amount0Out *big.Int
	Amount1Out *big.Int
}

func (v *UniswapV2Verifier) matchSwapEvents(events []dbevent.ContractEvent, providerAddress string) []uniswapV2SwapEvent {
	eventSpec := v.pairABI.Events["Swap"]
	pair := common.HexToAddress(providerAddress)
	out := make([]uniswapV2SwapEvent, 0)
	for _, event := range events {
		if event.ContractAddress != pair {
			continue
		}
		if len(event.RLPLog.Topics) == 0 || event.RLPLog.Topics[0] != eventSpec.ID {
			continue
		}
		var decoded struct {
			Amount0In  *big.Int
			Amount1In  *big.Int
			Amount0Out *big.Int
			Amount1Out *big.Int
		}
		if err := v.pairABI.UnpackIntoInterface(&decoded, "Swap", event.RLPLog.Data); err != nil {
			continue
		}
		if len(event.RLPLog.Topics) < 3 {
			continue
		}
		out = append(out, uniswapV2SwapEvent{
			To:         common.BytesToAddress(event.RLPLog.Topics[2].Bytes()),
			Amount0In:  decoded.Amount0In,
			Amount1In:  decoded.Amount1In,
			Amount0Out: decoded.Amount0Out,
			Amount1Out: decoded.Amount1Out,
		})
	}
	return out
}

func matchUniswapSwapEvent(events []uniswapV2SwapEvent, receiver string, legs []dbscanner.InteractionAssetLeg) (bool, *uniswapV2SwapEvent) {
	for _, event := range events {
		if event.To.Hex() != receiver {
			continue
		}
		if matchesUniswapLegs(event, legs) {
			copied := event
			return true, &copied
		}
	}
	return false, nil
}

func matchesUniswapLegs(event uniswapV2SwapEvent, legs []dbscanner.InteractionAssetLeg) bool {
	var want0, want1 string
	for _, leg := range legs {
		if leg.TokenSide != nil && *leg.TokenSide == "token0" {
			want0 = bigIntString(leg.AmountBorrowed)
		}
		if leg.TokenSide != nil && *leg.TokenSide == "token1" {
			want1 = bigIntString(leg.AmountBorrowed)
		}
	}
	if want0 != "" {
		if event.Amount0Out == nil || event.Amount0Out.String() != want0 {
			return false
		}
	}
	if want1 != "" {
		if event.Amount1Out == nil || event.Amount1Out.String() != want1 {
			return false
		}
	}
	return want0 != "" || want1 != ""
}
