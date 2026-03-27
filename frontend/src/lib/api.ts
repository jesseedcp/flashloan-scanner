import type { FindingItem, TransactionDetailResponse } from '../types'

async function request<T>(input: string): Promise<T> {
  const response = await fetch(input)
  if (!response.ok) {
    const payload = (await response.json().catch(() => null)) as { error?: string } | null
    throw new Error(payload?.error ?? `Request failed: ${response.status}`)
  }
  return response.json() as Promise<T>
}

export function getTransactionDetail(txHash: string, chainId: number) {
  return request<TransactionDetailResponse>(`/api/v1/transactions/${txHash}?chain_id=${chainId}`)
}

export function getJobResults(jobId: string) {
  return request<FindingItem[]>(`/api/v1/jobs/${jobId}/results`)
}
