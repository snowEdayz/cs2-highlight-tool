package release

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchUnifiedLatest_DoesNotSendAuthorizationHeader(t *testing.T) {
	receivedAuth := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"project": {"id":"cs2-highlight-tool-v2","name":"cs2-highlight-tool-v2"},
			"dependencies": {
				"advancedfx": {
					"name":"advancedfx",
					"repo":"advancedfx/advancedfx",
					"latest_tag":"v1.2.3",
					"latest": {
						"tag_name":"v1.2.3",
						"assets":[{"name":"hlae_1_2_3.zip","url":"dl.example.com/hlae.zip","github_url":"https://github.com/advancedfx/advancedfx/releases/download/v1.2.3/hlae_1_2_3.zip"}]
					}
				}
			}
		}`))
	}))
	defer server.Close()

	info, err := FetchUnifiedLatest(server.URL)
	if err != nil {
		t.Fatalf("FetchUnifiedLatest error: %v", err)
	}
	if receivedAuth != "" {
		t.Fatalf("Authorization header = %q, want empty", receivedAuth)
	}
	hlae, ok := info.ComponentInfoBySource("hlae", SourceGitHub)
	if !ok || hlae == nil || hlae.TagName != "v1.2.3" {
		t.Fatalf("unexpected info: %#v", info)
	}
	if got := AssetDownloadURL(hlae.Assets[0]); got != "https://dl.example.com/hlae.zip" {
		t.Fatalf("asset download url = %q", got)
	}
}

func TestFetchUnifiedLatest_ComponentMappingAndAssetURLs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"project": {"id":"cs2-highlight-tool-v2","name":"cs2-highlight-tool-v2"},
			"dependencies": {
				"advancedfx": {
					"name":"advancedfx",
					"repo":"advancedfx/advancedfx",
					"latest_tag":"v2.189.9",
					"latest": {
						"tag_name":"v2.189.9",
						"assets":[{
							"name":"hlae_2_189_9.zip",
							"url":"dl.snowblog.xyz/cs2-highlight-tool-v2/advancedfx/advancedfx/v2.189.9/hlae_2_189_9.zip",
							"github_url":"https://github.com/advancedfx/advancedfx/releases/download/v2.189.9/hlae_2_189_9.zip",
							"mirror_url":"https://gh-proxy.org/https://github.com/advancedfx/advancedfx/releases/download/v2.189.9/hlae_2_189_9.zip"
						}]
					}
				},
				"cs2-server-plugin": {
					"name":"cs2-server-plugin",
					"repo":"hkslover/cs2-server-plugin",
					"latest_tag":"v0.0.2",
					"latest": {
						"tag_name":"v0.0.2",
						"assets":[{
							"name":"cs2-server-plugin-v0.0.2-windows.zip",
							"url":"dl.snowblog.xyz/cs2-highlight-tool-v2/hkslover/cs2-server-plugin/v0.0.2/cs2-server-plugin-v0.0.2-windows.zip",
							"github_url":"https://github.com/hkslover/cs2-server-plugin/releases/download/v0.0.2/cs2-server-plugin-v0.0.2-windows.zip",
							"mirror_url":"https://gh-proxy.org/https://github.com/hkslover/cs2-server-plugin/releases/download/v0.0.2/cs2-server-plugin-v0.0.2-windows.zip"
						}]
					}
				}
			}
		}`))
	}))
	defer server.Close()

	latest, err := FetchUnifiedLatest(server.URL)
	if err != nil {
		t.Fatalf("FetchUnifiedLatest error: %v", err)
	}

	hlae, ok := latest.ComponentInfoBySource("hlae", SourceGitHub)
	if !ok || hlae == nil {
		t.Fatalf("hlae info missing: %#v", hlae)
	}
	if got := hlae.Assets[0].URL; got != "https://dl.snowblog.xyz/cs2-highlight-tool-v2/advancedfx/advancedfx/v2.189.9/hlae_2_189_9.zip" {
		t.Fatalf("hlae asset url = %q", got)
	}

	plugin, ok := latest.ComponentInfoBySource("plugin", SourceGitHub)
	if !ok || plugin == nil {
		t.Fatalf("plugin info missing: %#v", plugin)
	}
	if plugin.Source != SourceGitHub {
		t.Fatalf("plugin source = %q, want %q", plugin.Source, SourceGitHub)
	}
	if plugin.Repo != "hkslover/cs2-server-plugin" {
		t.Fatalf("plugin repo = %q", plugin.Repo)
	}
	asset := plugin.Assets[0]
	if asset.GitHubURL == "" || asset.MirrorURL == "" {
		t.Fatalf("plugin asset links incomplete: %#v", asset)
	}

	if _, ok := latest.ComponentInfoBySource("self_update", SourceGitHub); ok {
		t.Fatal("self_update should not be available when manifest has no app dependency")
	}
	if got := latest.ComponentSourceError("self_update", SourceGitHub); got != "" {
		t.Fatalf("self_update source error = %q, want empty", got)
	}
}

func TestSelectAppExeAsset(t *testing.T) {
	info := &Info{Assets: []Asset{
		{Name: "notes.txt", DownloadURL: "https://example.test/notes.txt"},
		{Name: "cs2-highlight-tool-windows-amd64.exe", DownloadURL: "https://example.test/app.exe"},
	}}
	asset, ok := SelectAppExeAsset(info)
	if !ok || AssetDownloadURL(asset) != "https://example.test/app.exe" {
		t.Fatalf("SelectAppExeAsset() = %#v, %v", asset, ok)
	}
}

func TestSelectHLAEAsset(t *testing.T) {
	info := &Info{Assets: []Asset{
		{Name: "hlae_2.170.0.zip.asc", DownloadURL: "https://example.test/sig"},
		{Name: "hlae_2.170.0.zip", DownloadURL: "https://example.test/hlae.zip"},
	}}
	asset, ok := SelectHLAEAsset(info)
	if !ok || AssetDownloadURL(asset) != "https://example.test/hlae.zip" {
		t.Fatalf("SelectHLAEAsset() = %#v, %v", asset, ok)
	}
}

func TestSelectPluginAssetPrefersZip(t *testing.T) {
	info := &Info{Assets: []Asset{
		{Name: "plugin.dll", DownloadURL: "https://example.test/plugin.dll"},
		{Name: "plugin.zip", DownloadURL: "https://example.test/plugin.zip"},
	}}
	asset, ok := SelectPluginAsset(info)
	if !ok || AssetDownloadURL(asset) != "https://example.test/plugin.zip" {
		t.Fatalf("SelectPluginAsset() = %#v, %v", asset, ok)
	}
}
