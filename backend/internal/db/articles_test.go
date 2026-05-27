package db

import (
	"testing"
	"time"
)

func TestCursorRoundTrip(t *testing.T) {
	now := time.Date(2026, 5, 27, 14, 30, 0, 123456789, time.UTC)
	id := int64(987654321)
	enc := encodeCursor(now, id)
	gotT, gotID, err := decodeCursor(enc)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !gotT.Equal(now) {
		t.Errorf("time mismatch: got %v want %v", gotT, now)
	}
	if gotID != id {
		t.Errorf("id mismatch: got %d want %d", gotID, id)
	}
}

func TestCursorBadInput(t *testing.T) {
	if _, _, err := decodeCursor("not-base64!"); err == nil {
		t.Error("expected error on bad cursor")
	}
}
