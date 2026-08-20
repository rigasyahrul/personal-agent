// web/src/lib/api/types.ts
export interface Project {
  id: string
  vault_id?: string | null
  vault_name?: string
  name: string
  note_count: number
  session_count?: number
  due_count?: number
}

export interface Vault {
  id: string
  name: string
  created_at: string
  updated_at: string
}

export interface HomeResponse {
  projects: Project[]
  due_count?: number
  generated_at: string
}

export interface Session {
  id: string
  title: string
  status: string
  provider: string
  model_id: string
  home?: string
  vault_id?: string | null
  project_id?: string | null
  created_at?: string
  updated_at?: string
  tool_grants?: { workspace_files?: boolean } | null
  tool_grants_json?: string | null
}

export interface ModelOption {
  provider: string
  model_id: string
}

export interface ModelsResponse {
  models: ModelOption[]
}

export interface CreateSessionInput {
  home: 'project'
  title: string
  provider: string
  model_id: string
  model_parameters: Record<string, unknown>
  tool_grants: { workspace_files: boolean }
}

export interface ChatMessage {
  sequence: number
  role: string
  content: string
  changed_path?: string
}

export interface RunStatus {
  status: string
}

export interface NoteTreeEntry {
  kind: 'folder' | 'file' | string
  path: string
  note_id?: string
}

export interface NoteDetail {
  id?: string
  note_id?: string
  relative_path: string
  body: string
  rendered_html?: string
}

export interface WorkspaceEntry {
  path: string
  kind: 'file' | 'directory' | string
}

export interface WorkspaceTree {
  entries: WorkspaceEntry[]
}

export interface WorkspaceFile {
  path: string
  kind: 'file' | string
  content?: string
}

export interface PromotePayload {
  workspace_path: string
  target_relative_path: string
  review_mode: 'none' | 'whole' | 'bites' | string
}

export interface PromoteResult {
  operation_id: string
}

export interface OperationStatus {
  operation_id: string
  badge: string
  pending_id?: string
  retry_cards?: boolean
  publication_status?: string
}

export interface ReviewItem {
  id: string
  project_id: string
  note_id?: string
  kind: string
  prompt: string
  answer?: string | null
  row_version?: number
  [key: string]: unknown
}

export interface ReviewQueue {
  scope: string
  caught_up: boolean
  items: ReviewItem[]
}
