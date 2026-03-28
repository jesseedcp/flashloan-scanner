import { useEffect, useMemo, useState } from 'react'
import {
  Background,
  BaseEdge,
  Controls,
  EdgeLabelRenderer,
  Handle,
  MarkerType,
  Position,
  ReactFlow,
  ReactFlowProvider,
  getSmoothStepPath,
  useNodesInitialized,
  useReactFlow,
  type Edge,
  type EdgeProps,
  type Node,
  type NodeProps,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'

type FlowTone = 'borrow' | 'swap' | 'repay'
type LaneNodeRole = 'borrow_source' | 'receiver' | 'hop' | 'repayment_target' | 'topup_source'
type DiagramSlotRole = LaneNodeRole | 'receiver_settlement'
type DiagramPathTrack = 'borrow' | 'execute' | 'return' | 'topup' | 'repay'

export type FundFlowLaneNode = {
  id: string
  roles: LaneNodeRole[]
  title: string
  subtitle: string
  address?: string
}

export type FundFlowLaneSegment = {
  id: string
  from: string
  to: string
  action: string
  asset: string
  amount: string
  tone: FlowTone
}

export type FundFlowLane = {
  id: string
  label: string
  sublabel?: string
  nodes: FundFlowLaneNode[]
  segments: FundFlowLaneSegment[]
}

export type FundFlowDiagramModel = {
  lanes: FundFlowLane[]
  emptyLabel: string
}

export type InvocationFlowNode = {
  id: string
  depth: number
  title: string
  detail: string
  from?: string
  to?: string
  phase: 'entry' | 'callback' | 'repayment' | 'settlement' | 'error' | 'event' | 'neutral'
  phaseLabel: string
  badges: string[]
  meta: string[]
  children: InvocationFlowNode[]
}

type FundFlowDiagramProps = {
  model: FundFlowDiagramModel
}

type InvocationFlowTreeProps = {
  nodes: InvocationFlowNode[]
  emptyLabel: string
  raw?: boolean
}

type DiagramSlot = {
  key: string
  role: DiagramSlotRole
  title: string
  subtitle: string
  address?: string
  x: number
  y: number
}

type DiagramPath = {
  id: string
  fromKey: string
  toKey: string
  tone: FlowTone
  track: DiagramPathTrack
  label: string
  verboseLabel: string
  labelOffsetY?: number
}

type ExecutionBranch = {
  id: string
  segments: FundFlowLaneSegment[]
  returnsToReceiver: boolean
  visibleDepth: number
}

type LaneRowLayout = {
  lane: FundFlowLane
  top: number
  height: number
  slots: DiagramSlot[]
  paths: DiagramPath[]
}

type UnifiedDiagramLayout = {
  width: number
  height: number
  rows: LaneRowLayout[]
}

type FundFlowGraphNodeData =
  | {
      kind: 'lane-label'
      title: string
      subtitle?: string
    }
  | {
      kind: 'slot'
      title: string
      subtitle: string
      address?: string
      role: DiagramSlotRole
      compact: boolean
    }

type FundFlowGraphEdgeData = {
  label: string
  verboseLabel: string
  tone: FlowTone
  track: DiagramPathTrack
  showLabel: boolean
  labelOffsetY?: number
}

const FUND_FLOW_NODE_WIDTH = 188
const FUND_FLOW_NODE_HEIGHT = 52

export function FundFlowDiagram({ model }: FundFlowDiagramProps) {
  const [expanded, setExpanded] = useState(false)
  const diagram = useMemo(() => buildUnifiedDiagramLayout(model.lanes), [model.lanes])

  if (diagram.rows.length === 0) {
    return <div className="forensic-empty-box">{model.emptyLabel}</div>
  }

  useEffect(() => {
    if (!expanded) {
      return undefined
    }
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setExpanded(false)
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [expanded])

  const onPreviewKeyDown = (event: React.KeyboardEvent<HTMLDivElement>) => {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault()
      setExpanded(true)
    }
  }

  return (
    <>
      <div className="forensic-diagram-shell">
        <div
          className="forensic-flow-preview"
          role="button"
          tabIndex={0}
          onClick={() => setExpanded(true)}
          onKeyDown={onPreviewKeyDown}
          aria-label="放大查看资金流向图"
        >
          <div className="forensic-flow-preview-head">
            <span className="forensic-flow-preview-chip">Overview</span>
            <span className="forensic-flow-preview-hint">点击放大</span>
          </div>
          <FundFlowLegend compact />
          <div className="forensic-flow-preview-canvas">
            <FundFlowViewport diagram={diagram} mode="preview" />
          </div>
        </div>
      </div>

      {expanded ? (
        <div className="forensic-flow-modal-backdrop" onClick={() => setExpanded(false)} role="presentation">
          <div
            className="forensic-flow-modal"
            role="dialog"
            aria-modal="true"
            aria-label="资金流向图放大查看"
            onClick={(event) => event.stopPropagation()}
          >
            <div className="forensic-flow-modal-header">
              <div>
                <strong>Fund Flow</strong>
                <p>主图只保留借入、关键执行分支、补足与归还闭环；放大后再看完整路径和金额标签。</p>
                <FundFlowLegend />
              </div>
              <button type="button" className="forensic-flow-modal-close" onClick={() => setExpanded(false)} aria-label="关闭">
                关闭
              </button>
            </div>

            <div className="forensic-flow-modal-scroll">
              <div className="forensic-flow-board">
                <FundFlowViewport diagram={diagram} mode="expanded" />
              </div>
            </div>
          </div>
        </div>
      ) : null}
    </>
  )
}

function FundFlowLegend({ compact = false }: { compact?: boolean }) {
  return (
    <div className={`forensic-flow-legend${compact ? ' compact' : ''}`}>
      <div className="forensic-flow-legend-group">
        <span className="forensic-flow-legend-label">线颜色</span>
        <div className="forensic-flow-legend-items">
          <span className="forensic-flow-legend-item">
            <span className="forensic-flow-legend-line tone-borrow" />
            <span>蓝色: 借入</span>
          </span>
          <span className="forensic-flow-legend-item">
            <span className="forensic-flow-legend-line tone-swap" />
            <span>紫色: 执行 / 交换</span>
          </span>
          <span className="forensic-flow-legend-item">
            <span className="forensic-flow-legend-line tone-repay" />
            <span>橙色: 补足 / 归还</span>
          </span>
        </div>
      </div>

      <div className="forensic-flow-legend-group">
        <span className="forensic-flow-legend-label">节点底色</span>
        <div className="forensic-flow-legend-items">
          <span className="forensic-flow-legend-item">
            <span className="forensic-flow-legend-chip role-borrow-source" />
            <span>浅蓝: 借出方 / Provider</span>
          </span>
          <span className="forensic-flow-legend-item">
            <span className="forensic-flow-legend-chip role-hop" />
            <span>灰白: Receiver / 中间执行</span>
          </span>
          <span className="forensic-flow-legend-item">
            <span className="forensic-flow-legend-chip role-repay" />
            <span>浅橙: 补足来源 / 归还目标</span>
          </span>
        </div>
      </div>
    </div>
  )
}

function FundFlowViewport({
  diagram,
  mode,
}: {
  diagram: UnifiedDiagramLayout
  mode: 'preview' | 'expanded'
}) {
  const compact = mode === 'preview'
  const graph = useMemo(() => buildReactFlowGraph(diagram, compact), [diagram, compact])
  const height = compact
    ? Math.max(260, Math.min(360, Math.round(diagram.height * 0.84)))
    : Math.max(480, Math.min(760, diagram.height + 80))

  return (
    <ReactFlowProvider>
      <div className={`forensic-flow-react-shell ${mode}`} style={{ height }}>
        <ReactFlow
          nodes={graph.nodes}
          edges={graph.edges}
          nodeTypes={fundFlowNodeTypes}
          edgeTypes={fundFlowEdgeTypes}
          fitView
          fitViewOptions={{ padding: compact ? 0.18 : 0.2 }}
          minZoom={0.2}
          maxZoom={1.5}
          nodesDraggable={false}
          nodesConnectable={false}
          elementsSelectable={false}
          edgesFocusable={false}
          nodesFocusable={false}
          panOnDrag={!compact}
          zoomOnScroll={!compact}
          zoomOnPinch={!compact}
          zoomOnDoubleClick={!compact}
          preventScrolling={false}
          proOptions={{ hideAttribution: true }}
          className={`forensic-flow-react ${mode}`}
        >
          <FundFlowAutoFit compact={compact} graphKey={`${mode}:${graph.nodes.length}:${graph.edges.length}:${diagram.width}:${diagram.height}`} />
          {!compact ? (
            <>
              <Controls showInteractive={false} position="top-right" />
              <Background gap={24} color="rgba(148, 163, 184, 0.16)" />
            </>
          ) : null}
        </ReactFlow>
      </div>
    </ReactFlowProvider>
  )
}

function FundFlowAutoFit({
  compact,
  graphKey,
}: {
  compact: boolean
  graphKey: string
}) {
  const nodesInitialized = useNodesInitialized()
  const { fitView } = useReactFlow()

  useEffect(() => {
    if (!nodesInitialized) {
      return
    }
    const frame = window.requestAnimationFrame(() => {
      fitView({
        padding: compact ? 0.26 : 0.2,
        includeHiddenNodes: true,
        minZoom: compact ? 0.08 : 0.2,
        maxZoom: compact ? 0.92 : 1.2,
        duration: 0,
      })
    })
    return () => window.cancelAnimationFrame(frame)
  }, [compact, fitView, graphKey, nodesInitialized])

  return null
}

function FundFlowNodeView({ data }: NodeProps<Node<FundFlowGraphNodeData>>) {
  if (data.kind === 'lane-label') {
    return (
      <div className="forensic-flow-lane-chip" title={data.subtitle || data.title}>
        <strong>{data.title}</strong>
        {data.subtitle ? <span>{data.subtitle}</span> : null}
      </div>
    )
  }

  return (
    <div className={`forensic-flow-node role-${resolveSlotClass(data.role)} ${data.compact ? 'is-compact' : ''}`} title={data.address || data.subtitle}>
      <Handle type="target" id="left" position={Position.Left} className="forensic-flow-handle" />
      <Handle type="source" id="right" position={Position.Right} className="forensic-flow-handle" />
      <Handle type="target" id="bottom" position={Position.Bottom} className="forensic-flow-handle" />
      <Handle type="source" id="top" position={Position.Top} className="forensic-flow-handle" />

      <span className="forensic-flow-node-dot" aria-hidden="true" />
      <div className="forensic-flow-node-copy">
        <strong>{truncateLabel(data.title, data.compact ? 24 : 28)}</strong>
        {!data.compact ? <span>{truncateLabel(data.subtitle, 30)}</span> : null}
      </div>
    </div>
  )
}

function FundFlowEdgeView({
  id,
  data,
  markerEnd,
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
}: EdgeProps<Edge<FundFlowGraphEdgeData>>) {
  const tone = data?.tone ?? 'swap'
  const color = toneColor(tone)
  const [edgePath, labelX, labelY] = getSmoothStepPath({
    sourceX,
    sourceY,
    targetX,
    targetY,
    sourcePosition,
    targetPosition,
    borderRadius: tone === 'swap' ? 22 : 18,
    offset: data?.track === 'topup' ? 30 : 24,
  })

  return (
    <>
      <BaseEdge id={id} path={edgePath} markerEnd={markerEnd} style={{ stroke: color, strokeWidth: 2.5 }} />
      {data?.showLabel ? (
        <EdgeLabelRenderer>
          <div
            className={`forensic-flow-edge-label tone-${tone}`}
            style={{
              transform: `translate(-50%, -50%) translate(${labelX}px, ${labelY + (data.labelOffsetY ?? 0)}px)`,
            }}
            title={data.verboseLabel}
          >
            {truncateLabel(data.label, 26)}
          </div>
        </EdgeLabelRenderer>
      ) : null}
    </>
  )
}

const fundFlowNodeTypes = {
  fundFlowNode: FundFlowNodeView,
}

const fundFlowEdgeTypes = {
  fundFlowEdge: FundFlowEdgeView,
}

export function InvocationFlowTree({ nodes, emptyLabel, raw = false }: InvocationFlowTreeProps) {
  if (nodes.length === 0) {
    return <div className="forensic-empty-box">{emptyLabel}</div>
  }

  return (
    <div className={`invocation-tree ${raw ? 'is-raw' : ''}`}>
      <div
        className={`invocation-tree-scroll ${raw ? 'is-raw' : ''}`}
        tabIndex={0}
        role="region"
        aria-label={raw ? '完整内部调用链滚动区域' : '调用调用树滚动区域'}
      >
        <ul className="invocation-tree-list">
          {nodes.map((node) => (
            <InvocationTreeItem key={node.id} node={node} raw={raw} />
          ))}
        </ul>
      </div>
    </div>
  )
}

function InvocationTreeItem({ node, raw }: { node: InvocationFlowNode; raw: boolean }) {
  const route = node.from && node.to ? `${shortenValue(node.from)} -> ${shortenValue(node.to)}` : undefined

  return (
    <li className={`invocation-tree-item phase-${node.phase}`}>
      <div className="invocation-node-row">
        <div className="invocation-node-topline">
          <span className="invocation-node-phase">{node.phaseLabel}</span>
          {node.badges.map((badge) => (
            <span key={`${node.id}-${badge}`} className="invocation-node-badge">
              {badge}
            </span>
          ))}
        </div>

        <div className="invocation-node-content">
          <div className="invocation-node-head">
            <strong>{node.title}</strong>
            <span className="invocation-node-depth">L{node.depth}</span>
          </div>
          <p className="invocation-node-detail">{node.detail}</p>
          {route ? (
            <p className="invocation-node-route" title={`${node.from} -> ${node.to}`}>
              {route}
            </p>
          ) : null}
          {node.meta.length > 0 ? (
            <p className={`invocation-node-meta ${raw ? 'is-raw' : ''}`}>{node.meta.join(' | ')}</p>
          ) : null}
        </div>
      </div>

      {node.children.length > 0 ? (
        <ul className="invocation-tree-list">
          {node.children.map((child) => (
            <InvocationTreeItem key={child.id} node={child} raw={raw} />
          ))}
        </ul>
      ) : null}
    </li>
  )
}

function buildReactFlowGraph(diagram: UnifiedDiagramLayout, compact: boolean) {
  const nodes: Array<Node<FundFlowGraphNodeData>> = []
  const edges: Array<Edge<FundFlowGraphEdgeData>> = []

  diagram.rows.forEach((row) => {
    nodes.push({
      id: `lane-label:${row.lane.id}`,
      type: 'fundFlowNode',
      position: { x: 20, y: row.top + 4 },
      selectable: false,
      draggable: false,
      data: {
        kind: 'lane-label',
        title: row.lane.label,
        subtitle: compact ? undefined : row.lane.sublabel,
      },
    })

    row.slots.forEach((slot) => {
      nodes.push({
        id: slot.key,
        type: 'fundFlowNode',
        position: { x: slot.x, y: slot.y + 26 },
        selectable: false,
        draggable: false,
        data: {
          kind: 'slot',
          title: slot.title,
          subtitle: slot.subtitle,
          address: slot.address,
          role: slot.role,
          compact,
        },
      })
    })

    row.paths.forEach((path) => {
      const handles = resolveHandles(path.track)
      edges.push({
        id: path.id,
        source: path.fromKey,
        sourceHandle: handles.sourceHandle,
        target: path.toKey,
        targetHandle: handles.targetHandle,
        type: 'fundFlowEdge',
        selectable: false,
        markerEnd: {
          type: MarkerType.ArrowClosed,
          color: toneColor(path.tone),
          width: 16,
          height: 16,
        },
        data: {
          label: path.label,
          verboseLabel: path.verboseLabel,
          tone: path.tone,
          track: path.track,
          showLabel: !compact,
        },
      })
    })
  })

  return { nodes, edges }
}

function resolveHandles(track: DiagramPathTrack) {
  if (track === 'topup') {
    return {
      sourceHandle: 'top',
      targetHandle: 'bottom',
    }
  }
  return {
    sourceHandle: 'right',
    targetHandle: 'left',
  }
}

function toneColor(tone: FlowTone) {
  switch (tone) {
    case 'borrow':
      return '#2563eb'
    case 'repay':
      return '#f59e0b'
    default:
      return '#8b5cf6'
  }
}

function buildUnifiedDiagramLayout(lanes: FundFlowLane[]): UnifiedDiagramLayout {
  const rowGap = 28
  let top = 12
  let maxWidth = 1120
  const rows: LaneRowLayout[] = []

  lanes.forEach((lane) => {
    const row = buildLaneRowLayout(lane, top)
    rows.push(row)
    const rowWidth = row.slots.reduce((largest, slot) => Math.max(largest, slot.x), 0) + FUND_FLOW_NODE_WIDTH + 56
    maxWidth = Math.max(maxWidth, rowWidth)
    top += row.height + rowGap
  })

  return {
    width: maxWidth,
    height: Math.max(top - rowGap + 24, 220),
    rows,
  }
}

function buildLaneRowLayout(lane: FundFlowLane, top: number): LaneRowLayout {
  const nodeMap = new Map(lane.nodes.map((node) => [node.id, node] as const))
  const receiverNode = lane.nodes.find((node) => node.roles.includes('receiver')) ?? lane.nodes[0]

  if (!receiverNode) {
    return { lane, top, height: 140, slots: [], paths: [] }
  }

  const borrowSegment = lane.segments.find((segment) => segment.tone === 'borrow' && segment.to === receiverNode.id)
  const repaySegment = lane.segments.find((segment) => segment.tone === 'repay' && segment.from === receiverNode.id)
  const topupSegments = lane.segments.filter((segment) => segment.tone === 'repay' && segment.to === receiverNode.id)
  const executionBranches = buildExecutionBranches(receiverNode.id, lane.segments.filter((segment) => segment.tone === 'swap'))

  const borrowSourceNode = borrowSegment ? nodeMap.get(borrowSegment.from) : undefined
  const repaymentTargetNode = repaySegment ? nodeMap.get(repaySegment.to) : undefined
  const topupNodes = topupSegments
    .map((segment) => nodeMap.get(segment.from))
    .filter((node): node is FundFlowLaneNode => Boolean(node))
    .filter((node, index, list) => list.findIndex((item) => item.id === node.id) === index)

  const branchCount = Math.max(executionBranches.length, 1)
  const branchGapY = 124
  const branchAnchorY = top + 138
  const branchYs = executionBranches.length > 0
    ? executionBranches.map((_, index) => branchAnchorY + (index - (executionBranches.length - 1) / 2) * branchGapY)
    : [branchAnchorY]
  const branchMinY = Math.min(branchAnchorY, ...branchYs)
  const branchMaxY = Math.max(branchAnchorY, ...branchYs)

  const slots: DiagramSlot[] = []
  const paths: DiagramPath[] = []

  const borrowX = 56
  const receiverEntryX = 388
  const hopStartX = 780
  const hopGap = 288
  const maxVisibleDepth = Math.max(1, ...executionBranches.map((branch) => branch.visibleDepth))
  const showSettlementReceiver = executionBranches.some((branch) => branch.returnsToReceiver)
  const receiverSettleX = showSettlementReceiver ? hopStartX + maxVisibleDepth * hopGap + 96 : receiverEntryX
  const repaymentX = Math.max(receiverSettleX + 356, hopStartX + Math.max(maxVisibleDepth - 1, 0) * hopGap + 420)
  const topupY = branchMaxY + 146

  const receiverEntryKey = `${lane.id}:receiver-entry`
  const receiverSettleKey = showSettlementReceiver ? `${lane.id}:receiver-settle` : receiverEntryKey

  slots.push(createSlot(receiverEntryKey, 'receiver', receiverNode, receiverEntryX, branchAnchorY))

  if (showSettlementReceiver) {
    slots.push(createSlot(receiverSettleKey, 'receiver_settlement', receiverNode, receiverSettleX, branchAnchorY))
  }

  if (borrowSourceNode && borrowSegment) {
    slots.push(createSlot(`${lane.id}:borrow-source`, 'borrow_source', borrowSourceNode, borrowX, branchAnchorY))
    paths.push({
      id: `${lane.id}:borrow`,
      fromKey: `${lane.id}:borrow-source`,
      toKey: receiverEntryKey,
      tone: 'borrow',
      track: 'borrow',
      label: formatCompactFlowLabel(borrowSegment),
      verboseLabel: formatVerboseFlowLabel(borrowSegment),
      labelOffsetY: -16,
    })
  }

  executionBranches.forEach((branch, branchIndex) => {
    const branchY = branchYs[branchIndex] ?? branchAnchorY
    let previousKey = receiverEntryKey
    let visibleIndex = 0

    branch.segments.forEach((segment, segmentIndex) => {
      const isReturn = sameSlotTarget(segment.to, receiverNode.id)
      let nextKey = receiverSettleKey

      if (!isReturn) {
        const node = nodeMap.get(segment.to)
        if (!node) {
          return
        }
        nextKey = `${lane.id}:branch:${branchIndex}:step:${segmentIndex}`
        slots.push(createSlot(
          nextKey,
          'hop',
          node,
          hopStartX + visibleIndex * hopGap,
          branchY,
        ))
        visibleIndex += 1
      }

      paths.push({
        id: `${lane.id}:execute:${branch.id}:${segment.id}`,
        fromKey: previousKey,
        toKey: nextKey,
        tone: 'swap',
        track: isReturn ? 'return' : 'execute',
        label: formatCompactFlowLabel(segment),
        verboseLabel: formatVerboseFlowLabel(segment),
        labelOffsetY: computeBranchLabelOffset(branchIndex, branchCount, isReturn),
      })

      previousKey = nextKey
    })
  })

  if (repaymentTargetNode && repaySegment) {
    slots.push(createSlot(`${lane.id}:repayment-target`, 'repayment_target', repaymentTargetNode, repaymentX, branchAnchorY))
    paths.push({
      id: `${lane.id}:repay`,
      fromKey: receiverSettleKey,
      toKey: `${lane.id}:repayment-target`,
      tone: 'repay',
      track: 'repay',
      label: formatCompactFlowLabel(repaySegment),
      verboseLabel: formatVerboseFlowLabel(repaySegment),
      labelOffsetY: -16,
    })
  }

  distributeTopupPositions(topupNodes.length, receiverSettleX)
    .forEach((x, index) => {
      const node = topupNodes[index]
      const segment = topupSegments.find((item) => item.from === node.id)
      if (!node || !segment) {
        return
      }
      const key = `${lane.id}:topup:${index}`
      slots.push(createSlot(key, 'topup_source', node, x, topupY))
      paths.push({
        id: `${lane.id}:topup:${segment.id}`,
        fromKey: key,
        toKey: receiverSettleKey,
        tone: 'repay',
        track: 'topup',
        label: formatCompactFlowLabel(segment),
        verboseLabel: formatVerboseFlowLabel(segment),
        labelOffsetY: 18,
      })
    })

  const height = topupNodes.length > 0
    ? Math.max(326, topupY - top + 126)
    : Math.max(232, branchMaxY - branchMinY + 208 + (branchCount > 1 ? 12 : 0))

  return {
    lane,
    top,
    height,
    slots,
    paths,
  }
}

function createSlot(
  key: string,
  role: DiagramSlotRole,
  node: FundFlowLaneNode,
  x: number,
  y: number,
): DiagramSlot {
  return {
    key,
    role,
    title: node.title,
    subtitle: node.subtitle,
    address: node.address,
    x,
    y,
  }
}

function buildExecutionBranches(receiverId: string, swapSegments: FundFlowLaneSegment[]) {
  const outgoing = new Map<string, FundFlowLaneSegment[]>()
  swapSegments.forEach((segment) => {
    const existing = outgoing.get(segment.from) ?? []
    existing.push(segment)
    outgoing.set(segment.from, existing)
  })

  const branches: ExecutionBranch[] = []
  const seenSignatures = new Set<string>()

  const pushBranch = (segments: FundFlowLaneSegment[]) => {
    if (segments.length === 0 || branches.length >= 6) {
      return
    }
    const signature = segments.map((segment) => segment.id).join('>')
    if (seenSignatures.has(signature)) {
      return
    }
    seenSignatures.add(signature)
    const visibleDepth = segments.filter((segment) => !sameSlotTarget(segment.to, receiverId)).length
    branches.push({
      id: `branch-${branches.length}`,
      segments,
      returnsToReceiver: sameSlotTarget(segments[segments.length - 1]?.to, receiverId),
      visibleDepth,
    })
  }

  const walk = (segment: FundFlowLaneSegment, path: FundFlowLaneSegment[], seen: Set<string>) => {
    if (path.length >= 4 || sameSlotTarget(segment.to, receiverId)) {
      pushBranch(path)
      return
    }

    const nextSegments = (outgoing.get(segment.to) ?? [])
      .filter((candidate) => !seen.has(candidate.id))
      .slice(0, 3)

    if (nextSegments.length === 0) {
      pushBranch(path)
      return
    }

    nextSegments.forEach((nextSegment) => {
      const nextSeen = new Set(seen)
      nextSeen.add(nextSegment.id)
      walk(nextSegment, [...path, nextSegment], nextSeen)
    })
  }

  ;(outgoing.get(receiverId) ?? []).slice(0, 4).forEach((seed) => {
    walk(seed, [seed], new Set([seed.id]))
  })

  if (branches.length === 0 && swapSegments.length > 0) {
    pushBranch([swapSegments[0]])
  }

  return branches
}

function distributeTopupPositions(count: number, targetX: number) {
  if (count === 0) {
    return []
  }
  const gap = 236
  const start = Math.max(668, targetX - ((count - 1) * gap) / 2)
  return Array.from({ length: count }, (_, index) => start + index * gap)
}

function sameSlotTarget(left?: string, right?: string) {
  return (left ?? '').trim().toLowerCase() === (right ?? '').trim().toLowerCase()
}

function computeBranchLabelOffset(branchIndex: number, branchCount: number, isReturn: boolean) {
  if (isReturn) {
    return -18
  }
  if (branchCount <= 1) {
    return -14
  }
  const center = (branchCount - 1) / 2
  const direction = branchIndex < center ? -1 : branchIndex > center ? 1 : 0
  if (direction === 0) {
    return -16
  }
  return direction * 20
}

function resolveSlotClass(role: DiagramSlotRole) {
  if (role === 'receiver_settlement') {
    return 'receiver'
  }
  if (role === 'borrow_source') {
    return 'borrow-source'
  }
  if (role === 'repayment_target') {
    return 'repayment-target'
  }
  if (role === 'topup_source') {
    return 'topup-source'
  }
  if (role === 'receiver') {
    return 'receiver'
  }
  return 'hop'
}

function formatCompactFlowLabel(segment?: FundFlowLaneSegment) {
  if (!segment) {
    return ''
  }
  return `${segment.amount} ${segment.asset}`.trim()
}

function formatVerboseFlowLabel(segment?: FundFlowLaneSegment) {
  if (!segment) {
    return ''
  }
  return segment.action ? `${segment.action} · ${segment.amount} ${segment.asset}` : formatCompactFlowLabel(segment)
}

function shortenValue(value: string) {
  if (value.length <= 18) {
    return value
  }
  return `${value.slice(0, 10)}...${value.slice(-6)}`
}

function truncateLabel(value: string, maxLength: number) {
  if (value.length <= maxLength) {
    return value
  }
  return `${value.slice(0, maxLength - 3)}...`
}
