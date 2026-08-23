// web/src/lib/api/instructions.ts
import { request } from './client'

const enc = (value: string) => encodeURIComponent(value)

export type InstructionName = 'soul' | 'system' | 'agents'

export type InstructionFile = {
  content: string
}

export function getGlobalInstruction(name: InstructionName): Promise<InstructionFile> {
  return request<InstructionFile>(
    `/api/v1/global/instructions/${enc(name)}`,
  ) as Promise<InstructionFile>
}

export function putGlobalInstruction(
  name: InstructionName,
  content: string,
): Promise<InstructionFile> {
  return request<InstructionFile>(`/api/v1/global/instructions/${enc(name)}`, {
    method: 'PUT',
    body: { content },
  }) as Promise<InstructionFile>
}

export function getProjectInstruction(
  projectId: string,
  name: InstructionName,
): Promise<InstructionFile> {
  return request<InstructionFile>(
    `/api/v1/projects/${enc(projectId)}/instructions/${enc(name)}`,
  ) as Promise<InstructionFile>
}

export function putProjectInstruction(
  projectId: string,
  name: InstructionName,
  content: string,
): Promise<InstructionFile> {
  return request<InstructionFile>(
    `/api/v1/projects/${enc(projectId)}/instructions/${enc(name)}`,
    {
      method: 'PUT',
      body: { content },
    },
  ) as Promise<InstructionFile>
}
