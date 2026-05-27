package envsetup

import (
	"errors"
	"testing"
)

func TestResolveDownloadSource_UsesUnifiedSource(t *testing.T) {
	orig := preferredReleaseSourceFn
	t.Cleanup(func() {
		preferredReleaseSourceFn = orig
	})
	preferredReleaseSourceFn = func() (string, string, error) {
		return "CN", "CN", nil
	}

	result, err := resolveDownloadSource(nil)
	if err != nil {
		t.Fatalf("resolveDownloadSource error: %v", err)
	}
	if result.Source != normalizeDownloadSource("github") {
		t.Fatalf("source = %q, want %q", result.Source, normalizeDownloadSource("github"))
	}
	if result.CountryCode != "CN" {
		t.Fatalf("country_code = %q, want %q", result.CountryCode, "CN")
	}
	if result.Message == "" {
		t.Fatal("message should not be empty")
	}
}

func TestResolveDownloadSource_GeoIPFailureDefaultsToGitHub(t *testing.T) {
	orig := preferredReleaseSourceFn
	t.Cleanup(func() {
		preferredReleaseSourceFn = orig
	})
	preferredReleaseSourceFn = func() (string, string, error) {
		return "", "", errors.New("geoip unavailable")
	}

	result, err := resolveDownloadSource(nil)
	if err != nil {
		t.Fatalf("resolveDownloadSource error: %v", err)
	}
	if result.Source != normalizeDownloadSource("github") {
		t.Fatalf("source = %q, want %q", result.Source, normalizeDownloadSource("github"))
	}
	if result.CountryCode != "" {
		t.Fatalf("country_code = %q, want empty", result.CountryCode)
	}
}
