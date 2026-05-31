<script lang="ts">
  import { app } from '../stores/app.svelte';

  let newCatName = $state('');
  let newCatCritical = $state(false);
  let newFeedUrl = $state('');
  let newFeedName = $state('');
  let newFeedCategory = $state<number | null>(null);
  let busy = $state(false);

  // Per-form error state so errors don't bleed between sections.
  let catError = $state<string | null>(null);
  let feedError = $state<string | null>(null);
  let listError = $state<string | null>(null);

  // Inline two-step confirmation — stores the id pending deletion.
  let deletingCatId = $state<number | null>(null);
  let deletingFeedId = $state<number | null>(null);

  async function addCategory(e: Event) {
    e.preventDefault();
    if (!newCatName.trim()) return;
    busy = true;
    catError = null;
    try {
      await app.createCategory(newCatName.trim(), newCatCritical);
      newCatName = '';
      newCatCritical = false;
    } catch (err) {
      catError =
        err instanceof Error && err.message
          ? err.message
          : "Couldn't create the category — check your connection and try again.";
    } finally {
      busy = false;
    }
  }

  async function addFeed(e: Event) {
    e.preventDefault();
    if (!newFeedUrl.trim() || newFeedCategory === null) return;
    busy = true;
    feedError = null;
    try {
      await app.createFeed(newFeedCategory, newFeedUrl.trim(), newFeedName.trim() || undefined);
      newFeedUrl = '';
      newFeedName = '';
    } catch (err) {
      feedError =
        err instanceof Error && err.message
          ? err.message
          : "Couldn't add that feed. Make sure the URL points to a valid RSS or Atom feed.";
    } finally {
      busy = false;
    }
  }

  async function removeCategory(id: number) {
    busy = true;
    listError = null;
    deletingCatId = null;
    try {
      await app.deleteCategory(id);
    } catch (err) {
      listError =
        err instanceof Error && err.message
          ? err.message
          : "Couldn't delete the category. Try refreshing the page.";
    } finally {
      busy = false;
    }
  }

  async function removeFeed(id: number) {
    busy = true;
    listError = null;
    deletingFeedId = null;
    try {
      await app.deleteFeed(id);
    } catch (err) {
      listError =
        err instanceof Error && err.message
          ? err.message
          : "Couldn't delete the feed. Try refreshing the page.";
    } finally {
      busy = false;
    }
  }

  async function refreshFeed(id: number) {
    busy = true;
    listError = null;
    try {
      await app.refreshFeed(id);
    } catch (err) {
      listError =
        err instanceof Error && err.message
          ? err.message
          : "Couldn't refresh the feed. Check your connection and try again.";
    } finally {
      busy = false;
    }
  }

  async function toggleCritical(id: number, isCritical: boolean) {
    listError = null;
    try {
      await app.toggleCategoryCritical(id, isCritical);
    } catch (err) {
      listError =
        err instanceof Error && err.message
          ? err.message
          : "Couldn't update the category. Try refreshing the page.";
    }
  }

  $effect(() => {
    if (newFeedCategory === null && app.categories.length > 0) {
      newFeedCategory = app.categories[0].id;
    }
  });
</script>

<div class="safe-x mx-auto max-w-4xl px-4 py-6 sm:px-6">
  <h2 class="mb-1 text-xl font-semibold">Feeds</h2>
  <p class="mb-6 text-sm text-slate-400">
    Manage categories and the RSS / Atom feeds inside each one.
  </p>

  <!-- Add category -->
  <section class="mb-8 rounded-lg border border-slate-800 bg-slate-900/40 p-4">
    <h3 class="mb-3 text-sm font-medium text-slate-200">Add category</h3>
    {#if catError}
      <div
        role="alert"
        class="mb-3 rounded border border-rose-700/60 bg-rose-900/30 px-3 py-2 text-sm text-rose-200"
      >
        {catError}
      </div>
    {/if}
    <form class="flex flex-wrap items-end gap-3" onsubmit={addCategory}>
      <label class="flex flex-col text-xs text-slate-400">
        Name
        <input
          class="field-input mt-1 rounded border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none"
          placeholder="AI"
          bind:value={newCatName}
          oninput={() => (catError = null)}
          required
        />
      </label>
      <label class="flex items-center gap-2 text-xs text-slate-400">
        <input type="checkbox" bind:checked={newCatCritical} />
        Critical (push to Telegram immediately)
      </label>
      <button
        type="submit"
        class="btn btn-sm btn-primary"
        disabled={busy || !newCatName.trim()}
      >
        Add category
      </button>
    </form>
  </section>

  <!-- Add feed -->
  <section class="mb-8 rounded-lg border border-slate-800 bg-slate-900/40 p-4">
    <h3 class="mb-3 text-sm font-medium text-slate-200">Add feed</h3>
    {#if feedError}
      <div
        role="alert"
        class="mb-3 rounded border border-rose-700/60 bg-rose-900/30 px-3 py-2 text-sm text-rose-200"
      >
        {feedError}
      </div>
    {/if}
    <form class="grid grid-cols-1 gap-3 md:grid-cols-[1fr_1fr_10rem_auto]" onsubmit={addFeed}>
      <label class="flex flex-col text-xs text-slate-400">
        URL
        <input
          type="url"
          class="field-input mt-1 rounded border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none"
          placeholder="https://example.com/feed.xml"
          bind:value={newFeedUrl}
          oninput={() => (feedError = null)}
          required
        />
      </label>
      <label class="flex flex-col text-xs text-slate-400">
        Name (optional)
        <input
          class="field-input mt-1 rounded border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none"
          placeholder="auto"
          bind:value={newFeedName}
        />
      </label>
      <label class="flex flex-col text-xs text-slate-400">
        Category
        <select
          class="field-input mt-1 rounded border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-sky-500 focus:outline-none"
          bind:value={newFeedCategory}
        >
          {#each app.categories as c (c.id)}
            <option value={c.id}>{c.name}</option>
          {/each}
        </select>
      </label>
      <button
        type="submit"
        class="btn btn-sm btn-primary w-full md:w-auto md:self-end"
        disabled={busy || !newFeedUrl.trim() || newFeedCategory === null}
      >
        Add feed
      </button>
    </form>
  </section>

  <!-- Category + feed list -->
  <section>
    <h3 class="mb-3 text-sm font-medium text-slate-200">Categories &amp; feeds</h3>

    {#if listError}
      <div
        role="alert"
        class="mb-4 rounded border border-rose-700/60 bg-rose-900/30 px-3 py-2 text-sm text-rose-200"
      >
        {listError}
      </div>
    {/if}

    <div class="space-y-4">
      {#each app.categories as cat (cat.id)}
        <div class="rounded-lg border border-slate-800 bg-slate-900/40">
          <header class="flex flex-col gap-3 px-4 py-3 sm:flex-row sm:items-center sm:justify-between">
            <div class="flex min-w-0 flex-wrap items-center gap-2 sm:gap-3">
              <h4 class="text-sm font-medium text-slate-100">{cat.name}</h4>
              <span class="text-xs text-slate-500">/{cat.slug}</span>
              <label class="flex min-h-[2.75rem] items-center gap-2 text-xs text-slate-400">
                <input
                  type="checkbox"
                  checked={cat.is_critical}
                  disabled={busy}
                  aria-label={`Mark ${cat.name} as critical`}
                  onchange={(e) =>
                    void toggleCritical(cat.id, (e.currentTarget as HTMLInputElement).checked)}
                />
                critical
              </label>
            </div>

            <!-- Inline two-step deletion for category -->
            {#if deletingCatId === cat.id}
              <div class="flex items-center gap-2">
                <span class="text-xs text-rose-300">
                  Delete category and all its feeds?
                </span>
                <button
                  type="button"
                  class="btn btn-sm btn-danger"
                  disabled={busy}
                  onclick={() => void removeCategory(cat.id)}
                >
                  Yes, delete
                </button>
                <button
                  type="button"
                  class="btn btn-sm btn-ghost"
                  onclick={() => (deletingCatId = null)}
                >
                  Cancel
                </button>
              </div>
            {:else}
              <button
                type="button"
                class="btn btn-sm btn-danger w-full sm:w-auto"
                disabled={busy}
                onclick={() => (deletingCatId = cat.id)}
              >
                Delete category
              </button>
            {/if}
          </header>

          <ul class="border-t border-slate-800/80">
            {#each app.feeds.filter((f) => f.category_id === cat.id) as feed (feed.id)}
              <li
                class="flex flex-col gap-3 px-4 py-3 sm:flex-row sm:items-center sm:justify-between"
              >
                <div class="min-w-0 flex-1">
                  <div class="truncate text-slate-100">{feed.name}</div>
                  <div class="truncate text-xs text-slate-500">{feed.url}</div>
                </div>
                <div class="flex shrink-0 flex-wrap items-center gap-2">
                  {#if feed.error_count > 0}
                    <span class="text-xs text-amber-400">{feed.error_count} fetch error{feed.error_count === 1 ? '' : 's'}</span>
                  {/if}
                  {#if !feed.is_active}
                    <span class="text-xs text-rose-400" title="Polling paused after repeated errors">paused — too many errors</span>
                  {/if}
                  <button
                    type="button"
                    class="btn btn-sm btn-ghost flex-1 sm:flex-none"
                    disabled={busy}
                    onclick={() => void refreshFeed(feed.id)}
                  >
                    Refresh
                  </button>

                  <!-- Inline two-step deletion for feed -->
                  {#if deletingFeedId === feed.id}
                    <span class="text-xs text-rose-300">Delete this feed?</span>
                    <button
                      type="button"
                      class="btn btn-sm btn-danger"
                      disabled={busy}
                      onclick={() => void removeFeed(feed.id)}
                    >
                      Yes, delete
                    </button>
                    <button
                      type="button"
                      class="btn btn-sm btn-ghost"
                      onclick={() => (deletingFeedId = null)}
                    >
                      Cancel
                    </button>
                  {:else}
                    <button
                      type="button"
                      class="btn btn-sm btn-danger flex-1 sm:flex-none"
                      disabled={busy}
                      onclick={() => (deletingFeedId = feed.id)}
                    >
                      Delete
                    </button>
                  {/if}
                </div>
              </li>
            {:else}
              <li class="px-4 py-3 text-xs text-slate-500">
                No feeds in this category yet.
                <button
                  type="button"
                  class="ml-1 text-sky-400 underline underline-offset-2"
                  onclick={() => document.querySelector<HTMLInputElement>('input[type="url"]')?.focus()}
                >
                  Add one above.
                </button>
              </li>
            {/each}
          </ul>
        </div>
      {:else}
        <p class="text-sm text-slate-500">No categories yet — add one above to get started.</p>
      {/each}
    </div>
  </section>
</div>
