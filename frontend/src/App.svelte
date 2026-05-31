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
    app.selectedArticleId = null;
    mobilePane = 'list';
  }
  function showDetail() {
    mobilePane = 'detail';
  }

  function formatLastSync(d: Date): string {
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }
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
      {#if app.refreshNotice}
        <span class="shrink-0 text-xs text-emerald-400" role="status" aria-live="polite">
          {app.refreshNotice}
        </span>
      {:else if app.lastRefreshedAt}
        <span class="hidden text-xs text-slate-500 sm:inline" title="Last sync">
          {formatLastSync(app.lastRefreshedAt)}
        </span>
      {/if}
      <button
        type="button"
        class="btn btn-sm btn-ghost"
        title="Refresh"
        disabled={app.refreshing}
        aria-busy={app.refreshing}
        onclick={() => void app.refreshAll()}
        aria-label={app.refreshing ? 'Refreshing feeds' : 'Refresh feeds'}
      >
        <span class="inline-block" class:icon-spin={app.refreshing} aria-hidden="true">↻</span>
        <span class="hidden sm:inline">{app.refreshing ? 'Refreshing…' : 'Refresh'}</span>
      </button>

      <!-- View switcher — mutually exclusive, so tab semantics are appropriate -->
      <div role="tablist" aria-label="View">
        <button
          type="button"
          role="tab"
          class="btn btn-sm btn-ghost"
          class:btn-active={app.view === 'reader'}
          aria-selected={app.view === 'reader'}
          onclick={() => app.setView('reader')}
        >
          Reader
        </button>
        <button
          type="button"
          role="tab"
          class="btn btn-sm btn-ghost"
          class:btn-active={app.view === 'feeds'}
          aria-selected={app.view === 'feeds'}
          onclick={() => app.setView('feeds')}
        >
          Feeds
        </button>
      </div>
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
      role="alert"
      aria-live="assertive"
    >
      <span class="min-w-0 flex-1">{app.error}</span>
      <button type="button" class="btn btn-sm btn-ghost shrink-0" onclick={() => (app.error = null)}>
        Dismiss
      </button>
    </div>
  {/if}
</div>
