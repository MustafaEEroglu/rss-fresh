// Wire types — must match backend/internal/db/*.go JSON shapes EXACTLY.
// See `.memory_bank/system_architecture.md` for the contract.

export interface Category {
  id: number;
  name: string;
  slug: string;
  is_critical: boolean;
  created_at?: string;
  feed_count?: number;
  unread_count?: number;
}

export interface Feed {
  id: number;
  category_id: number;
  name: string;
  url: string;
  last_fetched_at: string | null;
  error_count: number;
  is_active: boolean;
  created_at?: string;
}

export interface Article {
  id: number;
  feed_id: number;
  feed_name: string;
  category_id: number;
  category_slug: string;
  guid?: string;
  title: string;
  url: string;
  author?: string;
  content?: string;
  summary?: string;
  published_at: string | null;
  fetched_at: string;
  is_read: boolean;
  is_saved: boolean;
}

export interface ArticleListResponse {
  items: Article[];
  next_cursor: string | null;
}

export type ArticleFilterMode = 'unread' | 'read' | 'saved';

export interface ArticleFilter {
  category_id?: number;
  feed_id?: number;
  unread?: boolean;
  read?: boolean;
  saved?: boolean;
  limit?: number;
  cursor?: string;
}
