import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { Link, useParams, useSearchParams } from 'react-router-dom'
import {
  FundFlowDiagram,
  InvocationFlowTree,
  type FundFlowDiagramModel,
  type InvocationFlowNode,
} from '../components/TransactionForensicsDiagrams'
import { LanguageToggle } from '../components/LanguageToggle'
import { getTransactionDetail } from '../lib/api'
import { useI18n } from '../lib/i18n'
import type { TransactionDetailResponse } from '../types'

type AddressSummary = {
  address: string
  roleCodes: string[]
  roleLabels: string[]
}

type BasicInfoRow = {
  label: string
  value: ReactNode
  action?: ReactNode
}

type BalanceChangeRow = {
  id: string
  address: string
  roleLabel: string
  assetAddress: string
  assetLabel: string
  recordType: string
  borrowed: string
  repaid: string
  fee: string
  direction: string
  strict: boolean
}

type DetailCopy = ReturnType<typeof getDetailCopy>
type TraceFrame = NonNullable<NonNullable<TransactionDetailResponse['trace_summary']>['frames']>[number]

export function TransactionDetail() {
  const { t, protocolName } = useI18n()
  const { txHash = '' } = useParams()
  const [searchParams] = useSearchParams()
  const [data, setData] = useState<TransactionDetailResponse | null>(null)
  const [error, setError] = useState<string>()
  const [loading, setLoading] = useState(true)
  const chainId = Number(searchParams.get('chain_id') ?? '1')
  const isEnglish = t.yes === 'Yes'
  const copy = getDetailCopy(isEnglish)

  useEffect(() => {
    let active = true
    setLoading(true)
    setError(undefined)

    getTransactionDetail(txHash, chainId)
      .then((payload) => {
        if (active) {
          setData(payload)
        }
      })
      .catch((reason: Error) => {
        if (active) {
          setError(reason.message)
        }
      })
      .finally(() => {
        if (active) {
          setLoading(false)
        }
      })

    return () => {
      active = false
    }
  }, [chainId, txHash])

  const addressSummaries = useMemo(
    () => (data ? resolveAddressSummaries(data, isEnglish) : []),
    [data, isEnglish],
  )
  const addressMap = useMemo(
    () => new Map(addressSummaries.map((item) => [item.address.toLowerCase(), item] as const)),
    [addressSummaries],
  )
  const balanceRows = useMemo(
    () => (data ? buildBalanceChangeRows(data, addressMap, copy) : []),
    [addressMap, copy, data],
  )
  const fundFlowModel = useMemo(
    () => {
      if (!data) {
        return { lanes: [], emptyLabel: copy.fundFlowEmpty }
      }
      if (data.fund_flow_graph) {
        return normalizeBackendFundFlowModel(data.fund_flow_graph, copy)
      }
      return buildFundFlowModel(data, addressMap, copy)
    },
    [addressMap, copy, data],
  )
  const invocationFlowNodes = useMemo(
    () => (data ? buildInvocationNodes(data, addressMap, copy, false) : []),
    [addressMap, copy, data],
  )

  if (loading || error || !data) {
    return (
      <main className="page-shell detail-shell forensic-shell">
        <div className="detail-topbar">
          <Link to="/">← {t.backToConsole}</Link>
          <LanguageToggle />
        </div>

        {loading ? <section className="panel">{t.loadingTransactionDetail}</section> : null}
        {error ? <section className="panel error-text">{error}</section> : null}
      </main>
    )
  }

  const balanceSentence = buildBalanceSentence(balanceRows, copy)
  const browserFieldRows = buildBrowserFieldRows(data, addressMap, protocolName, t, copy)

  return (
    <main className="page-shell detail-shell forensic-shell">
      <div className="detail-topbar">
        <Link to="/">← {t.backToConsole}</Link>
        <LanguageToggle />
      </div>

      <section className="panel forensic-section">
        <SectionHeader title={copy.basicInformationTitle} copy={copy.basicInformationCopy} compact />
        <TransactionHashRail value={data.tx_hash} label={copy.transactionHashLabel} copy={copy} />
        <BrowserInfoPanel rows={browserFieldRows} inputData={data.input_data} copy={copy} />
      </section>

      <section className="panel forensic-section">
        <SectionHeader title={copy.balanceChangesTitle} copy={copy.balanceChangesCopy} />
        <div className="forensic-table-shell">
          <table className="forensic-table">
            <thead>
              <tr>
                <th>{copy.addressLabel}</th>
                <th>{copy.addressRoleLabel}</th>
                <th>{copy.assetLabel}</th>
                <th>{copy.assetRecordTypeLabel}</th>
                <th className="align-right">{copy.borrowedLabel}</th>
                <th className="align-right">{copy.repaidLabel}</th>
                <th className="align-right">{copy.premiumFeeLabel}</th>
                <th>{copy.directionLabel}</th>
                <th>{copy.strictRecordLabel}</th>
              </tr>
            </thead>
            <tbody>
              {balanceRows.length === 0 ? (
                <tr>
                  <td colSpan={9} className="forensic-empty-cell">{copy.noData}</td>
                </tr>
              ) : (
                balanceRows.map((row) => (
                  <tr key={row.id}>
                    <td><AddressText value={row.address} /></td>
                    <td>{row.roleLabel}</td>
                    <td><AddressText value={row.assetAddress} fallback={row.assetLabel} /></td>
                    <td>{row.recordType}</td>
                    <td className="align-right mono-inline">{row.borrowed}</td>
                    <td className="align-right mono-inline">{row.repaid}</td>
                    <td className="align-right mono-inline">{row.fee}</td>
                    <td>{row.direction}</td>
                    <td>
                      <span className={`forensic-inline-flag ${row.strict ? 'is-strong' : ''}`}>
                        {row.strict ? copy.strictYes : copy.strictNo}
                      </span>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
        <p className="forensic-section-note">{balanceSentence}</p>
      </section>

      <section className="panel forensic-section">
        <SectionHeader title={copy.fundFlowTitle} copy={copy.fundFlowCopy} />
        <FundFlowDiagram model={fundFlowModel} />
      </section>

      <section className="panel forensic-section">
        <SectionHeader title={copy.invocationFlowTitle} copy={copy.invocationFlowCopy} />
        <InvocationFlowTree nodes={invocationFlowNodes} emptyLabel={copy.invocationFlowEmpty} />
      </section>

      <section className="panel forensic-section">
        <SectionHeader title={copy.rawEvidenceTitle} copy={copy.rawEvidenceCopy} compact />
        <div className="forensic-raw-panel">
          <div className="forensic-raw-panel-head">
            <strong>{copy.rawProtocolFieldsTitle}</strong>
            <p>{copy.rawProtocolFieldsCopy}</p>
          </div>
          <div className="forensic-raw-panel-body">
            {data.interactions.map((interaction) => (
              <article key={interaction.interaction_id} className="forensic-raw-block">
                <div className="forensic-raw-block-head">
                  <div>
                    <strong>{protocolName(interaction.protocol)} · {interaction.entrypoint}</strong>
                    <p>{interaction.interaction_id}</p>
                  </div>
                  <span className={`forensic-status-pill verdict-${interaction.strict ? 'strict' : interaction.verified ? 'verified' : 'candidate'}`}>
                    {interaction.strict ? t.strict : interaction.verified ? t.verified : t.candidate}
                  </span>
                </div>
                <DefinitionGrid rows={buildInteractionFieldRows(interaction, protocolName, t, copy)} />
                <div className="forensic-table-shell compact">
                  <table className="forensic-table">
                    <thead>
                      <tr>
                        <th>{copy.assetLabel}</th>
                        <th>{copy.assetRecordTypeLabel}</th>
                        <th className="align-right">{copy.borrowedLabel}</th>
                        <th className="align-right">{copy.repaidLabel}</th>
                        <th className="align-right">{copy.premiumFeeLabel}</th>
                        <th>{copy.settlementModeLabel}</th>
                        <th>{copy.strictRecordLabel}</th>
                      </tr>
                    </thead>
                    <tbody>
                      {interaction.legs.map((leg) => (
                        <tr key={`${interaction.interaction_id}-${leg.leg_index}`}>
                          <td><AddressText value={leg.asset_address} /></td>
                          <td>{formatAssetRole(leg.asset_role, copy)}</td>
                          <td className="align-right mono-inline">{normalizeAmount(leg.amount_borrowed)}</td>
                          <td className="align-right mono-inline">{normalizeAmount(leg.amount_repaid)}</td>
                          <td className="align-right mono-inline">{formatFeeDisplay(leg.premium_amount, leg.fee_amount)}</td>
                          <td>{leg.settlement_mode || copy.notAvailable}</td>
                          <td>
                            <span className={`forensic-inline-flag ${leg.strict_leg ? 'is-strong' : ''}`}>
                              {leg.strict_leg ? copy.strictYes : copy.strictNo}
                            </span>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </article>
            ))}
          </div>
        </div>

      </section>
    </main>
  )
}

function SectionHeader({
  title,
  copy,
  compact,
}: {
  title: string
  copy: string
  compact?: boolean
}) {
  const [primaryTitle, secondaryTitle] = splitSectionTitle(title)

  return (
    <div className={`forensic-section-header${compact ? ' compact' : ''}`}>
      <div className="forensic-section-header-main">
        <h2>
          <span>{primaryTitle}</span>
          {secondaryTitle ? <small>{secondaryTitle}</small> : null}
        </h2>
        <p>{copy}</p>
      </div>
    </div>
  )
}

function DefinitionGrid({ rows }: { rows: BasicInfoRow[] }) {
  return (
    <dl className="forensic-definition-grid">
      {rows.map((row) => (
        <div key={row.label} className="forensic-definition-row">
          <dt>{row.label}</dt>
          <dd>{row.value}</dd>
        </div>
      ))}
    </dl>
  )
}

function BrowserInfoPanel({
  rows,
  inputData,
  copy,
}: {
  rows: BasicInfoRow[]
  inputData?: string
  copy: DetailCopy
}) {
  return (
    <div className="browser-info-panel">
      <BrowserInfoGrid rows={rows} columns={2} />
      <details className="browser-input-disclosure">
        <summary>
          <span>{copy.inputDataTitle}</span>
          <CopyButton value={inputData || copy.notAvailable} idleLabel={copy.copy} activeLabel={copy.copied} compact />
        </summary>
        <div className="browser-input-disclosure-body">
          <pre className="forensic-code-block browser-input-code">{inputData || copy.notAvailable}</pre>
        </div>
      </details>
    </div>
  )
}

function BrowserInfoGrid({ rows, columns }: { rows: BasicInfoRow[]; columns: number }) {
  return (
    <dl
      className="browser-info-grid"
      style={{ gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))` }}
    >
      {rows.map((row) => (
        <div key={row.label} className="browser-info-cell">
          <dt>
            <span>{row.label}</span>
            {row.action ? <span className="browser-info-cell-action">{row.action}</span> : null}
          </dt>
          <dd>{row.value}</dd>
        </div>
      ))}
    </dl>
  )
}

function splitSectionTitle(title: string) {
  const [primaryTitle, secondaryTitle] = title.split('/').map((item) => item.trim())
  return [primaryTitle, secondaryTitle] as const
}

function AddressText({ value, fallback }: { value?: string; fallback?: string }) {
  if (!value) {
    return <span className="forensic-muted">{fallback ?? 'N/A'}</span>
  }

  const displayValue = fallback ?? value

  return (
    <span className="forensic-address-text" title={value}>
      <span className="forensic-address-text-inner">{displayValue}</span>
    </span>
  )
}

function AddressList({ values }: { values: string[] }) {
  if (values.length === 0) {
    return <span className="forensic-muted">N/A</span>
  }

  return (
    <div className="forensic-address-list">
      {values.map((value) => (
        <AddressText key={value} value={value} />
      ))}
    </div>
  )
}

function AddressValueWithCopy({
  value,
  fallback,
  copy,
}: {
  value: string
  fallback?: string
  copy: DetailCopy
}) {
  return (
    <div className="forensic-address-field">
      <AddressText value={value} fallback={fallback} />
      <CopyButton value={value} idleLabel={copy.copy} activeLabel={copy.copied} compact />
    </div>
  )
}

function TransactionHashRail({
  value,
  label,
  copy,
}: {
  value: string
  label: string
  copy: DetailCopy
}) {
  return (
    <div className="forensic-hash-rail">
      <span className="forensic-hash-rail-label">{label}</span>
      <div className="forensic-hash-rail-value">
        <AddressText value={value} fallback={value} />
        <CopyButton value={value} idleLabel={copy.copy} activeLabel={copy.copied} compact />
      </div>
    </div>
  )
}

function CopyButton({
  value,
  idleLabel,
  activeLabel,
  compact,
}: {
  value: string
  idleLabel: string
  activeLabel: string
  compact?: boolean
}) {
  const [copied, setCopied] = useState(false)

  const handleCopy = async () => {
    if (!navigator?.clipboard?.writeText) {
      return
    }
    await navigator.clipboard.writeText(value)
    setCopied(true)
    window.setTimeout(() => setCopied(false), 1000)
  }

  return (
    <button
      type="button"
      className={`forensic-copy-button${compact ? ' compact' : ''}`}
      onClick={() => void handleCopy()}
      aria-label={copied ? activeLabel : idleLabel}
      title={copied ? activeLabel : idleLabel}
    >
      <span className="forensic-copy-icon" aria-hidden="true">
        <svg viewBox="0 0 20 20" fill="none">
          <rect x="6.5" y="3.5" width="10" height="12" rx="2.5" stroke="currentColor" strokeWidth="1.5" />
          <rect x="3.5" y="6.5" width="10" height="10" rx="2.5" stroke="currentColor" strokeWidth="1.5" />
        </svg>
      </span>
    </button>
  )
}

function buildInteractionFieldRows(
  interaction: TransactionDetailResponse['interactions'][number],
  protocolName: (protocol: string) => string,
  t: ReturnType<typeof useI18n>['t'],
  copy: DetailCopy,
) {
  return [
    { label: copy.protocolLabel, value: protocolName(interaction.protocol) },
    { label: copy.protocolEntrypointLabel, value: interaction.entrypoint || copy.notAvailable },
    { label: copy.callbackSeenLabel, value: interaction.callback_seen ? t.yes : t.no },
    { label: copy.repaymentSeenLabel, value: interaction.repayment_seen ? t.yes : t.no },
    { label: copy.settlementSeenLabel, value: interaction.settlement_seen ? t.yes : t.no },
    { label: copy.debtOpeningLabel, value: interaction.contains_debt_opening ? t.yes : t.no },
    { label: copy.methodSelectorLabel, value: interaction.raw_method_selector || copy.notAvailable },
    { label: copy.exclusionReasonLabel, value: interaction.exclusion_reason || copy.notAvailable },
    { label: copy.verificationNotesLabel, value: interaction.verification_notes || copy.notAvailable },
  ] satisfies BasicInfoRow[]
}

function buildBasicInformationRows(
  data: TransactionDetailResponse,
  addressMap: Map<string, AddressSummary>,
  protocolName: (protocol: string) => string,
  t: ReturnType<typeof useI18n>['t'],
  copy: DetailCopy,
): BasicInfoRow[] {
  const primaryInteraction = data.interactions[0]
  const receiverOrCallback = uniqueNonEmpty([
    primaryInteraction?.callback_target,
    primaryInteraction?.receiver_address,
    data.to_address,
  ])

  return [
    { label: copy.scanVerdictLabel, value: buildVerdictLabel(data, t) },
    { label: copy.identificationTypeLabel, value: copy.identificationTypeValue },
    { label: copy.hitProtocolsLabel, value: data.protocols.map((item) => protocolName(item)).join(', ') || copy.notAvailable },
    {
      label: copy.protocolEntrypointLabel,
      value: primaryInteraction ? `${protocolName(primaryInteraction.protocol)} · ${primaryInteraction.entrypoint}` : (data.method_selector || copy.notAvailable),
    },
    { label: copy.transactionHashLabel, value: <AddressText value={data.tx_hash} fallback={data.tx_hash} /> },
    { label: copy.blockLabel, value: data.block_number || copy.notAvailable },
    { label: copy.timeLabel, value: formatUnixTimestamp(data.timestamp, copy) },
    { label: copy.senderLabel, value: <AddressText value={data.from_address || primaryInteraction?.initiator} /> },
    { label: copy.receiverCallbackLabel, value: <AddressList values={receiverOrCallback} /> },
    { label: copy.positionInBlockLabel, value: data.tx_index !== undefined ? String(data.tx_index) : copy.notAvailable },
    { label: copy.gasUsedLabel, value: data.gas_used || copy.notAvailable },
    { label: copy.gasPriceLabel, value: formatGasPrice(data.effective_gas_price, copy) },
    { label: copy.transactionFeeLabel, value: formatTransactionFee(data.transaction_fee, copy) },
    {
      label: copy.traceStatusLabel,
      value: data.trace_summary ? formatTraceStatus(data.trace_summary.status, copy) : copy.notAvailable,
    },
    {
      label: copy.internalFramesLabel,
      value: data.trace_summary?.frames?.length !== undefined ? String(data.trace_summary.frames.length) : copy.notAvailable,
    },
  ]
}

function buildBalanceChangeRows(
  data: TransactionDetailResponse,
  addressMap: Map<string, AddressSummary>,
  copy: DetailCopy,
): BalanceChangeRow[] {
  const rows: BalanceChangeRow[] = []

  data.interactions.forEach((interaction) => {
    interaction.legs.forEach((leg) => {
      const borrowed = normalizeAmount(leg.amount_borrowed)
      const repaid = normalizeAmount(leg.amount_repaid)
      const fee = formatFeeDisplay(leg.premium_amount, leg.fee_amount)
      const borrowerAddress = firstNonEmpty(interaction.callback_target, interaction.receiver_address, interaction.initiator, interaction.provider_address)
      const repaymentAddress = firstNonEmpty(leg.repaid_to_address, interaction.provider_address, interaction.pair_address, interaction.factory_address)
      const borrowerRole = resolveAddressRoleLabel(borrowerAddress, addressMap, 'receiver', copy)
      const repaymentRole = resolveAddressRoleLabel(repaymentAddress, addressMap, 'repayment_target', copy)
      const recordType = formatAssetRole(leg.asset_role, copy)

      if (borrowerAddress && borrowerAddress === repaymentAddress) {
        rows.push({
          id: `${interaction.interaction_id}-${leg.leg_index}-merged`,
          address: borrowerAddress,
          roleLabel: borrowerRole,
          assetAddress: leg.asset_address,
          assetLabel: assetDisplayLabel(leg.asset_address),
          recordType,
          borrowed,
          repaid,
          fee,
          direction: formatDirectionLabel(borrowed, repaid, fee, copy),
          strict: leg.strict_leg,
        })
        return
      }

      if (borrowed !== '0' && borrowerAddress) {
        rows.push({
          id: `${interaction.interaction_id}-${leg.leg_index}-borrow`,
          address: borrowerAddress,
          roleLabel: borrowerRole,
          assetAddress: leg.asset_address,
          assetLabel: assetDisplayLabel(leg.asset_address),
          recordType,
          borrowed,
          repaid: '0',
          fee: fee === '0' ? '0' : fee,
          direction: copy.borrowDirectionLabel,
          strict: leg.strict_leg,
        })
      }

      if ((repaid !== '0' || fee !== '0') && repaymentAddress) {
        rows.push({
          id: `${interaction.interaction_id}-${leg.leg_index}-repay`,
          address: repaymentAddress,
          roleLabel: repaymentRole,
          assetAddress: leg.asset_address,
          assetLabel: assetDisplayLabel(leg.asset_address),
          recordType,
          borrowed: '0',
          repaid,
          fee,
          direction: formatDirectionLabel('0', repaid, fee, copy),
          strict: leg.strict_leg,
        })
      }
    })
  })

  return rows.sort((left, right) => {
    const roleDelta = getRolePriorityByLabel(left.roleLabel, copy) - getRolePriorityByLabel(right.roleLabel, copy)
    if (roleDelta !== 0) {
      return roleDelta
    }
    return left.address.localeCompare(right.address)
  })
}

function buildFundFlowModel(
  data: TransactionDetailResponse,
  addressMap: Map<string, AddressSummary>,
  copy: DetailCopy,
): FundFlowDiagramModel {
  const orderedTraceFlows = sortFundFlowsByTrace(data)
    .filter((flow) => isMeaningfulFundFlow(flow.action, flow.amount))
    .filter((flow) => !isZeroAddress(flow.source) && !isZeroAddress(flow.target))
  const traceAvailable = data.trace_summary?.status === 'available' && orderedTraceFlows.length > 0
  const lanes: FundFlowDiagramModel['lanes'] = []

  data.interactions.forEach((interaction, interactionIndex) => {
    const receiver = firstNonEmpty(interaction.callback_target, interaction.receiver_address, interaction.initiator)
    if (!receiver) {
      return
    }

    interaction.legs.forEach((leg) => {
      const lane = traceAvailable
        ? buildFundFlowLane(data, interaction, leg, interactionIndex, receiver, orderedTraceFlows, addressMap, copy)
        : buildFallbackFundFlowLane(data, interaction, leg, interactionIndex, receiver, addressMap, copy)
      if (lane) {
        lanes.push(lane)
      }
    })
  })

  return {
    lanes,
    emptyLabel: copy.fundFlowEmpty,
  }
}

function buildFundFlowLane(
  data: TransactionDetailResponse,
  interaction: TransactionDetailResponse['interactions'][number],
  leg: TransactionDetailResponse['interactions'][number]['legs'][number],
  interactionIndex: number,
  receiver: string,
  orderedTraceFlows: ReturnType<typeof sortFundFlowsByTrace>,
  addressMap: Map<string, AddressSummary>,
  copy: DetailCopy,
): FundFlowDiagramModel['lanes'][number] | null {
  const nodes = new Map<string, FundFlowDiagramModel['lanes'][number]['nodes'][number]>()
  const segments: FundFlowDiagramModel['lanes'][number]['segments'] = []
  const borrowedAsset = normalizeAddress(leg.asset_address)
  const borrowedAmount = normalizeAmount(leg.amount_borrowed)
  const repaidAmount = normalizeAmount(leg.amount_repaid)
  const feeAmount = sumRawAmounts(leg.premium_amount, leg.fee_amount)
  const expectedRepayment = repaidAmount !== '0'
    ? sumRawAmounts(repaidAmount, leg.premium_amount, leg.fee_amount)
    : sumRawAmounts(borrowedAmount, leg.premium_amount, leg.fee_amount)
  const indexedFlows = orderedTraceFlows.map((flow, order) => ({ ...flow, order }))

  const pushNode = (
    address: string,
    role: 'borrow_source' | 'receiver' | 'hop' | 'repayment_target' | 'topup_source',
    titleOverride?: string,
  ) => {
    const id = address.toLowerCase()
    const existing = nodes.get(id)
    const stage = flowRoleToStage(role)
    const title = titleOverride ?? resolveFlowNodeDisplayTitle(address, stage, addressMap.get(id), copy, data.from_address)
    if (!existing) {
      nodes.set(id, {
        id,
        roles: [role],
        title,
        subtitle: shortenHash(address),
        address,
      })
      return id
    }

    if (!existing.roles.includes(role)) {
      existing.roles.push(role)
    }
    if (existing.title === existing.subtitle && title !== existing.subtitle) {
      existing.title = title
    }
    nodes.set(id, existing)
    return id
  }

  const pushSegment = (
    id: string,
    from: string,
    to: string,
    action: string,
    asset: string,
    amount: string,
    tone: 'borrow' | 'swap' | 'repay',
  ) => {
    segments.push({ id, from, to, action, asset, amount, tone })
  }

  const borrowCandidates = indexedFlows.filter((flow) =>
    sameAddress(flow.target, receiver)
    && sameAddress(flow.asset_address, borrowedAsset)
  )
  const borrowFlow = pickPreferredIndexedFlow(
    borrowCandidates.filter((flow) => sameAddress(flow.source, interaction.provider_address) && normalizeAmount(flow.amount) === borrowedAmount),
    borrowCandidates.filter((flow) => !sameAddress(flow.source, data.from_address) && normalizeAmount(flow.amount) === borrowedAmount),
    borrowCandidates.filter((flow) => normalizeAmount(flow.amount) === borrowedAmount),
    borrowCandidates.filter((flow) => !sameAddress(flow.source, data.from_address)),
    borrowCandidates,
  )
  const borrowSource = borrowFlow?.source ?? firstNonEmpty(interaction.provider_address, interaction.factory_address, interaction.pair_address)

  if (borrowSource && borrowedAmount !== '0') {
    pushSegment(
      `${interaction.interaction_id}-${leg.leg_index}-borrow-${interactionIndex}`,
      pushNode(borrowSource, 'borrow_source'),
      pushNode(receiver, 'receiver'),
      copy.borrowDirectionLabel,
      assetDisplayLabel(leg.asset_address),
      compactRawAmount(borrowedAmount),
      'borrow',
    )
  }

  const repayCandidates = indexedFlows.filter((flow) =>
    sameAddress(flow.source, receiver)
    && sameAddress(flow.asset_address, borrowedAsset)
  )
  const repayFlow = pickPreferredIndexedFlow(
    repayCandidates.filter((flow) => sameAddress(flow.target, leg.repaid_to_address) && normalizeAmount(flow.amount) === expectedRepayment),
    repayCandidates.filter((flow) => sameAddress(flow.target, interaction.provider_address) && normalizeAmount(flow.amount) === expectedRepayment),
    repayCandidates.filter((flow) => normalizeAmount(flow.amount) === expectedRepayment),
    repayCandidates.filter((flow) => sameAddress(flow.target, interaction.provider_address)),
    repayCandidates,
  )
  const repaymentTarget = repayFlow?.target ?? firstNonEmpty(leg.repaid_to_address, interaction.provider_address, interaction.pair_address, interaction.factory_address)

  const topUpCandidates = indexedFlows.filter((flow) =>
    sameAddress(flow.target, receiver)
    && sameAddress(flow.asset_address, borrowedAsset)
    && !sameAddress(flow.source, borrowSource)
    && !sameAddress(flow.source, repaymentTarget)
    && (borrowFlow ? flow.order > borrowFlow.order : true)
  )
  const topUpFlows = pickKeyTopupFlows({
    flows: topUpCandidates,
    sender: data.from_address,
    expectedRepayment,
  })

  const swapSegments = extractKeyExecutionSegments({
    receiver,
    indexedFlows,
    borrowSource,
    repaymentTarget,
    topupSources: topUpFlows.map((flow) => flow.source),
    startAfterOrder: borrowFlow?.order ?? -1,
  })

  swapSegments.forEach((flow, index) => {
    const fromRole = sameAddress(flow.source, receiver) ? 'receiver' : 'hop'
    const toRole = sameAddress(flow.target, receiver) ? 'receiver' : 'hop'
      pushSegment(
        `${interaction.interaction_id}-${leg.leg_index}-swap-${interactionIndex}-${index}`,
        pushNode(flow.source, fromRole),
        pushNode(flow.target, toRole),
        humanizeFlowAction(flow.action, copy),
        assetDisplayLabel(flow.asset_address),
        compactRawAmount(normalizeAmount(flow.amount)),
        'swap',
      )
    })

  topUpFlows.forEach((topUpFlow, index) => {
    pushSegment(
      `${interaction.interaction_id}-${leg.leg_index}-topup-${interactionIndex}-${index}`,
      pushNode(topUpFlow.source, 'topup_source', sameAddress(topUpFlow.source, data.from_address) ? copy.senderLabel : undefined),
      pushNode(receiver, 'receiver'),
      copy.repayDirectionLabel,
      assetDisplayLabel(topUpFlow.asset_address),
      compactRawAmount(normalizeAmount(topUpFlow.amount)),
      'repay',
    )
  })

  if (repaymentTarget && (expectedRepayment !== '0' || feeAmount !== '0')) {
    pushSegment(
      `${interaction.interaction_id}-${leg.leg_index}-repay-${interactionIndex}`,
      pushNode(receiver, 'receiver'),
      pushNode(repaymentTarget, 'repayment_target'),
      copy.repayDirectionLabel,
      assetDisplayLabel(leg.asset_address),
      compactRawAmount(expectedRepayment !== '0' ? expectedRepayment : feeAmount),
      'repay',
    )
  }

  const laneSegments = dedupeLaneSegments(segments)
  if (laneSegments.length === 0) {
    return null
  }

  return {
    id: `${interaction.interaction_id}-${leg.leg_index}`,
    label: assetDisplayLabel(leg.asset_address),
    sublabel: `${copy.borrowDirectionLabel} ${compactRawAmount(borrowedAmount)} · ${swapSegments.length}${copy.isEnglish ? ' key execution path(s)' : ' 条关键执行支线'} · ${copy.repayDirectionLabel} ${compactRawAmount(expectedRepayment !== '0' ? expectedRepayment : feeAmount)}`,
    nodes: Array.from(nodes.values()),
    segments: laneSegments,
  }
}

function buildFallbackFundFlowLane(
  data: TransactionDetailResponse,
  interaction: TransactionDetailResponse['interactions'][number],
  leg: TransactionDetailResponse['interactions'][number]['legs'][number],
  interactionIndex: number,
  receiver: string,
  addressMap: Map<string, AddressSummary>,
  copy: DetailCopy,
): FundFlowDiagramModel['lanes'][number] | null {
  const nodes = new Map<string, FundFlowDiagramModel['lanes'][number]['nodes'][number]>()
  const borrowedAmount = normalizeAmount(leg.amount_borrowed)
  const expectedRepayment = sumRawAmounts(
    leg.amount_repaid !== '0' ? leg.amount_repaid : leg.amount_borrowed,
    leg.premium_amount,
    leg.fee_amount,
  )
  const provider = firstNonEmpty(interaction.provider_address, interaction.pair_address, interaction.factory_address)
  const repaymentTarget = firstNonEmpty(leg.repaid_to_address, interaction.provider_address, interaction.pair_address, interaction.factory_address)

  const pushNode = (
    address: string,
    role: 'borrow_source' | 'receiver' | 'hop' | 'repayment_target' | 'topup_source',
  ) => {
    const id = address.toLowerCase()
    if (!nodes.has(id)) {
      const stage = flowRoleToStage(role)
      nodes.set(id, {
        id,
        roles: [role],
        title: resolveFlowNodeDisplayTitle(address, stage, addressMap.get(id), copy, data.from_address),
        subtitle: shortenHash(address),
        address,
      })
    }
    return id
  }

  const segments: FundFlowDiagramModel['lanes'][number]['segments'] = []

  if (provider && borrowedAmount !== '0') {
    segments.push({
      id: `${interaction.interaction_id}-${leg.leg_index}-fallback-borrow-${interactionIndex}`,
      from: pushNode(provider, 'borrow_source'),
      to: pushNode(receiver, 'receiver'),
      action: copy.borrowDirectionLabel,
      asset: assetDisplayLabel(leg.asset_address),
      amount: compactRawAmount(borrowedAmount),
      tone: 'borrow',
    })
  }

  if (repaymentTarget && expectedRepayment !== '0') {
    segments.push({
      id: `${interaction.interaction_id}-${leg.leg_index}-fallback-repay-${interactionIndex}`,
      from: pushNode(receiver, 'receiver'),
      to: pushNode(repaymentTarget, 'repayment_target'),
      action: copy.repayDirectionLabel,
      asset: assetDisplayLabel(leg.asset_address),
      amount: compactRawAmount(expectedRepayment),
      tone: 'repay',
    })
  }

  if (segments.length === 0) {
    return null
  }

  return {
    id: `${interaction.interaction_id}-${leg.leg_index}`,
    label: assetDisplayLabel(leg.asset_address),
    sublabel: copy.isEnglish ? 'Trace unavailable, fallback lane only' : 'Trace 不可用，仅展示最小闭环',
    nodes: Array.from(nodes.values()),
    segments,
  }
}

function normalizeBackendFundFlowModel(
  graph: NonNullable<TransactionDetailResponse['fund_flow_graph']>,
  copy: DetailCopy,
): FundFlowDiagramModel {
  return {
    emptyLabel: graph.empty_label || copy.fundFlowEmpty,
    lanes: (graph.lanes ?? []).map((lane) => ({
      id: lane.id,
      label: assetDisplayLabel(lane.asset_address || lane.label),
      sublabel: lane.sublabel,
      nodes: lane.nodes.map((node) => ({
        id: node.id,
        roles: node.roles as Array<'borrow_source' | 'receiver' | 'hop' | 'repayment_target' | 'topup_source'>,
        title: normalizeBackendFundFlowNodeTitle(node, copy),
        subtitle: node.address ? shortenHash(node.address) : node.subtitle,
        address: node.address,
      })),
      segments: lane.segments.map((segment) => ({
        id: segment.id,
        from: segment.from,
        to: segment.to,
        action: normalizeBackendFundFlowAction(segment.action, copy),
        asset: assetDisplayLabel(segment.asset_address || segment.asset),
        amount: segment.amount,
        tone: segment.tone,
      })),
    })),
  }
}

function normalizeBackendFundFlowNodeTitle(
  node: NonNullable<NonNullable<NonNullable<TransactionDetailResponse['fund_flow_graph']>['lanes']>[number]['nodes']>[number],
  copy: DetailCopy,
) {
  if (node.address) {
    const known = KNOWN_ADDRESS_LABELS[node.address.toLowerCase()]
    if (known) {
      return known
    }
  }
  const roles = node.roles ?? []
  if (roles.includes('receiver') && node.title === 'Receiver / Callback') {
    return copy.receiverCallbackNodeLabel
  }
  if (roles.includes('repayment_target') && node.title === 'Repayment Target') {
    return copy.repaymentTargetNodeLabel
  }
  if (roles.includes('borrow_source') && node.title === 'Provider / Pool') {
    return copy.providerPoolLabel
  }
  if (roles.includes('topup_source') && node.title === 'Top-up Source') {
    return copy.isEnglish ? 'Top-up Source' : '补足来源'
  }
  return node.title || copy.notAvailable
}

function normalizeBackendFundFlowAction(action: string, copy: DetailCopy) {
  switch (action) {
    case 'Borrow':
      return copy.borrowDirectionLabel
    case 'Top-up':
      return copy.topupDirectionLabel
    case 'Repay':
      return copy.repayDirectionLabel
    case 'Swap':
      return copy.swapDirectionLabel
    default:
      return action || copy.unknownDirectionLabel
  }
}

function buildInvocationNodes(
  data: TransactionDetailResponse,
  addressMap: Map<string, AddressSummary>,
  copy: DetailCopy,
  raw: boolean,
): InvocationFlowNode[] {
  const frames = data.trace_summary?.frames ?? []
  if (frames.length === 0) {
    return []
  }

  const sortedFrames = [...frames].sort((left, right) => {
    if (left.call_index !== right.call_index) {
      return left.call_index - right.call_index
    }
    return left.depth - right.depth
  })
  const frameIds = new Set(sortedFrames.map((frame) => frame.id))
  const childrenByParent = new Map<string, typeof sortedFrames>()
  sortedFrames.forEach((frame) => {
    const parentKey = frame.parent_id && frameIds.has(frame.parent_id) ? frame.parent_id : '__root__'
    const siblings = childrenByParent.get(parentKey) ?? []
    siblings.push(frame)
    childrenByParent.set(parentKey, siblings)
  })

  const buildNode = (frame: (typeof sortedFrames)[number]): InvocationFlowNode | null => {
    const childNodes = (childrenByParent.get(frame.id) ?? [])
      .map((child) => buildNode(child))
      .filter((child): child is InvocationFlowNode => Boolean(child))

    const important =
      raw ||
      frame.depth <= 1 ||
      Boolean(frame.token_action) ||
      Boolean(frame.error || frame.revert_reason) ||
      Boolean(frame.tags?.includes('callback_path')) ||
      Boolean(frame.tags?.includes('repayment_path')) ||
      frame.type.toLowerCase().includes('event')

    if (!raw && !important && childNodes.length === 0) {
      return null
    }

    const phase = classifyFramePhase(frame)
    return {
      id: frame.id,
      depth: frame.depth,
      title: frame.token_action || frame.method_selector || frame.type,
      detail: describeFrame(frame, addressMap, copy),
      from: frame.from,
      to: frame.to,
      phase,
      phaseLabel: resolvePhaseLabel(phase, copy),
      badges: buildFrameBadges(frame, copy),
      meta: buildFrameMeta(frame, copy, raw),
      children: childNodes,
    }
  }

  return (childrenByParent.get('__root__') ?? [])
    .map((frame) => buildNode(frame))
    .filter((node): node is InvocationFlowNode => Boolean(node))
}

function buildBrowserFieldRows(
  data: TransactionDetailResponse,
  addressMap: Map<string, AddressSummary>,
  protocolName: (protocol: string) => string,
  t: ReturnType<typeof useI18n>['t'],
  copy: DetailCopy,
): BasicInfoRow[] {
  const primaryInteraction = data.interactions[0]
  const receiverOrCallback = uniqueNonEmpty([
    primaryInteraction?.callback_target,
    primaryInteraction?.receiver_address,
    data.to_address,
  ])

  return [
    { label: copy.scanVerdictLabel, value: buildVerdictLabel(data, t) },
    { label: copy.identificationTypeLabel, value: copy.identificationTypeValue },
    { label: copy.hitProtocolsLabel, value: data.protocols.map((item) => protocolName(item)).join(', ') || copy.notAvailable },
    {
      label: copy.protocolEntrypointLabel,
      value: primaryInteraction ? `${protocolName(primaryInteraction.protocol)} · ${primaryInteraction.entrypoint}` : (data.method_selector || copy.notAvailable),
    },
    { label: copy.timeLabel, value: formatUnixTimestamp(data.timestamp, copy) },
    { label: copy.chainLabel, value: String(data.chain_id) },
    { label: copy.blockLabel, value: data.block_number },
    { label: copy.positionInBlockLabel, value: data.tx_index !== undefined ? String(data.tx_index) : copy.notAvailable },
    { label: copy.transactionStatusLabel, value: formatTransactionStatus(data.status, copy) },
    { label: copy.senderLabel, value: <AddressText value={data.from_address || primaryInteraction?.initiator} /> },
    { label: copy.receiverCallbackLabel, value: <AddressList values={receiverOrCallback} /> },
    { label: copy.transactionToLabel, value: <AddressText value={data.to_address} /> },
    { label: copy.methodSelectorLabel, value: data.method_selector || copy.notAvailable },
    { label: copy.gasUsedLabel, value: data.gas_used || copy.notAvailable },
    { label: copy.gasPriceLabel, value: formatGasPrice(data.effective_gas_price, copy) },
    { label: copy.transactionFeeLabel, value: formatTransactionFee(data.transaction_fee, copy) },
    { label: copy.traceStatusLabel, value: data.trace_summary ? formatTraceStatus(data.trace_summary.status, copy) : copy.notAvailable },
    { label: copy.internalFramesLabel, value: data.trace_summary?.frames?.length !== undefined ? String(data.trace_summary.frames.length) : copy.notAvailable },
  ]
}

function buildSummarySentence(
  data: TransactionDetailResponse,
  protocolName: (protocol: string) => string,
  t: ReturnType<typeof useI18n>['t'],
  copy: DetailCopy,
) {
  const firstInteraction = data.interactions[0]
  const protocolLabel = firstInteraction ? protocolName(firstInteraction.protocol) : (data.protocols[0] ? protocolName(data.protocols[0]) : copy.notAvailable)
  const entrypoint = firstInteraction?.entrypoint || data.method_selector || copy.notAvailable
  return copy.isEnglish
    ? `This transaction is classified as a ${protocolLabel} flash loan interaction. The main entrypoint is ${entrypoint}, and the scanner keeps the verdict at ${buildVerdictLabel(data, t)}.`
    : `本笔交易识别为 ${protocolLabel} 闪电贷交互，主入口为 ${entrypoint}，扫描结论为${buildVerdictLabel(data, t)}。`
}

function buildBalanceSentence(rows: BalanceChangeRow[], copy: DetailCopy) {
  const strictCount = rows.filter((row) => row.strict).length
  const addressCount = new Set(rows.map((row) => row.address.toLowerCase())).size
  return copy.isEnglish
    ? `Balance changes now stay on the flash-loan loop itself: ${addressCount} key addresses and ${strictCount} strict asset records explain what was borrowed, what was repaid, and where fees landed.`
    : `资产变化现在只围绕闪电贷闭环本身展开：${addressCount} 个关键地址和 ${strictCount} 条严格资产记录说明了借入、归还与费用落点。`
}

function resolveAddressSummaries(data: TransactionDetailResponse, isEnglish: boolean) {
  const roleMap = new Map<string, Set<string>>()
  const appendRole = (address?: string, roleCode?: string) => {
    const normalized = (address ?? '').trim()
    if (!normalized || !roleCode) {
      return
    }
    const existing = roleMap.get(normalized) ?? new Set<string>()
    existing.add(roleCode)
    roleMap.set(normalized, existing)
  }

  if (data.summary?.addresses?.length) {
    data.summary.addresses.forEach((item) => {
      item.roles.forEach((roleCode) => appendRole(item.address, roleCode))
    })
  }

  if (roleMap.size === 0) {
    data.interactions.forEach((interaction) => {
      appendRole(interaction.initiator, 'initiator')
      appendRole(interaction.receiver_address, 'receiver')
      appendRole(interaction.callback_target, 'callback_target')
      appendRole(interaction.provider_address, 'provider')
      appendRole(interaction.factory_address, 'factory')
      appendRole(interaction.pair_address, 'pair')
      appendRole(interaction.on_behalf_of, 'on_behalf_of')
      interaction.legs.forEach((leg) => appendRole(leg.repaid_to_address, 'repayment_target'))
    })
  }

  return Array.from(roleMap.entries())
    .map(([address, roles]) => {
      const roleCodes = Array.from(roles.values()).sort((left, right) => getRolePriority(left) - getRolePriority(right))
      return {
        address,
        roleCodes,
        roleLabels: roleCodes.map((roleCode) => roleCodeToLabel(roleCode, isEnglish)),
      }
    })
    .sort((left, right) => {
      const priorityDelta = getRolePriority(left.roleCodes[0] ?? '') - getRolePriority(right.roleCodes[0] ?? '')
      if (priorityDelta !== 0) {
        return priorityDelta
      }
      return left.address.localeCompare(right.address)
    })
}

function buildVerdictLabel(data: TransactionDetailResponse, t: ReturnType<typeof useI18n>['t']) {
  if (data.strict) {
    return t.strict
  }
  if (data.verified) {
    return t.verified
  }
  return t.candidate
}

function resolveVerdictTone(data: TransactionDetailResponse) {
  if (data.strict) {
    return 'strict'
  }
  if (data.verified) {
    return 'verified'
  }
  return 'candidate'
}

function roleCodeToLabel(roleCode: string, isEnglish: boolean) {
  switch (roleCode) {
    case 'initiator':
      return isEnglish ? 'Initiator' : '发起地址'
    case 'on_behalf_of':
      return isEnglish ? 'On-Behalf-Of' : '代偿关系地址'
    case 'receiver':
      return isEnglish ? 'Receiver' : '接收地址'
    case 'callback_target':
      return isEnglish ? 'Callback Target' : '回调地址'
    case 'provider':
      return isEnglish ? 'Provider / Pool' : 'Provider / Pool'
    case 'factory':
      return isEnglish ? 'Factory' : '工厂合约'
    case 'pair':
      return isEnglish ? 'Pair / Pool' : '交易对 / 池子'
    case 'repayment_target':
      return isEnglish ? 'Repayment Target' : '归还目标'
    default:
      return roleCode
  }
}

function formatAssetRole(role: string | undefined, copy: DetailCopy) {
  switch (role) {
    case 'flash_loan_asset':
      return copy.borrowedAssetLabel
    case 'repayment_asset':
      return copy.repaymentAssetLabel
    case 'fee_asset':
      return copy.feeAssetLabel
    case 'premium_asset':
      return copy.premiumAssetLabel
    case 'collateral_asset':
      return copy.collateralAssetLabel
    case 'debt_asset':
      return copy.debtAssetLabel
    default:
      return role ? role.replace(/_/g, ' ') : copy.notAvailable
  }
}

function resolveAddressRoleLabel(
  address: string | undefined,
  addressMap: Map<string, AddressSummary>,
  fallbackCode: string,
  copy: DetailCopy,
) {
  if (!address) {
    return copy.notAvailable
  }
  const matched = addressMap.get(address.toLowerCase())
  if (matched?.roleLabels?.length) {
    return matched.roleLabels[0]
  }
  return roleCodeToLabel(fallbackCode, copy.isEnglish)
}

function resolveFlowStage(
  address: string,
  direction: 'source' | 'target',
  addressMap: Map<string, AddressSummary>,
): 'provider' | 'receiver' | 'intermediary' | 'repayment' {
  const roleCodes = addressMap.get(address.toLowerCase())?.roleCodes ?? []

  if (direction === 'source') {
    if (roleCodes.includes('provider')) {
      return 'provider'
    }
    if (roleCodes.includes('receiver') || roleCodes.includes('callback_target') || roleCodes.includes('initiator')) {
      return 'receiver'
    }
    return 'intermediary'
  }

  if (roleCodes.includes('repayment_target') || roleCodes.includes('provider')) {
    return 'repayment'
  }
  if (roleCodes.includes('receiver') || roleCodes.includes('callback_target') || roleCodes.includes('initiator')) {
    return 'receiver'
  }
  if (roleCodes.includes('factory') || roleCodes.includes('pair')) {
    return 'intermediary'
  }
  return 'intermediary'
}

function resolveStageLabel(stage: 'provider' | 'receiver' | 'intermediary' | 'repayment', copy: DetailCopy) {
  switch (stage) {
    case 'provider':
      return copy.providerPoolLabel
    case 'receiver':
      return copy.receiverCallbackNodeLabel
    case 'intermediary':
      return copy.intermediaryNodeLabel
    case 'repayment':
      return copy.repaymentTargetNodeLabel
  }
}

function resolveFlowNodeTitle(stage: 'provider' | 'receiver' | 'intermediary' | 'repayment', summary: AddressSummary | undefined, copy: DetailCopy) {
  if (stage === 'receiver') {
    return summary?.roleLabels.find((item) => item.includes('回调') || item.includes('Receiver') || item.includes('Callback')) ?? copy.receiverCallbackNodeLabel
  }
  if (stage === 'provider') {
    return summary?.roleLabels.find((item) => item.includes('Provider') || item.includes('流动性')) ?? copy.providerPoolLabel
  }
  if (stage === 'repayment') {
    return summary?.roleLabels.find((item) => item.includes('归还') || item.includes('Repayment') || item.includes('Provider')) ?? copy.repaymentTargetNodeLabel
  }
  return summary?.roleLabels.find((item) => item.includes('工厂') || item.includes('Pair') || item.includes('Pool')) ?? copy.intermediaryNodeLabel
}

function humanizeFlowAction(action: string, copy: DetailCopy) {
  const normalized = action.toLowerCase()
  if (normalized.includes('swap')) {
    return copy.swapDirectionLabel
  }
  if (normalized.includes('repay') || normalized.includes('return')) {
    return copy.repayDirectionLabel
  }
  if (normalized.includes('borrow') || normalized.includes('flash')) {
    return copy.borrowDirectionLabel
  }
  return action || copy.unknownDirectionLabel
}

function resolveFlowTone(
  action: string,
  sourceStage: 'provider' | 'receiver' | 'intermediary' | 'repayment',
  targetStage: 'provider' | 'receiver' | 'intermediary' | 'repayment',
): 'borrow' | 'swap' | 'repay' {
  const normalized = action.toLowerCase()
  if (normalized.includes('swap')) {
    return 'swap'
  }
  if (normalized.includes('repay') || normalized.includes('return') || targetStage === 'repayment') {
    return 'repay'
  }
  if (normalized.includes('borrow') || normalized.includes('flash') || sourceStage === 'provider') {
    return 'borrow'
  }
  return 'swap'
}

function mergeFlowStage(
  current: 'provider' | 'receiver' | 'intermediary' | 'repayment',
  incoming: 'provider' | 'receiver' | 'intermediary' | 'repayment',
) {
  const priority: Record<'provider' | 'receiver' | 'intermediary' | 'repayment', number> = {
    receiver: 0,
    provider: 1,
    repayment: 2,
    intermediary: 3,
  }
  return priority[incoming] < priority[current] ? incoming : current
}

function isMeaningfulFundFlow(action: string, amount: string) {
  const normalizedAction = action.toLowerCase()
  if (normalizeAmount(amount) === '0') {
    return false
  }
  return ['transfer', 'transferfrom', 'withdraw', 'deposit', 'swap'].some((keyword) => normalizedAction.includes(keyword))
}

function isZeroAddress(address: string) {
  return address.toLowerCase() === '0x0000000000000000000000000000000000000000'
}

function summarizeIntermediaryCandidates(
  flows: NonNullable<NonNullable<TransactionDetailResponse['trace_summary']>['asset_flows']>,
  provider: string | undefined,
  receiver: string | undefined,
) {
  if (!provider || !receiver) {
    return []
  }

  const providerLower = provider.toLowerCase()
  const receiverLower = receiver.toLowerCase()
  const scores = new Map<string, number>()

  flows.forEach((flow) => {
    const sourceLower = flow.source.toLowerCase()
    const targetLower = flow.target.toLowerCase()
    const touchesReceiver = sourceLower === receiverLower || targetLower === receiverLower
    if (!touchesReceiver) {
      return
    }

    ;[flow.source, flow.target].forEach((address) => {
      const lowered = address.toLowerCase()
      if (lowered === receiverLower || lowered === providerLower || isZeroAddress(address)) {
        return
      }
      scores.set(address, (scores.get(address) ?? 0) + 1)
    })
  })

  return Array.from(scores.entries())
    .map(([address, score]) => ({ address, score }))
    .sort((left, right) => right.score - left.score)
}

function sortFundFlowsByTrace(data: TransactionDetailResponse) {
  const frameOrder = new Map(
    (data.trace_summary?.frames ?? []).map((frame) => [frame.id, frame.call_index * 10 + frame.depth] as const),
  )

  return [...(data.trace_summary?.asset_flows ?? [])].sort((left, right) => {
    const leftOrder = frameOrder.get(left.frame_id) ?? Number.MAX_SAFE_INTEGER
    const rightOrder = frameOrder.get(right.frame_id) ?? Number.MAX_SAFE_INTEGER
    if (leftOrder !== rightOrder) {
      return leftOrder - rightOrder
    }
    return left.frame_id.localeCompare(right.frame_id)
  })
}

function dedupeLaneSegments(segments: FundFlowDiagramModel['lanes'][number]['segments']) {
  const grouped = new Map<string, FundFlowDiagramModel['lanes'][number]['segments'][number]>()
  segments.forEach((segment) => {
    const key = [segment.from, segment.to, segment.asset, segment.tone].join('|')
    if (!grouped.has(key)) {
      grouped.set(key, segment)
    }
  })
  return Array.from(grouped.values())
}

function pickPreferredIndexedFlow<T extends { order: number }>(...groups: T[][]) {
  for (const flows of groups) {
    if (flows.length === 0) {
      continue
    }
    return [...flows].sort((left, right) => left.order - right.order)[0]
  }
  return undefined
}

function pickPreferredFlow<T extends { source: string; amount: string }>(...groups: T[][]) {
  for (const flows of groups) {
    if (flows.length === 0) {
      continue
    }
    return [...flows].sort((left, right) => compareFlowPreference(left, right))[0]
  }
  return undefined
}

function compareFlowPreference(
  left: { source: string; amount: string },
  right: { source: string; amount: string },
) {
  const leftPriority = isLikelyUserAddress(left.source) ? 1 : 0
  const rightPriority = isLikelyUserAddress(right.source) ? 1 : 0
  if (leftPriority !== rightPriority) {
    return leftPriority - rightPriority
  }

  const leftLength = normalizeAmount(right.amount).length - normalizeAmount(left.amount).length
  if (leftLength !== 0) {
    return leftLength
  }

  return left.source.localeCompare(right.source)
}

function resolveFlowNodeDisplayTitle(
  address: string,
  stage: 'provider' | 'receiver' | 'intermediary' | 'repayment',
  summary: AddressSummary | undefined,
  copy: DetailCopy,
  senderAddress?: string,
) {
  if (senderAddress && sameAddress(address, senderAddress)) {
    return copy.senderLabel
  }
  const knownLabel = KNOWN_ADDRESS_LABELS[address.toLowerCase()]
  if (knownLabel) {
    return knownLabel
  }
  if (summary?.roleLabels.length) {
    return resolveFlowNodeTitle(stage, summary, copy)
  }
  if (stage === 'receiver') {
    return copy.receiverCallbackNodeLabel
  }
  return shortenHash(address)
}

function sumRawAmounts(...values: Array<string | undefined>) {
  const meaningful = values.map((value) => normalizeAmount(value)).filter((value) => value !== '0')
  if (meaningful.length === 0) {
    return '0'
  }
  try {
    return meaningful.reduce((total, value) => total + BigInt(value), 0n).toString()
  } catch {
    return meaningful[0]
  }
}

function flowRoleToStage(role: 'borrow_source' | 'receiver' | 'hop' | 'repayment_target' | 'topup_source') {
  switch (role) {
    case 'borrow_source':
      return 'provider' as const
    case 'receiver':
      return 'receiver' as const
    case 'repayment_target':
    case 'topup_source':
      return 'repayment' as const
    default:
      return 'intermediary' as const
  }
}

function extractKeyExecutionSegments(args: {
  receiver: string
  indexedFlows: Array<ReturnType<typeof sortFundFlowsByTrace>[number] & { order: number }>
  borrowSource?: string
  repaymentTarget?: string
  topupSources: string[]
  startAfterOrder: number
}) {
  const topupSourceSet = new Set(args.topupSources.map((value) => normalizeAddress(value)))
  const seenFrameIDs = new Set<string>()
  const segments: typeof args.indexedFlows = []
  const seeds = args.indexedFlows
    .filter((flow) =>
      flow.order > args.startAfterOrder
      && sameAddress(flow.source, args.receiver)
      && !sameAddress(flow.target, args.receiver)
      && !sameAddress(flow.target, args.repaymentTarget)
      && !sameAddress(flow.target, args.borrowSource)
      && !topupSourceSet.has(normalizeAddress(flow.target))
      && !flow.action.toLowerCase().includes('approve'),
    )
    .slice(0, 3)

  const pushIfNew = (flow: (typeof args.indexedFlows)[number] | undefined) => {
    if (!flow || seenFrameIDs.has(flow.frame_id)) {
      return false
    }
    seenFrameIDs.add(flow.frame_id)
    segments.push(flow)
    return true
  }

  seeds.forEach((seed) => {
    if (!pushIfNew(seed)) {
      return
    }
    let current = seed
    let depth = 0
    while (depth < 2) {
      depth += 1
      const next = args.indexedFlows.find((flow) =>
        flow.order > current.order
        && sameAddress(flow.source, current.target)
        && !flow.action.toLowerCase().includes('approve')
        && !sameAddress(flow.target, args.borrowSource)
        && !topupSourceSet.has(normalizeAddress(flow.target)),
      )
      if (!next || !pushIfNew(next)) {
        break
      }
      if (sameAddress(next.target, args.receiver) || sameAddress(next.target, args.repaymentTarget)) {
        break
      }
      current = next
    }
  })

  return segments
}

function pickKeyTopupFlows(args: {
  flows: Array<ReturnType<typeof sortFundFlowsByTrace>[number] & { order: number }>
  sender?: string
  expectedRepayment: string
}) {
  const sorted = [...args.flows].sort((left, right) => {
    const leftSenderPriority = sameAddress(left.source, args.sender) ? 0 : 1
    const rightSenderPriority = sameAddress(right.source, args.sender) ? 0 : 1
    if (leftSenderPriority !== rightSenderPriority) {
      return leftSenderPriority - rightSenderPriority
    }
    const leftAmountMatches = normalizeAmount(left.amount) === args.expectedRepayment ? 0 : 1
    const rightAmountMatches = normalizeAmount(right.amount) === args.expectedRepayment ? 0 : 1
    if (leftAmountMatches !== rightAmountMatches) {
      return leftAmountMatches - rightAmountMatches
    }
    return left.order - right.order
  })

  const seen = new Set<string>()
  return sorted.filter((flow) => {
    const key = `${normalizeAddress(flow.source)}|${normalizeAddress(flow.asset_address)}|${normalizeAmount(flow.amount)}`
    if (seen.has(key)) {
      return false
    }
    seen.add(key)
    return true
  }).slice(0, 3)
}

function normalizeAddress(value?: string) {
  return (value ?? '').trim().toLowerCase()
}

function sameAddress(left?: string, right?: string) {
  const leftAddress = normalizeAddress(left)
  const rightAddress = normalizeAddress(right)
  if (!leftAddress || !rightAddress) {
    return false
  }
  return leftAddress === rightAddress
}

function isLikelyUserAddress(address: string) {
  return !isZeroAddress(address) && !address.toLowerCase().startsWith('0x000000')
}

function compactRawAmount(value?: string) {
  const normalized = normalizeAmount(value)
  if (normalized.length <= 10) {
    return normalized
  }
  return `${normalized.slice(0, 6)}...${normalized.slice(-4)}`
}

function classifyFramePhase(frame: TraceFrame) {
  if (frame.error || frame.revert_reason || frame.tags?.includes('error')) {
    return 'error'
  }
  if (frame.depth === 0) {
    return 'entry'
  }
  if (frame.type.toLowerCase().includes('event')) {
    return 'event'
  }
  if (frame.tags?.includes('callback_path')) {
    return 'callback'
  }
  if (frame.tags?.includes('repayment_path')) {
    return 'repayment'
  }
  if (frame.token_action && ['approve', 'transfer', 'withdraw', 'deposit'].includes(frame.token_action.toLowerCase())) {
    return 'settlement'
  }
  return 'neutral'
}

function resolvePhaseLabel(
  phase: InvocationFlowNode['phase'],
  copy: DetailCopy,
) {
  switch (phase) {
    case 'entry':
      return copy.phaseEntryLabel
    case 'callback':
      return copy.phaseCallbackLabel
    case 'repayment':
      return copy.phaseRepaymentLabel
    case 'settlement':
      return copy.phaseSettlementLabel
    case 'error':
      return copy.phaseErrorLabel
    case 'event':
      return copy.phaseEventLabel
    default:
      return copy.phaseInternalLabel
  }
}

function buildFrameBadges(
  frame: TraceFrame,
  copy: DetailCopy,
) {
  const badges = [frame.type]
  if (frame.token_action) {
    badges.push(frame.token_action)
  }
  if (frame.tags?.includes('callback_path')) {
    badges.push(copy.callbackTag)
  }
  if (frame.tags?.includes('repayment_path')) {
    badges.push(copy.repaymentTag)
  }
  return badges.slice(0, 3)
}

function buildFrameMeta(
  frame: TraceFrame,
  copy: DetailCopy,
  raw: boolean,
) {
  const meta = [`${copy.frameLabel} ${frame.call_index + 1}`]
  if (raw && frame.method_selector) {
    meta.push(`${copy.methodSelectorLabel} ${frame.method_selector}`)
  }
  if (frame.asset_address) {
    meta.push(`${assetDisplayLabel(frame.asset_address)} ${normalizeAmount(frame.token_amount)}`)
  }
  if (raw && frame.id) {
    meta.push(frame.id)
  }
  return meta
}

function describeFrame(
  frame: TraceFrame,
  addressMap: Map<string, AddressSummary>,
  copy: DetailCopy,
) {
  if (frame.error || frame.revert_reason) {
    return frame.revert_reason || frame.error || copy.notAvailable
  }

  const fromRole = addressMap.get(frame.from.toLowerCase())?.roleLabels[0]
  const toRole = addressMap.get(frame.to.toLowerCase())?.roleLabels[0]

  if (frame.tags?.includes('callback_path')) {
    return copy.isEnglish
      ? `Callback-side execution expands from ${fromRole ?? shortenHash(frame.from)} to ${toRole ?? shortenHash(frame.to)}.`
      : `回调侧执行从 ${fromRole ?? shortenHash(frame.from)} 展开到 ${toRole ?? shortenHash(frame.to)}。`
  }
  if (frame.tags?.includes('repayment_path')) {
    return copy.isEnglish
      ? `Repayment-side execution moves from ${fromRole ?? shortenHash(frame.from)} to ${toRole ?? shortenHash(frame.to)}.`
      : `归还侧执行从 ${fromRole ?? shortenHash(frame.from)} 走向 ${toRole ?? shortenHash(frame.to)}。`
  }
  if (frame.token_action) {
    return copy.isEnglish
      ? `${frame.token_action} touches ${assetDisplayLabel(frame.asset_address)} with amount ${normalizeAmount(frame.token_amount)}.`
      : `${frame.token_action} 涉及 ${assetDisplayLabel(frame.asset_address)}，数量为 ${normalizeAmount(frame.token_amount)}。`
  }
  return copy.isEnglish
    ? `Internal ${frame.type.toLowerCase()} between ${fromRole ?? shortenHash(frame.from)} and ${toRole ?? shortenHash(frame.to)}.`
    : `${fromRole ?? shortenHash(frame.from)} 与 ${toRole ?? shortenHash(frame.to)} 之间的内部 ${frame.type.toLowerCase()}。`
}

function formatDirectionLabel(borrowed: string, repaid: string, fee: string, copy: DetailCopy) {
  if (borrowed !== '0' && repaid !== '0') {
    return copy.borrowRepayDirectionLabel
  }
  if (repaid !== '0' && fee !== '0') {
    return copy.repayFeeDirectionLabel
  }
  if (borrowed !== '0') {
    return copy.borrowDirectionLabel
  }
  if (repaid !== '0') {
    return copy.repayDirectionLabel
  }
  if (fee !== '0') {
    return copy.feeDirectionLabel
  }
  return copy.unknownDirectionLabel
}

function formatFeeDisplay(premium?: string, fee?: string) {
  const premiumValue = normalizeAmount(premium)
  const feeValue = normalizeAmount(fee)
  if (premiumValue === '0' && feeValue === '0') {
    return '0'
  }
  if (premiumValue !== '0' && feeValue !== '0') {
    return `${premiumValue} / ${feeValue}`
  }
  return premiumValue !== '0' ? premiumValue : feeValue
}

function formatUnixTimestamp(timestamp: number | undefined, copy: DetailCopy) {
  if (!timestamp) {
    return copy.notAvailable
  }
  return new Date(timestamp * 1000).toLocaleString(copy.isEnglish ? 'en-US' : 'zh-CN')
}

function formatGasPrice(value: string | undefined, copy: DetailCopy) {
  if (!value) {
    return copy.notAvailable
  }
  return `${formatUnits(value, 9, 4)} Gwei`
}

function formatTransactionFee(value: string | undefined, copy: DetailCopy) {
  if (!value) {
    return copy.notAvailable
  }
  return `${formatUnits(value, 18, 6)} ETH`
}

function formatTraceStatus(status: string, copy: DetailCopy) {
  switch (status) {
    case 'available':
      return copy.traceAvailableLabel
    case 'error':
      return copy.traceErrorLabel
    default:
      return copy.traceUnavailableLabel
  }
}

function formatTransactionStatus(status: number | undefined, copy: DetailCopy) {
  if (status === undefined) {
    return copy.notAvailable
  }
  if (status === 1) {
    return copy.transactionSuccessLabel
  }
  if (status === 0) {
    return copy.transactionFailedLabel
  }
  return String(status)
}

const KNOWN_ASSET_LABELS: Record<string, string> = {
  '0x514910771af9ca656af840dff83e8264ecf986ca': 'LINK',
  '0xc02aa39b223fe8d0a0e5c4f27ead9083c756cc2': 'WETH',
  '0x5e8c8a7243651db1384c0ddfdbe39761e8e7e51a': 'aEthLINK',
  '0x4d5f47fa6a74757f35c14fd3a6ef8e3c9bc514e8': 'aEthWETH',
  '0x7effd7b47bfd17e52fb7559d3f924201b9dbff3d': 'varDebtLINK',
}

const KNOWN_ADDRESS_LABELS: Record<string, string> = {
  '0x87870bca3f3fd6335c3f4ce8392d69350b4fa4e2': 'Aave Pool V3',
  '0xadc0a53095a0af87f3aa29fe0715b5c28016364e': 'Aave Swap Collateral Adapter V3',
  '0x5d4f3c6fa16908609bac31ff148bd002aa6b8c83': 'Uniswap V3: LINK 2',
  '0x6a000f20005980200259b80c5102003040001068': 'ParaSwap Augustus V6.2',
  '0x4d5f47fa6a74757f35c14fd3a6ef8e3c9bc514e8': 'aEthWETH',
  '0x5e8c8a7243651db1384c0ddfdbe39761e8e7e51a': 'aEthLINK',
}

function assetDisplayLabel(address?: string) {
  if (!address) {
    return 'N/A'
  }
  return KNOWN_ASSET_LABELS[address.toLowerCase()] ?? shortenHash(address)
}

function normalizeAmount(value?: string) {
  if (!value || value.trim() === '') {
    return '0'
  }
  return value
}

function shortenHash(value: string) {
  if (value.length <= 18) {
    return value
  }
  return `${value.slice(0, 10)}...${value.slice(-6)}`
}

function uniqueNonEmpty(values: Array<string | undefined>) {
  const seen = new Set<string>()
  return values.filter((value): value is string => {
    const normalized = (value ?? '').trim()
    if (!normalized || seen.has(normalized)) {
      return false
    }
    seen.add(normalized)
    return true
  })
}

function firstNonEmpty(...values: Array<string | undefined>) {
  return values.find((value) => Boolean((value ?? '').trim()))
}

function getRolePriority(roleCode: string) {
  switch (roleCode) {
    case 'initiator':
      return 0
    case 'provider':
      return 1
    case 'callback_target':
      return 2
    case 'receiver':
      return 3
    case 'repayment_target':
      return 4
    case 'pair':
      return 5
    case 'factory':
      return 6
    case 'on_behalf_of':
      return 7
    default:
      return 99
  }
}

function getRolePriorityByLabel(roleLabel: string, copy: DetailCopy) {
  if (roleLabel === copy.providerPoolLabel) {
    return 1
  }
  if (roleLabel === copy.receiverCallbackNodeLabel) {
    return 2
  }
  if (roleLabel === copy.repaymentTargetNodeLabel) {
    return 4
  }
  return 99
}

function formatUnits(value: string, decimals: number, precision: number) {
  try {
    const bigValue = BigInt(value)
    const negative = bigValue < 0n
    const base = 10n ** BigInt(decimals)
    const absolute = negative ? -bigValue : bigValue
    const whole = absolute / base
    const fraction = absolute % base
    if (fraction === 0n) {
      return `${negative ? '-' : ''}${whole.toString()}`
    }
    const padded = fraction.toString().padStart(decimals, '0').slice(0, precision).replace(/0+$/, '')
    return `${negative ? '-' : ''}${whole.toString()}${padded ? `.${padded}` : ''}`
  } catch {
    return value
  }
}

function getDetailCopy(isEnglish: boolean) {
  return {
    isEnglish,
    pageKicker: isEnglish ? 'Flash Loan Forensics' : '闪电贷识别取证页',
    basicInformationTitle: 'Basic Information / 基本信息',
    basicInformationCopy: isEnglish
      ? 'Confirm what this transaction is, which protocol entrypoint matched, and which execution context the scanner is relying on.'
      : '用于确认这笔交易的识别结果、协议入口与执行上下文。',
    balanceChangesTitle: 'Balance Changes / 资产变化',
    balanceChangesCopy: isEnglish
      ? 'Focus only on borrow, repay, fee, and strict asset records so the flash-loan loop is explicit.'
      : '用于查看关键地址在各资产上的借入、归还与费用变化。',
    fundFlowTitle: 'Fund Flow / 资产流向图',
    fundFlowCopy: isEnglish
      ? 'Show how borrowed assets move across the provider, callback path, intermediary hops, and repayment target.'
      : '用于说明借入资产如何在关键参与方之间流动，并最终形成归还闭环。',
    invocationFlowTitle: 'Invocation Flow / 调用调用树',
    invocationFlowCopy: isEnglish
      ? 'Show how internal calls expand from protocol entry into callback execution, repayment, settlement, and error branches.'
      : '用于查看内部调用如何从协议入口展开到回调与归还路径。',
    rawEvidenceTitle: 'Raw Evidence / 原始证据',
    rawEvidenceCopy: isEnglish
      ? 'Keep browser-style fields and full raw call data folded out of the main reading path.'
      : '用于在追问时回看原始协议字段、完整调用链与浏览器式底稿。',
    scanVerdictLabel: isEnglish ? 'Scan Verdict' : '扫描结论',
    identificationTypeLabel: isEnglish ? 'Identification Type' : '识别类型',
    identificationTypeValue: 'Flash Loan Interaction',
    hitProtocolsLabel: isEnglish ? 'Matched Protocols' : '命中协议',
    protocolEntrypointLabel: isEnglish ? 'Protocol Entrypoint' : '协议入口',
    transactionHashLabel: isEnglish ? 'Transaction Hash' : '交易哈希',
    blockLabel: isEnglish ? 'Block' : '区块号',
    timeLabel: isEnglish ? 'Time' : '时间',
    senderLabel: isEnglish ? 'Sender' : '发起地址',
    receiverCallbackLabel: isEnglish ? 'Receiver / Callback' : '接收 / 回调地址',
    positionInBlockLabel: isEnglish ? 'Position in Block' : '区块内位置',
    gasUsedLabel: isEnglish ? 'Gas Used' : 'Gas Used',
    gasPriceLabel: isEnglish ? 'Gas Price' : 'Gas Price',
    transactionFeeLabel: isEnglish ? 'Transaction Fee' : 'Transaction Fee',
    traceStatusLabel: isEnglish ? 'Trace Status' : 'Trace 状态',
    internalFramesLabel: isEnglish ? 'Internal Frames' : '内部调用帧数',
    addressLabel: isEnglish ? 'Address' : '地址',
    addressRoleLabel: isEnglish ? 'Address Role' : '地址角色',
    assetLabel: isEnglish ? 'Asset' : '资产',
    assetRecordTypeLabel: isEnglish ? 'Asset Record Type' : '资产记录类型',
    borrowedLabel: isEnglish ? 'Borrowed' : '借入数量',
    repaidLabel: isEnglish ? 'Repaid' : '归还数量',
    premiumFeeLabel: 'premium / fee',
    directionLabel: isEnglish ? 'Direction' : '变化方向',
    strictRecordLabel: isEnglish ? 'Strict Record' : '是否为严格资产记录',
    strictYes: isEnglish ? 'Strict' : '严格',
    strictNo: isEnglish ? 'Normal' : '普通',
    borrowDirectionLabel: isEnglish ? 'Borrow' : '借入',
    repayDirectionLabel: isEnglish ? 'Repay' : '归还',
    swapDirectionLabel: isEnglish ? 'Swap' : '交换',
    topupDirectionLabel: isEnglish ? 'Top-up' : '补足',
    feeDirectionLabel: isEnglish ? 'Fee' : '费用',
    borrowRepayDirectionLabel: isEnglish ? 'Borrow -> Repay' : '借入 -> 归还',
    repayFeeDirectionLabel: isEnglish ? 'Repay + Fee' : '归还 + 费用',
    unknownDirectionLabel: isEnglish ? 'Observed Change' : '关键变化',
    borrowedAssetLabel: isEnglish ? 'Borrowed Asset' : '借入资产',
    repaymentAssetLabel: isEnglish ? 'Repayment Asset' : '归还资产',
    feeAssetLabel: isEnglish ? 'Fee Record' : '手续费记录',
    premiumAssetLabel: isEnglish ? 'Premium Record' : '溢价记录',
    collateralAssetLabel: isEnglish ? 'Collateral Asset' : '抵押资产',
    debtAssetLabel: isEnglish ? 'Debt Record' : '债务记录',
    providerPoolLabel: isEnglish ? 'Provider / Pool' : 'Provider / Pool',
    receiverCallbackNodeLabel: isEnglish ? 'Receiver / Callback' : 'Receiver / Callback',
    intermediaryNodeLabel: isEnglish ? 'Intermediary Protocol / Pair / Pool' : '中间协议 / Pair / Pool',
    repaymentTargetNodeLabel: isEnglish ? 'Repayment Target' : '归还目标',
    fundFlowEmpty: isEnglish ? 'No fund-flow path is available for this transaction.' : '当前没有可用于展示的资产流向路径。',
    invocationFlowEmpty: isEnglish ? 'No invocation trace is available for this transaction.' : '当前没有可用于展示的内部调用树。',
    rawProtocolFieldsTitle: isEnglish ? 'Raw Protocol Interaction Fields' : '原始协议交互字段',
    rawProtocolFieldsCopy: isEnglish
      ? 'Keep the interaction-level scanner fields and leg tables here for deeper follow-up.'
      : '保留协议交互级别的扫描字段和资产记录底稿，供继续追问时查看。',
    inputDataTitle: 'Input Data',
    inputDataCopy: isEnglish ? 'Keep raw calldata folded by default.' : 'Input Data 作为原始字段折叠保留。',
    browserFieldsTitle: isEnglish ? 'Browser-style Raw Fields' : '浏览器式原始字段',
    browserFieldsCopy: isEnglish
      ? 'Fallback browser fields stay here so they do not compete with the forensic storyline.'
      : '把浏览器式原始字段下沉到这里，避免它们抢占取证主线。',
    otherInformationTitle: isEnglish ? 'Other Information' : '其他信息',
    transactionStatusLabel: isEnglish ? 'Transaction Status' : '交易状态',
    transactionToLabel: isEnglish ? 'Transaction To' : '交易接收地址',
    chainLabel: isEnglish ? 'Chain' : '链',
    protocolLabel: isEnglish ? 'Protocol' : '协议',
    methodSelectorLabel: isEnglish ? 'Method Selector' : '方法选择器',
    callbackSeenLabel: isEnglish ? 'Callback Seen' : '回调命中',
    repaymentSeenLabel: isEnglish ? 'Repayment Seen' : '归还命中',
    settlementSeenLabel: isEnglish ? 'Settlement Seen' : '结算命中',
    debtOpeningLabel: isEnglish ? 'Debt Opening' : '存在开债',
    exclusionReasonLabel: isEnglish ? 'Exclusion Reason' : '排除原因',
    verificationNotesLabel: isEnglish ? 'Verification Notes' : '验证说明',
    settlementModeLabel: isEnglish ? 'Settlement Mode' : '结算模式',
    traceAvailableLabel: isEnglish ? 'available' : '可用',
    traceUnavailableLabel: isEnglish ? 'unavailable' : '不可用',
    traceErrorLabel: isEnglish ? 'error' : '拉取失败',
    transactionSuccessLabel: isEnglish ? 'success' : '成功',
    transactionFailedLabel: isEnglish ? 'failed' : '失败',
    phaseEntryLabel: isEnglish ? 'Protocol Entry' : '协议入口',
    phaseCallbackLabel: isEnglish ? 'Callback Subtree' : '回调子树',
    phaseRepaymentLabel: isEnglish ? 'Repayment Path' : '归还路径',
    phaseSettlementLabel: isEnglish ? 'Settlement' : '结算节点',
    phaseErrorLabel: isEnglish ? 'Error' : '错误节点',
    phaseEventLabel: isEnglish ? 'Event' : '事件节点',
    phaseInternalLabel: isEnglish ? 'Internal Call' : '内部调用',
    callbackTag: isEnglish ? 'callback' : 'callback',
    repaymentTag: isEnglish ? 'repayment' : 'repayment',
    frameLabel: isEnglish ? 'frame' : 'frame',
    copy: isEnglish ? 'Copy' : '复制',
    copied: isEnglish ? 'Copied' : '已复制',
    noData: isEnglish ? 'No data available.' : '暂无可展示数据。',
    notAvailable: isEnglish ? 'N/A' : '暂无',
  }
}
