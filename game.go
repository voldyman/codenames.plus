package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"strings"

	"github.com/markbates/pkger"
)

var (
	DefaultWords       = []string{}
	NsfwWords          = []string{}
	DuetWords          = []string{}
	UndercoverWords    = []string{}
	CustomWords        = []string{}
	BoardTypeToWordSet = map[BoardType]*[]string{}
)

func init() {
	DefaultWords = readWords("/server/words.txt")
	NsfwWords = readWords("/server/nsfw-words.txt")
	DuetWords = readWords("/server/duet-words.txt")
	UndercoverWords = readWords("/server/duet-words.txt")
	CustomWords = readWords("/server/custom-words.txt")

	BoardTypeToWordSet = map[BoardType]*[]string{
		BoardTypeDefault:    &DefaultWords,
		BoardTypeDuet:       &DuetWords,
		BoardTypeCustom:     &CustomWords,
		BoardTypeNsfw:       &NsfwWords,
		BoardTypeUndercover: &UndercoverWords,
	}
}

func readWords(file string) []string {
	f, err := pkger.Open(file)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	txt, err := ioutil.ReadAll(f)
	if err != nil {
		fmt.Println("unable to read file", file)
		panic(err)
	}
	return strings.Split(string(txt), "\n")
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

type GameLog struct {
	Event     string `json:"event,omitempty"`
	Team      string `json:"team,omitempty"`
	Word      string `json:"word,omitempty"`
	Type      string `json:"type,omitempty"`
	Clue      *Clue  `json:"clue,omitempty"`
	EndedTurn bool   `json:"endedTurn"`
}

type Game struct {
	TimerAmount float64 `json:"timerAmount"`
	WordPool    int     `json:"wordPool"`

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
	Turn   string    `json:"turn"`
	Over   bool      `json:"over"`
	Winner *string   `json:"winner"`
	Timer  float64   `json:"timer"`
	Board  [][]Tile  `json:"board"`
	Log    []GameLog `json:"log"`
	Clue   *Clue     `json:"clue"`

	turnsTaken int
}

func NewGame(bt BoardType, timerAmount float64) *Game {
	blueTiles := 9
	redTiles := 8

	turn := TeamBlue
	if rand.Intn(100)%2 == 0 {
		turn = TeamRed
		blueTiles = 8
		redTiles = 9
	}

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
		Timer:  timerAmount,
		Board:  generateBoard(bt, turn),
		Log:    []GameLog{},
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
