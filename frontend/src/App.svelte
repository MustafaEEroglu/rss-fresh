<script lang="ts">
  import { onMount } from 'svelte';
  import Sidebar from './lib/components/Sidebar.svelte';
  import ArticleList from './lib/components/ArticleList.svelte';
  import ArticleReader from './lib/components/ArticleReader.svelte';
  import FeedManager from './lib/components/FeedManager.svelte';
  import { app } from './lib/stores/app.svelte';

  let mobilePane = $state<'sidebar' | 'list' | 'detail'>('list');

  onMount(() => {
    void app.bootstrap();
  });

  function showSidebar() {
    mobilePane = 'sidebar';
  }
  function showList() {
    app.pruneArticlesToFilter();
    mobilePane = 'list';
  }
  function showDetail() {
    mobilePane = 'detail';
  }

  $effect(() => {
    if (app.selectedArticleId !== null) {
      mobilePane = 'detail';
    }
  });
</script>

<div class="app-shell flex h-full flex-col bg-slate-950 text-slate-100">
  <header
    class="safe-top safe-x flex shrink-0 items-center justify-between gap-2 border-b border-slate-800 bg-slate-900/80 px-3 py-2 backdrop-blur md:px-4"
  >
    <div class="flex min-w-0 items-center gap-2">
      <button
        type="button"
        class="btn btn-icon btn-ghost md:hidden"
        aria-label="Open sidebar"
        onclick={showSidebar}
      >
        ☰
      </button>
      <h1 class="truncate text-base font-semibold tracking-tight">RSS-Fresh</h1>
      {#if !app.online}
        <span
          class="shrink-0 rounded-full bg-amber-500/15 px-2 py-0.5 text-xs font-medium text-amber-300"
          >offline</span
        >
      {/if}
    </div>
    <div class="flex shrink-0 items-center gap-1.5 sm:gap-2">
      <button
        type="button"
        class="btn btn-sm btn-ghost"
        title="Refresh"
        onclick={() => app.refreshAll()}
        aria-label="Refresh feeds"
      >
        <span aria-hidden="true">↻</span>
        <span class="hidden sm:inline">Refresh</span>
      </button>
      <button
        type="button"
        class="btn btn-sm btn-ghost"
        class:btn-active={app.view === 'reader'}
        aria-pressed={app.view === 'reader'}
        onclick={() => app.setView('reader')}
      >
        Reader
      </button>
      <button
        type="button"
        class="btn btn-sm btn-ghost"
        class:btn-active={app.view === 'feeds'}
        aria-pressed={app.view === 'feeds'}
        onclick={() => app.setView('feeds')}
      >
        Feeds
      </button>
    </div>
  </header>

  {#if app.view === 'reader'}
    <main class="grid min-h-0 flex-1 grid-cols-1 md:grid-cols-[16rem_minmax(0,1fr)_minmax(0,2fr)]">
      <aside
        class="border-r border-slate-800 bg-slate-900/40 md:block"
        class:hidden={mobilePane !== 'sidebar'}
      >
        <Sidebar onPickCategory={() => showList()} />
      </aside>

      <section
        class="border-r border-slate-800 md:block"
        class:hidden={mobilePane !== 'list'}
      >
        <ArticleList onPickArticle={() => showDetail()} />
      </section>

      <section class="md:block" class:hidden={mobilePane !== 'detail'}>
        <ArticleReader onBack={() => showList()} />
      </section>
    </main>
  {:else}
    <main class="min-h-0 flex-1 overflow-y-auto">
      <FeedManager />
    </main>
  {/if}

  {#if app.error}
    <div
      class="safe-bottom safe-x flex shrink-0 items-center justify-between gap-3 border-t border-rose-700/60 bg-rose-900/30 px-3 py-2 text-sm text-rose-200"
      role="status"
    >
      <span class="min-w-0 flex-1">{app.error}</span>
      <button type="button" class="btn btn-sm btn-ghost shrink-0" onclick={() => (app.error = null)}>
        Dismiss
      </button>
    </div>
  {/if}
</div>
