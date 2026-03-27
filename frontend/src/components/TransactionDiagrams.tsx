import { useRef, useState } from 'react'
import { linkHorizontal } from 'd3-shape'

type AddressSummary = {
  address: string
  roles: string[]
  roleCodes: string[]
}

type AddressGraphView = {
  left: AddressSummary[]
  right: AddressSummary[]
  flows: Array<{
    id: string
    protocol: string
    assetAddress: string
    amount: string
    source: string
    target: string
  }>
}

type SequenceLane = {
  id: string
  label: string
  caption: string
}

type SequenceStep = {
  id: string
  from: string
  to: string
  title: string
  detail: string
  tone: 'neutral' | 'accent' | 'danger'
}

type SequenceDiagramView = {
  lanes: SequenceLane[]
  steps: SequenceStep[]
}

type AddressGraphProps = {
  view: AddressGraphView
  txHash: string
  protocolsLabel: string
  identifiedFlowsLabel: string
  borrowedLabel: string
  repaidLabel: string
  shortenHash: (value: string) => string
}

type SequenceDiagramProps = {
  view: SequenceDiagramView
}

const relationPath = linkHorizontal<
  { source: { x: number; y: number }; target: { x: number; y: number } },
  { x: number; y: number }
>()
  .x((point) => point.x)
  .y((point) => point.y)

export function TransactionAddressGraph({
  view,
  txHash,
  protocolsLabel,
  identifiedFlowsLabel,
  shortenHash,
}: AddressGraphProps) {
  const shellRef = useRef<HTMLDivElement | null>(null)
  const [hoveredValue, setHoveredValue] = useState<{ value: string; x: number; y: number }>()
  const width = 900
  const leftX = 88
  const centerX = width / 2
  const rightX = width - 88
  const rowGap = 86
  const top = 56
  const nodeWidth = 176
  const nodeHeight = 56
  const hubWidth = 236
  const hubHeight = 112
  const graphHeight = top + Math.max(view.left.length, view.right.length, 1) * rowGap + 108
  const hubY = top + Math.max(view.left.length, view.right.length, 1) * rowGap * 0.5 - hubHeight / 2

  const leftNodes = view.left.map((item, index) => ({
    ...item,
    x: leftX,
    y: top + index * rowGap,
  }))
  const rightNodes = view.right.map((item, index) => ({
    ...item,
    x: rightX,
    y: top + index * rowGap,
  }))

  const updateHoveredValue = (
    event: React.MouseEvent<Element>,
    value: string,
  ) => {
    const shell = shellRef.current
    if (!shell) {
      return
    }
    const rect = shell.getBoundingClientRect()
    setHoveredValue({
      value,
      x: event.clientX - rect.left,
      y: event.clientY - rect.top,
    })
  }

  return (
    <div ref={shellRef} className="svg-diagram-shell address-graph-shell">
      <svg className="svg-diagram" viewBox={`0 0 ${width} ${graphHeight}`} role="img" aria-label={identifiedFlowsLabel}>
        <defs>
          <marker id="relation-arrow" markerWidth="8" markerHeight="8" refX="7" refY="4" orient="auto">
            <path d="M0,0 L8,4 L0,8 Z" fill="rgba(17,17,17,0.26)" />
          </marker>
        </defs>

        {leftNodes.map((node) => (
          <path
            key={`left-line-${node.address}`}
            d={relationPath({
              source: { x: node.x + nodeWidth, y: node.y + nodeHeight / 2 },
              target: { x: centerX - hubWidth / 2, y: hubY + hubHeight / 2 },
            }) ?? undefined}
            fill="none"
            stroke="rgba(17,17,17,0.18)"
            strokeWidth="1.5"
            markerEnd="url(#relation-arrow)"
          />
        ))}

        {rightNodes.map((node) => (
          <path
            key={`right-line-${node.address}`}
            d={relationPath({
              source: { x: centerX + hubWidth / 2, y: hubY + hubHeight / 2 },
              target: { x: node.x, y: node.y + nodeHeight / 2 },
            }) ?? undefined}
            fill="none"
            stroke="rgba(17,17,17,0.18)"
            strokeWidth="1.5"
            markerEnd="url(#relation-arrow)"
          />
        ))}

        {leftNodes.map((node) => (
          <g key={`left-node-${node.address}`} transform={`translate(${node.x}, ${node.y})`}>
            <rect width={nodeWidth} height={nodeHeight} rx="18" fill="rgba(255,255,255,0.92)" stroke="rgba(17,17,17,0.12)" />
            <text x="14" y="22" className="svg-node-label">{node.roles[0]}</text>
            <text
              x="14"
              y="40"
              className="svg-node-value svg-hover-value"
              onMouseEnter={(event) => updateHoveredValue(event, node.address)}
              onMouseMove={(event) => updateHoveredValue(event, node.address)}
              onMouseLeave={() => setHoveredValue(undefined)}
            >
              {shortenHash(node.address)}
            </text>
          </g>
        ))}

        {rightNodes.map((node) => (
          <g key={`right-node-${node.address}`} transform={`translate(${node.x - nodeWidth}, ${node.y})`}>
            <rect width={nodeWidth} height={nodeHeight} rx="18" fill="rgba(255,255,255,0.92)" stroke="rgba(17,17,17,0.12)" />
            <text x="14" y="22" className="svg-node-label">{node.roles[0]}</text>
            <text
              x="14"
              y="40"
              className="svg-node-value svg-hover-value"
              onMouseEnter={(event) => updateHoveredValue(event, node.address)}
              onMouseMove={(event) => updateHoveredValue(event, node.address)}
              onMouseLeave={() => setHoveredValue(undefined)}
            >
              {shortenHash(node.address)}
            </text>
          </g>
        ))}

        <g transform={`translate(${centerX - hubWidth / 2}, ${hubY})`}>
          <rect width={hubWidth} height={hubHeight} rx="28" fill="rgba(255,255,255,0.96)" stroke="rgba(17,17,17,0.16)" />
          <text x={hubWidth / 2} y="32" textAnchor="middle" className="svg-node-label centered">{protocolsLabel}</text>
          <text
            x={hubWidth / 2}
            y="60"
            textAnchor="middle"
            className="svg-hub-value svg-hover-value"
            onMouseEnter={(event) => updateHoveredValue(event, txHash)}
            onMouseMove={(event) => updateHoveredValue(event, txHash)}
            onMouseLeave={() => setHoveredValue(undefined)}
          >
            {shortenHash(txHash)}
          </text>
          <text x={hubWidth / 2} y="84" textAnchor="middle" className="svg-node-label centered">
            {identifiedFlowsLabel} · {view.flows.length}
          </text>
        </g>
      </svg>

      {hoveredValue ? (
        <div
          className="address-graph-tooltip"
          style={{
            left: hoveredValue.x,
            top: hoveredValue.y,
          }}
        >
          {hoveredValue.value}
        </div>
      ) : null}

      <div className="address-flow-ledger">
        <div className="address-flow-ledger-header">
          <span>{identifiedFlowsLabel}</span>
          <strong>{view.flows.length}</strong>
        </div>
        <div className="address-flow-list">
          {view.flows.map((flow) => (
            <article key={flow.id} className="address-flow-row">
              <div className="address-flow-topline">
                <strong>{flow.protocol}</strong>
                <span
                  className="address-hover-value"
                  onMouseEnter={(event) => updateHoveredValue(event, flow.assetAddress)}
                  onMouseMove={(event) => updateHoveredValue(event, flow.assetAddress)}
                  onMouseLeave={() => setHoveredValue(undefined)}
                >
                  {shortenHash(flow.assetAddress)}
                </span>
              </div>
              <p className="address-flow-route">
                <span
                  className="address-hover-value"
                  onMouseEnter={(event) => updateHoveredValue(event, flow.source)}
                  onMouseMove={(event) => updateHoveredValue(event, flow.source)}
                  onMouseLeave={() => setHoveredValue(undefined)}
                >
                  {shortenHash(flow.source)}
                </span>
                {' '}→{' '}
                <span
                  className="address-hover-value"
                  onMouseEnter={(event) => updateHoveredValue(event, flow.target)}
                  onMouseMove={(event) => updateHoveredValue(event, flow.target)}
                  onMouseLeave={() => setHoveredValue(undefined)}
                >
                  {shortenHash(flow.target)}
                </span>
              </p>
              <p className="address-flow-amount">{flow.amount}</p>
            </article>
          ))}
        </div>
      </div>
    </div>
  )
}

export function TransactionSequenceDiagram({ view }: SequenceDiagramProps) {
  const [hoveredValue, setHoveredValue] = useState<{ value: string; x: number; y: number }>()
  const shellRef = useRef<HTMLDivElement | null>(null)
  const laneWidth = 176
  const width = Math.max(760, view.lanes.length * laneWidth)
  const laneStartY = 52
  const laneBodyTop = 86
  const stepGap = 86
  const height = laneBodyTop + view.steps.length * stepGap + 48
  const laneCenters = view.lanes.map((_, index) => 88 + index * laneWidth)

  const updateHoveredValue = (
    event: React.MouseEvent<Element>,
    value: string,
  ) => {
    const shell = shellRef.current
    if (!shell) {
      return
    }
    const rect = shell.getBoundingClientRect()
    setHoveredValue({
      value,
      x: event.clientX - rect.left,
      y: event.clientY - rect.top,
    })
  }

  return (
    <div ref={shellRef} className="svg-diagram-shell">
      <svg className="svg-diagram" viewBox={`0 0 ${width} ${height}`} role="img" aria-label="sequence diagram">
        <defs>
          <marker id="sequence-arrow" markerWidth="8" markerHeight="8" refX="7" refY="4" orient="auto">
            <path d="M0,0 L8,4 L0,8 Z" fill="rgba(17,17,17,0.28)" />
          </marker>
        </defs>

        {view.lanes.map((lane, index) => (
          <g key={lane.id}>
            <rect
              x={laneCenters[index] - 72}
              y={laneStartY - 24}
              width="144"
              height="48"
              rx="16"
              fill="rgba(255,255,255,0.92)"
              stroke="rgba(17,17,17,0.12)"
            />
            <text x={laneCenters[index]} y={laneStartY - 4} textAnchor="middle" className="svg-node-value">{lane.label}</text>
            <text
              x={laneCenters[index]}
              y={laneStartY + 14}
              textAnchor="middle"
              className="svg-node-label centered svg-hover-value"
              onMouseEnter={(event) => updateHoveredValue(event, lane.id.startsWith('0x') ? lane.id : lane.caption)}
              onMouseMove={(event) => updateHoveredValue(event, lane.id.startsWith('0x') ? lane.id : lane.caption)}
              onMouseLeave={() => setHoveredValue(undefined)}
            >
              {lane.caption}
            </text>
            <line
              x1={laneCenters[index]}
              y1={laneBodyTop}
              x2={laneCenters[index]}
              y2={height - 16}
              stroke="rgba(17,17,17,0.12)"
              strokeDasharray="4 6"
            />
          </g>
        ))}

        {view.steps.map((step, index) => {
          const y = laneBodyTop + index * stepGap + 18
          const fromIndex = view.lanes.findIndex((lane) => lane.id === step.from)
          const toIndex = view.lanes.findIndex((lane) => lane.id === step.to)
          const fromX = laneCenters[fromIndex]
          const toX = laneCenters[toIndex]
          const isSelf = fromIndex === toIndex
          const cardWidth = 154
          const cardHeight = 52
          const cardX = toX - cardWidth / 2
          const toneClass =
            step.tone === 'accent'
              ? 'rgba(13,140,114,0.12)'
              : step.tone === 'danger'
                ? 'rgba(190,79,59,0.12)'
                : 'rgba(255,255,255,0.9)'

          return (
            <g key={step.id}>
              {!isSelf ? (
                <>
                  <circle cx={fromX} cy={y} r="4" fill="#0d8c72" />
                  <path
                    d={relationPath({
                      source: { x: fromX, y },
                      target: { x: toX, y },
                    }) ?? undefined}
                    fill="none"
                    stroke="rgba(17,17,17,0.2)"
                    strokeWidth="1.5"
                    markerEnd="url(#sequence-arrow)"
                  />
                </>
              ) : (
                <path
                  d={`M ${fromX} ${y} c 26 0, 26 32, 0 32`}
                  fill="none"
                  stroke="rgba(17,17,17,0.2)"
                  strokeWidth="1.5"
                  markerEnd="url(#sequence-arrow)"
                />
              )}
              <rect
                x={cardX}
                y={y + 10}
                width={cardWidth}
                height={cardHeight}
                rx="14"
                fill={toneClass}
                stroke="rgba(17,17,17,0.1)"
              />
              <text x={cardX + 12} y={y + 30} className="svg-node-value small">{step.title}</text>
              <text
                x={cardX + 12}
                y={y + 46}
                className="svg-node-label svg-hover-value"
                onMouseEnter={(event) => updateHoveredValue(event, step.detail)}
                onMouseMove={(event) => updateHoveredValue(event, step.detail)}
                onMouseLeave={() => setHoveredValue(undefined)}
              >
                {truncateSvgText(step.detail, 26)}
              </text>
            </g>
          )
        })}
      </svg>

      {hoveredValue ? (
        <div
          className="address-graph-tooltip"
          style={{
            left: hoveredValue.x,
            top: hoveredValue.y,
          }}
        >
          {hoveredValue.value}
        </div>
      ) : null}
    </div>
  )
}

function truncateSvgText(value: string, maxLength: number) {
  if (value.length <= maxLength) {
    return value
  }
  return `${value.slice(0, maxLength - 1)}…`
}
