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
	"github.com/cpchain-network/flashloan-scanner/scanner/registry"
	"github.com/cpchain-network/flashloan-scanner/scanner/uniswapv2"
)

type UniswapV2CandidateExtractor struct {
	registry registry.Registry
	pairABI  abi.ABI
}

func NewUniswapV2CandidateExtractor(reg registry.Registry) (*UniswapV2CandidateExtractor, error) {
	pairABI, err := uniswapv2.PairABI()
	if err != nil {
		return nil, err
	}
	return &UniswapV2CandidateExtractor{
		registry: reg,
		pairABI:  pairABI,
	}, nil
}

func (e *UniswapV2CandidateExtractor) Protocol() scanner.Protocol {
	return scanner.ProtocolUniswapV2
}

func (e *UniswapV2CandidateExtractor) Extract(_ context.Context, chainID uint64, txs []scanner.ObservedTransaction) ([]scanner.CandidateInteraction, []scanner.InteractionLeg, error) {
	var interactions []scanner.CandidateInteraction
	var legs []scanner.InteractionLeg

	for _, tx := range txs {
		if tx.ToAddress == "" || !e.registry.IsOfficialUniswapV2Pair(chainID, tx.ToAddress) {
			continue
		}
		if len(tx.InputData) < 10 {
			continue
		}

		payload, err := hexutil.Decode(tx.InputData)
		if err != nil || len(payload) < 4 {
			continue
		}
		method, err := e.pairABI.MethodById(payload[:4])
		if err != nil || method.Name != "swap" {
			continue
		}

		interaction, newLegs, err := e.extractSwap(chainID, tx, payload, method)
		if err != nil {
			return nil, nil, err
		}
		if interaction == nil {
			continue
		}
		interactions = append(interactions, *interaction)
		legs = append(legs, newLegs...)
	}

	return interactions, legs, nil
}

type uniswapV2SwapArgs struct {
	Amount0Out *big.Int
	Amount1Out *big.Int
	To         common.Address
	Data       []byte
}

func (e *UniswapV2CandidateExtractor) extractSwap(chainID uint64, tx scanner.ObservedTransaction, payload []byte, method *abi.Method) (*scanner.CandidateInteraction, []scanner.InteractionLeg, error) {
	values, err := method.Inputs.Unpack(payload[4:])
	if err != nil {
		return nil, nil, err
	}
	if len(values) != 4 {
		return nil, nil, fmt.Errorf("unexpected swap input length: %d", len(values))
	}

	args := uniswapV2SwapArgs{
		Amount0Out: values[0].(*big.Int),
		Amount1Out: values[1].(*big.Int),
		To:         values[2].(common.Address),
		Data:       values[3].([]byte),
	}
	if len(args.Data) == 0 {
		return nil, nil, nil
	}

	pairMeta, err := e.registry.GetUniswapV2Pair(chainID, tx.ToAddress)
	if err != nil {
		return nil, nil, err
	}
	if pairMeta == nil {
		return nil, nil, nil
	}

	interactionID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("%d:%s:%d", tx.ChainID, tx.TxHash, 0)))
	dataNonEmpty := true
	interaction := &scanner.CandidateInteraction{
		InteractionID:      interactionID.String(),
		InteractionOrdinal: 0,
		ChainID:            tx.ChainID,
		TxHash:             tx.TxHash,
		BlockNumber:        tx.BlockNumber,
		Protocol:           scanner.ProtocolUniswapV2,
		Entrypoint:         "swap",
		ProviderAddress:    tx.ToAddress,
		FactoryAddress:     pairMeta.FactoryAddress,
		PairAddress:        pairMeta.PairAddress,
		ReceiverAddress:    args.To.Hex(),
		CallbackTarget:     args.To.Hex(),
		CandidateLevel:     scanner.CandidateLevelNormal,
		RawMethodSelector:  tx.MethodSelector,
		DataNonEmpty:       &dataNonEmpty,
	}

	var legs []scanner.InteractionLeg
	if args.Amount0Out != nil && args.Amount0Out.Sign() > 0 {
		legs = append(legs, scanner.InteractionLeg{
			InteractionID:  interaction.InteractionID,
			LegIndex:       len(legs),
			AssetAddress:   pairMeta.Token0,
			AssetRole:      "borrowed",
			TokenSide:      stringPtr("token0"),
			AmountOut:      stringPtr(args.Amount0Out.String()),
			AmountBorrowed: stringPtr(args.Amount0Out.String()),
		})
	}
	if args.Amount1Out != nil && args.Amount1Out.Sign() > 0 {
		legs = append(legs, scanner.InteractionLeg{
			InteractionID:  interaction.InteractionID,
			LegIndex:       len(legs),
			AssetAddress:   pairMeta.Token1,
			AssetRole:      "borrowed",
			TokenSide:      stringPtr("token1"),
			AmountOut:      stringPtr(args.Amount1Out.String()),
			AmountBorrowed: stringPtr(args.Amount1Out.String()),
		})
	}
	if len(legs) == 0 {
		return nil, nil, nil
	}
	return interaction, legs, nil
}
