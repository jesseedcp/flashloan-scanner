export type SocketStatus = 'connecting' | 'connected' | 'disconnected'
export type JobStatus = 'idle' | 'pending' | 'running' | 'completed' | 'failed'

export type FindingItem = {
  tx_hash: string
  protocol: string
  block_number: number
  candidate: boolean
  verified: boolean
  strict: boolean
  interaction_count?: number
  strict_interaction_count?: number
  protocol_count?: number
  protocols?: string[]
  created_at?: string
}

export type LogItem = {
  level: string
  message: string
  protocol?: string
  block_number?: number
  timestamp: string
}

export type ProtocolProgress = {
  protocol: string
  status: JobStatus
  start_block: number
  end_block: number
  current_block: number
  found_candidates: number
  found_verified: number
  found_strict: number
  latest_finding?: FindingItem
  error?: string
}

export type JobOverview = {
  job_id: string
  status: Exclude<JobStatus, 'idle'>
  chain_id: number
  start_block: number
  end_block: number
  trace_enabled: boolean
  total_candidates: number
  total_verified: number
  total_strict: number
  completed_protocols: number
  protocols: Record<string, ProtocolProgress>
  started_at?: string
  finished_at?: string
  error?: string
}

export type TransactionDetailResponse = {
  tx_hash: string
  chain_id: number
  block_number: string
  candidate: boolean
  verified: boolean
  strict: boolean
  interaction_count: number
  strict_interaction_count: number
  protocol_count: number
  protocols: string[]
  summary?: {
    addresses: Array<{
      address: string
      roles: string[]
    }>
    timeline: Array<{
      ordinal: number
      kind: string
      protocol?: string
      entrypoint?: string
      asset_address?: string
      asset_role?: string
      amount_borrowed?: string
      amount_repaid?: string
      strict?: boolean
      event_seen?: boolean
      callback_seen?: boolean
      settlement_seen?: boolean
      repayment_seen?: boolean
    }>
    conclusion: {
      verdict: string
      protocols: string[]
      interaction_count: number
      strict_interaction_count: number
      callback_seen_count: number
      settlement_seen_count: number
      repayment_seen_count: number
      strict_leg_count: number
      debt_opening_count: number
      exclusion_reasons: string[]
    }
  }
  trace_summary?: {
    status: string
    error?: string
    root_frame_id?: string
    frames?: Array<{
      id: string
      parent_id?: string
      depth: number
      call_index: number
      type: string
      from: string
      to: string
      method_selector?: string
      error?: string
      revert_reason?: string
      token_action?: string
      asset_address?: string
      token_amount?: string
      flow_source?: string
      flow_target?: string
      tags?: string[]
    }>
    sequence?: Array<{
      step: number
      frame_id: string
      parent_frame_id?: string
      depth: number
      from: string
      to: string
      method_selector?: string
      label: string
      detail: string
      token_action?: string
      asset_address?: string
      token_amount?: string
      error?: string
    }>
    asset_flows?: Array<{
      frame_id: string
      action: string
      asset_address: string
      source: string
      target: string
      amount: string
    }>
    interaction_evidence?: Array<{
      interaction_id: string
      protocol: string
      entrypoint: string
      verdict: string
      callback_seen: boolean
      settlement_seen: boolean
      repayment_seen: boolean
      contains_debt_opening: boolean
      callback_frame_ids?: string[]
      callback_subtree_ids?: string[]
      repayment_frame_ids?: string[]
      exclusion_reason?: string
      verification_notes?: string
      provider_address?: string
      receiver_address?: string
    }>
  }
  interactions: Array<{
    interaction_id: string
    protocol: string
    entrypoint: string
    provider_address: string
    factory_address: string
    pair_address: string
    receiver_address: string
    callback_target: string
    initiator: string
    on_behalf_of: string
    candidate_level: number
    verified: boolean
    strict: boolean
    callback_seen: boolean
    settlement_seen: boolean
    repayment_seen: boolean
    contains_debt_opening: boolean
    exclusion_reason: string
    verification_notes: string
    raw_method_selector: string
    legs: Array<{
      leg_index: number
      asset_address: string
      asset_role: string
      token_side: string
      amount_out: string
      amount_in: string
      amount_borrowed: string
      amount_repaid: string
      premium_amount: string
      fee_amount: string
      interest_rate_mode: string
      repaid_to_address: string
      opened_debt: boolean
      strict_leg: boolean
      event_seen: boolean
      settlement_mode: string
    }>
  }>
}

export type WSMessage = {
  type: string
  job_id?: string
  timestamp?: string
  payload: unknown
}
