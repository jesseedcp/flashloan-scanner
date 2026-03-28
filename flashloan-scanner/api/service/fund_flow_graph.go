package service

import (
	"fmt"
	"sort"
	"strings"
)

type FundFlowGraph struct {
	Status     string         `json:"status"`
	EmptyLabel string         `json:"empty_label,omitempty"`
	Lanes      []FundFlowLane `json:"lanes,omitempty"`
}

type FundFlowLane struct {
	ID           string            `json:"id"`
	Label        string            `json:"label"`
	AssetAddress string            `json:"asset_address,omitempty"`
	Sublabel     string            `json:"sublabel,omitempty"`
	Nodes        []FundFlowNode    `json:"nodes"`
	Segments     []FundFlowSegment `json:"segments"`
}

type FundFlowNode struct {
	ID       string   `json:"id"`
	Roles    []string `json:"roles"`
	Title    string   `json:"title"`
	Subtitle string   `json:"subtitle"`
	Address  string   `json:"address,omitempty"`
}

type FundFlowSegment struct {
	ID           string `json:"id"`
	From         string `json:"from"`
	To           string `json:"to"`
	Action       string `json:"action"`
	Asset        string `json:"asset"`
	AssetAddress string `json:"asset_address,omitempty"`
	Amount       string `json:"amount"`
	Tone         string `json:"tone"`
}

type orderedTraceAssetFlow struct {
	TraceAssetFlow
	Order int
}

var builtinAssetLabels = map[string]string{
	"0x514910771af9ca656af840dff83e8264ecf986ca": "LINK",
	"0xc02aa39b223fe8d0a0e5c4f27ead9083c756cc2":  "WETH",
	"0x5e8c8a7243651db1384c0ddfdbe39761e8e7e51a": "aEthLINK",
	"0x4d5f47fa6a74757f35c14fd3a6ef8e3c9bc514e8": "aEthWETH",
	"0x7effd7b47bfd17e52fb7559d3f924201b9dbff3d": "varDebtLINK",
}

var builtinAddressLabels = map[string]string{
	"0x87870bca3f3fd6335c3f4ce8392d69350b4fa4e2": "Aave Pool V3",
	"0xadc0a53095a0af87f3aa29fe0715b5c28016364e": "Aave Swap Collateral Adapter V3",
	"0x5d4f3c6fa16908609bac31ff148bd002aa6b8c83": "Uniswap V3: LINK 2",
	"0x6a000f20005980200259b80c5102003040001068": "ParaSwap Augustus V6.2",
	"0x4d5f47fa6a74757f35c14fd3a6ef8e3c9bc514e8": "aEthWETH",
	"0x5e8c8a7243651db1384c0ddfdbe39761e8e7e51a": "aEthLINK",
}

func buildFundFlowGraph(response *TransactionDetailResponse) *FundFlowGraph {
	graph := &FundFlowGraph{
		Status:     traceStatusUnavailable,
		EmptyLabel: "No fund-flow path is available for this transaction.",
	}
	if response == nil {
		return graph
	}
	if response.TraceSummary != nil && response.TraceSummary.Status != "" {
		graph.Status = response.TraceSummary.Status
	}

	orderedFlows := sortFundFlowsByTrace(response)
	meaningfulFlows := make([]orderedTraceAssetFlow, 0, len(orderedFlows))
	for _, flow := range orderedFlows {
		if !isMeaningfulFundFlow(flow.Action, flow.Amount) {
			continue
		}
		if isZeroAddress(flow.Source) || isZeroAddress(flow.Target) {
			continue
		}
		meaningfulFlows = append(meaningfulFlows, flow)
	}

	traceAvailable := response.TraceSummary != nil && response.TraceSummary.Status == traceStatusAvailable && len(meaningfulFlows) > 0
	evidenceByInteractionID := mapInteractionEvidence(response.TraceSummary)

	for interactionIndex, interaction := range response.Interactions {
		receiver := firstNonEmptyString(interaction.CallbackTarget, interaction.ReceiverAddress, interaction.Initiator)
		if receiver == "" {
			continue
		}

		interactionFlows := meaningfulFlows
		if traceAvailable {
			interactionFlows = filterFlowsByInteractionEvidence(meaningfulFlows, evidenceByInteractionID[interaction.InteractionID])
		}

		for _, leg := range interaction.Legs {
			var lane *FundFlowLane
			if traceAvailable {
				lane = buildFundFlowLane(response, interaction, leg, interactionIndex, receiver, interactionFlows)
			}
			if lane == nil {
				lane = buildFallbackFundFlowLane(response, interaction, leg, interactionIndex, receiver)
			}
			if lane != nil {
				graph.Lanes = append(graph.Lanes, *lane)
			}
		}
	}

	if len(graph.Lanes) > 0 {
		graph.EmptyLabel = ""
		if traceAvailable {
			graph.Status = traceStatusAvailable
		}
	}
	return graph
}

func buildFundFlowLane(
	response *TransactionDetailResponse,
	interaction InteractionDetailResult,
	leg InteractionLegDTO,
	interactionIndex int,
	receiver string,
	orderedTraceFlows []orderedTraceAssetFlow,
) *FundFlowLane {
	nodes := make(map[string]*FundFlowNode)
	segments := make([]FundFlowSegment, 0, 8)
	borrowedAsset := normalizeAddress(leg.AssetAddress)
	borrowedAmount := normalizeAmount(leg.AmountBorrowed)
	repaidAmount := normalizeAmount(leg.AmountRepaid)
	feeAmount := sumRawAmounts(leg.PremiumAmount, leg.FeeAmount)
	expectedRepayment := feeAmount
	if repaidAmount != "0" {
		expectedRepayment = sumRawAmounts(repaidAmount, leg.PremiumAmount, leg.FeeAmount)
	} else {
		expectedRepayment = sumRawAmounts(borrowedAmount, leg.PremiumAmount, leg.FeeAmount)
	}

	pushNode := func(address string, role string) string {
		id := normalizeAddress(address)
		if id == "" {
			return ""
		}
		title := resolveFundFlowNodeTitle(address, role, interaction, response)
		if existing, ok := nodes[id]; ok {
			if !containsString(existing.Roles, role) {
				existing.Roles = append(existing.Roles, role)
				sort.Strings(existing.Roles)
			}
			if existing.Title == existing.Subtitle && title != existing.Subtitle {
				existing.Title = title
			}
			return id
		}
		nodes[id] = &FundFlowNode{
			ID:       id,
			Roles:    []string{role},
			Title:    title,
			Subtitle: shortenHash(address),
			Address:  address,
		}
		return id
	}

	pushSegment := func(id string, from string, to string, action string, assetAddress string, amount string, tone string) {
		if from == "" || to == "" {
			return
		}
		segments = append(segments, FundFlowSegment{
			ID:           id,
			From:         from,
			To:           to,
			Action:       action,
			Asset:        resolveAssetDisplayLabel(assetAddress),
			AssetAddress: assetAddress,
			Amount:       compactRawAmount(amount),
			Tone:         tone,
		})
	}

	borrowCandidates := filterTraceFlows(orderedTraceFlows, func(flow orderedTraceAssetFlow) bool {
		return sameAddress(flow.Target, receiver) && sameAddress(flow.AssetAddress, borrowedAsset)
	})
	borrowFlow := pickPreferredOrderedFlow(
		filterTraceFlows(borrowCandidates, func(flow orderedTraceAssetFlow) bool {
			return sameAddress(flow.Source, interaction.ProviderAddress) && normalizeAmount(flow.Amount) == borrowedAmount
		}),
		filterTraceFlows(borrowCandidates, func(flow orderedTraceAssetFlow) bool {
			return !sameAddress(flow.Source, response.FromAddress) && normalizeAmount(flow.Amount) == borrowedAmount
		}),
		filterTraceFlows(borrowCandidates, func(flow orderedTraceAssetFlow) bool {
			return normalizeAmount(flow.Amount) == borrowedAmount
		}),
		filterTraceFlows(borrowCandidates, func(flow orderedTraceAssetFlow) bool {
			return !sameAddress(flow.Source, response.FromAddress)
		}),
		borrowCandidates,
	)
	borrowSource := firstNonEmptyString(
		stringOrEmpty(borrowFlow, func(flow orderedTraceAssetFlow) string { return flow.Source }),
		interaction.ProviderAddress,
		interaction.FactoryAddress,
		interaction.PairAddress,
	)

	if borrowSource != "" && borrowedAmount != "0" {
		pushSegment(
			fmt.Sprintf("%s-%d-borrow-%d", interaction.InteractionID, leg.LegIndex, interactionIndex),
			pushNode(borrowSource, "borrow_source"),
			pushNode(receiver, "receiver"),
			"Borrow",
			leg.AssetAddress,
			borrowedAmount,
			"borrow",
		)
	}

	repayCandidates := filterTraceFlows(orderedTraceFlows, func(flow orderedTraceAssetFlow) bool {
		return sameAddress(flow.Source, receiver) && sameAddress(flow.AssetAddress, borrowedAsset)
	})
	repayFlow := pickPreferredOrderedFlow(
		filterTraceFlows(repayCandidates, func(flow orderedTraceAssetFlow) bool {
			return sameAddress(flow.Target, leg.RepaidToAddress) && normalizeAmount(flow.Amount) == expectedRepayment
		}),
		filterTraceFlows(repayCandidates, func(flow orderedTraceAssetFlow) bool {
			return sameAddress(flow.Target, interaction.ProviderAddress) && normalizeAmount(flow.Amount) == expectedRepayment
		}),
		filterTraceFlows(repayCandidates, func(flow orderedTraceAssetFlow) bool {
			return normalizeAmount(flow.Amount) == expectedRepayment
		}),
		filterTraceFlows(repayCandidates, func(flow orderedTraceAssetFlow) bool {
			return sameAddress(flow.Target, interaction.ProviderAddress)
		}),
		repayCandidates,
	)
	repaymentTarget := firstNonEmptyString(
		stringOrEmpty(repayFlow, func(flow orderedTraceAssetFlow) string { return flow.Target }),
		leg.RepaidToAddress,
		interaction.ProviderAddress,
		interaction.PairAddress,
		interaction.FactoryAddress,
	)

	swapSegments := extractKeyExecutionSegments(
		receiver,
		orderedTraceFlows,
		borrowSource,
		repaymentTarget,
		intOrDefault(borrowFlow, func(flow orderedTraceAssetFlow) int { return flow.Order }, -1),
	)
	executionFrameIDs := make(map[string]struct{}, len(swapSegments))
	for _, flow := range swapSegments {
		executionFrameIDs[flow.FrameID] = struct{}{}
	}

	topUpCandidates := filterTraceFlows(orderedTraceFlows, func(flow orderedTraceAssetFlow) bool {
		if borrowFlow != nil && flow.Order <= borrowFlow.Order {
			return false
		}
		if !sameAddress(flow.Target, receiver) || sameAddress(flow.Source, receiver) {
			return false
		}
		if _, usedInExecution := executionFrameIDs[flow.FrameID]; usedInExecution {
			return false
		}
		if isDebtLikeAsset(flow.AssetAddress) {
			return false
		}
		if sameAddress(flow.Source, response.FromAddress) {
			return true
		}
		if normalizeAmount(flow.Amount) == expectedRepayment {
			return true
		}
		if sameAddress(flow.AssetAddress, borrowedAsset) {
			return true
		}
		return sameAddress(flow.Source, borrowSource) || sameAddress(flow.Source, repaymentTarget)
	})
	topUpFlows := pickKeyTopupFlows(topUpCandidates, response.FromAddress, expectedRepayment)

	for index, flow := range swapSegments {
		fromRole := "hop"
		if sameAddress(flow.Source, receiver) {
			fromRole = "receiver"
		}
		toRole := "hop"
		if sameAddress(flow.Target, receiver) {
			toRole = "receiver"
		}
		pushSegment(
			fmt.Sprintf("%s-%d-swap-%d-%d", interaction.InteractionID, leg.LegIndex, interactionIndex, index),
			pushNode(flow.Source, fromRole),
			pushNode(flow.Target, toRole),
			humanizeFundFlowAction(flow.Action),
			flow.AssetAddress,
			normalizeAmount(flow.Amount),
			"swap",
		)
	}

	for index, topUpFlow := range topUpFlows {
		pushSegment(
			fmt.Sprintf("%s-%d-topup-%d-%d", interaction.InteractionID, leg.LegIndex, interactionIndex, index),
			pushNode(topUpFlow.Source, "topup_source"),
			pushNode(receiver, "receiver"),
			"Top-up",
			topUpFlow.AssetAddress,
			normalizeAmount(topUpFlow.Amount),
			"repay",
		)
	}

	if repaymentTarget != "" && (expectedRepayment != "0" || feeAmount != "0") {
		pushSegment(
			fmt.Sprintf("%s-%d-repay-%d", interaction.InteractionID, leg.LegIndex, interactionIndex),
			pushNode(receiver, "receiver"),
			pushNode(repaymentTarget, "repayment_target"),
			"Repay",
			leg.AssetAddress,
			firstNonZeroString(expectedRepayment, feeAmount),
			"repay",
		)
	}

	laneSegments := dedupeFundFlowSegments(segments)
	if len(laneSegments) == 0 {
		return nil
	}

	laneNodes := make([]FundFlowNode, 0, len(nodes))
	for _, node := range nodes {
		laneNodes = append(laneNodes, *node)
	}
	sort.SliceStable(laneNodes, func(i, j int) bool {
		return laneNodes[i].Address < laneNodes[j].Address
	})

	return &FundFlowLane{
		ID:           fmt.Sprintf("%s-%d", interaction.InteractionID, leg.LegIndex),
		Label:        resolveAssetDisplayLabel(leg.AssetAddress),
		AssetAddress: leg.AssetAddress,
		Sublabel:     fmt.Sprintf("Borrow %s · %d key execution branch(es) · Repay %s", compactRawAmount(borrowedAmount), len(swapSegments), compactRawAmount(firstNonZeroString(expectedRepayment, feeAmount))),
		Nodes:        laneNodes,
		Segments:     laneSegments,
	}
}

func buildFallbackFundFlowLane(
	response *TransactionDetailResponse,
	interaction InteractionDetailResult,
	leg InteractionLegDTO,
	interactionIndex int,
	receiver string,
) *FundFlowLane {
	nodes := make(map[string]*FundFlowNode)
	borrowedAmount := normalizeAmount(leg.AmountBorrowed)
	expectedRepayment := sumRawAmounts(
		firstNonZeroString(leg.AmountRepaid, leg.AmountBorrowed),
		leg.PremiumAmount,
		leg.FeeAmount,
	)
	provider := firstNonEmptyString(interaction.ProviderAddress, interaction.PairAddress, interaction.FactoryAddress)
	repaymentTarget := firstNonEmptyString(leg.RepaidToAddress, interaction.ProviderAddress, interaction.PairAddress, interaction.FactoryAddress)

	pushNode := func(address string, role string) string {
		id := normalizeAddress(address)
		if id == "" {
			return ""
		}
		if _, ok := nodes[id]; !ok {
			nodes[id] = &FundFlowNode{
				ID:       id,
				Roles:    []string{role},
				Title:    resolveFundFlowNodeTitle(address, role, interaction, response),
				Subtitle: shortenHash(address),
				Address:  address,
			}
		}
		return id
	}

	segments := make([]FundFlowSegment, 0, 2)
	if provider != "" && borrowedAmount != "0" {
		segments = append(segments, FundFlowSegment{
			ID:           fmt.Sprintf("%s-%d-fallback-borrow-%d", interaction.InteractionID, leg.LegIndex, interactionIndex),
			From:         pushNode(provider, "borrow_source"),
			To:           pushNode(receiver, "receiver"),
			Action:       "Borrow",
			Asset:        resolveAssetDisplayLabel(leg.AssetAddress),
			AssetAddress: leg.AssetAddress,
			Amount:       compactRawAmount(borrowedAmount),
			Tone:         "borrow",
		})
	}
	if repaymentTarget != "" && expectedRepayment != "0" {
		segments = append(segments, FundFlowSegment{
			ID:           fmt.Sprintf("%s-%d-fallback-repay-%d", interaction.InteractionID, leg.LegIndex, interactionIndex),
			From:         pushNode(receiver, "receiver"),
			To:           pushNode(repaymentTarget, "repayment_target"),
			Action:       "Repay",
			Asset:        resolveAssetDisplayLabel(leg.AssetAddress),
			AssetAddress: leg.AssetAddress,
			Amount:       compactRawAmount(expectedRepayment),
			Tone:         "repay",
		})
	}

	if len(segments) == 0 {
		return nil
	}

	laneNodes := make([]FundFlowNode, 0, len(nodes))
	for _, node := range nodes {
		laneNodes = append(laneNodes, *node)
	}
	sort.SliceStable(laneNodes, func(i, j int) bool {
		return laneNodes[i].Address < laneNodes[j].Address
	})

	return &FundFlowLane{
		ID:           fmt.Sprintf("%s-%d", interaction.InteractionID, leg.LegIndex),
		Label:        resolveAssetDisplayLabel(leg.AssetAddress),
		AssetAddress: leg.AssetAddress,
		Sublabel:     fmt.Sprintf("Borrow %s · fallback repay %s", compactRawAmount(borrowedAmount), compactRawAmount(expectedRepayment)),
		Nodes:        laneNodes,
		Segments:     dedupeFundFlowSegments(segments),
	}
}

func mapInteractionEvidence(summary *TransactionTraceSummary) map[string]TraceInteractionEvidence {
	result := map[string]TraceInteractionEvidence{}
	if summary == nil {
		return result
	}
	for _, item := range summary.InteractionEvidence {
		result[item.InteractionID] = item
	}
	return result
}

func filterFlowsByInteractionEvidence(flows []orderedTraceAssetFlow, evidence TraceInteractionEvidence) []orderedTraceAssetFlow {
	allowedIDs := make(map[string]struct{})
	for _, id := range evidence.CallbackFrameIDs {
		allowedIDs[id] = struct{}{}
	}
	for _, id := range evidence.CallbackSubtreeIDs {
		allowedIDs[id] = struct{}{}
	}
	for _, id := range evidence.RepaymentFrameIDs {
		allowedIDs[id] = struct{}{}
	}
	if len(allowedIDs) == 0 {
		return flows
	}

	filtered := make([]orderedTraceAssetFlow, 0, len(flows))
	for _, flow := range flows {
		if _, ok := allowedIDs[flow.FrameID]; ok {
			filtered = append(filtered, flow)
		}
	}
	if len(filtered) == 0 {
		return flows
	}
	return filtered
}

func sortFundFlowsByTrace(response *TransactionDetailResponse) []orderedTraceAssetFlow {
	if response == nil || response.TraceSummary == nil {
		return nil
	}
	frameOrder := map[string]int{}
	for _, frame := range response.TraceSummary.Frames {
		frameOrder[frame.ID] = frame.CallIndex*10 + frame.Depth
	}
	out := make([]orderedTraceAssetFlow, 0, len(response.TraceSummary.AssetFlows))
	for _, flow := range response.TraceSummary.AssetFlows {
		order := frameOrder[flow.FrameID]
		out = append(out, orderedTraceAssetFlow{
			TraceAssetFlow: flow,
			Order:          order,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Order != out[j].Order {
			return out[i].Order < out[j].Order
		}
		return out[i].FrameID < out[j].FrameID
	})
	return out
}

func extractKeyExecutionSegments(
	receiver string,
	indexedFlows []orderedTraceAssetFlow,
	borrowSource string,
	repaymentTarget string,
	startAfterOrder int,
) []orderedTraceAssetFlow {
	adjacency := make(map[string][]orderedTraceAssetFlow)
	for _, flow := range indexedFlows {
		if flow.Order <= startAfterOrder {
			continue
		}
		if strings.Contains(strings.ToLower(flow.Action), "approve") {
			continue
		}
		sourceKey := normalizeAddress(flow.Source)
		adjacency[sourceKey] = append(adjacency[sourceKey], flow)
	}

	receiverSeeds := make([]orderedTraceAssetFlow, 0, 4)
	for _, flow := range adjacency[normalizeAddress(receiver)] {
		if sameAddress(flow.Target, receiver) || sameAddress(flow.Target, repaymentTarget) || sameAddress(flow.Target, borrowSource) {
			continue
		}
		receiverSeeds = append(receiverSeeds, flow)
		if len(receiverSeeds) == 4 {
			break
		}
	}

	seenFrameIDs := make(map[string]struct{})
	segments := make([]orderedTraceAssetFlow, 0, 10)
	var walkBranch func(flow orderedTraceAssetFlow, depth int)
	walkBranch = func(flow orderedTraceAssetFlow, depth int) {
		if depth > 3 {
			return
		}
		if _, seen := seenFrameIDs[flow.FrameID]; seen {
			return
		}
		seenFrameIDs[flow.FrameID] = struct{}{}
		segments = append(segments, flow)
		if sameAddress(flow.Target, receiver) || sameAddress(flow.Target, repaymentTarget) {
			return
		}

		nextCandidates := adjacency[normalizeAddress(flow.Target)]
		sort.SliceStable(nextCandidates, func(i, j int) bool {
			leftAfter := 1
			if nextCandidates[i].Order >= flow.Order {
				leftAfter = 0
			}
			rightAfter := 1
			if nextCandidates[j].Order >= flow.Order {
				rightAfter = 0
			}
			if leftAfter != rightAfter {
				return leftAfter < rightAfter
			}
			leftDistance := nextCandidates[i].Order - flow.Order
			if leftDistance < 0 {
				leftDistance = -leftDistance
			}
			rightDistance := nextCandidates[j].Order - flow.Order
			if rightDistance < 0 {
				rightDistance = -rightDistance
			}
			if leftDistance != rightDistance {
				return leftDistance < rightDistance
			}
			return nextCandidates[i].FrameID < nextCandidates[j].FrameID
		})
		branchCount := 0
		for _, candidate := range nextCandidates {
			if candidate.FrameID == flow.FrameID {
				continue
			}
			if sameAddress(candidate.Target, borrowSource) && !sameAddress(candidate.Target, receiver) {
				continue
			}
			walkBranch(candidate, depth+1)
			branchCount += 1
			if branchCount == 3 {
				break
			}
		}
	}

	for _, seed := range receiverSeeds {
		walkBranch(seed, 0)
	}

	sort.SliceStable(segments, func(i, j int) bool {
		if segments[i].Order != segments[j].Order {
			return segments[i].Order < segments[j].Order
		}
		return segments[i].FrameID < segments[j].FrameID
	})
	return segments
}

func pickKeyTopupFlows(flows []orderedTraceAssetFlow, sender string, expectedRepayment string) []orderedTraceAssetFlow {
	sorted := append([]orderedTraceAssetFlow(nil), flows...)
	sort.SliceStable(sorted, func(i, j int) bool {
		leftSenderPriority := 1
		if sameAddress(sorted[i].Source, sender) {
			leftSenderPriority = 0
		}
		rightSenderPriority := 1
		if sameAddress(sorted[j].Source, sender) {
			rightSenderPriority = 0
		}
		if leftSenderPriority != rightSenderPriority {
			return leftSenderPriority < rightSenderPriority
		}
		leftMatches := 1
		if normalizeAmount(sorted[i].Amount) == expectedRepayment {
			leftMatches = 0
		}
		rightMatches := 1
		if normalizeAmount(sorted[j].Amount) == expectedRepayment {
			rightMatches = 0
		}
		if leftMatches != rightMatches {
			return leftMatches < rightMatches
		}
		return sorted[i].Order < sorted[j].Order
	})

	seen := make(map[string]struct{})
	out := make([]orderedTraceAssetFlow, 0, 3)
	for _, flow := range sorted {
		key := normalizeAddress(flow.Source) + "|" + normalizeAddress(flow.AssetAddress) + "|" + normalizeAmount(flow.Amount)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, flow)
		if len(out) == 3 {
			break
		}
	}
	return out
}

func topUpSources(flows []orderedTraceAssetFlow) []string {
	result := make([]string, 0, len(flows))
	for _, flow := range flows {
		result = append(result, flow.Source)
	}
	return result
}

func isDebtLikeAsset(assetAddress string) bool {
	label := strings.ToLower(resolveAssetDisplayLabel(assetAddress))
	return strings.Contains(label, "debt") || strings.Contains(label, "vardebt")
}

func dedupeFundFlowSegments(segments []FundFlowSegment) []FundFlowSegment {
	grouped := make(map[string]FundFlowSegment)
	order := make([]string, 0, len(segments))
	for _, segment := range segments {
		key := strings.Join([]string{
			segment.From,
			segment.To,
			normalizeAddress(segment.AssetAddress),
			segment.Tone,
			segment.Action,
		}, "|")
		if _, exists := grouped[key]; exists {
			continue
		}
		grouped[key] = segment
		order = append(order, key)
	}
	out := make([]FundFlowSegment, 0, len(order))
	for _, key := range order {
		out = append(out, grouped[key])
	}
	return out
}

func resolveFundFlowNodeTitle(address string, role string, interaction InteractionDetailResult, response *TransactionDetailResponse) string {
	if sameAddress(address, response.FromAddress) {
		return "Sender"
	}
	if known := builtinAddressLabels[normalizeAddress(address)]; known != "" {
		return known
	}

	switch role {
	case "borrow_source":
		return protocolProviderLabel(interaction.Protocol)
	case "receiver":
		return "Receiver / Callback"
	case "repayment_target":
		if sameAddress(address, interaction.ProviderAddress) || sameAddress(address, interaction.PairAddress) {
			return protocolProviderLabel(interaction.Protocol)
		}
		return "Repayment Target"
	case "topup_source":
		return "Top-up Source"
	default:
		if sameAddress(address, interaction.PairAddress) {
			return protocolPairLabel(interaction.Protocol)
		}
		if sameAddress(address, interaction.FactoryAddress) {
			return "Factory"
		}
		return shortenHash(address)
	}
}

func protocolProviderLabel(protocol string) string {
	switch protocol {
	case "aave_v3":
		return "Aave Pool"
	case "balancer_v2":
		return "Balancer Vault"
	case "uniswap_v2":
		return "Uniswap V2 Pair"
	default:
		return "Provider / Pool"
	}
}

func protocolPairLabel(protocol string) string {
	switch protocol {
	case "uniswap_v2":
		return "Uniswap V2 Pair"
	default:
		return "Pair / Pool"
	}
}

func resolveAssetDisplayLabel(assetAddress string) string {
	if assetAddress == "" {
		return "N/A"
	}
	if known := builtinAssetLabels[normalizeAddress(assetAddress)]; known != "" {
		return known
	}
	return shortenHash(assetAddress)
}

func humanizeFundFlowAction(action string) string {
	normalized := strings.ToLower(strings.TrimSpace(action))
	switch {
	case strings.Contains(normalized, "swap"):
		return "Swap"
	case strings.Contains(normalized, "repay"), strings.Contains(normalized, "return"):
		return "Repay"
	case strings.Contains(normalized, "borrow"), strings.Contains(normalized, "flash"):
		return "Borrow"
	case strings.Contains(normalized, "transferfrom"):
		return "transferFrom"
	case strings.Contains(normalized, "transfer"):
		return "transfer"
	default:
		if action == "" {
			return "Transfer"
		}
		return action
	}
}

func isMeaningfulFundFlow(action string, amount string) bool {
	if normalizeAmount(amount) == "0" {
		return false
	}
	normalizedAction := strings.ToLower(action)
	return strings.Contains(normalizedAction, "transfer") ||
		strings.Contains(normalizedAction, "withdraw") ||
		strings.Contains(normalizedAction, "deposit") ||
		strings.Contains(normalizedAction, "swap")
}

func compactRawAmount(value string) string {
	normalized := normalizeAmount(value)
	if len(normalized) <= 10 {
		return normalized
	}
	return normalized[:6] + "..." + normalized[len(normalized)-4:]
}

func sumRawAmounts(values ...string) string {
	total := "0"
	for _, value := range values {
		normalized := normalizeAmount(value)
		if normalized == "0" {
			continue
		}
		if total == "0" {
			total = normalized
			continue
		}
		total = addDecimalStrings(total, normalized)
	}
	return total
}

func addDecimalStrings(left string, right string) string {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if left == "" {
		return normalizeAmount(right)
	}
	if right == "" {
		return normalizeAmount(left)
	}

	leftRunes := []byte(left)
	rightRunes := []byte(right)
	maxLen := len(leftRunes)
	if len(rightRunes) > maxLen {
		maxLen = len(rightRunes)
	}

	carry := 0
	result := make([]byte, 0, maxLen+1)
	for i := 0; i < maxLen; i++ {
		li := len(leftRunes) - 1 - i
		ri := len(rightRunes) - 1 - i
		digit := carry
		if li >= 0 {
			digit += int(leftRunes[li] - '0')
		}
		if ri >= 0 {
			digit += int(rightRunes[ri] - '0')
		}
		result = append(result, byte('0'+digit%10))
		carry = digit / 10
	}
	if carry > 0 {
		result = append(result, byte('0'+carry))
	}
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return string(result)
}

func normalizeAmount(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "0"
	}
	return value
}

func normalizeAddress(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func sameAddress(left string, right string) bool {
	leftAddress := normalizeAddress(left)
	rightAddress := normalizeAddress(right)
	if leftAddress == "" || rightAddress == "" {
		return false
	}
	return leftAddress == rightAddress
}

func isZeroAddress(address string) bool {
	return normalizeAddress(address) == "0x0000000000000000000000000000000000000000"
}

func shortenHash(value string) string {
	if len(value) <= 18 {
		return value
	}
	return value[:10] + "..." + value[len(value)-6:]
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func firstNonZeroString(values ...string) string {
	for _, value := range values {
		if normalizeAmount(value) != "0" {
			return normalizeAmount(value)
		}
	}
	return "0"
}

func containsString(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

func filterTraceFlows(flows []orderedTraceAssetFlow, predicate func(orderedTraceAssetFlow) bool) []orderedTraceAssetFlow {
	out := make([]orderedTraceAssetFlow, 0, len(flows))
	for _, flow := range flows {
		if predicate(flow) {
			out = append(out, flow)
		}
	}
	return out
}

func pickPreferredOrderedFlow(groups ...[]orderedTraceAssetFlow) *orderedTraceAssetFlow {
	for _, group := range groups {
		if len(group) == 0 {
			continue
		}
		best := group[0]
		for _, candidate := range group[1:] {
			if candidate.Order < best.Order {
				best = candidate
			}
		}
		copy := best
		return &copy
	}
	return nil
}

func stringOrEmpty[T any](value *T, mapper func(T) string) string {
	if value == nil {
		return ""
	}
	return mapper(*value)
}

func intOrDefault[T any](value *T, mapper func(T) int, fallback int) int {
	if value == nil {
		return fallback
	}
	return mapper(*value)
}
