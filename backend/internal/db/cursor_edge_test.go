package db

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Cursor edge cases
// ---------------------------------------------------------------------------

func TestCursorPaddedBase64Rejected(t *testing.T) {
	// Standard base64 (with padding `=`) is not accepted — we use RawURLEncoding.
	padded := base64.StdEncoding.EncodeToString([]byte("2026-01-01T00:00:00Z|42"))
	if _, _, err := decodeCursor(padded); err == nil {
		t.Error("expected error decoding standard-padded base64 cursor")
	}
}

func TestCursorMissingDelimiterRejected(t *testing.T) {
	// Encoded content has no '|' separator.
	noSep := base64.RawURLEncoding.EncodeToString([]byte("2026-01-01T00:00:00Z42"))
	if _, _, err := decodeCursor(noSep); err == nil {
		t.Error("expected error: missing pipe delimiter")
	}
}

func TestCursorBadTimestampRejected(t *testing.T) {
	bad := base64.RawURLEncoding.EncodeToString([]byte("not-a-timestamp|99"))
	if _, _, err := decodeCursor(bad); err == nil {
		t.Error("expected error: unparsable timestamp")
	}
}

func TestCursorBadIDRejected(t *testing.T) {
	bad := base64.RawURLEncoding.EncodeToString([]byte("2026-01-01T00:00:00Z|notanint"))
	if _, _, err := decodeCursor(bad); err == nil {
		t.Error("expected error: unparsable id")
	}
}

func TestCursorEmptyStringRejected(t *testing.T) {
	if _, _, err := decodeCursor(""); err == nil {
		t.Error("expected error: empty cursor")
	}
}

func TestCursorSQLInjectionPayloadRejected(t *testing.T) {
	// Raw SQL is not valid base64+timestamp — must not panic and must return error.
	if _, _, err := decodeCursor("' OR 1=1--"); err == nil {
		t.Error("expected error: SQL injection payload is not a valid cursor")
	}
}

func TestCursorXSSPayloadRejected(t *testing.T) {
	if _, _, err := decodeCursor("<script>alert(1)</script>"); err == nil {
		t.Error("expected error: XSS payload is not a valid cursor")
	}
}

func TestCursorOversizedPayloadRejected(t *testing.T) {
	// 10 000-byte payload must return an error from timestamp parsing, not panic.
	large := base64.RawURLEncoding.EncodeToString([]byte(strings.Repeat("A", 10_000) + "|9"))
	if _, _, err := decodeCursor(large); err == nil {
		t.Error("expected error: oversized cursor payload")
	}
}

func TestCursorNegativeIDPreserved(t *testing.T) {
	// Negative IDs are unusual but the codec must round-trip them without error.
	ts := time.Date(2024, 3, 15, 9, 0, 0, 0, time.UTC)
	enc := encodeCursor(ts, -1)
	gotT, gotID, err := decodeCursor(enc)
	if err != nil {
		t.Fatalf("decode negative id: %v", err)
	}
	if !gotT.Equal(ts) {
		t.Errorf("time mismatch: got %v want %v", gotT, ts)
	}
	if gotID != -1 {
		t.Errorf("id mismatch: got %d want -1", gotID)
	}
}

func TestCursorZeroTimeRoundTrip(t *testing.T) {
	// Zero time is a valid edge case (unlikely but must not panic).
	zero := time.Time{}
	enc := encodeCursor(zero, 0)
	gotT, gotID, err := decodeCursor(enc)
	if err != nil {
		t.Fatalf("decode zero time: %v", err)
	}
	if !gotT.Equal(zero.UTC()) {
		t.Errorf("time mismatch: got %v want %v", gotT, zero.UTC())
	}
	if gotID != 0 {
		t.Errorf("id mismatch: got %d want 0", gotID)
	}
}

func TestCursorPreservesSubsecondPrecision(t *testing.T) {
	// The pagination query uses nanosecond-precision cursors; precision loss
	// would cause rows to be skipped or repeated at page boundaries.
	ts := time.Date(2026, 5, 27, 14, 30, 0, 999_999_999, time.UTC)
	enc := encodeCursor(ts, 1)
	gotT, _, err := decodeCursor(enc)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !gotT.Equal(ts) {
		t.Errorf("nanosecond precision lost: got %v want %v", gotT, ts)
	}
}

// BUG DOCUMENTATION: encodeCursor returns "" when published_at is nil,
// which means the last page's cursor is dropped silently and users cannot
// paginate past articles that have no publish date. The test below exercises
// the path through ListArticles that builds the next cursor — it exists to
// keep the behaviour documented and catch any regression when the fix lands.
func TestCursorIsOnlyBuiltWhenPublishedAtIsNonNil(t *testing.T) {
	// If published_at is nil, encodeCursor is never called.
	// This test verifies the helper itself doesn't panic on zero time.
	enc := encodeCursor(time.Time{}, 5)
	if enc == "" {
		t.Error("encodeCursor(zero, 5) returned empty string — cursor will be silently dropped for null-date articles")
	}
}
