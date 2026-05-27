<script lang="ts">
  import { app } from '../stores/app.svelte';

  interface Props {
    onBack?: () => void;
  }
  let { onBack }: Props = $props();

  // Strip <script>/<style> and dangerous handlers from feed-supplied HTML.
  // Servers cannot be trusted with the user's reading surface, so we sanitise
  // before injecting via {@html}. Keep simple — this isn't a full sanitiser,
  // but it removes the highest-risk vectors.
  function sanitize(html: string | undefined | null): string {
    if (!html) return '';
    let s = html;
    s = s.replace(/<script\b[\s\S]*?<\/script>/gi, '');
    s = s.replace(/<style\b[\s\S]*?<\/style>/gi, '');
    s = s.replace(/\son\w+\s*=\s*"[^"]*"/gi, '');
    s = s.replace(/\son\w+\s*=\s*'[^']*'/gi, '');
    s = s.replace(/\son\w+\s*=\s*[^\s>]+/gi, '');
    s = s.replace(/javascript:/gi, '');
    return s;
  }
</script>

<div class="flex h-full flex-col">
  {#if !app.selectedArticle}
    <div class="flex flex-1 items-center justify-center px-6 text-center text-sm text-slate-500">
      Select an article to read.
    </div>
  {:else}
    {@const a = app.selectedArticle}
    <header
      class="flex shrink-0 items-center justify-between gap-2 border-b border-slate-800 bg-slate-900/40 px-3 py-2"
    >
      <button
        class="rounded p-1 text-slate-400 hover:bg-slate-800 hover:text-slate-100 md:hidden"
        aria-label="Back to list"
        onclick={() => onBack?.()}
      >
        ←
      </button>
      <div class="min-w-0 flex-1 truncate text-xs text-slate-400">
        {a.feed_name} • {a.category_slug}
      </div>
      <div class="flex shrink-0 items-center gap-1">
        <button
          class="rounded px-2 py-1 text-xs hover:bg-slate-800"
          onclick={() => app.toggleSaved(a.id, !a.is_saved)}
          aria-pressed={a.is_saved}
          title={a.is_saved ? 'Unsave' : 'Save'}
        >
          {a.is_saved ? '★ saved' : '☆ save'}
        </button>
        <button
          class="rounded px-2 py-1 text-xs hover:bg-slate-800"
          onclick={() => app.toggleRead(a.id, !a.is_read)}
          aria-pressed={a.is_read}
        >
          {a.is_read ? 'Mark unread' : 'Mark read'}
        </button>
        <a
          class="rounded px-2 py-1 text-xs hover:bg-slate-800"
          href={a.url}
          target="_blank"
          rel="noopener noreferrer"
        >
          Open ↗
        </a>
      </div>
    </header>

    <article class="scrollbar-thin flex-1 overflow-y-auto px-5 py-6">
      <h1 class="mb-2 text-2xl font-semibold leading-tight text-slate-50">
        {a.title}
      </h1>
      <p class="mb-6 text-xs text-slate-500">
        {#if a.author}{a.author} • {/if}
        {#if a.published_at}{new Date(a.published_at).toLocaleString()}{/if}
      </p>
      <div class="article-body text-slate-200">
        {@html sanitize(a.content || a.summary)}
      </div>
    </article>
  {/if}
</div>
