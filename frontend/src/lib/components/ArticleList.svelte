<script lang="ts">
  import { app } from '../stores/app.svelte';

  interface Props {
    onPickArticle?: () => void;
  }
  let { onPickArticle }: Props = $props();

  function pick(id: number) {
    app.selectArticle(id);
    onPickArticle?.();
  }

  function relative(ts: string | null): string {
    if (!ts) return '';
    const t = new Date(ts).getTime();
    const diff = Date.now() - t;
    const mins = Math.round(diff / 60000);
    if (mins < 1) return 'just now';
    if (mins < 60) return `${mins}m`;
    const hours = Math.round(mins / 60);
    if (hours < 24) return `${hours}h`;
    const days = Math.round(hours / 24);
    if (days < 30) return `${days}d`;
    return new Date(ts).toLocaleDateString();
  }
</script>

<div class="flex h-full flex-col">
  <div
    class="flex shrink-0 items-center justify-between border-b border-slate-800 bg-slate-900/40 px-3 py-2"
  >
    <h2 class="text-sm font-medium text-slate-200">
      {app.selectedCategory?.name ?? 'All categories'}
    </h2>
    <button
      class="rounded px-2 py-1 text-xs text-slate-300 hover:bg-slate-800 disabled:opacity-30"
      disabled={app.articles.every((a) => a.is_read)}
      onclick={() => app.markAllReadInView()}
    >
      Mark all read
    </button>
  </div>

  <ol class="scrollbar-thin flex-1 overflow-y-auto" aria-label="Article list">
    {#if app.loading && app.articles.length === 0}
      <li class="px-4 py-6 text-sm text-slate-500">Loading…</li>
    {:else if app.articles.length === 0}
      <li class="px-4 py-6 text-sm text-slate-500">
        Nothing here. Try switching filter or adding feeds.
      </li>
    {/if}

    {#each app.articles as art (art.id)}
      <li>
        <button
          class="block w-full border-b border-slate-800/60 px-3 py-3 text-left transition hover:bg-slate-800/40"
          class:bg-slate-800={app.selectedArticleId === art.id}
          onclick={() => pick(art.id)}
        >
          <div class="flex items-center justify-between gap-2">
            <span
              class="truncate text-xs"
              class:text-slate-500={art.is_read}
              class:text-sky-400={!art.is_read}
            >
              {art.feed_name}
            </span>
            <span class="shrink-0 text-xs text-slate-500">{relative(art.published_at)}</span>
          </div>
          <div
            class="mt-1 line-clamp-2 text-sm"
            class:text-slate-400={art.is_read}
            class:text-slate-100={!art.is_read}
            class:font-medium={!art.is_read}
          >
            {art.title}
          </div>
          {#if art.is_saved}
            <div class="mt-1 text-xs text-amber-300">★ saved</div>
          {/if}
        </button>
      </li>
    {/each}

    {#if app.nextCursor}
      <li class="p-3">
        <button
          class="w-full rounded border border-slate-800 px-3 py-2 text-xs text-slate-300 hover:bg-slate-800"
          onclick={() => app.loadMore()}
        >
          Load more
        </button>
      </li>
    {/if}
  </ol>
</div>
