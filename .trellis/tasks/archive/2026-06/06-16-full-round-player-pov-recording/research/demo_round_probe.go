package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	common "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
)

type playerStats struct {
	Name    string
	SteamID uint64
	Kills   int
	Deaths  int
	Slot    int
}

type deathEvent struct {
	Tick          int
	VictimName    string
	VictimSteamID uint64
	VictimSlot    int
	KillerName    string
	KillerSteamID uint64
	Weapon        string
}

type roundInfo struct {
	Number        int
	StartTick     int
	FreezeEndTick int
	EndTick       int
	OfficialEnd   int
	Reason        string
	Winner        string
	Slots         map[uint64]int
	Deaths        []deathEvent
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: go run demo_round_probe.go <demo.dem> [target name or steamid]")
		os.Exit(2)
	}
	demoPath := os.Args[1]
	targetQuery := ""
	if len(os.Args) >= 3 {
		targetQuery = strings.TrimSpace(strings.Join(os.Args[2:], " "))
	}

	players, rounds, tickRate, err := parseDemo(demoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse demo: %v\n", err)
		os.Exit(1)
	}
	if len(players) == 0 {
		fmt.Println("no players found")
		return
	}

	playerList := make([]playerStats, 0, len(players))
	for _, player := range players {
		playerList = append(playerList, *player)
	}
	sort.Slice(playerList, func(i, j int) bool {
		if playerList[i].Kills == playerList[j].Kills {
			return playerList[i].Name < playerList[j].Name
		}
		return playerList[i].Kills > playerList[j].Kills
	})

	fmt.Printf("demo=%s\n", demoPath)
	fmt.Printf("tick_rate=%.2f rounds=%d players=%d\n\n", tickRate, len(rounds), len(playerList))
	fmt.Println("players:")
	for _, player := range playerList {
		fmt.Printf("  %-24s steam=%d kills=%d deaths=%d last_slot=%d\n", player.Name, player.SteamID, player.Kills, player.Deaths, player.Slot)
	}

	target := resolveTarget(playerList, targetQuery)
	if target == nil {
		fmt.Println("\nno target matched")
		return
	}

	fmt.Printf("\nselected target: %s steam=%d kills=%d deaths=%d last_slot=%d\n", target.Name, target.SteamID, target.Kills, target.Deaths, target.Slot)
	fmt.Println("\nrecording simulation:")

	for _, round := range rounds {
		recordStart := round.FreezeEndTick
		startLabel := "freeze_end"
		if recordStart <= 0 {
			recordStart = round.StartTick
			startLabel = "round_start"
		}
		targetDeath := findTargetDeath(round.Deaths, target.SteamID, recordStart, round.EndTick)
		ignoredTargetDeath := findIgnoredTargetDeath(round.Deaths, target.SteamID, recordStart, round.EndTick)
		recordEnd := 0
		endLabel := ""
		if targetDeath != nil {
			recordEnd = targetDeath.Tick
			endLabel = "target_death"
		} else if round.EndTick > 0 {
			recordEnd = round.EndTick
			endLabel = "round_end"
		} else {
			recordEnd = round.OfficialEnd
			endLabel = "round_official_end"
		}

		status := "survived"
		if targetDeath != nil {
			status = fmt.Sprintf("died_by=%s weapon=%s", targetDeath.KillerName, targetDeath.Weapon)
		} else if ignoredTargetDeath != nil {
			status = fmt.Sprintf(
				"survived; ignored_post_round_death tick=%d killer=%s weapon=%s",
				ignoredTargetDeath.Tick,
				ignoredTargetDeath.KillerName,
				ignoredTargetDeath.Weapon,
			)
		}
		targetSlot := round.Slots[target.SteamID]
		fmt.Printf(
			"  round=%02d start=%d freeze_end=%d end=%d official_end=%d target_slot=%d record=%d->%d (%s->%s) %s\n",
			round.Number,
			round.StartTick,
			round.FreezeEndTick,
			round.EndTick,
			round.OfficialEnd,
			targetSlot,
			recordStart,
			recordEnd,
			startLabel,
			endLabel,
			status,
		)
	}
}

func parseDemo(demoPath string) (map[uint64]*playerStats, []roundInfo, float64, error) {
	f, err := os.Open(demoPath)
	if err != nil {
		return nil, nil, 0, err
	}
	defer f.Close()

	parser := dem.NewParser(f)
	defer parser.Close()

	players := make(map[uint64]*playerStats)
	roundsByNumber := make(map[int]*roundInfo)
	roundOrder := make([]int, 0, 32)
	currentRound := 0

	ensurePlayer := func(p *common.Player) *playerStats {
		if p == nil || p.SteamID64 == 0 {
			return nil
		}
		stats := players[p.SteamID64]
		if stats == nil {
			stats = &playerStats{SteamID: p.SteamID64}
			players[p.SteamID64] = stats
		}
		if p.Name != "" {
			stats.Name = p.Name
		}
		if p.UserID > 0 {
			stats.Slot = p.UserID + 1
		}
		return stats
	}

	roundForEvent := func() int {
		if currentRound > 0 {
			return currentRound
		}
		return parser.GameState().TotalRoundsPlayed() + 1
	}

	ensureRound := func(number int) *roundInfo {
		if number <= 0 {
			return nil
		}
		round := roundsByNumber[number]
		if round == nil {
			round = &roundInfo{Number: number, Slots: make(map[uint64]int)}
			roundsByNumber[number] = round
			roundOrder = append(roundOrder, number)
		}
		return round
	}

	snapshotPlayingSlots := func(round *roundInfo) {
		if round == nil {
			return
		}
		for _, player := range parser.GameState().Participants().Playing() {
			stats := ensurePlayer(player)
			if stats == nil || stats.SteamID == 0 || stats.Slot <= 0 {
				continue
			}
			round.Slots[stats.SteamID] = stats.Slot
		}
	}

	parser.RegisterEventHandler(func(_ events.RoundStart) {
		if parser.GameState().IsWarmupPeriod() {
			return
		}
		currentRound = parser.GameState().TotalRoundsPlayed() + 1
		round := ensureRound(currentRound)
		if round != nil {
			round.StartTick = positiveTick(parser.GameState().IngameTick())
			snapshotPlayingSlots(round)
		}
	})

	parser.RegisterEventHandler(func(_ events.RoundFreezetimeEnd) {
		if parser.GameState().IsWarmupPeriod() {
			return
		}
		round := ensureRound(roundForEvent())
		if round != nil {
			round.FreezeEndTick = positiveTick(parser.GameState().IngameTick())
			snapshotPlayingSlots(round)
		}
	})

	parser.RegisterEventHandler(func(e events.RoundEnd) {
		if parser.GameState().IsWarmupPeriod() {
			return
		}
		round := ensureRound(roundForEvent())
		if round != nil {
			round.EndTick = positiveTick(parser.GameState().IngameTick())
			round.Reason = fmt.Sprint(e.Reason)
			round.Winner = teamName(e.Winner)
		}
	})

	parser.RegisterEventHandler(func(_ events.RoundEndOfficial) {
		if parser.GameState().IsWarmupPeriod() {
			return
		}
		round := ensureRound(roundForEvent())
		if round != nil {
			round.OfficialEnd = positiveTick(parser.GameState().IngameTick())
		}
	})

	parser.RegisterEventHandler(func(e events.Kill) {
		if parser.GameState().IsWarmupPeriod() {
			return
		}
		if e.Killer == nil || e.Victim == nil || e.Victim.SteamID64 == 0 {
			return
		}
		if stats := ensurePlayer(e.Killer); stats != nil {
			stats.Kills++
		}
		if stats := ensurePlayer(e.Victim); stats != nil {
			stats.Deaths++
		}
		weapon := "unknown"
		if e.Weapon != nil {
			weapon = e.Weapon.String()
		}
		victimSlot := 0
		if e.Victim.UserID > 0 {
			victimSlot = e.Victim.UserID + 1
		}
		round := ensureRound(roundForEvent())
		if round != nil {
			round.Deaths = append(round.Deaths, deathEvent{
				Tick:          positiveTick(parser.GameState().IngameTick()),
				VictimName:    e.Victim.Name,
				VictimSteamID: e.Victim.SteamID64,
				VictimSlot:    victimSlot,
				KillerName:    e.Killer.Name,
				KillerSteamID: e.Killer.SteamID64,
				Weapon:        weapon,
			})
		}
	})

	if err := parser.ParseToEnd(); err != nil {
		return nil, nil, 0, err
	}

	sort.Ints(roundOrder)
	rounds := make([]roundInfo, 0, len(roundOrder))
	for _, number := range roundOrder {
		rounds = append(rounds, *roundsByNumber[number])
	}
	return players, rounds, parser.TickRate(), nil
}

func resolveTarget(players []playerStats, query string) *playerStats {
	if len(players) == 0 {
		return nil
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return &players[0]
	}
	if id, err := strconv.ParseUint(query, 10, 64); err == nil {
		for i := range players {
			if players[i].SteamID == id {
				return &players[i]
			}
		}
	}
	needle := strings.ToLower(query)
	for i := range players {
		if strings.ToLower(players[i].Name) == needle {
			return &players[i]
		}
	}
	for i := range players {
		if strings.Contains(strings.ToLower(players[i].Name), needle) {
			return &players[i]
		}
	}
	return nil
}

func findTargetDeath(deaths []deathEvent, steamID uint64, startTick int, endTick int) *deathEvent {
	for i := range deaths {
		if deaths[i].VictimSteamID == steamID &&
			deaths[i].Tick >= startTick &&
			(endTick <= 0 || deaths[i].Tick <= endTick) {
			return &deaths[i]
		}
	}
	return nil
}

func findIgnoredTargetDeath(deaths []deathEvent, steamID uint64, startTick int, endTick int) *deathEvent {
	for i := range deaths {
		if deaths[i].VictimSteamID != steamID {
			continue
		}
		if deaths[i].Tick < startTick || (endTick > 0 && deaths[i].Tick > endTick) {
			return &deaths[i]
		}
	}
	return nil
}

func positiveTick(tick int) int {
	if tick < 0 {
		return 0
	}
	return tick
}

func teamName(team common.Team) string {
	switch team {
	case common.TeamCounterTerrorists:
		return "ct"
	case common.TeamTerrorists:
		return "t"
	default:
		return fmt.Sprint(team)
	}
}
