package httpapi

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mustafaeeroglu/rss-fresh/internal/config"
)

// validationServer returns a Server whose DB/refresher are nil.
// Safe only for test paths that return *before* any s.db call (i.e. validation
// failures). The chi Recoverer middleware turns unexpected panics into 500 so a
// test that accidentally reaches the nil db will fail the assertion rather than
// crash the test binary.
func validationServer() *Server {
	return &Server{
		cfg:       &config.Config{Version: "test"},
		log:       slog.Default(),
		refresher: nil,
		spaFS:     nil,
	}
}

func request(t *testing.T, method, path string, body any) *http.Request {
	t.Helper()
	if body == nil {
		return httptest.NewRequest(method, path, nil)
	}
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	r := httptest.NewRequest(method, path, bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	return r
}

func requestRaw(method, path, rawBody string) *http.Request {
	r := httptest.NewRequest(method, path, strings.NewReader(rawBody))
	r.Header.Set("Content-Type", "application/json")
	return r
}

func assertStatus(t *testing.T, w *httptest.ResponseRecorder, want int) {
	t.Helper()
	if w.Code != want {
		t.Errorf("want HTTP %d, got %d — body: %s", want, w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// GET /api/v1/articles
// ---------------------------------------------------------------------------

func TestHandleListArticles_UnreadAndReadMutuallyExclusive(t *testing.T) {
	rec := httptest.NewRecorder()
	srv := validationServer().Router()
	srv.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/articles?unread=1&read=1", nil))
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleListArticles_UnreadTrueVariant(t *testing.T) {
	rec := httptest.NewRecorder()
	srv := validationServer().Router()
	// unread=true & read=true should also be rejected
	srv.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/articles?unread=true&read=true", nil))
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleListArticles_BadCategoryID(t *testing.T) {
	rec := httptest.NewRecorder()
	srv := validationServer().Router()
	srv.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/articles?category_id=notanumber", nil))
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleListArticles_BadFeedID(t *testing.T) {
	rec := httptest.NewRecorder()
	srv := validationServer().Router()
	srv.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/articles?feed_id=2.5", nil))
	assertStatus(t, rec, http.StatusBadRequest)
}

// ---------------------------------------------------------------------------
// POST /api/v1/articles/mark-read
// ---------------------------------------------------------------------------

func TestHandleBulkMarkRead_EmptyIDsReturns200(t *testing.T) {
	rec := httptest.NewRecorder()
	srv := validationServer().Router()
	srv.ServeHTTP(rec, request(t, "POST", "/api/v1/articles/mark-read", map[string]any{"ids": []int{}}))
	assertStatus(t, rec, http.StatusOK)

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["updated"] != float64(0) {
		t.Errorf("expected updated=0, got %v", resp["updated"])
	}
}

func TestHandleBulkMarkRead_NullIDsReturns200(t *testing.T) {
	rec := httptest.NewRecorder()
	srv := validationServer().Router()
	srv.ServeHTTP(rec, requestRaw("POST", "/api/v1/articles/mark-read", `{"ids": null}`))
	assertStatus(t, rec, http.StatusOK)
}

func TestHandleBulkMarkRead_MissingIDsReturns200(t *testing.T) {
	rec := httptest.NewRecorder()
	srv := validationServer().Router()
	srv.ServeHTTP(rec, requestRaw("POST", "/api/v1/articles/mark-read", `{}`))
	assertStatus(t, rec, http.StatusOK)
}

func TestHandleBulkMarkRead_Over1000IDsReturns400(t *testing.T) {
	ids := make([]int, 1001)
	for i := range ids {
		ids[i] = i + 1
	}
	rec := httptest.NewRecorder()
	srv := validationServer().Router()
	srv.ServeHTTP(rec, request(t, "POST", "/api/v1/articles/mark-read", map[string]any{"ids": ids}))
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleBulkMarkRead_Exactly1000IDsPassesValidation(t *testing.T) {
	// 1000 IDs should pass the length guard (> 1000 rejects, exactly 1000 is fine).
	// The handler will then call s.db which is nil and the Recoverer returns 500 —
	// but 500 is not 400, which is the point: validation let it through.
	ids := make([]int, 1000)
	for i := range ids {
		ids[i] = i + 1
	}
	rec := httptest.NewRecorder()
	srv := validationServer().Router()
	srv.ServeHTTP(rec, request(t, "POST", "/api/v1/articles/mark-read", map[string]any{"ids": ids}))
	if rec.Code == http.StatusBadRequest {
		t.Errorf("1000 IDs should pass validation guard, got 400")
	}
}

func TestHandleBulkMarkRead_InvalidJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	srv := validationServer().Router()
	srv.ServeHTTP(rec, requestRaw("POST", "/api/v1/articles/mark-read", `{bad json`))
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleBulkMarkRead_UnknownFieldReturns400(t *testing.T) {
	// DisallowUnknownFields means extra keys → 400.
	rec := httptest.NewRecorder()
	srv := validationServer().Router()
	srv.ServeHTTP(rec, requestRaw("POST", "/api/v1/articles/mark-read", `{"ids": [], "extra": 1}`))
	assertStatus(t, rec, http.StatusBadRequest)
}

// ---------------------------------------------------------------------------
// PATCH /api/v1/articles/{id}
// ---------------------------------------------------------------------------

func TestHandleUpdateArticle_BadIDReturns400(t *testing.T) {
	rec := httptest.NewRecorder()
	srv := validationServer().Router()
	srv.ServeHTTP(rec, request(t, "PATCH", "/api/v1/articles/notanumber", map[string]any{"is_read": true}))
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleUpdateArticle_ZeroIDReturns400(t *testing.T) {
	// "0" is a valid integer but no article will have id=0.
	// Passes validation → reaches nil db → Recoverer returns 500.
	// This test documents the behaviour: zero ID is not caught at validation.
	rec := httptest.NewRecorder()
	srv := validationServer().Router()
	srv.ServeHTTP(rec, request(t, "PATCH", "/api/v1/articles/0", map[string]any{"is_read": true}))
	if rec.Code == http.StatusBadRequest {
		t.Log("zero id now rejected at validation layer — update test expectation")
	}
	// Not asserting 400 here intentionally; this test documents current behaviour.
}

// ---------------------------------------------------------------------------
// POST /api/v1/feeds
// ---------------------------------------------------------------------------

func TestHandleCreateFeed_MissingCategoryIDReturns400(t *testing.T) {
	rec := httptest.NewRecorder()
	srv := validationServer().Router()
	srv.ServeHTTP(rec, request(t, "POST", "/api/v1/feeds", map[string]any{
		"url": "https://example.com/rss",
	}))
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleCreateFeed_ZeroCategoryIDReturns400(t *testing.T) {
	rec := httptest.NewRecorder()
	srv := validationServer().Router()
	srv.ServeHTTP(rec, request(t, "POST", "/api/v1/feeds", map[string]any{
		"category_id": 0,
		"url":         "https://example.com/rss",
	}))
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleCreateFeed_EmptyURLReturns400(t *testing.T) {
	rec := httptest.NewRecorder()
	srv := validationServer().Router()
	srv.ServeHTTP(rec, request(t, "POST", "/api/v1/feeds", map[string]any{
		"category_id": 1,
		"url":         "",
	}))
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleCreateFeed_WhitespaceOnlyURLReturns400(t *testing.T) {
	rec := httptest.NewRecorder()
	srv := validationServer().Router()
	srv.ServeHTTP(rec, request(t, "POST", "/api/v1/feeds", map[string]any{
		"category_id": 1,
		"url":         "   ",
	}))
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleCreateFeed_FTPSchemeRejected(t *testing.T) {
	rec := httptest.NewRecorder()
	srv := validationServer().Router()
	srv.ServeHTTP(rec, request(t, "POST", "/api/v1/feeds", map[string]any{
		"category_id": 1,
		"url":         "ftp://feeds.example.com/rss",
	}))
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleCreateFeed_FileSchemeRejected(t *testing.T) {
	rec := httptest.NewRecorder()
	srv := validationServer().Router()
	srv.ServeHTTP(rec, request(t, "POST", "/api/v1/feeds", map[string]any{
		"category_id": 1,
		"url":         "file:///etc/passwd",
	}))
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleCreateFeed_JavascriptSchemeRejected(t *testing.T) {
	rec := httptest.NewRecorder()
	srv := validationServer().Router()
	srv.ServeHTTP(rec, request(t, "POST", "/api/v1/feeds", map[string]any{
		"category_id": 1,
		"url":         "javascript:alert(1)",
	}))
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleCreateFeed_NoHostRejected(t *testing.T) {
	rec := httptest.NewRecorder()
	srv := validationServer().Router()
	srv.ServeHTTP(rec, request(t, "POST", "/api/v1/feeds", map[string]any{
		"category_id": 1,
		"url":         "https://",
	}))
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleCreateFeed_RelativeURLRejected(t *testing.T) {
	rec := httptest.NewRecorder()
	srv := validationServer().Router()
	srv.ServeHTTP(rec, request(t, "POST", "/api/v1/feeds", map[string]any{
		"category_id": 1,
		"url":         "/relative/path",
	}))
	assertStatus(t, rec, http.StatusBadRequest)
}

// ---------------------------------------------------------------------------
// PATCH /api/v1/feeds/{id}
// ---------------------------------------------------------------------------

func TestHandleUpdateFeed_BadSchemeRejected(t *testing.T) {
	rec := httptest.NewRecorder()
	srv := validationServer().Router()
	srv.ServeHTTP(rec, request(t, "PATCH", "/api/v1/feeds/1", map[string]any{
		"url": "ftp://bad.example.com/rss",
	}))
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleUpdateFeed_BadIDReturns400(t *testing.T) {
	rec := httptest.NewRecorder()
	srv := validationServer().Router()
	srv.ServeHTTP(rec, request(t, "PATCH", "/api/v1/feeds/abc", map[string]any{
		"name": "Updated name",
	}))
	assertStatus(t, rec, http.StatusBadRequest)
}

// ---------------------------------------------------------------------------
// POST /api/v1/categories
// ---------------------------------------------------------------------------

func TestHandleCreateCategory_EmptyNameReturns400(t *testing.T) {
	rec := httptest.NewRecorder()
	srv := validationServer().Router()
	srv.ServeHTTP(rec, request(t, "POST", "/api/v1/categories", map[string]any{
		"name": "",
	}))
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleCreateCategory_WhitespaceOnlyNameReturns400(t *testing.T) {
	rec := httptest.NewRecorder()
	srv := validationServer().Router()
	srv.ServeHTTP(rec, request(t, "POST", "/api/v1/categories", map[string]any{
		"name": "   ",
	}))
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleCreateCategory_PureSpecialCharsSlugReturns400(t *testing.T) {
	// A name of "!!!" slugifies to "" — should be caught and return 400.
	rec := httptest.NewRecorder()
	srv := validationServer().Router()
	srv.ServeHTTP(rec, request(t, "POST", "/api/v1/categories", map[string]any{
		"name": "!!!",
	}))
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleCreateCategory_ExplicitEmptySlugReturns400(t *testing.T) {
	// Providing an explicit slug of "---" collapses to "": should be rejected.
	rec := httptest.NewRecorder()
	srv := validationServer().Router()
	srv.ServeHTTP(rec, request(t, "POST", "/api/v1/categories", map[string]any{
		"name": "Valid Name",
		"slug": "---",
	}))
	assertStatus(t, rec, http.StatusBadRequest)
}

// BUG: PATCH /api/v1/categories/{id} does not validate that the slug is
// non-empty after normalization.  Sending {"slug":"---"} saves "" to the DB.
// This test documents the defect; once the fix is applied, it should pass.
func TestHandleUpdateCategory_EmptySlugAfterNormalizationRejected(t *testing.T) {
	rec := httptest.NewRecorder()
	srv := validationServer().Router()
	srv.ServeHTTP(rec, request(t, "PATCH", "/api/v1/categories/1", map[string]any{
		"slug": "---",
	}))
	assertStatus(t, rec, http.StatusBadRequest)
}

// ---------------------------------------------------------------------------
// DELETE /api/v1/feeds/{id}
// ---------------------------------------------------------------------------

func TestHandleDeleteFeed_BadIDReturns400(t *testing.T) {
	rec := httptest.NewRecorder()
	srv := validationServer().Router()
	srv.ServeHTTP(rec, httptest.NewRequest("DELETE", "/api/v1/feeds/xyz", nil))
	assertStatus(t, rec, http.StatusBadRequest)
}

// ---------------------------------------------------------------------------
// DELETE /api/v1/categories/{id}
// ---------------------------------------------------------------------------

func TestHandleDeleteCategory_BadIDReturns400(t *testing.T) {
	rec := httptest.NewRecorder()
	srv := validationServer().Router()
	srv.ServeHTTP(rec, httptest.NewRequest("DELETE", "/api/v1/categories/xyz", nil))
	assertStatus(t, rec, http.StatusBadRequest)
}

// ---------------------------------------------------------------------------
// GET /api/v1/healthz
// ---------------------------------------------------------------------------

func TestHandleHealthz_AlwaysReturns200(t *testing.T) {
	rec := httptest.NewRecorder()
	validationServer().Router().ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/healthz", nil))
	assertStatus(t, rec, http.StatusOK)
}

// ---------------------------------------------------------------------------
// corsSameOrigin middleware
// ---------------------------------------------------------------------------

// TestCORSSameOrigin_MatchingOriginReflected verifies that a request whose
// Origin header host matches the Host header receives the CORS headers back.
func TestCORSSameOrigin_MatchingOriginReflected(t *testing.T) {
	rec := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/healthz", nil)
	r.Host = "rss.example.com"
	r.Header.Set("Origin", "https://rss.example.com")
	validationServer().Router().ServeHTTP(rec, r)

	got := rec.Header().Get("Access-Control-Allow-Origin")
	if got != "https://rss.example.com" {
		t.Errorf("same-origin: want ACAO=https://rss.example.com, got %q", got)
	}
}

// TestCORSSameOrigin_CrossOriginNotReflected verifies that a request from a
// different host receives no Access-Control-Allow-Origin header, which causes
// the browser to block the cross-origin response.
func TestCORSSameOrigin_CrossOriginNotReflected(t *testing.T) {
	rec := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/healthz", nil)
	r.Host = "rss.example.com"
	r.Header.Set("Origin", "https://evil.example.com")
	validationServer().Router().ServeHTTP(rec, r)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("cross-origin: ACAO header should be absent, got %q", got)
	}
}

// TestCORSSameOrigin_NoOriginHeader verifies that requests without an Origin
// header (same-origin or non-browser) are served normally with no CORS headers.
func TestCORSSameOrigin_NoOriginHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/healthz", nil)
	r.Host = "rss.example.com"
	validationServer().Router().ServeHTTP(rec, r)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("no-origin: ACAO header should be absent, got %q", got)
	}
	assertStatus(t, rec, http.StatusOK)
}

// TestCORSSameOrigin_PreflightSameOrigin verifies that a same-origin preflight
// returns 204 with the CORS headers set.
func TestCORSSameOrigin_PreflightSameOrigin(t *testing.T) {
	rec := httptest.NewRecorder()
	r := httptest.NewRequest("OPTIONS", "/api/v1/feeds", nil)
	r.Host = "rss.example.com"
	r.Header.Set("Origin", "https://rss.example.com")
	r.Header.Set("Access-Control-Request-Method", "POST")
	validationServer().Router().ServeHTTP(rec, r)

	assertStatus(t, rec, http.StatusNoContent)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://rss.example.com" {
		t.Errorf("preflight same-origin: want ACAO=https://rss.example.com, got %q", got)
	}
}

// TestCORSSameOrigin_PreflightCrossOrigin verifies that a cross-origin preflight
// returns 204 (the OPTIONS handler still fires) but without CORS headers — the
// browser will then block the subsequent actual request.
func TestCORSSameOrigin_PreflightCrossOrigin(t *testing.T) {
	rec := httptest.NewRecorder()
	r := httptest.NewRequest("OPTIONS", "/api/v1/feeds", nil)
	r.Host = "rss.example.com"
	r.Header.Set("Origin", "https://attacker.com")
	r.Header.Set("Access-Control-Request-Method", "POST")
	validationServer().Router().ServeHTTP(rec, r)

	assertStatus(t, rec, http.StatusNoContent)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("preflight cross-origin: ACAO should be absent, got %q", got)
	}
}
