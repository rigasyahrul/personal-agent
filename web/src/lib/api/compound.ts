// web/src/lib/api/compound.ts
import { request } from './client'
import type { CompoundProposal, CreateCompoundInput, DecideCompoundInput } from './types'

const enc = (value: string) => encodeURIComponent(value)

export function createCompound(
  sessionId: string,
  body: CreateCompoundInput,
): Promise<CompoundProposal> {
  return request<CompoundProposal>(`/api/v1/sessions/${enc(sessionId)}/compound`, {
    method: 'POST',
    body,
  }) as Promise<CompoundProposal>
}

export function getCompound(
  sessionId: string,
  proposalId: string,
): Promise<CompoundProposal> {
  return request<CompoundProposal>(
    `/api/v1/sessions/${enc(sessionId)}/compound/${enc(proposalId)}`,
  ) as Promise<CompoundProposal>
}

export function decideCompound(
  sessionId: string,
  proposalId: string,
  body: DecideCompoundInput,
): Promise<CompoundProposal> {
  return request<CompoundProposal>(
    `/api/v1/sessions/${enc(sessionId)}/compound/${enc(proposalId)}/decide`,
    { method: 'POST', body },
  ) as Promise<CompoundProposal>
}
