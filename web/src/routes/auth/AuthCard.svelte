<!-- web/src/routes/auth/AuthCard.svelte -->
<script lang="ts">import type { Snippet } from 'svelte'; let { children }: { children: Snippet } = $props();</script>
<main class="auth-canvas"><section class="auth-card">{@render children()}</section></main>
