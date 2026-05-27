package endpoints

import (
	"errors"
	"testing"
)

func setGeoIPCountryCodeResolverForTest(t *testing.T, resolver func() (string, error)) {
	t.Helper()
	geoIPCountryCodeResolver = resolver
	t.Cleanup(func() {
		geoIPCountryCodeResolver = fetchGeoIPCountryCode
	})
}

func TestPreferredReleaseSource_CNUsesGitHub(t *testing.T) {
	setGeoIPCountryCodeResolverForTest(t, func() (string, error) { return "CN", nil })
	source, countryCode, err := PreferredReleaseSource()
	if err != nil {
		t.Fatalf("PreferredReleaseSource() error = %v", err)
	}
	if source != releaseSourceGitHub {
		t.Fatalf("source = %q, want %q", source, releaseSourceGitHub)
	}
	if countryCode != "CN" {
		t.Fatalf("countryCode = %q, want %q", countryCode, "CN")
	}
}

func TestPreferredReleaseSource_OutsideCNUsesGitHub(t *testing.T) {
	setGeoIPCountryCodeResolverForTest(t, func() (string, error) { return "US", nil })
	source, countryCode, err := PreferredReleaseSource()
	if err != nil {
		t.Fatalf("PreferredReleaseSource() error = %v", err)
	}
	if source != releaseSourceGitHub {
		t.Fatalf("source = %q, want %q", source, releaseSourceGitHub)
	}
	if countryCode != "US" {
		t.Fatalf("countryCode = %q, want %q", countryCode, "US")
	}
}

func TestPreferredReleaseSource_GeoIPFailureFallsBackToGitHub(t *testing.T) {
	setGeoIPCountryCodeResolverForTest(t, func() (string, error) { return "", errors.New("geoip unavailable") })
	source, countryCode, err := PreferredReleaseSource()
	if err == nil {
		t.Fatal("PreferredReleaseSource() error = nil, want non-nil")
	}
	if source != releaseSourceGitHub {
		t.Fatalf("source = %q, want %q", source, releaseSourceGitHub)
	}
	if countryCode != "" {
		t.Fatalf("countryCode = %q, want empty", countryCode)
	}
}

func TestBuildGHProxyURL_OnlyWrapsGitHub(t *testing.T) {
	githubURL := "https://github.com/advancedfx/advancedfx/releases/download/v2.189.9/hlae_2_189_9.zip"
	want := "https://gh-proxy.org/" + githubURL
	if got := BuildGHProxyURL(githubURL); got != want {
		t.Fatalf("BuildGHProxyURL(github) = %q, want %q", got, want)
	}

	nonGitHubURL := "https://download.example.com/advancedfx/hlae_2_189_9.zip"
	if got := BuildGHProxyURL(nonGitHubURL); got != nonGitHubURL {
		t.Fatalf("BuildGHProxyURL(non-github) = %q, want %q", got, nonGitHubURL)
	}
}

func TestManualURLFor_UsesSourceSpecificReleasePage(t *testing.T) {
	if got := ManualURLFor("hlae", releaseSourceGitHub); got != "https://github.com/advancedfx/advancedfx/releases" {
		t.Fatalf("ManualURLFor(hlae, github) = %q", got)
	}
	if got := ManualURLFor("self_update", releaseSourceGitHub); got != "https://github.com/hkslover/cs2-highlight-tool/releases" {
		t.Fatalf("ManualURLFor(self_update, github) = %q", got)
	}
}

func TestSupportedReleaseSources_GitHubOnly(t *testing.T) {
	sources := SupportedReleaseSources()
	if len(sources) != 1 || sources[0] != releaseSourceGitHub {
		t.Fatalf("sources = %#v, want [github]", sources)
	}
}
