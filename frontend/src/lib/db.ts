// Dexie offline cache. Mirrors server tables for offline reads + queues
// mutations through the Workbox Background Sync system. See
// `.memory_bank/system_architecture.md` § SW for the contract.
import Dexie, { type EntityTable } from 'dexie';
import type { Article, Category, Feed } from './types';

export interface OutboxItem {
  id?: number;
  method: 'POST' | 'PATCH' | 'DELETE';
  path: string;
  body: string | null;
  createdAt: number;
}

class RssFreshDB extends Dexie {
  articles!: EntityTable<Article, 'id'>;
  feeds!: EntityTable<Feed, 'id'>;
  categories!: EntityTable<Category, 'id'>;
  outbox!: EntityTable<OutboxItem, 'id'>;

  constructor() {
    super('rss-fresh-cache');
    this.version(1).stores({
      articles: 'id, feed_id, category_id, [category_id+published_at], is_read, is_saved, published_at',
      feeds: 'id, category_id',
      categories: 'id, slug',
      outbox: '++id, createdAt',
    });
  }
}

export const localDb = new RssFreshDB();

export async function cacheArticles(items: Article[]): Promise<void> {
  if (items.length === 0) return;
  await localDb.articles.bulkPut(items);
}

export async function cacheFeeds(items: Feed[]): Promise<void> {
  await localDb.transaction('rw', localDb.feeds, async () => {
    await localDb.feeds.clear();
    if (items.length > 0) await localDb.feeds.bulkPut(items);
  });
}

export async function cacheCategories(items: Category[]): Promise<void> {
  await localDb.transaction('rw', localDb.categories, async () => {
    await localDb.categories.clear();
    if (items.length > 0) await localDb.categories.bulkPut(items);
  });
}

export async function readCachedArticles(filter: {
  category_id?: number;
  unread?: boolean;
  saved?: boolean;
  limit?: number;
}): Promise<Article[]> {
  let coll = localDb.articles.orderBy('published_at').reverse();
  if (filter.category_id !== undefined) {
    coll = coll.filter((a) => a.category_id === filter.category_id);
  }
  if (filter.unread) {
    coll = coll.filter((a) => !a.is_read);
  }
  if (filter.saved) {
    coll = coll.filter((a) => a.is_saved);
  }
  return await coll.limit(filter.limit ?? 50).toArray();
}

export async function readCachedCategories(): Promise<Category[]> {
  return await localDb.categories.orderBy('name').toArray();
}

export async function readCachedFeeds(categoryId?: number): Promise<Feed[]> {
  if (categoryId !== undefined) {
    return await localDb.feeds.where('category_id').equals(categoryId).toArray();
  }
  return await localDb.feeds.toArray();
}

// Optimistic local update for instant UI feedback before the network ack.
export async function patchLocalArticle(
  id: number,
  patch: Partial<Pick<Article, 'is_read' | 'is_saved'>>,
): Promise<void> {
  await localDb.articles.update(id, patch);
}
