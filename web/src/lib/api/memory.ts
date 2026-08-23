// web/src/lib/api/memory.ts
import { request } from './client'

const enc = (value: string) => encodeURIComponent(value)

export function getProjectMemoryLessons(projectId: string): Promise<{ content: string }> {
  return request<{ content: string }>(
    `/api/v1/projects/${enc(projectId)}/memory/lessons`,
  ) as Promise<{ content: string }>
}
