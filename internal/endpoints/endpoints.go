package endpoints

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	FFmpegFixedDownloadURL = "https://gitee.com/hkslover/ffmpeg_release/releases/download/v8.0.1/ffmpeg-8.0.1-essentials_build.7z"
	ffmpegManualURL        = "https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.7z"
	defaultReleaseAPIURL   = "https://updates.snowblog.xyz/cs2-highlight-tool-v2.json"
	defaultGHProxyDomain   = "gh-proxy.org"
	ipSBGeoIPURL           = "https://api.ip.sb/geoip"

	releaseSourceGitHub = "github"
)

var (
	geoIPCountryCodeResolver = fetchGeoIPCountryCode
)

var componentRepoBySource = map[string]map[string]string{
	"app": {
		releaseSourceGitHub: "hkslover/cs2-highlight-tool",
	},
	"hlae": {
		releaseSourceGitHub: "advancedfx/advancedfx",
	},
	"plugin": {
		releaseSourceGitHub: "hkslover/cs2-server-plugin",
	},
}

func SupportedReleaseSources() []string {
	return []string{releaseSourceGitHub}
}

func normalizeSource(source string) string {
	source = strings.ToLower(strings.TrimSpace(source))
	switch source {
	case releaseSourceGitHub:
		return source
	default:
		return releaseSourceGitHub
	}
}

func PreferredReleaseSource() (source string, countryCode string, err error) {
	countryCode, err = geoIPCountryCodeResolver()
	if err != nil {
		return releaseSourceGitHub, "", err
	}
	countryCode = strings.ToUpper(strings.TrimSpace(countryCode))
	return releaseSourceGitHub, countryCode, nil
}

func releaseAPIOverride(source string) string {
	source = strings.ToUpper(strings.TrimSpace(source))
	candidates := []string{
		fmt.Sprintf("CS2_RELEASE_API_URL_%s", source),
		"CS2_RELEASE_API_URL",
		fmt.Sprintf("CS2_APP_RELEASE_API_URL_%s", source),
		"CS2_APP_RELEASE_API_URL",
	}
	for _, key := range candidates {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func appReleasePageOverride(source string) string {
	source = strings.ToUpper(strings.TrimSpace(source))
	sourceSpecificKey := fmt.Sprintf("CS2_APP_RELEASE_PAGE_URL_%s", source)
	if value := strings.TrimSpace(os.Getenv(sourceSpecificKey)); value != "" {
		return value
	}
	return strings.TrimSpace(os.Getenv("CS2_APP_RELEASE_PAGE_URL"))
}

func APIURLFor(kind string, source string) string {
	_ = kind
	source = normalizeSource(source)
	if override := releaseAPIOverride(source); override != "" {
		return override
	}
	return defaultReleaseAPIURL
}

func normalizeReleasePageKind(kind string) string {
	if kind == "self_update" {
		return "app"
	}
	return kind
}

func componentRepo(kind string, source string) string {
	source = normalizeSource(source)
	bySource, ok := componentRepoBySource[normalizeReleasePageKind(kind)]
	if !ok {
		return ""
	}
	if repo := strings.TrimSpace(bySource[source]); repo != "" {
		return repo
	}
	return strings.TrimSpace(bySource[releaseSourceGitHub])
}

func ReleasePageURL(repo string, source string) string {
	repo = strings.TrimSpace(repo)
	if repo == "" {
		return ""
	}
	source = normalizeSource(source)
	switch source {
	case releaseSourceGitHub:
		return "https://github.com/" + repo + "/releases"
	default:
		return ""
	}
}

func ManualURLFor(kind string, source string) string {
	if kind == "ffmpeg" {
		return ffmpegManualURL
	}
	source = normalizeSource(source)
	if normalizeReleasePageKind(kind) == "app" {
		if override := appReleasePageOverride(source); override != "" {
			return override
		}
	}
	repo := componentRepo(kind, source)
	return ReleasePageURL(repo, source)
}

func BuildGHProxyURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return rawURL
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	if host != "github.com" {
		return rawURL
	}
	if strings.HasPrefix(strings.ToLower(rawURL), "https://"+defaultGHProxyDomain+"/") {
		return rawURL
	}
	return "https://" + defaultGHProxyDomain + "/" + rawURL
}

// Deprecated: kept for compatibility with existing call sites.
func RewriteGitHubDownloadURL(rawURL string) string {
	return rawURL
}

func fetchGeoIPCountryCode() (string, error) {
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, ipSBGeoIPURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("geoip status code: %d", resp.StatusCode)
	}
	var payload struct {
		CountryCode string `json:"country_code"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1024*1024)).Decode(&payload); err != nil {
		return "", err
	}
	countryCode := strings.TrimSpace(payload.CountryCode)
	if countryCode == "" {
		return "", fmt.Errorf("geoip country_code is empty")
	}
	return strings.ToUpper(countryCode), nil
}
