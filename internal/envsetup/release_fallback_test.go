package envsetup

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cs2-highlight-tool-v2/internal/release"
)

func seedHLAEReleaseSnapshotForTest(svc *Service, asset release.Asset) {
	svc.mu.Lock()
	svc.releaseSnapshot = &release.UnifiedLatest{
		Components: map[string]map[string][]release.ComponentCandidate{
			release.ComponentKeyHLAE: {
				release.SourceGitHub: {
					{
						Repo:   "advancedfx/advancedfx",
						Source: release.SourceGitHub,
						OK:     true,
						Info: &release.Info{
							TagName: "v2.0.0",
							Repo:    "advancedfx/advancedfx",
							Source:  release.SourceGitHub,
							Assets:  []release.Asset{asset},
						},
					},
				},
			},
		},
	}
	svc.state.SourceStep.Source = string(defaultDownloadSource())
	svc.mu.Unlock()
}

func TestCollectReleaseAssetCandidates_CNUsesMirrorThenURL(t *testing.T) {
	svc := New(t.TempDir(), "1.0.0")
	svc.Startup(nil)
	seedHLAEReleaseSnapshotForTest(svc, release.Asset{
		Name:        "hlae_2_0_0.zip",
		DownloadURL: "https://files.example.com/url.zip",
		URL:         "https://files.example.com/url.zip",
		GitHubURL:   "https://files.example.com/github.zip",
		MirrorURL:   "https://files.example.com/mirror.zip",
	})
	svc.mu.Lock()
	svc.state.SourceStep.CountryCode = "CN"
	svc.mu.Unlock()

	candidates, err := svc.collectReleaseAssetCandidates(componentHLAE, defaultDownloadSource(), release.SelectHLAEAsset)
	if err != nil {
		t.Fatalf("collectReleaseAssetCandidates error: %v", err)
	}
	if len(candidates) != 2 {
		t.Fatalf("candidate count = %d, want 2", len(candidates))
	}
	if candidates[0].URLKind != urlKindMirror || candidates[0].AssetURL != "https://files.example.com/mirror.zip" {
		t.Fatalf("candidate[0] = %#v", candidates[0])
	}
	if candidates[1].URLKind != urlKindDirect || candidates[1].AssetURL != "https://files.example.com/url.zip" {
		t.Fatalf("candidate[1] = %#v", candidates[1])
	}
}

func TestCollectReleaseAssetCandidates_UnknownCountryUsesMirrorThenURL(t *testing.T) {
	svc := New(t.TempDir(), "1.0.0")
	svc.Startup(nil)
	seedHLAEReleaseSnapshotForTest(svc, release.Asset{
		Name:        "hlae_2_0_0.zip",
		DownloadURL: "https://files.example.com/url.zip",
		URL:         "https://files.example.com/url.zip",
		GitHubURL:   "https://files.example.com/github.zip",
		MirrorURL:   "https://files.example.com/mirror.zip",
	})
	svc.mu.Lock()
	svc.state.SourceStep.CountryCode = ""
	svc.mu.Unlock()

	candidates, err := svc.collectReleaseAssetCandidates(componentHLAE, defaultDownloadSource(), release.SelectHLAEAsset)
	if err != nil {
		t.Fatalf("collectReleaseAssetCandidates error: %v", err)
	}
	if len(candidates) != 2 {
		t.Fatalf("candidate count = %d, want 2", len(candidates))
	}
	if candidates[0].URLKind != urlKindMirror || candidates[0].AssetURL != "https://files.example.com/mirror.zip" {
		t.Fatalf("candidate[0] = %#v", candidates[0])
	}
	if candidates[1].URLKind != urlKindDirect || candidates[1].AssetURL != "https://files.example.com/url.zip" {
		t.Fatalf("candidate[1] = %#v", candidates[1])
	}
}

func TestCollectReleaseAssetCandidates_OutsideCNUsesGitHubOnly(t *testing.T) {
	svc := New(t.TempDir(), "1.0.0")
	svc.Startup(nil)
	seedHLAEReleaseSnapshotForTest(svc, release.Asset{
		Name:        "hlae_2_0_0.zip",
		DownloadURL: "https://files.example.com/url.zip",
		URL:         "https://files.example.com/url.zip",
		GitHubURL:   "https://files.example.com/github.zip",
		MirrorURL:   "https://files.example.com/mirror.zip",
	})
	svc.mu.Lock()
	svc.state.SourceStep.CountryCode = "US"
	svc.mu.Unlock()

	candidates, err := svc.collectReleaseAssetCandidates(componentHLAE, defaultDownloadSource(), release.SelectHLAEAsset)
	if err != nil {
		t.Fatalf("collectReleaseAssetCandidates error: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("candidate count = %d, want 1", len(candidates))
	}
	if candidates[0].URLKind != urlKindGitHub || candidates[0].AssetURL != "https://files.example.com/github.zip" {
		t.Fatalf("candidate = %#v", candidates[0])
	}
}

func TestCollectReleaseAssetCandidates_OutsideCNMissingGitHubURLFails(t *testing.T) {
	svc := New(t.TempDir(), "1.0.0")
	svc.Startup(nil)
	seedHLAEReleaseSnapshotForTest(svc, release.Asset{
		Name:        "hlae_2_0_0.zip",
		DownloadURL: "https://files.example.com/url.zip",
		URL:         "https://files.example.com/url.zip",
		MirrorURL:   "https://files.example.com/mirror.zip",
	})
	svc.mu.Lock()
	svc.state.SourceStep.CountryCode = "US"
	svc.mu.Unlock()

	_, err := svc.collectReleaseAssetCandidates(componentHLAE, defaultDownloadSource(), release.SelectHLAEAsset)
	if err == nil {
		t.Fatal("collectReleaseAssetCandidates error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "github_url") {
		t.Fatalf("error = %v, want github_url hint", err)
	}
}

func TestDownloadAndInstallWithFallback_CNDoesNotAttemptGitHubURL(t *testing.T) {
	urlHits := 0
	mirrorHits := 0
	githubHits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/url.zip":
			urlHits++
		case "/mirror.zip":
			mirrorHits++
		case "/github.zip":
			githubHits++
		default:
			http.NotFound(w, r)
			return
		}
		http.Error(w, "down", http.StatusBadGateway)
	}))
	defer server.Close()

	svc := New(t.TempDir(), "1.0.0")
	svc.Startup(nil)
	seedHLAEReleaseSnapshotForTest(svc, release.Asset{
		Name:        "hlae_2_0_0.zip",
		DownloadURL: server.URL + "/url.zip",
		URL:         server.URL + "/url.zip",
		GitHubURL:   server.URL + "/github.zip",
		MirrorURL:   server.URL + "/mirror.zip",
	})
	svc.mu.Lock()
	svc.state.SourceStep.CountryCode = "CN"
	svc.mu.Unlock()

	candidates, err := svc.collectReleaseAssetCandidates(componentHLAE, defaultDownloadSource(), release.SelectHLAEAsset)
	if err != nil {
		t.Fatalf("collectReleaseAssetCandidates error: %v", err)
	}

	err = svc.downloadAndInstallWithFallback(componentHLAE, "v2.0.0", candidates, func(path string) error {
		return nil
	})
	if err == nil {
		t.Fatal("downloadAndInstallWithFallback error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "(url)") || !strings.Contains(err.Error(), "(mirror_url)") {
		t.Fatalf("error = %v, want url kind tags", err)
	}
	if strings.Contains(err.Error(), "github_url") {
		t.Fatalf("error = %v, should not include github_url attempt", err)
	}
	if urlHits != 1 || mirrorHits != 1 || githubHits != 0 {
		t.Fatalf("hits url=%d mirror=%d github=%d", urlHits, mirrorHits, githubHits)
	}
}
