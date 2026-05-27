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

<div class="flex h-full flex-col bg-slate-950 text-slate-100">
  <header
    class="flex shrink-0 items-center justify-between border-b border-slate-800 bg-slate-900/60 px-4 py-2 backdrop-blur"
  >
    <div class="flex items-center gap-3">
      <button
        class="rounded p-2 text-slate-400 hover:bg-slate-800 hover:text-slate-100 md:hidden"
        aria-label="Open sidebar"
        onclick={showSidebar}
      >
        ☰
      </button>
      <h1 class="text-base font-semibold tracking-tight">RSS-Fresh</h1>
      {#if !app.online}
        <span class="rounded-full bg-amber-500/15 px-2 py-0.5 text-xs text-amber-300"
          >offline</span
        >
      {/if}
    </div>
    <div class="flex items-center gap-2">
      <button
        class="rounded px-2 py-1 text-xs text-slate-300 hover:bg-slate-800"
        title="Refresh"
        onclick={() => app.refreshAll()}
        aria-label="Refresh"
      >
        ↻ Refresh
      </button>
      <button
        class="rounded px-2 py-1 text-xs hover:bg-slate-800"
        class:bg-slate-800={app.view === 'reader'}
        onclick={() => app.setView('reader')}
      >
        Reader
      </button>
      <button
        class="rounded px-2 py-1 text-xs hover:bg-slate-800"
        class:bg-slate-800={app.view === 'feeds'}
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
      class="flex shrink-0 items-center justify-between border-t border-rose-700/60 bg-rose-900/30 px-4 py-2 text-sm text-rose-200"
      role="status"
    >
      <span>{app.error}</span>
      <button class="text-xs underline" onclick={() => (app.error = null)}>dismiss</button>
    </div>
  {/if}
</div>
