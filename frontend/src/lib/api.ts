import type { Article, ArticleFilter, ArticleListResponse, Category, Feed } from './types';

const BASE = '/api/v1';

class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
  ) {
    super(message);
  }
}

async function http<T>(method: string, path: string, body?: unknown): Promise<T> {
  const init: RequestInit = {
    method,
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
  };
  if (body !== undefined) init.body = JSON.stringify(body);
  const res = await fetch(`${BASE}${path}`, init);
  const text = await res.text();
  const data = text ? JSON.parse(text) : null;
  if (!res.ok) {
    const code = data?.code ?? 'http_error';
    const msg = data?.error ?? `HTTP ${res.status}`;
    throw new ApiError(res.status, code, msg);
  }
  return data as T;
}

export const api = {
  // Categories
  listCategories: () => http<{ items: Category[] }>('GET', '/categories').then((r) => r.items),
  createCategory: (name: string, slug?: string, is_critical = false) =>
    http<Category>('POST', '/categories', { name, slug, is_critical }),
  updateCategory: (id: number, patch: Partial<Pick<Category, 'name' | 'slug' | 'is_critical'>>) =>
    http<Category>('PATCH', `/categories/${id}`, patch),
  deleteCategory: (id: number) => http<void>('DELETE', `/categories/${id}`),

  // Feeds
  listFeeds: (categoryId?: number) =>
    http<{ items: Feed[] }>(
      'GET',
      categoryId !== undefined ? `/feeds?category_id=${categoryId}` : '/feeds',
    ).then((r) => r.items),
  createFeed: (category_id: number, url: string, name?: string) =>
    http<Feed>('POST', '/feeds', { category_id, url, name }),
  updateFeed: (id: number, patch: Partial<Pick<Feed, 'category_id' | 'name' | 'url' | 'is_active'>>) =>
    http<Feed>('PATCH', `/feeds/${id}`, patch),
  deleteFeed: (id: number) => http<void>('DELETE', `/feeds/${id}`),
  refreshFeed: (id: number) => http<void>('POST', `/feeds/${id}/refresh`),

  // Articles
  listArticles: (f: ArticleFilter): Promise<ArticleListResponse> => {
    const q = new URLSearchParams();
    if (f.category_id !== undefined) q.set('category_id', String(f.category_id));
    if (f.feed_id !== undefined) q.set('feed_id', String(f.feed_id));
    if (f.unread) q.set('unread', '1');
    if (f.saved) q.set('saved', '1');
    if (f.limit) q.set('limit', String(f.limit));
    if (f.cursor) q.set('cursor', f.cursor);
    const qs = q.toString();
    return http<ArticleListResponse>('GET', `/articles${qs ? '?' + qs : ''}`);
  },
  updateArticle: (id: number, patch: Partial<Pick<Article, 'is_read' | 'is_saved'>>) =>
    http<Article>('PATCH', `/articles/${id}`, patch),
  bulkMarkRead: (ids: number[]) =>
    http<{ updated: number }>('POST', '/articles/mark-read', { ids }),
};

export { ApiError };
