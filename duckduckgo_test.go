package duckduckgo

import (
	"testing"
)

func TestSearch(t *testing.T) {
	dg := New()

	res, err := dg.Search("golang", 20)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(res) == 0 {
		t.Fatal("expected non-empty result")
	}
}
