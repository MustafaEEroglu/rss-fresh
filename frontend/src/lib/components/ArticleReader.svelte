<script lang="ts">
  import { app } from '../stores/app.svelte';

  interface Props {
    onBack?: () => void;
  }
  let { onBack }: Props = $props();

  // Strip <script>/<style> and dangerous handlers from feed-supplied HTML.
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
      class="safe-x flex shrink-0 flex-col gap-2 border-b border-slate-800 bg-slate-900/40 px-3 py-2"
    >
      <div class="flex items-center gap-2">
        <button
          type="button"
          class="btn btn-icon btn-ghost shrink-0 md:hidden"
          aria-label="Back to list"
          onclick={() => onBack?.()}
        >
          ←
        </button>
        <div class="min-w-0 flex-1 truncate text-xs font-medium text-slate-400">
          {a.feed_name} • {a.category_slug}
        </div>
      </div>

      <!-- Icon actions on narrow screens; text labels on md+. -->
      <div class="flex items-center gap-2">
        <button
          type="button"
          class="btn btn-sm flex-1 md:hidden"
          class:btn-active={a.is_saved}
          onclick={() => app.toggleSaved(a.id, !a.is_saved)}
          aria-pressed={a.is_saved}
          aria-label={a.is_saved ? 'Unsave article' : 'Save article'}
        >
          {a.is_saved ? '★ Saved' : '☆ Save'}
        </button>
        <button
          type="button"
          class="btn btn-sm flex-1 md:hidden"
          onclick={() => app.toggleRead(a.id, !a.is_read)}
          aria-pressed={a.is_read}
          aria-label={a.is_read ? 'Mark as unread' : 'Mark as read'}
        >
          {a.is_read ? '↩ Unread' : '✓ Read'}
        </button>
        <a
          class="btn btn-sm flex-1 md:hidden"
          href={a.url}
          target="_blank"
          rel="noopener noreferrer"
          aria-label="Open original article"
        >
          Open ↗
        </a>

        <div class="hidden items-center gap-2 md:flex md:ml-auto">
          <button
            type="button"
            class="btn btn-sm btn-ghost"
            class:btn-active={a.is_saved}
            onclick={() => app.toggleSaved(a.id, !a.is_saved)}
            aria-pressed={a.is_saved}
          >
            {a.is_saved ? '★ Saved' : '☆ Save'}
          </button>
          <button
            type="button"
            class="btn btn-sm btn-ghost"
            onclick={() => app.toggleRead(a.id, !a.is_read)}
            aria-pressed={a.is_read}
          >
            {a.is_read ? 'Mark unread' : 'Mark read'}
          </button>
          <a
            class="btn btn-sm btn-ghost"
            href={a.url}
            target="_blank"
            rel="noopener noreferrer"
          >
            Open ↗
          </a>
        </div>
      </div>
    </header>

    <article class="scrollbar-thin safe-x flex-1 overflow-y-auto px-3 py-5 md:px-5 md:py-6">
      <h1 class="mb-2 text-xl font-semibold leading-tight text-slate-50 md:text-2xl">
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
