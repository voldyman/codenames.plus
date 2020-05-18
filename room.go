package main

import (
	"math/rand"

	"github.com/sirupsen/logrus"
)

var (
	ConsensusSingle = "single"
	ConsensusAll    = "consensus"
	ConsensusTypes  = buildSet(ConsensusAll, ConsensusSingle)
)

var (
	ModeCasual = "casual"
	ModeTimed  = "timed"
	ModeTypes  = buildSet(ModeCasual, ModeTimed)
)

var (
	DifficultyNormal = "normal"
	DifficultyHard   = "hard"
	DifficultyTypes  = buildSet(DifficultyNormal, DifficultyHard)
)

type BoardType int

const (
	BoardTypeDefault BoardType = 1 << iota
	BoardTypeDuet
	BoardTypeUndercover
	BoardTypeCustom
	BoardTypeNsfw
)

var (
	PlayerRoleGuesser   = "guesser"
	PlayerRoleSpyMaster = "spymaster"
	PlayerRoleSpectator = "spectator"
)

type Room struct {
	Name       string             `json:"room"`
	Password   string             `json:"password"`
	Players    map[string]*Player `json:"players"`
	Difficulty string             `json:"difficulty"`
	Mode       string             `json:"mode"`
	Consesus   string             `json:"consensus"`
	Game       *Game              `json:"game"`

	boardType BoardType
}

func NewRoom(name, password string) *Room {
	return &Room{
		Name:       name,
		Password:   password,
		Players:    map[string]*Player{},
		Difficulty: DifficultyNormal,
		Mode:       ModeCasual,
		Consesus:   ConsensusSingle,
		Game:       NewGame(BoardTypeDefault),
		boardType:  BoardTypeDefault,
	}
}

func (r *Room) Join(playerID, name string) bool {
	if _, ok := r.Player(playerID); ok {
		return true
	}

	if r.hasPlayer(name) {
		name = name + "_"
	}
	randTeam := TeamBlue
	if rand.Intn(20)%2 == 0 {
		randTeam = TeamRed
	}
	r.Players[playerID] = &Player{
		ID:            playerID,
		NickName:      name,
		Room:          r.Name,
		Team:          randTeam,
		GuessProposal: nil,
		Role:          PlayerRoleGuesser,
	}
	return true
}

func (r *Room) hasPlayer(name string) bool {
	for _, p := range r.Players {
		if p.NickName == name {
			return true
		}
	}
	return false
}

func (r *Room) Leave(playerID string) bool {
	delete(r.Players, playerID)
	return true
}

func (r *Room) ChangeTeam(playerID, team string) {
	player, ok := r.Player(playerID)
	if !ok {
		log.WithFields(logrus.Fields{
			"PlayerID": playerID,
			"RoomName": r.Name,
			"Team":     team,
		}).Warn("player tried to change teams but they are not in the room")

		return
	}
	player.Team = team
}

func (r *Room) RandomizeTeams(playerID string) {
	players := []*Player{}
	for _, p := range r.Players {
		players = append(players, p)
	}

	for i := 0; i < len(players)/2; i++ {
		players[i].Team = TeamBlue
	}

	for i := len(players) / 2; i < len(players); i++ {
		players[i].Team = TeamRed
	}

	rand.Shuffle(len(players), func(i, j int) { players[i], players[j] = players[j], players[i] })
}

func (r *Room) NewGame() {
	r.Game = NewGame(r.boardType)
}

func (r *Room) SwitchRole(playerID, role string) (string, bool) {
	p, ok := r.Player(playerID)
	if !ok {
		log.WithFields(logrus.Fields{
			"PlayerID": playerID,
			"RoomName": r.Name,
			"Role":     role,
		}).Warn("player tried to switch roles but they are not in the room")
		return "player not a member of the room", false
	}
	p.Role = role
	if role == PlayerRoleSpectator {
		p.Team = "undecided"
	}
	return role, true
}

func (r *Room) ChangeDifficulty(playerID, difficulty string) {
	player := r.PlayerLogged(playerID, "player tried to change difficulty but failed successfully")
	if player.Role == PlayerRoleSpectator {
		return
	}

	r.Difficulty = difficulty
}

func (r *Room) SwitchMode(playerID, mode string) {
	if _, ok := ModeTypes[mode]; !ok {
		return
	}

	player := r.PlayerLogged(playerID, "player tried to change modes but failed successfully")
	if player.Role == PlayerRoleSpectator {
		return
	}

	r.Mode = mode
}

func (r *Room) SwitchConsensus(playerID, consensus string) {
	player := r.PlayerLogged(playerID, "player tried to ask for consent but failed successfully")
	if player.Role == PlayerRoleSpectator {
		return
	}

	r.Consesus = consensus
}

func (r *Room) EndTurn(playerID string) {
	logEntry := GameLog{
		Event: "endTurn",
	}
	if r.Game.Turn == TeamBlue {
		r.Game.Turn = TeamRed
		logEntry.Team = TeamRed
	} else {
		r.Game.Turn = TeamBlue
		logEntry.Team = TeamBlue
	}
	r.Game.Log = append(r.Game.Log, logEntry)
}

func (r *Room) SelectTile(playerID string, i, j int) {
	p, ok := r.Player(playerID)
	if !ok {
		log.WithFields(logrus.Fields{
			"PlayerID": playerID,
			"RoomName": r.Name,
		}).Warn("player tried to click tile but they are not in the room")
		return
	}
	if p.Team != r.Game.Turn {
		log.WithFields(logrus.Fields{
			"PlayerID":   playerID,
			"PlayerName": p.NickName,
			"RoomName":   r.Name,
			"PlayerTeam": p.Team,
			"TurnTeam":   r.Game.Turn,
		}).Info("player tried to click tile but it's not their turn")
		return
	}

	if p.Role == PlayerRoleSpectator {
		log.WithFields(logrus.Fields{
			"PlayerID":   playerID,
			"PlayerName": p.NickName,
			"RoomName":   r.Name,
			"Role":       p.Role,
		}).Info("specator being naughty and clicking words.")
		return
	}

	if p.Role == PlayerRoleSpyMaster {
		log.WithFields(logrus.Fields{
			"PlayerID":   playerID,
			"PlayerName": p.NickName,
			"RoomName":   r.Name,
			"Role":       p.Role,
		}).Info("player tried to click tile but they are the spymaster")
		return
	}

	if r.Game.Clue == nil {
		// no clue, can't play
		log.WithFields(logrus.Fields{
			"PlayerID":   playerID,
			"PlayerName": p.NickName,
			"RoomName":   r.Name,
		}).Info("player tried to click tile but they are is no clue")
		return
	} else if r.Game.turnsTaken >= r.Game.Clue.Count+1 {
		// can only make clue+1 turns max
		log.WithFields(logrus.Fields{
			"PlayerID":   playerID,
			"PlayerName": p.NickName,
			"RoomName":   r.Name,
			"TurnsTake":  r.Game.turnsTaken,
			"ClueCount":  r.Game.Clue.Count,
		}).Info("player tried to click time but they don't have clues left")
		return
	}

	tile := &r.Game.Board[i][j]

	log.WithFields(logrus.Fields{
		"PlayerID":   playerID,
		"PlayerName": p.NickName,
		"RoomName":   r.Name,
		"Tile":       tile.Word,
		"Flipped":    tile.Flipped,
		"Type":       tile.Type,
	}).Info("player flipping tile")

	if tile.Flipped {
		return
	}

	if r.Consesus == ConsensusAll && !r.playerHasConsensus(p, i, j) {
		log.WithFields(logrus.Fields{
			"PlayerID":   playerID,
			"PlayerName": p.NickName,
			"RoomName":   r.Name,
			"Tile":       tile.Word,
		}).Info("player tried to flip tile but they don't have consensus")
		return
	}

	tile.Flipped = true

	logEntry := GameLog{
		Event: "flipTile",
		Word:  tile.Word,
		Type:  tile.Type,
		Team:  p.Team,
	}

	switch tile.Type {
	case TileTypeBlack:
		r.Game.Over = true
		ot := otherTeam(p.Team)
		r.Game.Winner = &ot
		logEntry.EndedTurn = true

	case TileTypeNeutral:
		r.switchTurns()
		logEntry.EndedTurn = true

	case TileTypeBlue:
		r.Game.Blue--
		if p.Team == TeamBlue {
			r.Game.turnsTaken++
		} else {
			r.switchTurns()
			logEntry.EndedTurn = true
		}

	case TileTypeRed:
		r.Game.Red--
		if p.Team == TeamRed {
			r.Game.turnsTaken++
		} else {
			r.switchTurns()
			logEntry.EndedTurn = true
		}
	}

	r.clearGuessProposals()

	if r.Game.Clue != nil && r.Game.turnsTaken >= r.Game.Clue.Count+1 {
		r.switchTurns()
		logEntry.EndedTurn = true
	}
	r.Game.Log = append(r.Game.Log, logEntry)
}

func (r *Room) playerHasConsensus(p *Player, i, j int) bool {
	word := r.Game.Board[i][j].Word
	p.GuessProposal = &word
	for _, tp := range r.teamPlayers(p.Team) {
		if tp.Role == PlayerRoleSpyMaster || tp.Role == PlayerRoleSpectator {
			continue
		}
		if tp.GuessProposal == nil || *tp.GuessProposal != word {
			return false
		}
	}
	return true
}

func otherTeam(team string) string {
	if team == TeamBlue {
		return TeamRed
	}
	return TeamBlue
}

func (r *Room) switchTurns() {
	log.WithFields(logrus.Fields{
		"FromTeam": r.Game.Turn,
		"ToTeam":   otherTeam(r.Game.Turn),
	}).Info("Switching teams")

	r.Game.Turn = otherTeam(r.Game.Turn)
	r.Game.turnsTaken = 0
	r.Game.Clue = nil
}
func (r *Room) clearGuessProposals() {
	for _, tp := range r.teamPlayers(r.Game.Turn) {
		tp.GuessProposal = nil
	}
}

func (r *Room) teamPlayers(team string) []*Player {
	players := []*Player{}
	for id, p := range r.Players {
		if p == nil {
			log.WithField("NullPlayerID", id).Warn("found null player in map")
			delete(r.Players, id)
			continue
		}
		if p.Team == team {
			players = append(players, p)
		}
	}
	return players
}

func (r *Room) DeclareClue(playerID, clue string, count int) {
	r.Game.Clue = &Clue{
		Word:  clue,
		Count: count,
	}
	r.Game.Log = append(r.Game.Log, GameLog{
		Event: "declareClue",
		Clue:  r.Game.Clue,
		Team:  r.Game.Turn,
	})
}

func (r *Room) ChangeCards(playerID, pack string) {
	if pack == "base" {
		r.Game.Base = !r.Game.Base
		r.boardType = r.boardType ^ BoardTypeDefault
	}
	if pack == "duet" {
		r.Game.Duet = !r.Game.Duet
		r.boardType = r.boardType ^ BoardTypeDuet
	}
	if pack == "undercover" {
		r.Game.Undercover = !r.Game.Undercover
		r.boardType = r.boardType ^ BoardTypeUndercover
	}
	if pack == "custom" {
		r.Game.Custom = !r.Game.Custom
		r.boardType = r.boardType ^ BoardTypeCustom
	}
	if pack == "nsfw" {
		r.Game.Nsfw = !r.Game.Nsfw
		r.boardType = r.boardType ^ BoardTypeNsfw
	}
	r.Game.WordPool = wordpoolSize(r.boardType)
}

func (r *Room) GameState() *gameState {
	return &gameState{
		Room:       r.Name,
		Game:       r.Game,
		Difficulty: r.Difficulty,
		Consensus:  r.Consesus,
		Mode:       r.Mode,
		Players:    r.Players,
	}
}

func (r *Room) PlayerGameState(playerID string) *gameState {
	p, ok := r.Players[playerID]
	if !ok {
		return r.GameState()
	}
	return &gameState{
		Room:       r.Name,
		Game:       r.Game,
		Difficulty: r.Difficulty,
		Consensus:  r.Consesus,
		Mode:       r.Mode,
		Players:    r.Players,
		Team:       p.Team,
	}
}

func (r *Room) ChangeTimer(playerID string, value float64) {
	player := r.PlayerLogged(playerID, "player tried to change time zones but failed successfully")
	if player.Role == PlayerRoleSpectator {
		return
	}

	r.Game.TimerAmount = value * 60
}

// PlayerLogged ... Get a player and automatically log if not ok
func (r *Room) PlayerLogged(playerID string, errorMsg string) *Player {
	player, ok := r.Players[playerID]

	if !ok {
		log.WithFields(logrus.Fields{
			"PlayerID": playerID,
			"RoomName": r.Name,
		}).Warn(errorMsg)
	}

	return player
}

func (r *Room) Player(playerID string) (*Player, bool) {
	player, ok := r.Players[playerID]
	return player, ok
}

func buildSet(elem ...string) map[string]struct{} {
	result := map[string]struct{}{}
	for _, e := range elem {
		result[e] = struct{}{}
	}
	return result
}

type gameState struct {
	Room       string             `json:"room"`
	Players    map[string]*Player `json:"players"`
	Game       *Game              `json:"game"`
	Difficulty string             `json:"difficulty"`
	Mode       string             `json:"mode"`
	Consensus  string             `json:"consensus"`
	Team       string             `json:"team"`
}
