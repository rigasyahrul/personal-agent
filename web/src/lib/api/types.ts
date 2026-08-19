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
