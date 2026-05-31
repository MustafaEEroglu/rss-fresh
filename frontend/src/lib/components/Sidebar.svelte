<script lang="ts">
  import { app } from '../stores/app.svelte';
  import ArticleFilterBar from './ArticleFilterBar.svelte';

  interface Props {
    onPickCategory?: () => void;
  }
  let { onPickCategory }: Props = $props();

  function pick(id: number | null) {
    app.setCategory(id);
    onPickCategory?.();
  }
</script>

<nav class="flex h-full flex-col" aria-label="Categories">
  <div class="safe-x hidden border-b border-slate-800 px-3 py-3 md:block">
    <h2 class="text-xs font-semibold uppercase tracking-wider text-slate-400">Filter</h2>
    <div class="mt-2">
      <ArticleFilterBar />
    </div>
  </div>

  <div class="scrollbar-thin flex-1 overflow-y-auto">
    <button
      type="button"
      class="nav-row text-sm"
      class:bg-slate-800={app.selectedCategoryId === null}
      onclick={() => pick(null)}
    >
      <span>All</span>
      <span class="text-xs text-slate-400">
        {app.categories.reduce((s, c) => s + (c.unread_count ?? 0), 0)}
      </span>
    </button>

    {#each app.categories as cat (cat.id)}
      <button
        type="button"
        class="nav-row text-sm"
        class:bg-slate-800={app.selectedCategoryId === cat.id}
        onclick={() => pick(cat.id)}
      >
        <span class="flex min-w-0 items-center gap-2 truncate">
          {#if cat.is_critical}
            <span class="text-rose-400" aria-label="Critical" role="img">●</span>
          {/if}
          <span class="truncate">{cat.name}</span>
        </span>
        <span class="shrink-0 text-xs text-slate-400">
          {cat.unread_count ?? 0}
        </span>
      </button>
    {/each}

    {#if app.categories.length === 0}
      <p class="px-3 py-4 text-xs text-slate-500">
        No categories yet. Switch to <strong>Feeds</strong> to add one.
      </p>
    {/if}
  </div>

  <div
    class="safe-bottom safe-x shrink-0 border-t border-slate-800 px-3 py-2 text-xs text-slate-500"
  >
    {app.feeds.length} feed{app.feeds.length === 1 ? '' : 's'} • {app.online ? 'online' : 'offline'}
  </div>
</nav>
