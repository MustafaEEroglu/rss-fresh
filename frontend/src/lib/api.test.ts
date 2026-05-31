/**
 * Frontend unit tests.
 * Stack: vitest + jsdom (no Svelte runtime needed - tests pure TS helpers).
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { api, ApiError } from './api';

function mockFetch(status: number, body: unknown, contentType = 'application/json') {
  const text = typeof body === 'string' ? body : JSON.stringify(body);
  vi.stubGlobal(
    'fetch',
    vi.fn().mockResolvedValue({
      ok: status >= 200 && status < 300,
      status,
      headers: new Headers({ 'Content-Type': contentType }),
      text: () => Promise.resolve(text),
    }),
  );
}

function getCalledParams(): URLSearchParams {
  const url = vi.mocked(fetch).mock.calls[0][0] as string;
  const q = url.includes('?') ? url.split('?')[1] : '';
  return new URLSearchParams(q);
}

beforeEach(() => {
  vi.stubGlobal('fetch', vi.fn());
});

afterEach(() => {
  vi.unstubAllGlobals();
});

// ---------------------------------------------------------------------------
// api.http - error handling
// ---------------------------------------------------------------------------

describe('api.http - error handling', () => {
  it('throws ApiError with correct status and code on 4xx', async () => {
    mockFetch(404, { code: 'not_found', error: 'article not found' });
    await expect(api.updateArticle(999, { is_read: true })).rejects.toMatchObject({
      status: 404,
      code: 'not_found',
      message: 'article not found',
    });
  });

  it('throws ApiError on 5xx with fallback code http_error', async () => {
    mockFetch(500, { error: 'internal' });
    await expect(api.listCategories()).rejects.toMatchObject({
      status: 500,
      code: 'http_error',
    });
  });

  it('handles empty response body on 204 without throwing', async () => {
    mockFetch(204, '');
    await expect(api.deleteCategory(1)).resolves.not.toThrow();
  });

  it('throws SyntaxError when response body is non-JSON HTML', async () => {
    // Cloudflare error pages return HTML. http() calls JSON.parse which throws.
    // This means a 502 from Cloudflare will surface as SyntaxError, not ApiError.
    mockFetch(200, '<html>502 Bad Gateway</html>', 'text/html');
    await expect(api.listCategories()).rejects.toBeInstanceOf(SyntaxError);
  });

  it('includes credentials:include on every request', async () => {
    mockFetch(200, { items: [] });
    await api.listCategories();
    const call = vi.mocked(fetch).mock.calls[0];
    expect((call[1] as RequestInit).credentials).toBe('include');
  });
});

// ---------------------------------------------------------------------------
// api.listArticles - query-string building
// ---------------------------------------------------------------------------

describe('api.listArticles - query string', () => {
  it('sets unread=1 for unread filter, does not set read', async () => {
    mockFetch(200, { items: [], next_cursor: null });
    await api.listArticles({ unread: true });
    const p = getCalledParams();
    expect(p.get('unread')).toBe('1');
    expect(p.has('read')).toBe(false);
  });

  it('sets read=1 for read filter, does not set unread', async () => {
    mockFetch(200, { items: [], next_cursor: null });
    await api.listArticles({ read: true });
    const p = getCalledParams();
    expect(p.get('read')).toBe('1');
    expect(p.has('unread')).toBe(false);
  });

  it('sets saved=1 for saved filter', async () => {
    mockFetch(200, { items: [], next_cursor: null });
    await api.listArticles({ saved: true });
    const p = getCalledParams();
    expect(p.get('saved')).toBe('1');
  });

  it('omits all filter flags when none specified', async () => {
    mockFetch(200, { items: [], next_cursor: null });
    await api.listArticles({ limit: 10 });
    const p = getCalledParams();
    expect(p.has('unread')).toBe(false);
    expect(p.has('read')).toBe(false);
    expect(p.has('saved')).toBe(false);
  });

  it('appends cursor when provided', async () => {
    mockFetch(200, { items: [], next_cursor: null });
    await api.listArticles({ cursor: 'abc123' });
    const p = getCalledParams();
    expect(p.get('cursor')).toBe('abc123');
  });

  it('appends category_id when provided', async () => {
    mockFetch(200, { items: [], next_cursor: null });
    await api.listArticles({ category_id: 7 });
    const p = getCalledParams();
    expect(p.get('category_id')).toBe('7');
  });
});

// ---------------------------------------------------------------------------
// ApiError class
// ---------------------------------------------------------------------------

describe('ApiError', () => {
  it('is an instance of Error', () => {
    const e = new ApiError(400, 'bad_request', 'oops');
    expect(e).toBeInstanceOf(Error);
  });

  it('exposes status and code as properties', () => {
    const e = new ApiError(409, 'conflict', 'slug exists');
    expect(e.status).toBe(409);
    expect(e.code).toBe('conflict');
    expect(e.message).toBe('slug exists');
  });
});

// ---------------------------------------------------------------------------
// BUG DOCUMENTATION: markAllReadInView only marks the loaded page
// ---------------------------------------------------------------------------
//
// AppState.markAllReadInView() collects IDs from this.articles (at most 50 in memory)
// and calls api.bulkMarkRead(ids). If the server has 200 unread articles, only
// 50 get marked. After the call the UI clears to empty, but 150 remain unread
// on the server and reappear on next refresh.
//
// Fix: add a server-side "mark all unread as read" endpoint, or drain all pages
// before marking.
describe('BUG: markAllReadInView pagination gap', () => {
  it('only sends IDs for the currently loaded page, not all unread on server', async () => {
    const capturedBodies: unknown[] = [];
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation((_url: string, init: RequestInit) => {
        if (init.method === 'POST' && typeof init.body === 'string') {
          capturedBodies.push(JSON.parse(init.body));
        }
        return Promise.resolve({
          ok: true,
          status: 200,
          text: () => Promise.resolve(JSON.stringify({ updated: 50 })),
        });
      }),
    );

    // Only 50 articles loaded - simulates one page out of 200 unread on server.
    const loadedIds = Array.from({ length: 50 }, (_, i) => i + 1);
    await api.bulkMarkRead(loadedIds);

    expect(capturedBodies).toHaveLength(1);
    const sentIds = (capturedBodies[0] as { ids: number[] }).ids;
    // Only 50 IDs sent - remaining 150 server-side unread articles are untouched.
    expect(sentIds).toHaveLength(50);
  });
});
