// Reactive app state via Svelte 5 runes inside a class.
// Single source of truth for the UI. Loads from Dexie first, then API,
// then writes the API result back to Dexie ("offline-first reconcile").
import { api, ApiError } from '../api';
import {
  cacheArticles,
  cacheCategories,
  cacheFeeds,
  patchLocalArticle,
  readCachedArticles,
  readCachedCategories,
  readCachedFeeds,
} from '../db';
import type { Article, ArticleFilterMode, Category, Feed } from '../types';

type View = 'reader' | 'feeds';

function filterToQuery(mode: ArticleFilterMode): {
  unread?: boolean;
  read?: boolean;
  saved?: boolean;
} {
  switch (mode) {
    case 'unread':
      return { unread: true };
    case 'read':
      return { read: true };
    case 'saved':
      return { saved: true };
  }
}

class AppState {
  categories = $state<Category[]>([]);
  feeds = $state<Feed[]>([]);
  articles = $state<Article[]>([]);
  selectedCategoryId = $state<number | null>(null);
  selectedArticleId = $state<number | null>(null);
  articleFilter = $state<ArticleFilterMode>('unread');
  view = $state<View>('reader');
  online = $state<boolean>(navigator.onLine);
  loading = $state<boolean>(false);
  error = $state<string | null>(null);
  nextCursor = $state<string | null>(null);

  selectedArticle = $derived(
    this.articles.find((a) => a.id === this.selectedArticleId) ?? null,
  );

  selectedCategory = $derived(
    this.categories.find((c) => c.id === this.selectedCategoryId) ?? null,
  );

  feedsForSelected = $derived(
    this.selectedCategoryId == null
      ? this.feeds
      : this.feeds.filter((f) => f.category_id === this.selectedCategoryId),
  );

  constructor() {
    if (typeof window !== 'undefined') {
      window.addEventListener('online', () => (this.online = true));
      window.addEventListener('offline', () => (this.online = false));
    }
  }

  setView(v: View) {
    this.view = v;
  }

  setCategory(id: number | null) {
    this.selectedCategoryId = id;
    this.selectedArticleId = null;
    void this.loadArticles();
  }

  setArticleFilter(mode: ArticleFilterMode) {
    if (this.articleFilter === mode) return;
    this.articleFilter = mode;
    this.selectedArticleId = null;
    void this.loadArticles();
  }

  private listQuery() {
    return {
      category_id: this.selectedCategoryId ?? undefined,
      ...filterToQuery(this.articleFilter),
      limit: 50 as const,
    };
  }

  /** Drop list rows that no longer match the active filter (call when returning to the list). */
  pruneArticlesToFilter() {
    if (this.articleFilter === 'unread') {
      this.articles = this.articles.filter((a) => !a.is_read);
    } else if (this.articleFilter === 'read') {
      this.articles = this.articles.filter((a) => a.is_read);
    } else if (this.articleFilter === 'saved') {
      this.articles = this.articles.filter((a) => a.is_saved);
    }
    if (
      this.selectedArticleId !== null &&
      !this.articles.some((a) => a.id === this.selectedArticleId)
    ) {
      this.selectedArticleId = null;
    }
  }

  selectArticle(id: number) {
    this.selectedArticleId = id;
    const a = this.articles.find((x) => x.id === id);
    if (a && !a.is_read) {
      void this.toggleRead(id, true);
    }
  }

  async bootstrap() {
    this.loading = true;
    try {
      this.categories = await readCachedCategories();
      this.feeds = await readCachedFeeds();
      this.articles = await readCachedArticles({
        ...filterToQuery(this.articleFilter),
        limit: 50,
      });
    } catch {
      // Dexie not ready or empty; ignore.
    }
    await this.refreshAll();
    this.loading = false;
  }

  async refreshAll() {
    if (!navigator.onLine) return;
    try {
      const [cats, fds, list] = await Promise.all([
        api.listCategories(),
        api.listFeeds(),
        api.listArticles(this.listQuery()),
      ]);
      this.categories = cats;
      this.feeds = fds;
      this.articles = list.items;
      this.nextCursor = list.next_cursor;
      void cacheCategories(cats);
      void cacheFeeds(fds);
      void cacheArticles(list.items);
    } catch (err) {
      this.error = err instanceof ApiError ? err.message : 'network error';
    }
  }

  async loadArticles() {
    this.loading = true;
    this.error = null;
    try {
      const list = await api.listArticles(this.listQuery());
      this.articles = list.items;
      this.nextCursor = list.next_cursor;
      void cacheArticles(list.items);
    } catch (err) {
      // Fall back to local cache if offline.
      this.articles = await readCachedArticles({
        category_id: this.selectedCategoryId ?? undefined,
        ...filterToQuery(this.articleFilter),
        limit: 50,
      });
      this.nextCursor = null;
      this.error = err instanceof ApiError ? err.message : null;
    } finally {
      this.loading = false;
    }
  }

  async loadMore() {
    if (!this.nextCursor) return;
    try {
      const list = await api.listArticles({
        ...this.listQuery(),
        cursor: this.nextCursor,
      });
      this.articles = [...this.articles, ...list.items];
      this.nextCursor = list.next_cursor;
      void cacheArticles(list.items);
    } catch (err) {
      this.error = err instanceof ApiError ? err.message : 'load more failed';
    }
  }

  async toggleRead(id: number, isRead: boolean) {
    this.articles = this.articles.map((a) => (a.id === id ? { ...a, is_read: isRead } : a));
    void patchLocalArticle(id, { is_read: isRead });
    if (this.articleFilter === 'read' && !isRead) {
      this.articles = this.articles.filter((a) => a.id !== id);
      if (this.selectedArticleId === id) this.selectedArticleId = null;
    }
    try {
      await api.updateArticle(id, { is_read: isRead });
    } catch {
      // Mutations are queued by Workbox Background Sync — UI already optimistic.
    }
    void this.refreshCategoryCounts();
  }

  async toggleSaved(id: number, isSaved: boolean) {
    this.articles = this.articles.map((a) => (a.id === id ? { ...a, is_saved: isSaved } : a));
    void patchLocalArticle(id, { is_saved: isSaved });
    try {
      await api.updateArticle(id, { is_saved: isSaved });
    } catch {
      /* queued */
    }
  }

  async markAllReadInView() {
    const ids = this.articles.filter((a) => !a.is_read).map((a) => a.id);
    if (ids.length === 0) return;
    try {
      await api.bulkMarkRead(ids);
      if (this.articleFilter === 'unread') {
        this.articles = [];
        this.selectedArticleId = null;
      } else {
        this.articles = this.articles.map((a) => ({ ...a, is_read: true }));
      }
    } catch {
      /* queued */
    }
    void this.refreshCategoryCounts();
  }

  async refreshCategoryCounts() {
    if (!navigator.onLine) return;
    try {
      this.categories = await api.listCategories();
      void cacheCategories(this.categories);
    } catch {
      /* ignore */
    }
  }

  async createCategory(name: string, isCritical: boolean) {
    const c = await api.createCategory(name, undefined, isCritical);
    this.categories = [...this.categories, c].sort((a, b) => a.name.localeCompare(b.name));
    void cacheCategories(this.categories);
    return c;
  }

  async deleteCategory(id: number) {
    await api.deleteCategory(id);
    this.categories = this.categories.filter((c) => c.id !== id);
    this.feeds = this.feeds.filter((f) => f.category_id !== id);
    if (this.selectedCategoryId === id) this.selectedCategoryId = null;
    void cacheCategories(this.categories);
    void cacheFeeds(this.feeds);
  }

  async toggleCategoryCritical(id: number, isCritical: boolean) {
    const c = await api.updateCategory(id, { is_critical: isCritical });
    this.categories = this.categories.map((x) => (x.id === id ? { ...x, ...c } : x));
    void cacheCategories(this.categories);
  }

  async createFeed(categoryId: number, url: string, name?: string) {
    const f = await api.createFeed(categoryId, url, name);
    this.feeds = [...this.feeds, f].sort((a, b) => a.name.localeCompare(b.name));
    void cacheFeeds(this.feeds);
    return f;
  }

  async deleteFeed(id: number) {
    await api.deleteFeed(id);
    this.feeds = this.feeds.filter((f) => f.id !== id);
    void cacheFeeds(this.feeds);
  }

  async refreshFeed(id: number) {
    await api.refreshFeed(id);
  }
}

export const app = new AppState();
