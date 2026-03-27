import { useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'
import { Link, useParams, useSearchParams } from 'react-router-dom'
import { LanguageToggle } from '../components/LanguageToggle'
import { TransactionAddressGraph, TransactionSequenceDiagram } from '../components/TransactionDiagrams'
import { useI18n } from '../lib/i18n'
import { getTransactionDetail } from '../lib/api'
import type { TransactionDetailResponse } from '../types'

type AddressSummary = {
  address: string
  roles: string[]
  roleCodes: string[]
}

type ConclusionView = {
  title: string
  summary: string
  reasons: ConclusionReason[]
}

type ConclusionTarget =
  | 'detail-summary'
  | 'trace-evidence'
  | 'verification-evidence'
  | 'interaction-details'
  | 'asset-flow-details'

type ConclusionReason = {
  id: string
  text: string
  target: ConclusionTarget
}

type TimelineEntry = NonNullable<TransactionDetailResponse['summary']>['timeline'][number]
type TraceSummary = NonNullable<TransactionDetailResponse['trace_summary']>
type TraceFrame = NonNullable<TraceSummary['frames']>[number]
type TraceEvidence = NonNullable<TraceSummary['interaction_evidence']>[number]

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

export function TransactionDetail() {
  const { t, boolLabel, protocolName } = useI18n()
  const { txHash = '' } = useParams()
  const [searchParams] = useSearchParams()
  const [data, setData] = useState<TransactionDetailResponse | null>(null)
  const [error, setError] = useState<string>()
  const [loading, setLoading] = useState(true)
  const chainId = Number(searchParams.get('chain_id') ?? '1')

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

  const addressSummaries = useMemo(() => (data ? resolveAddressSummaries(data, t) : []), [data, t])
  const conclusion = useMemo(() => (data ? resolveConclusion(data, t, protocolName) : null), [data, t, protocolName])
  const timeline = useMemo(() => (data ? resolveTimeline(data) : []), [data])
  const addressGraph = useMemo(() => (data ? resolveAddressGraph(data, addressSummaries, t) : null), [data, addressSummaries, t])
  const sequenceDiagram = useMemo(() => (
    data ? resolveSequenceDiagram(data, addressSummaries, t, protocolName) : null
  ), [data, addressSummaries, t, protocolName])
  const traceEvidence = useMemo(() => (data ? resolveTraceEvidence(data) : []), [data])
  const traceFrames = useMemo(() => (data?.trace_summary?.frames ?? []), [data])
  const traceFrameShellRef = useRef<HTMLDivElement | null>(null)
  const detailSummaryRef = useRef<HTMLElement | null>(null)
  const traceEvidenceRef = useRef<HTMLElement | null>(null)
  const verificationEvidenceRef = useRef<HTMLElement | null>(null)
  const interactionDetailsRef = useRef<HTMLElement | null>(null)
  const assetFlowDetailsRef = useRef<HTMLDivElement | null>(null)
  const highlightTimerRef = useRef<number | null>(null)
  const [highlightTarget, setHighlightTarget] = useState<ConclusionTarget | null>(null)
  const [hoveredTraceValue, setHoveredTraceValue] = useState<{ value: string; x: number; y: number }>()

  const updateHoveredTraceValue = (
    event: React.MouseEvent<HTMLElement>,
    value: string,
  ) => {
    const shell = traceFrameShellRef.current
    if (!shell) {
      return
    }
    const rect = shell.getBoundingClientRect()
    setHoveredTraceValue({
      value,
      x: event.clientX - rect.left,
      y: event.clientY - rect.top,
    })
  }

  useEffect(() => () => {
    if (highlightTimerRef.current !== null) {
      window.clearTimeout(highlightTimerRef.current)
    }
  }, [])

  const scrollToConclusionTarget = (target: ConclusionTarget) => {
    const targetMap: Record<ConclusionTarget, HTMLElement | null> = {
      'detail-summary': detailSummaryRef.current,
      'trace-evidence': traceEvidenceRef.current,
      'verification-evidence': verificationEvidenceRef.current,
      'interaction-details': interactionDetailsRef.current,
      'asset-flow-details': assetFlowDetailsRef.current,
    }
    const element = targetMap[target]
    if (!element) {
      return
    }
    element.scrollIntoView({ behavior: 'smooth', block: 'start' })
    setHighlightTarget(target)
    if (highlightTimerRef.current !== null) {
      window.clearTimeout(highlightTimerRef.current)
    }
    highlightTimerRef.current = window.setTimeout(() => {
      setHighlightTarget((current) => (current === target ? null : current))
      highlightTimerRef.current = null
    }, 1800)
  }

  return (
    <main className="page-shell detail-shell">
      <div className="detail-topbar">
        <Link to="/">← {t.backToConsole}</Link>
        <LanguageToggle />
      </div>

      {loading ? <section className="panel">{t.loadingTransactionDetail}</section> : null}
      {error ? <section className="panel error-text">{error}</section> : null}

      {data ? (
        <>
          <section
            ref={detailSummaryRef}
            className={`panel detail-header detail-anchor-section ${highlightTarget === 'detail-summary' ? 'detail-anchor-highlight' : ''}`}
          >
            <p className="eyebrow">{t.protocolEvidence}</p>
            <AutoFitHashTitle value={data.tx_hash} />
            <div className="detail-meta">
              <span>{t.block} {data.block_number}</span>
              <span>{t.candidate} {boolLabel(data.candidate)}</span>
              <span>{t.verified} {boolLabel(data.verified)}</span>
              <span>{t.strict} {boolLabel(data.strict)}</span>
            </div>
            <p className="panel-copy">{t.protocolEvidenceCopy}</p>

            <div className="detail-summary-grid">
              <SummaryMetric label={t.protocolCountLabel} value={String(data.protocol_count)} />
              <SummaryMetric label={t.interactionCountLabel} value={String(data.interaction_count)} />
              <SummaryMetric label={t.strictInteractionCountLabel} value={String(data.strict_interaction_count)} />
              <SummaryMetric label={t.finalVerdictLabel} value={buildVerdictLabel(data, t)} />
            </div>
            {conclusion ? (
              <div className="detail-inline-conclusion">
                <article className="conclusion-summary-card">
                  <span>{t.detectionConclusion}</span>
                  <p>{conclusion.summary}</p>
                </article>

                <article className="conclusion-reasons-card">
                  <span>{t.keyReasons}</span>
                  <ul className="reason-list">
                    {conclusion.reasons.length === 0 ? <li>{t.noReasons}</li> : null}
                    {conclusion.reasons.map((reason) => (
                      <li key={reason.id}>
                        <button
                          type="button"
                          className="reason-link"
                          onClick={() => scrollToConclusionTarget(reason.target)}
                        >
                          {reason.text}
                        </button>
                      </li>
                    ))}
                  </ul>
                </article>
              </div>
            ) : null}
          </section>

          {addressGraph ? (
            <section className="panel detail-subpanel">
              <div className="panel-header compact">
                <div>
                  <p className="eyebrow">{t.addressGraph}</p>
                  <h2>{t.addressGraph}</h2>
                  <p className="panel-copy">{t.addressGraphCopy}</p>
                </div>
              </div>

              <TransactionAddressGraph
                view={addressGraph}
                txHash={data.tx_hash}
                protocolsLabel={data.protocols.map((item) => protocolName(item)).join(', ') || t.notAvailable}
                identifiedFlowsLabel={t.identifiedFlows}
                borrowedLabel={t.borrowedAmount}
                repaidLabel={t.repaidAmount}
                shortenHash={shortenHash}
              />

              <div className="panel-header compact detail-section-header merged-address-header">
                <div>
                  <p className="eyebrow">{t.keyAddresses}</p>
                  <h3>{t.keyAddresses}</h3>
                  <p className="panel-copy">{t.keyAddressesCopy}</p>
                </div>
              </div>

              {addressSummaries.length === 0 ? (
                <p className="muted-text">{t.noAddressData}</p>
              ) : (
                <div className="address-grid merged-address-grid">
                  {addressSummaries.map((item) => (
                    <article key={item.address} className="address-card">
                      <div className="address-card-top">
                        <span className="address-role-label">{t.addressRoles}</span>
                        <div className="count-stack">
                          {item.roles.map((role) => (
                            <span key={`${item.address}-${role}`}>{role}</span>
                          ))}
                        </div>
                      </div>
                      <p className="mono-block">{item.address}</p>
                    </article>
                  ))}
                </div>
              )}
            </section>
          ) : null}

          {sequenceDiagram ? (
            <section className="panel detail-subpanel">
              <div className="panel-header compact">
                <div>
                  <p className="eyebrow">{t.sequenceDiagram}</p>
                  <h2>{t.sequenceDiagram}</h2>
                  <p className="panel-copy">{t.sequenceDiagramCopy}</p>
                </div>
              </div>

              <TransactionSequenceDiagram view={sequenceDiagram} />
            </section>
          ) : null}

          {data.trace_summary ? (
            <section className="detail-dual-grid trace-detail-grid">
              <section
                ref={traceEvidenceRef}
                className={`panel detail-subpanel trace-call-chain-panel detail-anchor-section ${highlightTarget === 'trace-evidence' ? 'detail-anchor-highlight' : ''}`}
              >
                <div className="panel-header compact">
                  <div>
                    <p className="eyebrow">{t.traceEvidence}</p>
                    <div className="trace-title-row">
                      <h2>{t.traceEvidence}</h2>
                      {data.trace_summary.status !== 'available' ? (
                        <strong className={`trace-status-badge trace-status-${data.trace_summary.status}`}>
                          {formatTraceStatus(data.trace_summary.status, t)}
                        </strong>
                      ) : null}
                    </div>
                    <p className="panel-copy">{t.traceEvidenceCopy}</p>
                  </div>
                </div>

                {data.trace_summary.status !== 'available' ? (
                  <div className="trace-error-box">
                    <p className="trace-error-summary">{formatTraceErrorSummary(data.trace_summary, t)}</p>
                    {data.trace_summary.error ? (
                      <p className="trace-error-raw">{data.trace_summary.error}</p>
                    ) : null}
                  </div>
                ) : (
                  <div className="evidence-list">
                    {traceEvidence.map((item) => (
                      <article key={item.interaction_id} className="evidence-card">
                        <div className="panel-header compact">
                          <div>
                            <p className="eyebrow">{protocolName(item.protocol)}</p>
                            <h3>{item.entrypoint}</h3>
                          </div>
                          <div className="count-stack">
                            <span>{mapVerdictToLabel(item.verdict, t)}</span>
                          </div>
                        </div>

                        <div className="evidence-grid">
                          <EvidenceItem label={t.callbackSeenLabel} passed={item.callback_seen} yesLabel={t.yes} noLabel={t.no} />
                          <EvidenceItem label={t.settlementSeenLabel} passed={item.settlement_seen} yesLabel={t.yes} noLabel={t.no} />
                          <EvidenceItem label={t.repaymentSeenLabel} passed={item.repayment_seen} yesLabel={t.yes} noLabel={t.no} />
                          <EvidenceItem label={t.debtOpeningLabel} passed={item.contains_debt_opening} yesLabel={t.yes} noLabel={t.no} negative />
                        </div>

                        <div className="detail-inline-list trace-meta-list">
                          <div>
                            <strong>{t.callbackFrames}</strong>
                            <p>{item.callback_frame_ids?.length ?? 0}</p>
                          </div>
                          <div>
                            <strong>{t.callbackSubtree}</strong>
                            <p>{item.callback_subtree_ids?.length ?? 0}</p>
                          </div>
                          <div>
                            <strong>{t.repaymentFrames}</strong>
                            <p>{item.repayment_frame_ids?.length ?? 0}</p>
                          </div>
                          <div>
                            <strong>{t.exclusionReason}</strong>
                            <p>{item.exclusion_reason || t.notAvailable}</p>
                          </div>
                        </div>
                      </article>
                    ))}
                  </div>
                )}
              </section>

              <section className="panel detail-subpanel">
                <div className="panel-header compact">
                  <div>
                    <p className="eyebrow">{t.callChain}</p>
                    <h2>{t.callChain}</h2>
                    <p className="panel-copy">{t.callChainCopy}</p>
                  </div>
                </div>

                {traceFrames.length === 0 ? (
                  <div className="trace-error-box">
                    <p className="trace-error-summary">{formatTraceErrorSummary(data.trace_summary, t, t.noTraceFrames)}</p>
                    {data.trace_summary.error ? (
                      <p className="trace-error-raw">{data.trace_summary.error}</p>
                    ) : null}
                  </div>
                ) : (
                  <div ref={traceFrameShellRef} className="trace-frame-list trace-frame-scroll">
                    {traceFrames.map((frame) => (
                      (() => {
                        const traceTags = [
                          frame.token_action,
                          frame.tags?.includes('callback_path') ? t.callbackSubtree : '',
                          frame.tags?.includes('repayment_path') ? t.repaymentFrames : '',
                          frame.error ? t.traceError : '',
                        ].filter(Boolean)

                        return (
                          <article
                            key={frame.id}
                            className={`trace-frame-card trace-depth-${Math.min(frame.depth, 4)} ${frame.tags?.includes('callback_path') ? 'trace-callback' : ''} ${frame.tags?.includes('repayment_path') ? 'trace-repayment' : ''} ${frame.tags?.includes('error') ? 'trace-error' : ''}`}
                          >
                            <div className="trace-frame-head">
                              <span>{frame.call_index + 1}</span>
                              <strong>{frame.token_action || frame.method_selector || frame.type}</strong>
                            </div>
                            <p className="mono-block compact">
                              <span
                                className="address-hover-value"
                                onMouseEnter={(event) => updateHoveredTraceValue(event, frame.from)}
                                onMouseMove={(event) => updateHoveredTraceValue(event, frame.from)}
                                onMouseLeave={() => setHoveredTraceValue(undefined)}
                              >
                                {shortenHash(frame.from)}
                              </span>
                              {' '}→{' '}
                              <span
                                className="address-hover-value"
                                onMouseEnter={(event) => updateHoveredTraceValue(event, frame.to)}
                                onMouseMove={(event) => updateHoveredTraceValue(event, frame.to)}
                                onMouseLeave={() => setHoveredTraceValue(undefined)}
                              >
                                {shortenHash(frame.to)}
                              </span>
                            </p>
                            {traceTags.length > 0 ? (
                              <div className="count-stack trace-tag-stack">
                                {traceTags.map((tag) => (
                                  <span key={`${frame.id}-${tag}`}>{tag}</span>
                                ))}
                              </div>
                            ) : null}
                            {buildTraceFrameCopy(frame) ? (
                              <p className="trace-frame-copy">{buildTraceFrameCopy(frame)}</p>
                            ) : null}
                          </article>
                        )
                      })()
                    ))}
                    {hoveredTraceValue ? (
                      <div
                        className="address-graph-tooltip"
                        style={{
                          left: hoveredTraceValue.x,
                          top: hoveredTraceValue.y,
                        }}
                      >
                        {hoveredTraceValue.value}
                      </div>
                    ) : null}
                  </div>
                )}
              </section>
            </section>
          ) : null}

          <section className="panel detail-subpanel">
            <div className="panel-header compact">
              <div>
                <p className="eyebrow">{t.processTimeline}</p>
                <h2>{t.processTimeline}</h2>
                <p className="panel-copy">{t.processTimelineCopy}</p>
              </div>
            </div>

            <div className="timeline-list">
              {timeline.map((step) => (
                <article key={`${step.ordinal}-${step.kind}-${step.entrypoint ?? step.asset_address ?? step.protocol ?? ''}`} className="timeline-item">
                  <div className="timeline-marker">{step.ordinal}</div>
                  <div className="timeline-content">
                    <div className="timeline-topline">
                      <strong>{formatTimelineTitle(step, t)}</strong>
                      {step.protocol ? <span>{protocolName(step.protocol)}</span> : null}
                    </div>
                    <p className="timeline-copy">{formatTimelineDetail(step, t, protocolName)}</p>
                  </div>
                </article>
              ))}
            </div>
          </section>

          <section
            ref={verificationEvidenceRef}
            className={`panel detail-subpanel detail-anchor-section ${highlightTarget === 'verification-evidence' ? 'detail-anchor-highlight' : ''}`}
          >
              <div className="panel-header compact">
                <div>
                  <p className="eyebrow">{t.verificationEvidence}</p>
                  <h2>{t.verificationEvidence}</h2>
                  <p className="panel-copy">{t.verificationEvidenceCopy}</p>
                </div>
              </div>

              <div className="evidence-list">
                {data.interactions.map((interaction) => (
                  <article key={`${interaction.interaction_id}-evidence`} className="evidence-card">
                    <div className="panel-header compact">
                      <div>
                        <p className="eyebrow">{protocolName(interaction.protocol)}</p>
                        <h3>{interaction.entrypoint}</h3>
                      </div>
                      <div className="count-stack">
                        <span>{interaction.strict ? t.strict : t.nonStrict}</span>
                      </div>
                    </div>

                    <div className="evidence-grid">
                      <EvidenceItem label={t.callbackSeenLabel} passed={interaction.callback_seen} yesLabel={t.yes} noLabel={t.no} />
                      <EvidenceItem label={t.settlementSeenLabel} passed={interaction.settlement_seen} yesLabel={t.yes} noLabel={t.no} />
                      <EvidenceItem label={t.repaymentSeenLabel} passed={interaction.repayment_seen} yesLabel={t.yes} noLabel={t.no} />
                      <EvidenceItem label={t.debtOpeningLabel} passed={interaction.contains_debt_opening} yesLabel={t.yes} noLabel={t.no} negative />
                    </div>

                    <div className="detail-inline-list">
                      <div>
                        <strong>{t.methodSelector}</strong>
                        <p className="mono-block compact">{interaction.raw_method_selector || t.notAvailable}</p>
                      </div>
                      <div>
                        <strong>{t.exclusionReason}</strong>
                        <p>{interaction.exclusion_reason || t.notAvailable}</p>
                      </div>
                      <div>
                        <strong>{t.verificationNotes}</strong>
                        <p>{interaction.verification_notes || t.notAvailable}</p>
                      </div>
                    </div>
                  </article>
                ))}
              </div>
          </section>

          <section
            ref={interactionDetailsRef}
            className={`panel detail-subpanel detail-anchor-section ${highlightTarget === 'interaction-details' ? 'detail-anchor-highlight' : ''}`}
          >
            <div className="panel-header compact">
              <div>
                <p className="eyebrow">{t.interactionEvidence}</p>
                <h2>{t.interactionEvidence}</h2>
                <p className="panel-copy">{t.interactionEvidenceCopy}</p>
              </div>
            </div>
          </section>

          {data.interactions.map((interaction, index) => (
            <section
              key={interaction.interaction_id}
              className="panel interaction-panel"
            >
              <div className="panel-header compact">
                <div>
                  <p className="eyebrow">{protocolName(interaction.protocol)}</p>
                  <h2>{interaction.entrypoint}</h2>
                  <p className="panel-copy">{t.interactionSummary}</p>
                </div>
                <div className="count-stack">
                  <span>{t.candidateLevel} {interaction.candidate_level}</span>
                  <span>{interaction.verified ? t.verified : t.unverified}</span>
                  <span>{interaction.strict ? t.strict : t.nonStrict}</span>
                </div>
              </div>

              <div className="detail-grid">
                <div>
                  <strong>{t.provider}</strong>
                  <p>{interaction.provider_address || t.notAvailable}</p>
                </div>
                <div>
                  <strong>{t.receiver}</strong>
                  <p>{interaction.receiver_address || t.notAvailable}</p>
                </div>
                <div>
                  <strong>{t.callbackTarget}</strong>
                  <p>{interaction.callback_target || t.notAvailable}</p>
                </div>
                <div>
                  <strong>{t.verificationNotes}</strong>
                  <p>{interaction.verification_notes || t.notAvailable}</p>
                </div>
              </div>

              <div
                ref={index === 0 ? assetFlowDetailsRef : undefined}
                className={`panel-header compact detail-section-header detail-anchor-section ${highlightTarget === 'asset-flow-details' && index === 0 ? 'detail-anchor-highlight' : ''}`}
              >
                <div>
                  <p className="eyebrow">{t.assetFlow}</p>
                  <h3>{t.assetFlow}</h3>
                </div>
              </div>

              <div className="table-shell detail-table-shell">
                <table className="detail-data-table">
                  <thead>
                    <tr>
                      <th>{t.leg}</th>
                      <th>{t.asset}</th>
                      <th>{t.role}</th>
                      <th>{t.borrowedAmount}</th>
                      <th>{t.repaidAmount}</th>
                      <th>{t.premiumFee}</th>
                      <th>{t.settlementModeLabel}</th>
                      <th>{t.strictEvent}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {interaction.legs.map((leg) => (
                      <tr key={`${interaction.interaction_id}-${leg.leg_index}`}>
                        <td>{leg.leg_index}</td>
                        <td className="detail-asset-cell">
                          <span className="mono-inline">{shortenHash(leg.asset_address)}</span>
                        </td>
                        <td>{formatAssetRole(leg.asset_role, t)}</td>
                        <td className="mono-inline">{leg.amount_borrowed || '0'}</td>
                        <td className="mono-inline">{leg.amount_repaid || '0'}</td>
                        <td className="mono-inline">{`${leg.premium_amount || '0'} / ${leg.fee_amount || '0'}`}</td>
                        <td>{leg.settlement_mode || t.notAvailable}</td>
                        <td>{`${leg.strict_leg ? t.strictLeg : t.nonStrictLeg} · ${leg.event_seen ? t.eventSeen : t.eventMissing}`}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </section>
          ))}
        </>
      ) : null}
    </main>
  )
}

function SummaryMetric({ label, value }: { label: string; value: string }) {
  return (
    <article className="detail-summary-card">
      <span>{label}</span>
      <strong>{value}</strong>
    </article>
  )
}

function EvidenceItem({
  label,
  passed,
  yesLabel,
  noLabel,
  negative = false,
}: {
  label: string
  passed: boolean
  yesLabel: string
  noLabel: string
  negative?: boolean
}) {
  const positiveState = negative ? !passed : passed
  return (
    <div className={`evidence-item ${positiveState ? 'pass' : 'fail'}`}>
      <span>{label}</span>
      <strong>{passed ? yesLabel : noLabel}</strong>
    </div>
  )
}

function buildVerdictLabel(data: TransactionDetailResponse, t: ReturnType<typeof useI18n>['t']) {
  if (data.summary?.conclusion?.verdict) {
    return mapVerdictToLabel(data.summary.conclusion.verdict, t)
  }
  if (data.strict) {
    return t.strict
  }
  if (data.verified) {
    return t.verified
  }
  return t.candidate
}

function resolveConclusion(
  data: TransactionDetailResponse,
  t: ReturnType<typeof useI18n>['t'],
  protocolName: (protocol: string) => string,
): ConclusionView {
  const buildReason = (
    id: string,
    text: string,
    target: ConclusionTarget,
  ): ConclusionReason => ({
    id,
    text,
    target,
  })
  const summaryData = data.summary?.conclusion
  const protocols = (summaryData?.protocols ?? data.protocols).map((item) => protocolName(item)).join(', ')
  const callbackCount = summaryData?.callback_seen_count ?? data.interactions.filter((item) => item.callback_seen).length
  const repaymentCount = summaryData?.repayment_seen_count ?? data.interactions.filter((item) => item.repayment_seen).length
  const settlementCount = summaryData?.settlement_seen_count ?? data.interactions.filter((item) => item.settlement_seen).length
  const debtOpenings = summaryData?.debt_opening_count ?? data.interactions.filter((item) => item.contains_debt_opening).length
  const strictLegCount = summaryData?.strict_leg_count ?? data.interactions.reduce((sum, interaction) => (
    sum + interaction.legs.filter((leg) => leg.strict_leg).length
  ), 0)
  const exclusionReasons = summaryData?.exclusion_reasons ?? data.interactions
    .map((interaction) => interaction.exclusion_reason.trim())
    .filter(Boolean)

  const title = summaryData ? mapVerdictToLabel(summaryData.verdict, t) : buildVerdictLabel(data, t)
  let summary = ''
  if ((summaryData?.verdict ?? '').toLowerCase() === 'strict' || data.strict) {
    summary = `命中 ${protocols}，共识别 ${data.strict_interaction_count} 条严格交互，回调与归还证据完整，扫描器最终判定为严格通过。`
  } else if ((summaryData?.verdict ?? '').toLowerCase() === 'verified' || data.verified) {
    summary = `命中 ${protocols}，交易已通过验证，但严格证据仍不完整，因此结论停留在验证通过。`
  } else {
    summary = `命中 ${protocols} 的初筛信号，但验证证据不足，当前仅保留为初筛命中。`
  }

  if (t.yes === 'Yes') {
    if ((summaryData?.verdict ?? '').toLowerCase() === 'strict' || data.strict) {
      summary = `Matched ${protocols} with ${data.strict_interaction_count} strict interactions. Callback and repayment evidence are complete, so the scanner marks this transaction as strict.`
    } else if ((summaryData?.verdict ?? '').toLowerCase() === 'verified' || data.verified) {
      summary = `Matched ${protocols} and passed verification, but the strict evidence remains incomplete, so the final verdict stays at verified.`
    } else {
      summary = `Matched the initial signals on ${protocols}, but verification evidence is still insufficient, so the transaction remains an initial hit only.`
    }
  }

  const totalInteractions = summaryData?.interaction_count ?? data.interaction_count
  const totalProtocols = summaryData?.protocols.length ?? data.protocol_count

  const reasons = t.yes === 'Yes'
    ? [
      buildReason('protocol-count', `Protocols involved: ${totalProtocols}`, 'detail-summary'),
      buildReason('interaction-count', `Protocol interactions found: ${totalInteractions}`, 'interaction-details'),
      buildReason('callback-evidence', `Callback-path evidence: ${callbackCount} / ${totalInteractions} interactions`, 'trace-evidence'),
      buildReason('repayment-evidence', `Repayment-path evidence: ${repaymentCount} / ${totalInteractions} interactions`, 'trace-evidence'),
      buildReason('settlement-evidence', `Settlement-complete evidence: ${settlementCount} / ${totalInteractions} interactions`, 'verification-evidence'),
      buildReason('strict-legs', `Strict asset records: ${strictLegCount}`, 'asset-flow-details'),
    ]
    : [
      buildReason('protocol-count', `涉及协议：${totalProtocols} 个`, 'detail-summary'),
      buildReason('interaction-count', `识别到的协议交互：${totalInteractions} 条`, 'interaction-details'),
      buildReason('callback-evidence', `回调路径证据：${callbackCount} / ${totalInteractions} 条交互命中`, 'trace-evidence'),
      buildReason('repayment-evidence', `归还路径证据：${repaymentCount} / ${totalInteractions} 条交互命中`, 'trace-evidence'),
      buildReason('settlement-evidence', `结算完成证据：${settlementCount} / ${totalInteractions} 条交互命中`, 'verification-evidence'),
      buildReason('strict-legs', `严格资产记录：${strictLegCount} 条`, 'asset-flow-details'),
    ]

  if (debtOpenings > 0) {
    reasons.push(buildReason(
      'debt-openings',
      t.yes === 'Yes' ? `Debt openings detected: ${debtOpenings}` : `开债信号：${debtOpenings} 条`,
      'verification-evidence',
    ))
  }
  exclusionReasons.forEach((reason) => {
    reasons.push(buildReason(
      `exclusion-${reason}`,
      t.yes === 'Yes' ? `Exclusion note: ${reason}` : `排除说明：${reason}`,
      'verification-evidence',
    ))
  })

  return {
    title,
    summary,
    reasons,
  }
}

function resolveAddressSummaries(
  data: TransactionDetailResponse,
  t: ReturnType<typeof useI18n>['t'],
): AddressSummary[] {
  if (data.summary?.addresses?.length) {
    return data.summary.addresses
      .map((item) => {
        const roleCodes = item.roles
          .slice()
          .sort((left, right) => getRolePriority(left) - getRolePriority(right))
        return {
          address: item.address,
          roleCodes,
          roles: roleCodes.map((role) => mapRoleCodeToLabel(role, t)),
        }
      })
      .sort(compareAddressSummaries)
  }
  return buildAddressSummariesFallback(data, t)
}

function buildAddressSummariesFallback(
  data: TransactionDetailResponse,
  t: ReturnType<typeof useI18n>['t'],
): AddressSummary[] {
  const roleMap = new Map<string, Set<string>>()

  const append = (address: string, role: string) => {
    const normalized = address.trim()
    if (!normalized) {
      return
    }
    if (!roleMap.has(normalized)) {
      roleMap.set(normalized, new Set<string>())
    }
    roleMap.get(normalized)?.add(role)
  }

  data.interactions.forEach((interaction) => {
    append(interaction.initiator, t.roleInitiator)
    append(interaction.on_behalf_of, t.roleOnBehalfOf)
    append(interaction.receiver_address, t.roleReceiver)
    append(interaction.callback_target, t.roleCallbackTarget)
    append(interaction.provider_address, t.roleProvider)
    append(interaction.factory_address, t.roleFactory)
    append(interaction.pair_address, t.rolePair)

    interaction.legs.forEach((leg) => {
      append(leg.repaid_to_address, t.roleRepaymentTarget)
    })
  })

  return Array.from(roleMap.entries())
    .map(([address, roles]) => ({
      address,
      roleCodes: Array.from(roles.values()).sort((left, right) => getRolePriority(left) - getRolePriority(right)),
      roles: Array.from(roles.values())
        .sort((left, right) => getRolePriority(left) - getRolePriority(right))
        .map((role) => mapRoleCodeToLabel(role, t)),
    }))
    .sort(compareAddressSummaries)
}

function resolveTimeline(data: TransactionDetailResponse) {
  if (data.summary?.timeline?.length) {
    return data.summary.timeline
  }

  const fallback: NonNullable<TransactionDetailResponse['summary']>['timeline'] = []
  let ordinal = 1
  data.interactions.forEach((interaction) => {
    fallback.push({
      ordinal,
      kind: 'entrypoint',
      protocol: interaction.protocol,
      entrypoint: interaction.entrypoint,
    })
    ordinal += 1

    interaction.legs.forEach((leg) => {
      fallback.push({
        ordinal,
        kind: 'asset_leg',
        protocol: interaction.protocol,
        asset_address: leg.asset_address,
        asset_role: leg.asset_role,
        amount_borrowed: leg.amount_borrowed,
        amount_repaid: leg.amount_repaid,
        strict: leg.strict_leg,
        event_seen: leg.event_seen,
      })
      ordinal += 1
    })

    fallback.push({
      ordinal,
      kind: 'evidence',
      protocol: interaction.protocol,
      strict: interaction.strict,
      callback_seen: interaction.callback_seen,
      settlement_seen: interaction.settlement_seen,
      repayment_seen: interaction.repayment_seen,
    })
    ordinal += 1
  })
  return fallback
}

function formatTimelineTitle(step: TimelineEntry, t: ReturnType<typeof useI18n>['t']) {
  switch (step.kind) {
    case 'entrypoint':
      return t.timelineEntrypoint
    case 'asset_leg':
      return t.timelineAssetLeg
    case 'evidence':
      return t.timelineEvidence
    default:
      return t.interactionSummary
  }
}

function formatTimelineDetail(
  step: TimelineEntry,
  t: ReturnType<typeof useI18n>['t'],
  protocolName: (protocol: string) => string,
) {
  const protocolLabel = step.protocol ? protocolName(step.protocol) : t.notAvailable

  switch (step.kind) {
    case 'entrypoint':
      return t.yes === 'Yes'
        ? `Entered ${protocolLabel} through ${step.entrypoint || t.notAvailable}.`
        : `通过 ${step.entrypoint || t.notAvailable} 进入 ${protocolLabel}。`
    case 'asset_leg':
      return t.yes === 'Yes'
        ? `Asset ${shortenHash(step.asset_address || '')} acts as ${formatAssetRole(step.asset_role, t)}. Borrowed ${step.amount_borrowed || '0'} and repaid ${step.amount_repaid || '0'}, marked as ${step.strict ? t.strictLeg : t.nonStrictLeg} with ${step.event_seen ? t.eventSeen : t.eventMissing}.`
        : `资产 ${shortenHash(step.asset_address || '')} 作为${formatAssetRole(step.asset_role, t)}，借出 ${step.amount_borrowed || '0'}，归还 ${step.amount_repaid || '0'}，判定为${step.strict ? t.strictLeg : t.nonStrictLeg}，并且${step.event_seen ? t.eventSeen : t.eventMissing}。`
    case 'evidence':
      return t.yes === 'Yes'
        ? `${t.callbackSeenLabel}: ${step.callback_seen ? t.yes : t.no}. ${t.settlementSeenLabel}: ${step.settlement_seen ? t.yes : t.no}. ${t.repaymentSeenLabel}: ${step.repayment_seen ? t.yes : t.no}. Final interaction state: ${step.strict ? t.strict : t.nonStrict}.`
        : `${t.callbackSeenLabel}：${step.callback_seen ? t.yes : t.no}；${t.settlementSeenLabel}：${step.settlement_seen ? t.yes : t.no}；${t.repaymentSeenLabel}：${step.repayment_seen ? t.yes : t.no}；该交互最终为${step.strict ? t.strict : t.nonStrict}。`
    default:
      return t.notAvailable
  }
}

function mapRoleCodeToLabel(code: string, t: ReturnType<typeof useI18n>['t']) {
  switch (code) {
    case 'initiator':
      return t.roleInitiator
    case 'on_behalf_of':
      return t.roleOnBehalfOf
    case 'receiver':
      return t.roleReceiver
    case 'callback_target':
      return t.roleCallbackTarget
    case 'provider':
      return t.roleProvider
    case 'factory':
      return t.roleFactory
    case 'pair':
      return t.rolePair
    case 'repayment_target':
      return t.roleRepaymentTarget
    case 'trace_source':
      return t.yes === 'Yes' ? 'trace source' : 'trace 源地址'
    case 'trace_target':
      return t.yes === 'Yes' ? 'trace target' : 'trace 目标地址'
    case 'trace_participant':
      return t.yes === 'Yes' ? 'trace participant' : 'trace 参与方'
    default:
      return code
  }
}

function formatAssetRole(role: string | undefined, t: ReturnType<typeof useI18n>['t']) {
  if (!role) {
    return t.notAvailable
  }
  switch (role) {
    case 'flash_loan_asset':
      return t.yes === 'Yes' ? 'flash loan asset' : '闪电贷资产'
    case 'repayment_asset':
      return t.yes === 'Yes' ? 'repayment asset' : '偿还资产'
    case 'fee_asset':
      return t.yes === 'Yes' ? 'fee asset' : '手续费资产'
    case 'premium_asset':
      return t.yes === 'Yes' ? 'premium asset' : '溢价资产'
    case 'collateral_asset':
      return t.yes === 'Yes' ? 'collateral asset' : '抵押资产'
    case 'debt_asset':
      return t.yes === 'Yes' ? 'debt asset' : '债务资产'
    default:
      return role.replace(/_/g, ' ')
  }
}

function mapVerdictToLabel(verdict: string, t: ReturnType<typeof useI18n>['t']) {
  switch (verdict) {
    case 'strict':
      return t.strict
    case 'verified':
      return t.verified
    default:
      return t.candidate
  }
}

function resolveAddressGraph(
  data: TransactionDetailResponse,
  addressSummaries: AddressSummary[],
  t: ReturnType<typeof useI18n>['t'],
): AddressGraphView {
  if (data.trace_summary?.status === 'available' && data.trace_summary.asset_flows?.length) {
    const flowSourceOnly = new Set<string>()
    const flowTargetOnly = new Set<string>()
    const flowAddresses = new Set<string>()
    data.trace_summary.asset_flows.forEach((flow) => {
      flowAddresses.add(flow.source)
      flowAddresses.add(flow.target)
      flowSourceOnly.add(flow.source)
      flowTargetOnly.add(flow.target)
    })

    const matchedMap = new Map(
      addressSummaries
        .filter((item) => flowAddresses.has(item.address))
        .map((item) => [item.address.toLowerCase(), item] as const),
    )

    const allAddresses = Array.from(flowAddresses.values())
      .map((address) => {
        const matched = matchedMap.get(address.toLowerCase())
        if (matched) {
          return matched
        }
        const roleCodes = flowSourceOnly.has(address) && !flowTargetOnly.has(address)
          ? ['trace_source']
          : flowTargetOnly.has(address) && !flowSourceOnly.has(address)
            ? ['trace_target']
            : ['trace_participant']
        return {
          address,
          roleCodes,
          roles: [mapRoleCodeToLabel(roleCodes[0], t)],
        }
      })
      .sort(compareAddressSummaries)

    const left = allAddresses.filter((item) => getGraphSide(item.roleCodes[0]) === 'left')
    const right = allAddresses.filter((item) => getGraphSide(item.roleCodes[0]) === 'right')

    return {
      left,
      right,
      flows: data.trace_summary.asset_flows.map((flow) => ({
        id: flow.frame_id,
        protocol: flow.action,
        assetAddress: flow.asset_address,
        amount: flow.amount,
        source: flow.source,
        target: flow.target,
      })),
    }
  }

  const left = addressSummaries.filter((item) => getGraphSide(item.roleCodes[0]) === 'left').slice(0, 4)
  const right = addressSummaries.filter((item) => getGraphSide(item.roleCodes[0]) === 'right').slice(0, 4)
  const seen = new Set<string>()
  const flows: AddressGraphView['flows'] = []

  data.interactions.forEach((interaction) => {
    interaction.legs.forEach((leg) => {
      const key = [
        interaction.protocol,
        leg.asset_address,
        leg.amount_borrowed || '0',
        leg.amount_repaid || '0',
      ].join(':')
      if (seen.has(key)) {
        return
      }
      seen.add(key)
      flows.push({
        id: key,
        protocol: interaction.protocol,
        assetAddress: leg.asset_address,
        amount: leg.amount_borrowed || leg.amount_repaid || '0',
        source: interaction.provider_address || interaction.pair_address || interaction.factory_address || data.tx_hash,
        target: interaction.receiver_address || interaction.callback_target || data.tx_hash,
      })
    })
  })

  return {
    left,
    right,
    flows,
  }
}

function resolveSequenceDiagram(
  data: TransactionDetailResponse,
  addressSummaries: AddressSummary[],
  t: ReturnType<typeof useI18n>['t'],
  protocolName: (protocol: string) => string,
): SequenceDiagramView {
  if (data.trace_summary?.status === 'available' && data.trace_summary.sequence?.length) {
    const laneMap = new Map<string, SequenceLane>()
    const findAddressSummary = (address: string) => addressSummaries.find((item) => item.address.toLowerCase() === address.toLowerCase())
    const pushLane = (address: string) => {
      if (!address || laneMap.has(address)) {
        return
      }
      const matched = findAddressSummary(address)
      laneMap.set(address, {
        id: address,
        label: matched?.roles[0] ?? t.protocol,
        caption: shortenHash(address),
      })
    }

    data.trace_summary.sequence.forEach((step) => {
      pushLane(step.from)
      pushLane(step.to)
    })
    const lanes = Array.from(laneMap.values()).slice(0, 6)
    const laneIDs = new Set(lanes.map((lane) => lane.id))
    const steps = data.trace_summary.sequence
      .filter((step) => laneIDs.has(step.from) && laneIDs.has(step.to))
      .slice(0, 18)
      .map((step): SequenceStep => ({
        id: step.frame_id,
        from: step.from,
        to: step.to,
        title: step.label || step.method_selector || step.token_action || 'call',
        detail: step.detail,
        tone: step.error ? 'danger' : step.token_action ? 'accent' : 'neutral',
      }))
    if (lanes.length > 0 && steps.length > 0) {
      return { lanes, steps }
    }
  }

  const addressByRole = new Map<string, AddressSummary>()
  addressSummaries.forEach((item) => {
    item.roleCodes.forEach((roleCode) => {
      if (!addressByRole.has(roleCode)) {
        addressByRole.set(roleCode, item)
      }
    })
  })

  const lanes: SequenceLane[] = []
  const pushLane = (id: string, label: string, caption: string) => {
    if (!lanes.some((lane) => lane.id === id)) {
      lanes.push({ id, label, caption })
    }
  }

  const initiator = addressByRole.get('initiator')
  const provider = addressByRole.get('provider')
  const receiver = addressByRole.get('receiver')
  const callbackTarget = addressByRole.get('callback_target')
  const repaymentTarget = addressByRole.get('repayment_target')

  if (initiator) {
    pushLane('initiator', initiator.roles[0] ?? t.roleInitiator, shortenHash(initiator.address))
  }
  pushLane(
    'protocol',
    t.protocol,
    data.protocols.map((item) => protocolName(item)).join(', ') || t.notAvailable,
  )
  if (provider && !lanes.some((lane) => lane.caption === shortenHash(provider.address))) {
    pushLane('provider', provider.roles[0] ?? t.roleProvider, shortenHash(provider.address))
  }
  if (receiver) {
    pushLane('receiver', receiver.roles[0] ?? t.roleReceiver, shortenHash(receiver.address))
  }
  if (callbackTarget && !lanes.some((lane) => lane.caption === shortenHash(callbackTarget.address))) {
    pushLane('callback_target', callbackTarget.roles[0] ?? t.roleCallbackTarget, shortenHash(callbackTarget.address))
  }
  if (repaymentTarget) {
    pushLane('repayment_target', repaymentTarget.roles[0] ?? t.roleRepaymentTarget, shortenHash(repaymentTarget.address))
  }

  const steps: SequenceStep[] = []
  const entrySource = initiator ? 'initiator' : 'protocol'
  const interactionTarget = provider ? 'provider' : 'protocol'

  data.interactions.forEach((interaction, interactionIndex) => {
    steps.push({
      id: `${interaction.interaction_id}-entry`,
      from: entrySource,
      to: interactionTarget,
      title: t.timelineEntrypoint,
      detail: t.yes === 'Yes'
        ? `${protocolName(interaction.protocol)} receives ${interaction.entrypoint}.`
        : `${protocolName(interaction.protocol)} 收到入口调用 ${interaction.entrypoint}。`,
      tone: 'neutral',
    })

    interaction.legs.forEach((leg) => {
      const borrowTarget = receiver ? 'receiver' : callbackTarget ? 'callback_target' : interactionTarget
      steps.push({
        id: `${interaction.interaction_id}-borrow-${leg.leg_index}`,
        from: interactionTarget,
        to: borrowTarget,
        title: t.borrowedAmount,
        detail: t.yes === 'Yes'
          ? `${shortenHash(leg.asset_address)} borrowed ${leg.amount_borrowed || '0'}.`
          : `${shortenHash(leg.asset_address)} 借出 ${leg.amount_borrowed || '0'}。`,
        tone: 'accent',
      })

      const hasRepayment = (leg.amount_repaid || '0') !== '0'
      if (hasRepayment) {
        steps.push({
          id: `${interaction.interaction_id}-repay-${leg.leg_index}`,
          from: callbackTarget ? 'callback_target' : receiver ? 'receiver' : interactionTarget,
          to: repaymentTarget ? 'repayment_target' : interactionTarget,
          title: t.repaidAmount,
          detail: t.yes === 'Yes'
            ? `${shortenHash(leg.asset_address)} repaid ${leg.amount_repaid || '0'}.`
            : `${shortenHash(leg.asset_address)} 归还 ${leg.amount_repaid || '0'}。`,
          tone: 'neutral',
        })
      }
    })

    steps.push({
      id: `${interaction.interaction_id}-evidence-${interactionIndex}`,
      from: interactionTarget,
      to: interactionTarget,
      title: t.timelineEvidence,
      detail: t.yes === 'Yes'
        ? `${t.callbackSeenLabel} ${interaction.callback_seen ? t.yes : t.no}, ${t.repaymentSeenLabel} ${interaction.repayment_seen ? t.yes : t.no}, ${t.settlementSeenLabel} ${interaction.settlement_seen ? t.yes : t.no}.`
        : `${t.callbackSeenLabel}${interaction.callback_seen ? t.yes : t.no}，${t.repaymentSeenLabel}${interaction.repayment_seen ? t.yes : t.no}，${t.settlementSeenLabel}${interaction.settlement_seen ? t.yes : t.no}。`,
      tone: interaction.strict ? 'accent' : 'danger',
    })
  })

  return { lanes, steps }
}

function getRolePriority(role: string) {
  switch (role) {
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
    case 'on_behalf_of':
      return 5
    case 'factory':
      return 6
    case 'pair':
      return 7
    case 'trace_source':
      return 8
    case 'trace_target':
      return 9
    case 'trace_participant':
      return 10
    default:
      return 99
  }
}

function resolveTraceEvidence(data: TransactionDetailResponse): TraceEvidence[] {
  return data.trace_summary?.interaction_evidence ?? []
}

function formatTraceStatus(status: string, t: ReturnType<typeof useI18n>['t']) {
  switch (status) {
    case 'available':
      return t.traceStatusAvailable
    case 'error':
      return t.traceStatusError
    default:
      return t.traceStatusUnavailable
  }
}

function formatTraceErrorSummary(
  traceSummary: TraceSummary,
  t: ReturnType<typeof useI18n>['t'],
  fallback = '',
) {
  if (traceSummary.status === 'available') {
    return fallback || t.notAvailable
  }
  const raw = (traceSummary.error || '').toLowerCase()
  if (!raw) {
    return fallback || t.traceUnavailable
  }
  if (raw.includes('historical state is not available')) {
    return t.yes === 'Yes'
      ? 'The RPC cannot reconstruct the historical state required for tracing this transaction.'
      : '当前 RPC 无法重建这笔交易所需的历史状态，因此不能做 trace。'
  }
  if (raw.includes('too many requests') || raw.includes('rate') || raw.includes('429')) {
    return t.yes === 'Yes'
      ? 'The trace request was rate-limited by the RPC provider.'
      : 'trace 请求被 RPC 提供方限流了。'
  }
  if (raw.includes('method not found') || raw.includes('does not exist') || raw.includes('unsupported')) {
    return t.yes === 'Yes'
      ? 'The current RPC endpoint does not support debug_traceTransaction.'
      : '当前 RPC 端点不支持 debug_traceTransaction。'
  }
  return t.traceErrorDetail
}

function buildTraceFrameCopy(frame: TraceFrame) {
  if (frame.error) {
    return frame.error
  }
  if (frame.revert_reason) {
    return frame.revert_reason
  }
  if (frame.token_action) {
    return frame.token_amount ? `${frame.token_action} · ${frame.token_amount}` : frame.token_action
  }
  return ''
}

function getGraphSide(role: string | undefined) {
  switch (role) {
    case 'initiator':
    case 'provider':
    case 'factory':
    case 'on_behalf_of':
    case 'trace_source':
      return 'left'
    default:
      return 'right'
  }
}

function compareAddressSummaries(left: AddressSummary, right: AddressSummary) {
  const leftPriority = left.roleCodes.reduce((min, role) => Math.min(min, getRolePriority(role)), 99)
  const rightPriority = right.roleCodes.reduce((min, role) => Math.min(min, getRolePriority(role)), 99)
  if (leftPriority !== rightPriority) {
    return leftPriority - rightPriority
  }
  return left.address.localeCompare(right.address)
}

function AutoFitHashTitle({ value }: { value: string }) {
  const shellRef = useRef<HTMLDivElement | null>(null)
  const titleRef = useRef<HTMLHeadingElement | null>(null)

  useLayoutEffect(() => {
    const shell = shellRef.current
    const title = titleRef.current
    if (!shell || !title) {
      return
    }

    const maxFontSize = 24
    const minFontSize = 11

    const fit = () => {
      title.style.fontSize = `${maxFontSize}px`
      let nextSize = maxFontSize
      while (nextSize > minFontSize && title.scrollWidth > shell.clientWidth) {
        nextSize -= 1
        title.style.fontSize = `${nextSize}px`
      }
    }

    fit()

    const resizeObserver = new ResizeObserver(() => {
      fit()
    })
    resizeObserver.observe(shell)

    return () => {
      resizeObserver.disconnect()
    }
  }, [value])

  return (
    <div ref={shellRef} className="detail-hash-shell">
      <h1 ref={titleRef} className="detail-hash-title">{value}</h1>
    </div>
  )
}

function shortenHash(value: string) {
  if (!value) {
    return value
  }
  if (value.length <= 18) {
    return value
  }
  return `${value.slice(0, 10)}...${value.slice(-6)}`
}
