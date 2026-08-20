// web/src/lib/api/index.ts
import { request, api as baseApi } from './client'
import type {
  ChatMessage,
  CreateSessionInput,
  ModelsResponse,
  NoteDetail,
  NoteTreeEntry,
  OperationStatus,
  Project,
  PromotePayload,
  PromoteResult,
  RunStatus,
  Session,
  WorkspaceFile,
  WorkspaceTree,
} from './types'

export * from './client'
export * from './types'

const enc = (value: string) => encodeURIComponent(value)

export const api = {
  get: baseApi.get,
  post: baseApi.post,

  getProject: (projectId: string) =>
    request<Project>(`/api/v1/projects/${enc(projectId)}`) as Promise<Project>,

  listProjectNotes: (projectId: string) =>
    request<NoteTreeEntry[]>(`/api/v1/projects/${enc(projectId)}/tree`) as Promise<NoteTreeEntry[]>,

  getProjectNote: (_projectId: string, noteId: string) =>
    request<NoteDetail>(`/api/v1/notes/${enc(noteId)}`) as Promise<NoteDetail>,

  listModels: () =>
    request<ModelsResponse>('/api/v1/models') as Promise<ModelsResponse>,

  listProjectSessions: (projectId: string) =>
    request<Session[]>(`/api/v1/projects/${enc(projectId)}/sessions`) as Promise<Session[]>,

  createProjectSession: (projectId: string, input: CreateSessionInput) =>
    request<Session>(`/api/v1/projects/${enc(projectId)}/sessions`, {
      method: 'POST',
      body: input,
    }) as Promise<Session>,

  listMessages: (sessionId: string) =>
    request<ChatMessage[]>(`/api/v1/sessions/${enc(sessionId)}/messages`) as Promise<ChatMessage[]>,

  currentRun: (sessionId: string) =>
    request<RunStatus | null>(`/api/v1/sessions/${enc(sessionId)}/runs/current`) as Promise<RunStatus | null>,

  sendMessage: (sessionId: string, body: { content: string; request_key: string }) =>
    request(`/api/v1/sessions/${enc(sessionId)}/messages`, {
      method: 'POST',
      body,
    }),

  workspaceTree: (sessionId: string) =>
    request<WorkspaceTree>(`/api/v1/sessions/${enc(sessionId)}/workspace/tree`) as Promise<WorkspaceTree>,

  workspaceFile: (sessionId: string, path: string) =>
    request<WorkspaceFile>(
      `/api/v1/sessions/${enc(sessionId)}/workspace/file?path=${enc(path)}`,
    ) as Promise<WorkspaceFile>,

  promoteSession: (sessionId: string, payload: PromotePayload, key: string) =>
    request<PromoteResult>(`/api/v1/sessions/${enc(sessionId)}/promote`, {
      method: 'POST',
      body: payload,
      headers: { 'Idempotency-Key': key },
    }) as Promise<PromoteResult>,

  operationStatus: (operationId: string) =>
    request<OperationStatus>(`/api/v1/operations/${enc(operationId)}`) as Promise<OperationStatus>,

  retryReviewPending: (pendingId: string) =>
    request(`/api/v1/review/pending/${enc(pendingId)}/retry`, {
      method: 'POST',
      body: {},
    }),
}
