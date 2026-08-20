<!-- web/src/components/notes/NoteReader.svelte -->
<script lang="ts">
  import type { NoteDetail } from '../../lib/api/types'

  let {
    note,
  }: {
    note: NoteDetail
  } = $props()
</script>

<article class="prose prose-slate max-w-none">
  <h3 class="mb-3 text-lg font-semibold text-slate-900">{note.relative_path}</h3>
  {#if note.rendered_html}
    <!-- Server-provided safe HTML only -->
    {@html note.rendered_html}
  {:else}
    <pre class="whitespace-pre-wrap rounded-md bg-slate-50 p-4 text-sm text-slate-800">{note.body}</pre>
  {/if}
</article>
