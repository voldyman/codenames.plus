package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	DefaultWords    = []string{}
	NsfwWords       = []string{}
	DuetWords       = []string{}
	UndercoverWords = []string{}
	CustomWords     = []string{}
)

func init() {
	DefaultWords = readWords("server/words.txt")
	NsfwWords = readWords("server/nsfw-words.txt")
	DuetWords = readWords("server/duet-words.txt")
	UndercoverWords = readWords("server/duet-words.txt")
	CustomWords = readWords("server/custom-words.txt")

	BoardTypeToWordSet = map[BoardType]*[]string{
		BoardTypeDefault:    &DefaultWords,
		BoardTypeDuet:       &DuetWords,
		BoardTypeCustom:     &CustomWords,
		BoardTypeNsfw:       &NsfwWords,
		BoardTypeUndercover: &UndercoverWords,
	}
}

func readWords(file string) []string {
	txt, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Println("unable to read file", file)
		panic(err)
	}
	return strings.Split(string(txt), "\n")
}

var (
	ConsensusSingle = "single"
	ConsensusAll    = "consensus"
	ConsensusTypes  = buildSet(ConsensusAll, ConsensusSingle)
)

var (
	ModeCasual = "casual"
	ModeTimed  = "time"
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
	BoardTypeToWordSet = map[BoardType]*[]string{}
)
var (
	PlayerRoleGuesser   = "guesser"
	PlayerRoleSpyMaster = "spymaster"
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
	if _, ok := r.Players[playerID]; ok {
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
	p, ok := r.Player(playerID)
	if !ok {
		log.WithFields(logrus.Fields{
			"PlayerID": playerID,
			"RoomName": r.Name,
		}).Warn("player tried to leave but they are not in the room")
		return false
	}
	if p.Team == TeamBlue {
		r.Game.Blue--
	} else {
		r.Game.Red--
	}
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
	rand.Shuffle(len(players), func(i, j int) { players[i], players[j] = players[j], players[i] })

	for i := 0; i < len(players)/2; i++ {
		players[i].Team = TeamBlue
	}

	for i := len(players) / 2; i < len(players); i++ {
		players[i].Team = TeamRed
	}
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
	return role, true
}

func (r *Room) ChangeDifficulty(playerID, difficulty string) {
	r.Difficulty = difficulty
}

func (r *Room) SwitchMode(playerID, mode, timerAmount string) {
	if _, ok := ModeTypes[mode]; !ok {
		return
	}
	r.Mode = mode
	val, err := strconv.Atoi(timerAmount)
	if err != nil {
		val = int(5 * time.Minute)
	}
	r.Game.TimerAmount = int64(val)
}

func (r *Room) SwitchConsensus(playerID, consensus string) {
	r.Consesus = consensus
}

func (r *Room) EndTurn(playerID string) {
	if r.Game.Turn == TeamBlue {
		r.Game.Turn = TeamRed
	} else {
		r.Game.Turn = TeamBlue
	}
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

	if tile.Flipped {
		log.WithFields(logrus.Fields{
			"PlayerID":   playerID,
			"PlayerName": p.NickName,
			"RoomName":   r.Name,
			"Tile":       tile.Word,
		}).Info("player flipp an already flipped tile")
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

	switch tile.Type {
	case TileTypeBlack:
		r.Game.Over = true
		ot := otherTeam(p.Team)
		r.Game.Winner = &ot

	case TileTypeNeutral:
		r.switchTurns()

	case TileTypeBlue:
		r.Game.Blue--
		if p.Team == TeamBlue {
			r.Game.turnsTaken++
		} else {
			r.switchTurns()
		}

	case TileTypeRed:
		r.Game.Red--
		if p.Team == TeamRed {
			r.Game.turnsTaken++
		} else {
			r.switchTurns()
		}
	}

	if r.Game.turnsTaken >= r.Game.Clue.Count+1 {
		r.switchTurns()
	}
}

func (r *Room) playerHasConsensus(p *Player, i, j int) bool {
	word := r.Game.Board[i][j].Word
	p.GuessProposal = &word
	for _, tp := range r.teamPlayers(p.Team) {
		if tp.Role == PlayerRoleSpyMaster {
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
	for _, tp := range r.teamPlayers(r.Game.Turn) {
		tp.GuessProposal = nil
	}
	r.Game.Turn = otherTeam(r.Game.Turn)
	r.Game.turnsTaken = 0
	r.Game.Clue = nil
}

func (r *Room) teamPlayers(team string) []*Player {
	players := []*Player{}
	for _, p := range r.Players {
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

func (r *Room) ChangeTimer(playerID string, value int) {

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

type Player struct {
	ID            string  `json:"id"`
	NameAvailable bool    `json:"nameAvailable"`
	TempName      string  `json:"tempName"`
	Counter       int     `json:"counter"`
	NickName      string  `json:"nickname"`
	Room          string  `json:"room"`
	Team          string  `json:"team"`
	Role          string  `json:"role"`
	GuessProposal *string `json:"guessProposal"`
	Timeout       int     `json:"timeout"`
	AfkTimer      int     `json:"afkTimer"`
}

var (
	TileTypeBlue    = "blue"
	TileTypeRed     = "red"
	TileTypeBlack   = "death"
	TileTypeNeutral = "neutral"
)

type Tile struct {
	Word    string `json:"word"`
	Flipped bool   `json:"flipped"`
	Type    string `json:"type"`
}

var (
	TeamBlue = "blue"
	TeamRed  = "red"
)

type Clue struct {
	Word  string `json:"word"`
	Count int    `json:"count"`
}
type Game struct {
	TimerAmount int64 `json:"timerAmount"`
	WordPool    int   `json:"wordPool"`

	// Game types, these are kept here for the UI
	// actual board types are stored in the room
	Base       bool `json:"base"`
	Duet       bool `json:"duet"`
	Undercover bool `json:"undercover"`
	Custom     bool `json:"custom"`
	Nsfw       bool `json:"nsfw"`

	// player count
	Red  int `json:"red"`
	Blue int `json:"blue"`

	// game state
	Turn   string   `json:"turn"`
	Over   bool     `json:"over"`
	Winner *string  `json:"winner"`
	Timer  string   `json:"timer"`
	Board  [][]Tile `json:"board"`
	Log    []string `json:"log"`
	Clue   *Clue    `json:"clue"`

	turnsTaken int
}

func NewGame(bt BoardType) *Game {
	blueTiles := 9
	redTiles := 8

	turn := TeamBlue
	if rand.Intn(100)%2 == 0 {
		turn = TeamRed
		blueTiles = 8
		redTiles = 9
	}

	timerAmount := int64(5 * time.Minute)
	return &Game{
		TimerAmount: timerAmount,
		WordPool:    wordpoolSize(bt),

		Base:       isSet(bt, BoardTypeDefault),
		Duet:       isSet(bt, BoardTypeDuet),
		Undercover: isSet(bt, BoardTypeUndercover),
		Custom:     isSet(bt, BoardTypeCustom),
		Nsfw:       isSet(bt, BoardTypeNsfw),

		Red:  redTiles,
		Blue: blueTiles,

		Turn:   turn,
		Over:   false,
		Winner: nil,
		Timer:  strconv.FormatInt(timerAmount, 10),
		Board:  generateBoard(bt, turn),
		Log:    []string{},
		Clue:   nil,
	}
}

func wordpoolSize(bt BoardType) int {
	count := 0
	visitBoardType(bt, func(bt BoardType) {
		count += len(*BoardTypeToWordSet[bt])
	})
	return count
}

func generateBoard(bt BoardType, turn string) [][]Tile {
	totalWords := 25
	setsEnabled := getTotalSetsEnabled(bt)

	if setsEnabled == 0 {
		setsEnabled = 1
		bt = BoardTypeDefault
	}

	wordsPerSet := (totalWords / setsEnabled) + 1
	words := getWords(bt, wordsPerSet)
	linearTiles := generateLinearTiles(words, turn)

	rand.Shuffle(len(linearTiles), func(i, j int) { linearTiles[i], linearTiles[j] = linearTiles[j], linearTiles[i] })

	result := [][]Tile{}

	width := 5
	for i := 0; i < 5; i++ {
		x := i * width
		y := x + width
		result = append(result, linearTiles[x:y])
	}
	return result
}

func getTotalSetsEnabled(bt BoardType) int {
	setsEnabled := 0
	visitBoardType(bt, func(bt BoardType) {
		setsEnabled++
	})
	return setsEnabled
}

func visitBoardType(bt BoardType, visitor func(bt BoardType)) {
	if isSet(bt, BoardTypeDefault) {
		visitor(BoardTypeDefault)
	}
	if isSet(bt, BoardTypeCustom) {
		visitor(BoardTypeCustom)
	}
	if isSet(bt, BoardTypeDuet) {
		visitor(BoardTypeDuet)
	}
	if isSet(bt, BoardTypeNsfw) {
		visitor(BoardTypeNsfw)
	}
	if isSet(bt, BoardTypeUndercover) {
		visitor(BoardTypeUndercover)
	}
}

func getWords(bt BoardType, wordsPerSet int) []string {
	words := map[string]struct{}{}

	visitBoardType(bt, func(bt BoardType) {
		selectWords(*BoardTypeToWordSet[bt], wordsPerSet, words)
	})

	result := []string{}
	for k := range words {
		result = append(result, k)
	}
	return result
}

func isSet(bt, typ BoardType) bool {
	return (bt & typ) != 0
}

func selectWords(arr []string, count int, lookupTable map[string]struct{}) {
	for i := 0; i < count; {
		di := rand.Intn(len(arr))
		if _, ok := lookupTable[arr[di]]; ok {
			continue
		}
		lookupTable[arr[di]] = struct{}{}
		i++
	}

}
func generateLinearTiles(words []string, turn string) []Tile {
	linearTiles := make([]Tile, 25)

	wordIter := 0

	// the game has 1 black time
	// 9 for one team
	// 8 for the other team
	// rest are neutral

	linearTiles[wordIter] = Tile{
		Word:    words[wordIter],
		Type:    TileTypeBlack,
		Flipped: false,
	}
	wordIter++

	firstColor := TileTypeBlue
	secondColor := TileTypeRed
	if turn == TeamRed {
		firstColor = TileTypeRed
		secondColor = TileTypeBlue
	}

	for i := 0; i < 9; i++ {
		linearTiles[wordIter] = Tile{
			Word:    words[wordIter],
			Type:    firstColor,
			Flipped: false,
		}
		wordIter++
	}
	for i := 0; i < 8; i++ {
		linearTiles[wordIter] = Tile{
			Word:    words[wordIter],
			Type:    secondColor,
			Flipped: false,
		}
		wordIter++
	}

	for i := 0; i < 7; i++ {
		linearTiles[wordIter] = Tile{
			Word:    words[wordIter],
			Type:    TileTypeNeutral,
			Flipped: false,
		}
		wordIter++
	}
	return linearTiles
}
