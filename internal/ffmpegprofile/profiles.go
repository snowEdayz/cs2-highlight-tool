package ffmpegprofile

import (
	"fmt"
	"sort"
	"strings"
)

const (
	UserPresetAuto = "auto"
	UserPresetC1   = "c1"
	UserPresetN1   = "n1"
	UserPresetA1   = "a1"
	UserPresetI1   = "i1"
)

const (
	EditQualityStandard = "standard"
	EditQualityHigh     = "high"
	EditQualityUltra    = "ultra"
)

const (
	internalPresetN1H264 = "n1_h264"
	internalPresetA1H264 = "a1_h264"
	internalPresetI1H264 = "i1_h264"
)

type Profile struct {
	ID         string
	Encoder    string
	HLAEParams string
}

func NormalizeUserPreset(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case UserPresetAuto:
		return UserPresetAuto
	case UserPresetC1:
		return UserPresetC1
	case UserPresetN1:
		return UserPresetN1
	case UserPresetA1:
		return UserPresetA1
	case UserPresetI1:
		return UserPresetI1
	default:
		return UserPresetAuto
	}
}

func IsUserPreset(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case UserPresetAuto, UserPresetC1, UserPresetN1, UserPresetA1, UserPresetI1:
		return true
	default:
		return false
	}
}

func SupportedUserPresets() []string {
	return []string{UserPresetAuto, UserPresetC1, UserPresetN1, UserPresetA1, UserPresetI1}
}

func NormalizeEditQuality(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case EditQualityStandard:
		return EditQualityStandard
	case EditQualityHigh:
		return EditQualityHigh
	case EditQualityUltra:
		return EditQualityUltra
	default:
		return EditQualityHigh
	}
}

func DefaultAutoResolveOrder() []string {
	return []string{
		UserPresetN1,
		UserPresetA1,
		UserPresetI1,
		internalPresetN1H264,
		internalPresetA1H264,
		internalPresetI1H264,
		UserPresetC1,
	}
}

func ManualResolveOrder(userPreset string) []string {
	switch NormalizeUserPreset(userPreset) {
	case UserPresetN1:
		return []string{UserPresetN1, internalPresetN1H264, UserPresetC1}
	case UserPresetA1:
		return []string{UserPresetA1, internalPresetA1H264, UserPresetC1}
	case UserPresetI1:
		return []string{UserPresetI1, internalPresetI1H264, UserPresetC1}
	case UserPresetC1:
		return []string{UserPresetC1}
	default:
		return DefaultAutoResolveOrder()
	}
}

func ResolveOrder(userPreset string) []string {
	normalized := NormalizeUserPreset(userPreset)
	if normalized == UserPresetAuto {
		return DefaultAutoResolveOrder()
	}
	return ManualResolveOrder(normalized)
}

func ProfileByID(id string) (Profile, bool) {
	id = strings.ToLower(strings.TrimSpace(id))
	profile, ok := profileCatalog[id]
	return profile, ok
}

func HLAEProfileByID(id string) (string, string, error) {
	profile, ok := ProfileByID(id)
	if !ok {
		return "", "", fmt.Errorf("不支持的 video_preset: %s", id)
	}
	return profile.ID, profile.HLAEParams, nil
}

func BuildEditEncodeArgs(profileID string, quality string) ([]string, error) {
	profileID = strings.ToLower(strings.TrimSpace(profileID))
	quality = NormalizeEditQuality(quality)

	switch profileID {
	case UserPresetC1:
		crf := "16"
		switch quality {
		case EditQualityStandard:
			crf = "18"
		case EditQualityUltra:
			crf = "14"
		}
		return []string{"-c:v", "libx264", "-preset", "fast", "-crf", crf, "-pix_fmt", "yuv420p"}, nil
	case UserPresetN1:
		qp := "14"
		switch quality {
		case EditQualityStandard:
			qp = "20"
		case EditQualityUltra:
			qp = "10"
		}
		return []string{"-c:v", "hevc_nvenc", "-preset", "medium", "-tune", "hq", "-rc", "constqp", "-qp", qp, "-g", "120", "-pix_fmt", "yuv420p"}, nil
	case UserPresetA1:
		qp := "12"
		switch quality {
		case EditQualityStandard:
			qp = "20"
		case EditQualityUltra:
			qp = "8"
		}
		return []string{"-c:v", "hevc_amf", "-usage", "0", "-quality", "0", "-rc", "cqp", "-qp", qp, "-pix_fmt", "yuv420p"}, nil
	case UserPresetI1:
		qv := "12"
		switch quality {
		case EditQualityStandard:
			qv = "20"
		case EditQualityUltra:
			qv = "8"
		}
		return []string{"-c:v", "hevc_qsv", "-q:v", qv, "-preset", "veryfast", "-g", "120", "-pix_fmt", "nv12"}, nil
	case internalPresetN1H264:
		qp := "16"
		switch quality {
		case EditQualityStandard:
			qp = "22"
		case EditQualityUltra:
			qp = "12"
		}
		return []string{"-c:v", "h264_nvenc", "-preset", "medium", "-tune", "hq", "-rc", "constqp", "-qp", qp, "-g", "120", "-pix_fmt", "yuv420p"}, nil
	case internalPresetA1H264:
		qp := "14"
		switch quality {
		case EditQualityStandard:
			qp = "22"
		case EditQualityUltra:
			qp = "10"
		}
		return []string{"-c:v", "h264_amf", "-usage", "0", "-quality", "0", "-rc", "cqp", "-qp", qp, "-pix_fmt", "yuv420p"}, nil
	case internalPresetI1H264:
		qv := "14"
		switch quality {
		case EditQualityStandard:
			qv = "22"
		case EditQualityUltra:
			qv = "10"
		}
		return []string{"-c:v", "h264_qsv", "-q:v", qv, "-preset", "veryfast", "-g", "120", "-pix_fmt", "nv12"}, nil
	default:
		return nil, fmt.Errorf("不支持的剪辑编码预设: %s", profileID)
	}
}

func BuildRecordingEncodeArgs(profileID string, quality string) (string, error) {
	profileID = strings.ToLower(strings.TrimSpace(profileID))
	quality = NormalizeEditQuality(quality)

	switch profileID {
	case UserPresetC1:
		crf := "4"
		switch quality {
		case EditQualityStandard:
			crf = "10"
		case EditQualityUltra:
			crf = "2"
		}
		return "-c:v libx264 -preset 1 -crf " + crf + " -qmax 20 -g 120 -keyint_min 1 -pix_fmt yuv420p -x264-params ref=3:me=hex:subme=3:merange=12:b-adapt=1:aq-mode=2:aq-strength=0.9:no-fast-pskip=1", nil
	case UserPresetN1:
		qp := "14"
		switch quality {
		case EditQualityStandard:
			qp = "20"
		case EditQualityUltra:
			qp = "10"
		}
		return "-c:v hevc_nvenc -g 120 -preset medium -tune hq -rc constqp -qp " + qp + " -pix_fmt yuv420p", nil
	case UserPresetA1:
		qp := "12"
		switch quality {
		case EditQualityStandard:
			qp = "20"
		case EditQualityUltra:
			qp = "8"
		}
		return "-c:v hevc_amf -usage 0 -quality 0 -rc cqp -qp " + qp + " -pix_fmt yuv420p", nil
	case UserPresetI1:
		qv := "12"
		switch quality {
		case EditQualityStandard:
			qv = "20"
		case EditQualityUltra:
			qv = "8"
		}
		return "-c:v hevc_qsv -q:v " + qv + " -preset veryfast -g 120 -pix_fmt nv12", nil
	case internalPresetN1H264:
		qp := "16"
		switch quality {
		case EditQualityStandard:
			qp = "22"
		case EditQualityUltra:
			qp = "12"
		}
		return "-c:v h264_nvenc -g 120 -preset medium -tune hq -rc constqp -qp " + qp + " -pix_fmt yuv420p", nil
	case internalPresetA1H264:
		qp := "14"
		switch quality {
		case EditQualityStandard:
			qp = "22"
		case EditQualityUltra:
			qp = "10"
		}
		return "-c:v h264_amf -usage 0 -quality 0 -rc cqp -qp " + qp + " -pix_fmt yuv420p", nil
	case internalPresetI1H264:
		qv := "14"
		switch quality {
		case EditQualityStandard:
			qv = "22"
		case EditQualityUltra:
			qv = "10"
		}
		return "-c:v h264_qsv -q:v " + qv + " -preset veryfast -g 120 -pix_fmt nv12", nil
	default:
		return "", fmt.Errorf("不支持的 video_preset: %s", profileID)
	}
}

func KnownEncoders() []string {
	out := make([]string, 0, len(profileCatalog))
	seen := make(map[string]struct{}, len(profileCatalog))
	for _, profile := range profileCatalog {
		encoder := strings.TrimSpace(profile.Encoder)
		if encoder == "" {
			continue
		}
		if _, ok := seen[encoder]; ok {
			continue
		}
		seen[encoder] = struct{}{}
		out = append(out, encoder)
	}
	sort.Strings(out)
	return out
}

var profileCatalog = map[string]Profile{
	UserPresetC1: {
		ID:         UserPresetC1,
		Encoder:    "libx264",
		HLAEParams: "-c:v libx264 -preset 1 -crf 4 -qmax 20 -g 120 -keyint_min 1 -pix_fmt yuv420p -x264-params ref=3:me=hex:subme=3:merange=12:b-adapt=1:aq-mode=2:aq-strength=0.9:no-fast-pskip=1",
	},
	UserPresetN1: {
		ID:         UserPresetN1,
		Encoder:    "hevc_nvenc",
		HLAEParams: "-c:v hevc_nvenc -g 120 -preset medium -tune hq -rc constqp -qp 14 -pix_fmt yuv420p",
	},
	UserPresetA1: {
		ID:         UserPresetA1,
		Encoder:    "hevc_amf",
		HLAEParams: "-c:v hevc_amf -usage 0 -quality 0 -rc cqp -qp 12 -pix_fmt yuv420p",
	},
	UserPresetI1: {
		ID:         UserPresetI1,
		Encoder:    "hevc_qsv",
		HLAEParams: "-c:v hevc_qsv -q:v 12 -preset veryfast -g 120 -pix_fmt nv12",
	},
	internalPresetN1H264: {
		ID:         internalPresetN1H264,
		Encoder:    "h264_nvenc",
		HLAEParams: "-c:v h264_nvenc -g 120 -preset medium -tune hq -rc constqp -qp 16 -pix_fmt yuv420p",
	},
	internalPresetA1H264: {
		ID:         internalPresetA1H264,
		Encoder:    "h264_amf",
		HLAEParams: "-c:v h264_amf -usage 0 -quality 0 -rc cqp -qp 14 -pix_fmt yuv420p",
	},
	internalPresetI1H264: {
		ID:         internalPresetI1H264,
		Encoder:    "h264_qsv",
		HLAEParams: "-c:v h264_qsv -q:v 14 -preset veryfast -g 120 -pix_fmt nv12",
	},
}
