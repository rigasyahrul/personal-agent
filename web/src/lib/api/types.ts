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
  tool_grants?: { workspace_files?: boolean; session_files?: boolean } | null
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
  /** RFC3339 from API when present (domain.Message.created_at). */
  created_at?: string
  changed_path?: string
  run_id?: string
  tool_calls_json?: string
  tool_call_id?: string
}

export interface RunStatus {
  status: string
  id?: string
  started_at?: string
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

export type ReviewRating = 'again' | 'hard' | 'good' | 'easy'

export interface RateReviewPayload {
  rating: ReviewRating | string
  request_key: string
  row_version: number
  duration_ms: number
}

export interface Settings {
  timezone: string
  default_provider: string
  default_model_id: string
  backup_schedule: 'off' | 'daily' | string
  backup?: BackupStatus
  last_success?: BackupRun | null
  last_failure?: BackupRun | null
}

export interface BackupStatus {
  last_success?: BackupRun | null
  last_failure?: BackupRun | null
  sink_configured?: boolean
  schedule?: string
}

export interface BackupRun {
  id: string
  status: string
  cutoff_at?: string
  started_at?: string
  local_path?: string
  object_key?: string
  manifest_hash?: string
  completed_at?: string
  error?: string
}

export interface BackupListResponse {
  backups: BackupRun[]
  last_success?: BackupRun | null
  last_failure?: BackupRun | null
}

export interface UpdateSettingsInput {
  timezone: string
  default_provider: string
  default_model_id: string
  backup_schedule: 'off' | 'daily' | string
}

export interface CompoundItem {
  kind: string
  path: string
  action: string
  title?: string
  content: string
  content_sha256: string
}

export interface CompoundProposal {
  id: string
  status: string
  items: CompoundItem[]
  created_at: string
  decided_at?: string
  finished_at?: string
  error?: string
}

export interface CreateCompoundInput {
  request_key: string
  user_context?: string
  items?: CompoundItem[]
}

export interface DecideCompoundInput {
  request_key: string
  decision: 'approve' | 'reject'
  items?: CompoundItem[]
}
