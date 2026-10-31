package resolvers

import (
	"testing"
)

func TestResolvers(t *testing.T) {
	// Pixeldrain
	pdURL := "https://pixeldrain.com/u/abc12345"
	resolvedPD, err := ResolveLink(pdURL)
	if err != nil {
		t.Fatalf("Pixeldrain resolve error: %v", err)
	}
	expectedPD := "https://pixeldrain.com/api/file/abc12345?download"
	if resolvedPD != expectedPD {
		t.Errorf("expected '%s', got '%s'", expectedPD, resolvedPD)
	}

	// Buzzheavier
	bhURL := "https://buzzheavier.com/f/xyz987"
	resolvedBH, err := ResolveLink(bhURL)
	if err != nil {
		t.Fatalf("Buzzheavier resolve error: %v", err)
	}
	expectedBH := "https://buzzheavier.com/f/xyz987/download"
	if resolvedBH != expectedBH {
		t.Errorf("expected '%s', got '%s'", expectedBH, resolvedBH)
	}

	// Generic link fallback
	genURL := "https://example.com/test.zip"
	resolvedGen, err := ResolveLink(genURL)
	if err != nil {
		t.Fatalf("Generic resolve error: %v", err)
	}
	if resolvedGen != genURL {
		t.Errorf("expected unchanged '%s', got '%s'", genURL, resolvedGen)
	}
}
