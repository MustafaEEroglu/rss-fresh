<script lang="ts">
  import { app } from '../stores/app.svelte';
  import type { ArticleFilterMode } from '../types';

  const tabs: { mode: ArticleFilterMode; label: string }[] = [
    { mode: 'unread', label: 'Unread' },
    { mode: 'read', label: 'Read' },
    { mode: 'saved', label: 'Saved' },
  ];
</script>

<div class="segmented" role="group" aria-label="Article filter">
  {#each tabs as tab (tab.mode)}
    <button
      type="button"
      class="btn-segment"
      class:btn-segment-active={app.articleFilter === tab.mode}
      aria-pressed={app.articleFilter === tab.mode}
      onclick={() => app.setArticleFilter(tab.mode)}
    >
      {tab.label}
    </button>
  {/each}
</div>
