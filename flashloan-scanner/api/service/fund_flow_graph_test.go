package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildFundFlowGraphForAaveV3(t *testing.T) {
	response := &TransactionDetailResponse{
		FromAddress: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		TraceSummary: &TransactionTraceSummary{
			Status: traceStatusAvailable,
			Frames: []TraceFrameDTO{
				{ID: "borrow", CallIndex: 1, Depth: 1},
				{ID: "swap-out", CallIndex: 2, Depth: 2},
				{ID: "swap-back", CallIndex: 3, Depth: 2},
				{ID: "topup", CallIndex: 4, Depth: 2},
				{ID: "repay", CallIndex: 5, Depth: 2},
			},
			AssetFlows: []TraceAssetFlow{
				{FrameID: "borrow", Action: "transfer", AssetAddress: "0x1111111111111111111111111111111111111111", Source: "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Target: "0xcccccccccccccccccccccccccccccccccccccccc", Amount: "1000"},
				{FrameID: "swap-out", Action: "transferFrom", AssetAddress: "0x1111111111111111111111111111111111111111", Source: "0xcccccccccccccccccccccccccccccccccccccccc", Target: "0xdddddddddddddddddddddddddddddddddddddddd", Amount: "1000"},
				{FrameID: "swap-back", Action: "transfer", AssetAddress: "0x2222222222222222222222222222222222222222", Source: "0xdddddddddddddddddddddddddddddddddddddddd", Target: "0xcccccccccccccccccccccccccccccccccccccccc", Amount: "25"},
				{FrameID: "topup", Action: "transferFrom", AssetAddress: "0x1111111111111111111111111111111111111111", Source: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Target: "0xcccccccccccccccccccccccccccccccccccccccc", Amount: "1002"},
				{FrameID: "repay", Action: "transferFrom", AssetAddress: "0x1111111111111111111111111111111111111111", Source: "0xcccccccccccccccccccccccccccccccccccccccc", Target: "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Amount: "1002"},
			},
		},
		Interactions: []InteractionDetailResult{
			{
				InteractionID:     "aave-1",
				Protocol:          "aave_v3",
				Entrypoint:        "flashLoanSimple",
				ProviderAddress:   "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				ReceiverAddress:   "0xcccccccccccccccccccccccccccccccccccccccc",
				CallbackTarget:    "0xcccccccccccccccccccccccccccccccccccccccc",
				RepaymentSeen:     true,
				SettlementSeen:    true,
				CallbackSeen:      true,
				Verified:          true,
				Strict:            true,
				VerificationNotes: "strict",
				Legs: []InteractionLegDTO{
					{
						LegIndex:       0,
						AssetAddress:   "0x1111111111111111111111111111111111111111",
						AmountBorrowed: "1000",
						PremiumAmount:  "2",
					},
				},
			},
		},
	}

	graph := buildFundFlowGraph(response)
	require.NotNil(t, graph)
	require.Equal(t, traceStatusAvailable, graph.Status)
	require.Len(t, graph.Lanes, 1)
	require.Equal(t, "aave-1-0", graph.Lanes[0].ID)
	require.GreaterOrEqual(t, len(graph.Lanes[0].Segments), 5)
	require.Contains(t, collectLaneRoles(graph.Lanes[0]), "borrow_source")
	require.Contains(t, collectLaneRoles(graph.Lanes[0]), "receiver")
	require.Contains(t, collectLaneRoles(graph.Lanes[0]), "repayment_target")
	require.Contains(t, collectLaneRoles(graph.Lanes[0]), "topup_source")
	require.True(t, laneHasSegment(graph.Lanes[0], "0xdddddddddddddddddddddddddddddddddddddddd", "0xcccccccccccccccccccccccccccccccccccccccc", "swap"))
	require.True(t, laneHasSegment(graph.Lanes[0], "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "0xcccccccccccccccccccccccccccccccccccccccc", "repay"))
}

func TestBuildFundFlowGraphForBalancerV2(t *testing.T) {
	response := &TransactionDetailResponse{
		FromAddress: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		TraceSummary: &TransactionTraceSummary{
			Status: traceStatusAvailable,
			Frames: []TraceFrameDTO{
				{ID: "borrow", CallIndex: 1, Depth: 1},
				{ID: "swap-out", CallIndex: 2, Depth: 2},
				{ID: "swap-back", CallIndex: 3, Depth: 2},
				{ID: "repay", CallIndex: 4, Depth: 2},
			},
			AssetFlows: []TraceAssetFlow{
				{FrameID: "borrow", Action: "transfer", AssetAddress: "0x3333333333333333333333333333333333333333", Source: "0x9999999999999999999999999999999999999999", Target: "0xcccccccccccccccccccccccccccccccccccccccc", Amount: "2000"},
				{FrameID: "swap-out", Action: "transfer", AssetAddress: "0x3333333333333333333333333333333333333333", Source: "0xcccccccccccccccccccccccccccccccccccccccc", Target: "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", Amount: "2000"},
				{FrameID: "swap-back", Action: "transfer", AssetAddress: "0x4444444444444444444444444444444444444444", Source: "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", Target: "0xcccccccccccccccccccccccccccccccccccccccc", Amount: "50"},
				{FrameID: "repay", Action: "transfer", AssetAddress: "0x3333333333333333333333333333333333333333", Source: "0xcccccccccccccccccccccccccccccccccccccccc", Target: "0x9999999999999999999999999999999999999999", Amount: "2000"},
			},
		},
		Interactions: []InteractionDetailResult{
			{
				InteractionID:     "balancer-1",
				Protocol:          "balancer_v2",
				Entrypoint:        "flashLoan",
				ProviderAddress:   "0x9999999999999999999999999999999999999999",
				ReceiverAddress:   "0xcccccccccccccccccccccccccccccccccccccccc",
				CallbackTarget:    "0xcccccccccccccccccccccccccccccccccccccccc",
				RepaymentSeen:     true,
				SettlementSeen:    true,
				CallbackSeen:      true,
				Verified:          true,
				Strict:            true,
				VerificationNotes: "strict",
				Legs: []InteractionLegDTO{
					{
						LegIndex:       0,
						AssetAddress:   "0x3333333333333333333333333333333333333333",
						AmountBorrowed: "2000",
						FeeAmount:      "0",
					},
				},
			},
		},
	}

	graph := buildFundFlowGraph(response)
	require.NotNil(t, graph)
	require.Equal(t, traceStatusAvailable, graph.Status)
	require.Len(t, graph.Lanes, 1)
	require.Contains(t, collectLaneRoles(graph.Lanes[0]), "borrow_source")
	require.Contains(t, collectLaneRoles(graph.Lanes[0]), "receiver")
	require.Contains(t, collectLaneRoles(graph.Lanes[0]), "repayment_target")
	require.GreaterOrEqual(t, len(graph.Lanes[0].Segments), 3)
	require.True(t, laneHasSegment(graph.Lanes[0], "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", "0xcccccccccccccccccccccccccccccccccccccccc", "swap"))
}

func TestBuildFundFlowGraphForUniswapV2(t *testing.T) {
	response := &TransactionDetailResponse{
		FromAddress: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		TraceSummary: &TransactionTraceSummary{
			Status: traceStatusAvailable,
			Frames: []TraceFrameDTO{
				{ID: "borrow", CallIndex: 1, Depth: 1},
				{ID: "swap-out", CallIndex: 2, Depth: 2},
				{ID: "swap-back", CallIndex: 3, Depth: 2},
				{ID: "topup", CallIndex: 4, Depth: 2},
				{ID: "repay", CallIndex: 5, Depth: 2},
			},
			AssetFlows: []TraceAssetFlow{
				{FrameID: "borrow", Action: "transfer", AssetAddress: "0x5555555555555555555555555555555555555555", Source: "0xffffffffffffffffffffffffffffffffffffffff", Target: "0xcccccccccccccccccccccccccccccccccccccccc", Amount: "3000"},
				{FrameID: "swap-out", Action: "transfer", AssetAddress: "0x5555555555555555555555555555555555555555", Source: "0xcccccccccccccccccccccccccccccccccccccccc", Target: "0xabababababababababababababababababababab", Amount: "3000"},
				{FrameID: "swap-back", Action: "transfer", AssetAddress: "0x6666666666666666666666666666666666666666", Source: "0xabababababababababababababababababababab", Target: "0xcccccccccccccccccccccccccccccccccccccccc", Amount: "75"},
				{FrameID: "topup", Action: "transferFrom", AssetAddress: "0x5555555555555555555555555555555555555555", Source: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Target: "0xcccccccccccccccccccccccccccccccccccccccc", Amount: "3005"},
				{FrameID: "repay", Action: "transfer", AssetAddress: "0x5555555555555555555555555555555555555555", Source: "0xcccccccccccccccccccccccccccccccccccccccc", Target: "0xffffffffffffffffffffffffffffffffffffffff", Amount: "3005"},
			},
		},
		Interactions: []InteractionDetailResult{
			{
				InteractionID:     "uni-1",
				Protocol:          "uniswap_v2",
				Entrypoint:        "swap",
				ProviderAddress:   "0xffffffffffffffffffffffffffffffffffffffff",
				PairAddress:       "0xffffffffffffffffffffffffffffffffffffffff",
				FactoryAddress:    "0x1212121212121212121212121212121212121212",
				ReceiverAddress:   "0xcccccccccccccccccccccccccccccccccccccccc",
				CallbackTarget:    "0xcccccccccccccccccccccccccccccccccccccccc",
				RepaymentSeen:     true,
				SettlementSeen:    true,
				CallbackSeen:      true,
				Verified:          true,
				Strict:            true,
				VerificationNotes: "strict",
				Legs: []InteractionLegDTO{
					{
						LegIndex:       0,
						AssetAddress:   "0x5555555555555555555555555555555555555555",
						AmountBorrowed: "3000",
						AmountRepaid:   "3005",
					},
				},
			},
		},
	}

	graph := buildFundFlowGraph(response)
	require.NotNil(t, graph)
	require.Equal(t, traceStatusAvailable, graph.Status)
	require.Len(t, graph.Lanes, 1)
	require.Contains(t, collectLaneRoles(graph.Lanes[0]), "borrow_source")
	require.Contains(t, collectLaneRoles(graph.Lanes[0]), "receiver")
	require.Contains(t, collectLaneRoles(graph.Lanes[0]), "repayment_target")
	require.Contains(t, collectLaneRoles(graph.Lanes[0]), "topup_source")
	require.True(t, laneHasSegment(graph.Lanes[0], "0xabababababababababababababababababababab", "0xcccccccccccccccccccccccccccccccccccccccc", "swap"))
}

func collectLaneRoles(lane FundFlowLane) []string {
	roleSet := map[string]struct{}{}
	for _, node := range lane.Nodes {
		for _, role := range node.Roles {
			roleSet[role] = struct{}{}
		}
	}
	out := make([]string, 0, len(roleSet))
	for role := range roleSet {
		out = append(out, role)
	}
	return out
}

func laneHasSegment(lane FundFlowLane, from string, to string, tone string) bool {
	for _, segment := range lane.Segments {
		if sameAddress(segment.From, from) && sameAddress(segment.To, to) && segment.Tone == tone {
			return true
		}
	}
	return false
}
