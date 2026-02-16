package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	common "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	msg "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
)

type KillInfo struct {
	Round           int     `json:"round"`
	Tick            int     `json:"tick"`
	MapName         string  `json:"map_name"`
	VictimName      string  `json:"victim_name"`
	VictimID        int     `json:"victim_entity_id"`
	VictimSteamID   uint64  `json:"victim_steam_id"`
	VictimSide      string  `json:"victim_side"`
	VictimX         float64 `json:"victim_x"`
	VictimY         float64 `json:"victim_y"`
	VictimZ         float64 `json:"victim_z"`
	KillerName      string  `json:"killer_name"`
	KillerSteamID   uint64  `json:"killer_steam_id"`
	KillerSide      string  `json:"killer_side"`
	KillerX         float64 `json:"killer_x"`
	KillerY         float64 `json:"killer_y"`
	KillerZ         float64 `json:"killer_z"`
	WeaponName      string  `json:"weapon_name"`
	IsHeadshot      bool    `json:"is_headshot"`
	IsWallbang      bool    `json:"is_wallbang"`
	CanRender2DKill bool    `json:"can_render_2d_kill"`
}

type Segment struct {
	StartTick int        `json:"start_tick"`
	EndTick   int        `json:"end_tick"`
	Kills     []KillInfo `json:"kills"`
}

type PlayerInfo struct {
	Name     string `json:"name"`
	SteamID  uint64 `json:"steam_id"`
	EntityID int    `json:"entity_id"`
}

func parseDemoKills(demoPath string) (map[uint64]*PlayerInfo, map[int][]KillInfo, error) {
	f, err := os.Open(demoPath)
	if err != nil {
		return nil, nil, fmt.Errorf("打开 demo 失败: %w", err)
	}
	defer f.Close()

	parser := dem.NewParser(f)
	defer parser.Close()

	players := make(map[uint64]*PlayerInfo)
	kills := make(map[int][]KillInfo)
	currentRound := 0
	currentMapName := ""

	parser.RegisterNetMessageHandler(func(m *msg.CSVCMsg_ServerInfo) {
		if m == nil {
			return
		}
		if mapName := normalizeMapName(m.GetMapName()); mapName != "" {
			currentMapName = mapName
		}
	})
	parser.RegisterNetMessageHandler(func(m *msg.CNETMsg_SignonState) {
		if m == nil {
			return
		}
		if mapName := normalizeMapName(m.GetMapName()); mapName != "" {
			currentMapName = mapName
		}
	})

	// 注册回合开始事件
	parser.RegisterEventHandler(func(e events.RoundStart) {
		// TotalRoundsPlayed() 在首回合开始时是 0，这里统一转成从 1 开始的回合编号。
		currentRound = parser.GameState().TotalRoundsPlayed() + 1
	})

	// 注册击杀事件
	parser.RegisterEventHandler(func(e events.Kill) {
		if e.Killer == nil || e.Victim == nil {
			return
		}

		// 跳过热身
		if parser.GameState().IsWarmupPeriod() {
			return
		}

		// 记录玩家信息
		if _, exists := players[e.Killer.SteamID64]; !exists {
			players[e.Killer.SteamID64] = &PlayerInfo{
				Name:     e.Killer.Name,
				SteamID:  e.Killer.SteamID64,
				EntityID: e.Killer.EntityID,
			}
		}

		// 获取武器名称
		weaponName := "Unknown"
		if e.Weapon != nil {
			weaponName = e.Weapon.String()
		}

		killerPos := e.Killer.Position()
		victimPos := e.Victim.Position()
		canRender2D := hasRenderableCoordinates(killerPos.X, killerPos.Y, killerPos.Z) &&
			hasRenderableCoordinates(victimPos.X, victimPos.Y, victimPos.Z)

		// 记录击杀
		killInfo := KillInfo{
			Round:           currentRound,
			Tick:            parser.GameState().IngameTick(),
			MapName:         currentMapName,
			VictimName:      e.Victim.Name,
			VictimID:        e.Victim.EntityID,
			VictimSteamID:   e.Victim.SteamID64,
			VictimSide:      teamToSide(e.Victim.Team),
			VictimX:         victimPos.X,
			VictimY:         victimPos.Y,
			VictimZ:         victimPos.Z,
			KillerName:      e.Killer.Name,
			KillerSteamID:   e.Killer.SteamID64,
			KillerSide:      teamToSide(e.Killer.Team),
			KillerX:         killerPos.X,
			KillerY:         killerPos.Y,
			KillerZ:         killerPos.Z,
			WeaponName:      weaponName,
			IsHeadshot:      e.IsHeadshot,
			IsWallbang:      e.PenetratedObjects > 0,
			CanRender2DKill: canRender2D,
		}

		killsByRound := kills[int(e.Killer.SteamID64)]
		killsByRound = append(killsByRound, killInfo)
		kills[int(e.Killer.SteamID64)] = killsByRound
	})

	// 解析整个 demo
	if err := parser.ParseToEnd(); err != nil {
		return nil, nil, fmt.Errorf("解析 demo 失败: %w", err)
	}

	return players, kills, nil
}

func promptChoice(options []string, prompt string) int {
	reader := bufio.NewReader(os.Stdin)
	for {
		if prompt != "" {
			fmt.Print(prompt)
		}
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		idx, err := strconv.Atoi(input)
		if err == nil && idx >= 0 && idx < len(options) {
			return idx
		}
		printWarning("无效输入，请重试")
		if prompt != "" {
			fmt.Print(prompt)
		}
	}
}

func promptRounds(validRounds []int) []int {
	reader := bufio.NewReader(os.Stdin)
	for {
		colorYellow.Print("\n选择回合 (输入回合编号): ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		if input == "all" {
			return validRounds
		}

		parts := strings.Split(input, ",")
		var selected []int
		valid := true

		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			num, err := strconv.Atoi(part)
			if err != nil {
				valid = false
				break
			}

			found := false
			for _, r := range validRounds {
				if r == num {
					found = true
					break
				}
			}

			if !found {
				valid = false
				break
			}

			selected = append(selected, num)
		}

		if valid && len(selected) > 0 {
			return selected
		}

		printWarning("无效输入，请重试")
	}
}

func buildSegments(kills []KillInfo, preTicks, postTicks int) []Segment {
	if len(kills) == 0 {
		return nil
	}

	// 按 tick 排序
	sort.Slice(kills, func(i, j int) bool {
		return kills[i].Tick < kills[j].Tick
	})

	segments := []Segment{}

	for _, k := range kills {
		startTick := k.Tick - preTicks
		if startTick < 0 {
			startTick = 0
		}
		endTick := k.Tick + postTicks

		// 合并重叠的片段
		if len(segments) > 0 && startTick <= segments[len(segments)-1].EndTick {
			lastSeg := &segments[len(segments)-1]
			if endTick > lastSeg.EndTick {
				lastSeg.EndTick = endTick
			}
			lastSeg.Kills = append(lastSeg.Kills, k)
		} else {
			segments = append(segments, Segment{
				StartTick: startTick,
				EndTick:   endTick,
				Kills:     []KillInfo{k},
			})
		}
	}

	return segments
}

func buildVictimSegments(kills []KillInfo, preTicks, postTicks int) []Segment {
	if len(kills) == 0 {
		return nil
	}

	sortedKills := make([]KillInfo, len(kills))
	copy(sortedKills, kills)
	sort.Slice(sortedKills, func(i, j int) bool {
		return sortedKills[i].Tick < sortedKills[j].Tick
	})

	segments := make([]Segment, 0, len(sortedKills))
	for _, k := range sortedKills {
		startTick := k.Tick - preTicks
		if startTick < 0 {
			startTick = 0
		}
		endTick := k.Tick + postTicks
		segments = append(segments, Segment{
			StartTick: startTick,
			EndTick:   endTick,
			Kills:     []KillInfo{k},
		})
	}

	return segments
}

func segmentsToKills(segments []Segment) []KillInfo {
	if len(segments) == 0 {
		return nil
	}
	var kills []KillInfo
	for _, seg := range segments {
		if len(seg.Kills) > 0 {
			kills = append(kills, seg.Kills...)
		}
	}
	return kills
}

func normalizeMapName(name string) string {
	value := strings.TrimSpace(strings.ToLower(name))
	value = strings.TrimPrefix(value, "maps/")
	value = strings.TrimSuffix(value, ".vpk")
	return strings.TrimSpace(value)
}

func teamToSide(team common.Team) string {
	switch team {
	case common.TeamCounterTerrorists:
		return "ct"
	case common.TeamTerrorists:
		return "t"
	default:
		return "unknown"
	}
}

func hasRenderableCoordinates(x, y, z float64) bool {
	if math.IsNaN(x) || math.IsNaN(y) || math.IsNaN(z) {
		return false
	}
	if math.IsInf(x, 0) || math.IsInf(y, 0) || math.IsInf(z, 0) {
		return false
	}
	return !(x == 0 && y == 0 && z == 0)
}
