package app

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"cs2-highlight-tool-v2/internal/clipsjson"
	"cs2-highlight-tool-v2/internal/demo"
)

func (a *App) PreviewFullRoundPOV(demoPath, playerSteamID string) (*demo.FullRoundPOVPlan, error) {
	steamID, err := strconv.ParseUint(playerSteamID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("无效的 SteamID: %w", err)
	}
	return demo.ParseFullRoundPOVPlan(demoPath, steamID)
}

func buildFullRoundPOVSegmentsForPlugin(plan *demo.FullRoundPOVPlan, settings ClipSettings, tickRate float64) []clipsjson.FullRoundPOVSegment {
	if plan == nil || len(plan.Segments) == 0 {
		return nil
	}
	if tickRate <= 0 {
		tickRate = 64
	}
	segments := make([]clipsjson.FullRoundPOVSegment, 0, len(plan.Segments))
	for _, segment := range plan.Segments {
		if segment.RecordStartTick < 0 || segment.RecordEndTick < segment.RecordStartTick || segment.TargetSlot <= 0 {
			continue
		}
		endTick := fullRoundPOVRecordEndTick(segment, settings, tickRate)
		segments = append(segments, clipsjson.FullRoundPOVSegment{
			Round:              segment.Round,
			StartTick:          segment.RecordStartTick,
			EndTick:            endTick,
			Target:             strconv.Itoa(segment.TargetSlot),
			SpecMode:           1,
			SourceID:           buildFullRoundPOVSourceID(segment.Round, plan.PlayerSteamID),
			PlayerName:         strings.TrimSpace(plan.PlayerName),
			PlayerSteamID:      strings.TrimSpace(plan.PlayerSteamID),
			EndReason:          strings.TrimSpace(segment.EndReason),
			EnableVoice:        settings.EnableVoice,
			EnableSpecShowXray: settings.EnableSpecShowXray,
		})
	}
	return segments
}

func fullRoundPOVRecordEndTick(segment demo.FullRoundPOVSegment, settings ClipSettings, tickRate float64) int {
	endTick := segment.RecordEndTick
	if strings.TrimSpace(segment.EndReason) != demo.FullRoundPOVEndTargetDeath {
		return endTick
	}
	if settings.KillerPostSeconds <= 0 || tickRate <= 0 {
		return endTick
	}
	postTicks := int(math.Round(settings.KillerPostSeconds * tickRate))
	if postTicks <= 0 {
		return endTick
	}
	endTick += postTicks
	if segment.RoundEndTick > 0 && endTick > segment.RoundEndTick {
		return segment.RoundEndTick
	}
	return endTick
}

func buildFullRoundPOVSourceID(round int, playerSteamID string) string {
	steamID := strings.TrimSpace(playerSteamID)
	if steamID == "" {
		steamID = "unknown"
	}
	return "full_round_pov:r" + strconv.Itoa(round) + ":p" + steamID
}
