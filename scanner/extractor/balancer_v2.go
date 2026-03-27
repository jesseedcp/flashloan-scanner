package extractor

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/google/uuid"

	dbevent "github.com/cpchain-network/flashloan-scanner/database/event"
	"github.com/cpchain-network/flashloan-scanner/scanner"
	"github.com/cpchain-network/flashloan-scanner/scanner/balancer"
	"github.com/cpchain-network/flashloan-scanner/scanner/registry"
)

type BalancerV2CandidateExtractor struct {
	registry  registry.Registry
	vaultABI  abi.ABI
	eventView dbevent.ContractEventsView
}

func NewBalancerV2CandidateExtractor(reg registry.Registry, eventView dbevent.ContractEventsView) (*BalancerV2CandidateExtractor, error) {
	vaultABI, err := balancer.VaultABI()
	if err != nil {
		return nil, err
	}
	return &BalancerV2CandidateExtractor{
		registry:  reg,
		vaultABI:  vaultABI,
		eventView: eventView,
	}, nil
}

func (e *BalancerV2CandidateExtractor) Protocol() scanner.Protocol {
	return scanner.ProtocolBalancerV2
}

func (e *BalancerV2CandidateExtractor) Extract(_ context.Context, chainID uint64, txs []scanner.ObservedTransaction) ([]scanner.CandidateInteraction, []scanner.InteractionLeg, error) {
	var interactions []scanner.CandidateInteraction
	var legs []scanner.InteractionLeg

	for _, tx := range txs {
		interaction, newLegs, extracted, err := e.tryExtractFromTopLevel(tx)
		if err != nil {
			return nil, nil, err
		}
		if extracted {
			interactions = append(interactions, interaction)
			legs = append(legs, newLegs...)
			continue
		}

		derivedInteractions, derivedLegs, err := e.extractFromFlashLoanEvents(chainID, tx)
		if err != nil {
			return nil, nil, err
		}
		interactions = append(interactions, derivedInteractions...)
		legs = append(legs, derivedLegs...)
	}

	return interactions, legs, nil
}

type balancerFlashLoanArgs struct {
	Recipient common.Address
	Tokens    []common.Address
	Amounts   []*big.Int
	UserData  []byte
}

type balancerFlashLoanEvent struct {
	Provider  common.Address
	Recipient common.Address
	Token     common.Address
	Amount    *big.Int
	FeeAmount *big.Int
}

func (e *BalancerV2CandidateExtractor) tryExtractFromTopLevel(tx scanner.ObservedTransaction) (scanner.CandidateInteraction, []scanner.InteractionLeg, bool, error) {
	if tx.ToAddress == "" || !e.registry.IsOfficialBalancerVault(tx.ChainID, tx.ToAddress) {
		return scanner.CandidateInteraction{}, nil, false, nil
	}
	if len(tx.InputData) < 10 {
		return scanner.CandidateInteraction{}, nil, false, nil
	}

	payload, err := hexutil.Decode(tx.InputData)
	if err != nil || len(payload) < 4 {
		return scanner.CandidateInteraction{}, nil, false, nil
	}
	method, err := e.vaultABI.MethodById(payload[:4])
	if err != nil || method.Name != "flashLoan" {
		return scanner.CandidateInteraction{}, nil, false, nil
	}

	interaction, legs, err := e.extractFlashLoan(tx, payload, method)
	if err != nil {
		return scanner.CandidateInteraction{}, nil, false, err
	}
	return interaction, legs, true, nil
}

func (e *BalancerV2CandidateExtractor) extractFromFlashLoanEvents(chainID uint64, tx scanner.ObservedTransaction) ([]scanner.CandidateInteraction, []scanner.InteractionLeg, error) {
	if e.eventView == nil {
		return nil, nil, nil
	}

	events, err := e.eventView.ChainContractEventsByTxHash(fmt.Sprintf("%d", chainID), common.HexToHash(tx.TxHash))
	if err != nil {
		return nil, nil, err
	}

	flashLoanEvents := e.matchFlashLoanEvents(chainID, events)
	if len(flashLoanEvents) == 0 {
		return nil, nil, nil
	}

	type groupedInteraction struct {
		interaction scanner.CandidateInteraction
		legs        []scanner.InteractionLeg
	}

	var interactions []scanner.CandidateInteraction
	var legs []scanner.InteractionLeg
	groupOrder := make([]string, 0)
	grouped := make(map[string]*groupedInteraction)

	for _, event := range flashLoanEvents {
		groupKey := event.Provider.Hex() + "|" + event.Recipient.Hex()
		group, ok := grouped[groupKey]
		if !ok {
			ordinal := len(groupOrder)
			groupOrder = append(groupOrder, groupKey)
			interactionID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("%d:%s:%d", tx.ChainID, tx.TxHash, ordinal)))
			group = &groupedInteraction{
				interaction: scanner.CandidateInteraction{
					InteractionID:      interactionID.String(),
					InteractionOrdinal: ordinal,
					ChainID:            tx.ChainID,
					TxHash:             tx.TxHash,
					BlockNumber:        tx.BlockNumber,
					Protocol:           scanner.ProtocolBalancerV2,
					Entrypoint:         "flashLoan:event",
					ProviderAddress:    event.Provider.Hex(),
					ReceiverAddress:    event.Recipient.Hex(),
					CallbackTarget:     event.Recipient.Hex(),
					CandidateLevel:     scanner.CandidateLevelWeak,
					RawMethodSelector:  tx.MethodSelector,
				},
			}
			grouped[groupKey] = group
		}

		leg := scanner.InteractionLeg{
			InteractionID:  group.interaction.InteractionID,
			LegIndex:       len(group.legs),
			AssetAddress:   event.Token.Hex(),
			AssetRole:      "borrowed",
			AmountBorrowed: stringPtr(bigIntString(event.Amount)),
			FeeAmount:      stringPtr(bigIntString(event.FeeAmount)),
		}
		group.legs = append(group.legs, leg)
	}

	for _, key := range groupOrder {
		group := grouped[key]
		interactions = append(interactions, group.interaction)
		legs = append(legs, group.legs...)
	}

	return interactions, legs, nil
}

func (e *BalancerV2CandidateExtractor) matchFlashLoanEvents(chainID uint64, events []dbevent.ContractEvent) []balancerFlashLoanEvent {
	eventSpec := e.vaultABI.Events["FlashLoan"]
	out := make([]balancerFlashLoanEvent, 0)
	for _, event := range events {
		provider := event.ContractAddress.Hex()
		if !e.registry.IsOfficialBalancerVault(chainID, provider) {
			continue
		}
		if len(event.RLPLog.Topics) == 0 || event.RLPLog.Topics[0] != eventSpec.ID {
			continue
		}
		var decoded struct {
			Amount    *big.Int
			FeeAmount *big.Int
		}
		if err := e.vaultABI.UnpackIntoInterface(&decoded, "FlashLoan", event.RLPLog.Data); err != nil {
			continue
		}
		if len(event.RLPLog.Topics) < 3 {
			continue
		}
		out = append(out, balancerFlashLoanEvent{
			Provider:  event.ContractAddress,
			Recipient: common.BytesToAddress(event.RLPLog.Topics[1].Bytes()),
			Token:     common.BytesToAddress(event.RLPLog.Topics[2].Bytes()),
			Amount:    decoded.Amount,
			FeeAmount: decoded.FeeAmount,
		})
	}
	return out
}

func (e *BalancerV2CandidateExtractor) extractFlashLoan(tx scanner.ObservedTransaction, payload []byte, method *abi.Method) (scanner.CandidateInteraction, []scanner.InteractionLeg, error) {
	values, err := method.Inputs.Unpack(payload[4:])
	if err != nil {
		return scanner.CandidateInteraction{}, nil, err
	}
	if len(values) != 4 {
		return scanner.CandidateInteraction{}, nil, fmt.Errorf("unexpected flashLoan input length: %d", len(values))
	}

	args := balancerFlashLoanArgs{
		Recipient: values[0].(common.Address),
		Tokens:    values[1].([]common.Address),
		Amounts:   values[2].([]*big.Int),
		UserData:  values[3].([]byte),
	}
	interactionID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("%d:%s:%d", tx.ChainID, tx.TxHash, 0)))
	dataNonEmpty := len(args.UserData) > 0
	interaction := scanner.CandidateInteraction{
		InteractionID:      interactionID.String(),
		InteractionOrdinal: 0,
		ChainID:            tx.ChainID,
		TxHash:             tx.TxHash,
		BlockNumber:        tx.BlockNumber,
		Protocol:           scanner.ProtocolBalancerV2,
		Entrypoint:         "flashLoan",
		ProviderAddress:    tx.ToAddress,
		ReceiverAddress:    args.Recipient.Hex(),
		CallbackTarget:     args.Recipient.Hex(),
		CandidateLevel:     scanner.CandidateLevelNormal,
		RawMethodSelector:  tx.MethodSelector,
		DataNonEmpty:       &dataNonEmpty,
	}

	legs := make([]scanner.InteractionLeg, 0, len(args.Tokens))
	for i := range args.Tokens {
		amount := ""
		if i < len(args.Amounts) && args.Amounts[i] != nil {
			amount = args.Amounts[i].String()
		}
		legs = append(legs, scanner.InteractionLeg{
			InteractionID:  interaction.InteractionID,
			LegIndex:       i,
			AssetAddress:   args.Tokens[i].Hex(),
			AssetRole:      "borrowed",
			AmountBorrowed: stringPtr(amount),
		})
	}
	return interaction, legs, nil
}

func bigIntString(v *big.Int) string {
	if v == nil {
		return ""
	}
	return v.String()
}
