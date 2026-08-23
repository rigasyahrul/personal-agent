// web/src/lib/api/notes.ts
import { request } from './client'

const enc = (value: string) => encodeURIComponent(value)

export type NoteBacklinkDTO = {
  knowledge_id: string
  path: string
  title?: string
  snippet?: string
  kind?: string
  source_note_id?: string
}

export type NoteBacklink = {
  title: string
  path: string
  knowledgeId: string
  kind?: string
  sourceNoteId?: string
  snippet?: string
}

export function mapNoteBacklink(dto: NoteBacklinkDTO): NoteBacklink {
  return {
    title: dto.title ?? '',
    path: dto.path,
    knowledgeId: dto.knowledge_id,
    kind: dto.kind,
    sourceNoteId: dto.source_note_id,
    snippet: dto.snippet,
  }
}

export async function listProjectNoteBacklinks(
  projectId: string,
  noteId: string,
): Promise<NoteBacklink[]> {
  const data = await request<{ items?: NoteBacklinkDTO[] }>(
    `/api/v1/projects/${enc(projectId)}/notes/${enc(noteId)}/backlinks`,
  )
  return (data?.items ?? []).map(mapNoteBacklink)
}
