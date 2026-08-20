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
