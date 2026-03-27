package extractor

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/google/uuid"

	"github.com/cpchain-network/flashloan-scanner/scanner"
	"github.com/cpchain-network/flashloan-scanner/scanner/aave"
	"github.com/cpchain-network/flashloan-scanner/scanner/registry"
)

type AaveV3CandidateExtractor struct {
	registry registry.Registry
	poolABI  abi.ABI
}

func NewAaveV3CandidateExtractor(reg registry.Registry) (*AaveV3CandidateExtractor, error) {
	poolABI, err := aave.PoolABI()
	if err != nil {
		return nil, err
	}
	return &AaveV3CandidateExtractor{
		registry: reg,
		poolABI:  poolABI,
	}, nil
}

func (e *AaveV3CandidateExtractor) Protocol() scanner.Protocol {
	return scanner.ProtocolAaveV3
}

func (e *AaveV3CandidateExtractor) Extract(_ context.Context, chainID uint64, txs []scanner.ObservedTransaction) ([]scanner.CandidateInteraction, []scanner.InteractionLeg, error) {
	var interactions []scanner.CandidateInteraction
	var legs []scanner.InteractionLeg

	for _, tx := range txs {
		if tx.ToAddress == "" || !e.registry.IsOfficialAavePool(chainID, tx.ToAddress) {
			continue
		}
		if len(tx.InputData) < 10 {
			continue
		}

		payload, err := hexutil.Decode(tx.InputData)
		if err != nil || len(payload) < 4 {
			continue
		}
		method, err := e.poolABI.MethodById(payload[:4])
		if err != nil {
			continue
		}

		switch method.Name {
		case "flashLoan":
			interaction, newLegs, err := e.extractFlashLoan(tx, payload, method)
			if err != nil {
				return nil, nil, err
			}
			interactions = append(interactions, interaction)
			legs = append(legs, newLegs...)
		case "flashLoanSimple":
			interaction, newLegs, err := e.extractFlashLoanSimple(tx, payload, method)
			if err != nil {
				return nil, nil, err
			}
			interactions = append(interactions, interaction)
			legs = append(legs, newLegs...)
		}
	}

	return interactions, legs, nil
}

type flashLoanArgs struct {
	ReceiverAddress   common.Address
	Assets            []common.Address
	Amounts           []*big.Int
	InterestRateModes []*big.Int
	OnBehalfOf        common.Address
	Params            []byte
	ReferralCode      uint16
}

type flashLoanSimpleArgs struct {
	ReceiverAddress common.Address
	Asset           common.Address
	Amount          *big.Int
	Params          []byte
	ReferralCode    uint16
}

func (e *AaveV3CandidateExtractor) extractFlashLoan(tx scanner.ObservedTransaction, payload []byte, method *abi.Method) (scanner.CandidateInteraction, []scanner.InteractionLeg, error) {
	values, err := method.Inputs.Unpack(payload[4:])
	if err != nil {
		return scanner.CandidateInteraction{}, nil, err
	}
	if len(values) != 7 {
		return scanner.CandidateInteraction{}, nil, fmt.Errorf("unexpected flashLoan input length: %d", len(values))
	}

	args := flashLoanArgs{
		ReceiverAddress:   values[0].(common.Address),
		Assets:            values[1].([]common.Address),
		Amounts:           values[2].([]*big.Int),
		InterestRateModes: values[3].([]*big.Int),
		OnBehalfOf:        values[4].(common.Address),
		Params:            values[5].([]byte),
		ReferralCode:      values[6].(uint16),
	}
	interactionID := interactionUUID(tx.ChainID, tx.TxHash, 0)
	interaction := scanner.CandidateInteraction{
		InteractionID:      interactionID.String(),
		InteractionOrdinal: 0,
		ChainID:            tx.ChainID,
		TxHash:             tx.TxHash,
		BlockNumber:        tx.BlockNumber,
		Protocol:           scanner.ProtocolAaveV3,
		Entrypoint:         "flashLoan",
		ProviderAddress:    tx.ToAddress,
		ReceiverAddress:    args.ReceiverAddress.Hex(),
		OnBehalfOf:         args.OnBehalfOf.Hex(),
		CandidateLevel:     scanner.CandidateLevelNormal,
		RawMethodSelector:  tx.MethodSelector,
	}
	legs := make([]scanner.InteractionLeg, 0, len(args.Assets))
	for i := range args.Assets {
		mode := 0
		if i < len(args.InterestRateModes) && args.InterestRateModes[i] != nil {
			mode = int(args.InterestRateModes[i].Int64())
		}
		modeCopy := mode
		amount := ""
		if i < len(args.Amounts) && args.Amounts[i] != nil {
			amount = args.Amounts[i].String()
		}
		legs = append(legs, scanner.InteractionLeg{
			InteractionID:    interaction.InteractionID,
			LegIndex:         i,
			AssetAddress:     args.Assets[i].Hex(),
			AssetRole:        "borrowed",
			AmountBorrowed:   stringPtr(amount),
			InterestRateMode: &modeCopy,
		})
	}
	return interaction, legs, nil
}

func (e *AaveV3CandidateExtractor) extractFlashLoanSimple(tx scanner.ObservedTransaction, payload []byte, method *abi.Method) (scanner.CandidateInteraction, []scanner.InteractionLeg, error) {
	values, err := method.Inputs.Unpack(payload[4:])
	if err != nil {
		return scanner.CandidateInteraction{}, nil, err
	}
	if len(values) != 5 {
		return scanner.CandidateInteraction{}, nil, fmt.Errorf("unexpected flashLoanSimple input length: %d", len(values))
	}
	args := flashLoanSimpleArgs{
		ReceiverAddress: values[0].(common.Address),
		Asset:           values[1].(common.Address),
		Amount:          values[2].(*big.Int),
		Params:          values[3].([]byte),
		ReferralCode:    values[4].(uint16),
	}
	interactionID := interactionUUID(tx.ChainID, tx.TxHash, 0)
	mode := 0
	interaction := scanner.CandidateInteraction{
		InteractionID:      interactionID.String(),
		InteractionOrdinal: 0,
		ChainID:            tx.ChainID,
		TxHash:             tx.TxHash,
		BlockNumber:        tx.BlockNumber,
		Protocol:           scanner.ProtocolAaveV3,
		Entrypoint:         "flashLoanSimple",
		ProviderAddress:    tx.ToAddress,
		ReceiverAddress:    args.ReceiverAddress.Hex(),
		CandidateLevel:     scanner.CandidateLevelNormal,
		RawMethodSelector:  tx.MethodSelector,
	}
	legs := []scanner.InteractionLeg{{
		InteractionID:    interaction.InteractionID,
		LegIndex:         0,
		AssetAddress:     args.Asset.Hex(),
		AssetRole:        "borrowed",
		AmountBorrowed:   stringPtr(args.Amount.String()),
		InterestRateMode: &mode,
	}}
	return interaction, legs, nil
}

func interactionUUID(chainID uint64, txHash string, ordinal int) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("%d:%s:%d", chainID, txHash, ordinal)))
}

func stringPtr(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}
