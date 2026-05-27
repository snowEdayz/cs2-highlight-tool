package envsetup

import (
	"net/mail"
	"net/url"
	"os"
	"strings"

	"cs2-highlight-tool-v2/internal/release"
)

const debugStartupAdsEnv = "CS2_DEBUG_STARTUP_ADS"

func mapReleaseAds(items []release.AdItem) []StartupAd {
	if len(items) == 0 {
		return nil
	}
	out := make([]StartupAd, 0, len(items))
	for _, item := range items {
		out = append(out, StartupAd{
			ID:        item.ID,
			Enabled:   item.Enabled,
			Placement: item.Placement,
			ClickURL:  item.ClickURL,
			Sponsor:   item.Sponsor,
			Title:     item.Title,
			RichHTML:  item.RichHTML,
			ImageURL:  item.ImageURL,
			ImageAlt:  item.ImageAlt,
		})
	}
	return out
}

func normalizeExternalOpenURL(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", false
	}
	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	switch scheme {
	case "http", "https":
		if strings.TrimSpace(parsed.Host) == "" {
			return "", false
		}
		return parsed.String(), true
	case "mailto":
		return normalizeMailtoURL(parsed)
	default:
		return "", false
	}
}

func normalizeMailtoURL(parsed *url.URL) (string, bool) {
	if parsed == nil {
		return "", false
	}
	target := strings.TrimSpace(parsed.Opaque)
	if target == "" {
		target = strings.TrimSpace(parsed.Host)
	}
	if target == "" {
		target = strings.TrimSpace(strings.TrimPrefix(parsed.Path, "/"))
	}
	if target == "" || strings.ContainsAny(target, "\r\n") {
		return "", false
	}

	addressList := target
	if idx := strings.Index(addressList, "?"); idx >= 0 {
		addressList = addressList[:idx]
	}
	addressList = strings.TrimSpace(addressList)
	if addressList == "" {
		return "", false
	}
	for _, part := range strings.Split(addressList, ",") {
		addr := strings.TrimSpace(part)
		if addr == "" {
			return "", false
		}
		if _, err := mail.ParseAddress(addr); err != nil {
			return "", false
		}
	}

	return parsed.String(), true
}

func shouldUseDebugStartupAds() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(debugStartupAdsEnv)))
	switch value {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func debugStartupAds() []StartupAd {
	imageA := "data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' width='256' height='144'><rect width='100%25' height='100%25' fill='%230b3b2e'/><text x='50%25' y='50%25' dominant-baseline='middle' text-anchor='middle' fill='%23d1fae5' font-size='20'>广告位招租</text></svg>"
	imageB := "data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' width='256' height='144'><rect width='100%25' height='100%25' fill='%230b3b2e'/><text x='50%25' y='50%25' dominant-baseline='middle' text-anchor='middle' fill='%23d1fae5' font-size='20'>广告位招租</text></svg>"
	imageC := "data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' width='256' height='144'><rect width='100%25' height='100%25' fill='%233f2b96'/><text x='50%25' y='50%25' dominant-baseline='middle' text-anchor='middle' fill='%23f5f3ff' font-size='20'>Sponsor C</text></svg>"

	return []StartupAd{
		{
			ID:        "debug_sponsored_card_1",
			Enabled:   true,
			Placement: release.AdPlacementMainStepsTopBanner,
			ClickURL:  "mailto:hk_snow@yeah.net",
			Sponsor:   "",
			Title:     "广告位招租",
			RichHTML:  "<a href=\"mailto:hk_snow@yeah.net\" target=\"_blank\">联系我们</a>",
			ImageURL:  imageA,
			ImageAlt:  "广告位招租",
		},
		{
			ID:        "debug_sponsored_card_2",
			Enabled:   true,
			Placement: release.AdPlacementMainStepsTopBanner,
			ClickURL:  "https://example.com/sponsor/beta",
			Sponsor:   "Beta Tools",
			Title:     "测试广告位 B：多条轮播切换",
			RichHTML:  "<p>支持富文本，包含 <a href=\"https://example.com/sponsor/beta\" target=\"_blank\">外链示例</a>。</p>",
			ImageURL:  imageB,
			ImageAlt:  "Debug Sponsor B",
		},
		{
			ID:        "debug_sponsored_card_3",
			Enabled:   true,
			Placement: release.AdPlacementMainStepsTopBanner,
			ClickURL:  "https://example.com/sponsor/gamma",
			Sponsor:   "Gamma Hardware",
			Title:     "测试广告位 C：长标题与副文案",
			RichHTML:  "<p>这是一条更长的说明文本，用于观察在不同窗口宽度下的换行表现。</p>",
			ImageURL:  imageC,
			ImageAlt:  "Debug Sponsor C",
		},
	}
}
