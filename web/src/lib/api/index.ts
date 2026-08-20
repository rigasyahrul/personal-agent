// web/src/lib/api/index.ts
import { request, api as baseApi } from './client'
import type {
  BackupListResponse,
  BackupRun,
  ChatMessage,
  CreateSessionInput,
  ModelsResponse,
  NoteDetail,
  NoteTreeEntry,
  OperationStatus,
  Project,
  PromotePayload,
  PromoteResult,
  RateReviewPayload,
  ReviewQueue,
  RunStatus,
  Session,
  Settings,
  UpdateSettingsInput,
  WorkspaceFile,
  WorkspaceTree,
} from './types'

export * from './client'
export * from './types'

const enc = (value: string) => encodeURIComponent(value)

export const api = {
  get: baseApi.get,
  post: baseApi.post,

  listProjects: () =>
    request<Project[]>('/api/v1/projects') as Promise<Project[]>,

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
    // PromoteResult.operation_id is returned for operation status polling.
    request<PromoteResult>(`/api/v1/sessions/${enc(sessionId)}/promote`, {
      method: 'POST',
      body: payload,
      headers: { 'Idempotency-Key': key },
    }) as Promise<PromoteResult>,

  operationStatus: (operationId: string) =>
    request<OperationStatus>(`/api/v1/operations/${enc(operationId)}`) as Promise<OperationStatus>,

  getReviewQueue: (scope: string) =>
    // Review queue uses scope=all or scope=project:{id}.
    request<ReviewQueue>(`/api/v1/review/queue?scope=${enc(scope)}`) as Promise<ReviewQueue>,

  rateReviewItem: (itemId: string, payload: RateReviewPayload) =>
    request(`/api/v1/review/items/${enc(itemId)}/rate`, {
      method: 'POST',
      body: payload,
    }),

  suspendReviewItem: (itemId: string) =>
    request(`/api/v1/review/items/${enc(itemId)}/suspend`, {
      method: 'POST',
      body: {},
    }),

  retryReviewPending: (pendingId: string) =>
    request(`/api/v1/review/pending/${enc(pendingId)}/retry`, {
      method: 'POST',
      body: {},
    }),

  getSettings: () =>
    request<Settings>('/api/v1/settings') as Promise<Settings>,

  updateSettings: (input: UpdateSettingsInput) =>
    request<Settings>('/api/v1/settings', {
      method: 'PUT',
      body: input,
    }) as Promise<Settings>,

  listBackups: () =>
    request<BackupListResponse>('/api/v1/backups') as Promise<BackupListResponse>,

  createBackup: () =>
    request<BackupRun>('/api/v1/backups', {
      method: 'POST',
      body: {},
    }) as Promise<BackupRun>,
}
