package main

import (
	"fmt"
	"log"
)

type CodeNames struct {
	PlayerRooms map[string]*Room
	NameRooms   map[string]*Room
}

func NewCodeNames() *CodeNames {
	return &CodeNames{
		PlayerRooms: map[string]*Room{},
		NameRooms:   map[string]*Room{},
	}
}

func (cn *CodeNames) PlayerRoomName(playerID string) string {
	room, ok := cn.PlayerRooms[playerID]
	if !ok {
		return ""
	}
	return room.Name
}
func (cn *CodeNames) CreateRoom(playerID, nick, room, password string) (string, bool) {
	if room, ok := cn.PlayerRooms[playerID]; ok {
		_ = room
		// leave maybe?
	}

	if _, ok := cn.NameRooms[room]; ok {
		return fmt.Sprintf("room %s already exists.", room), false
	}

	cn.NameRooms[room] = NewRoom(room, password)
	return cn.JoinRoom(playerID, room, nick, password)
}

func (cn *CodeNames) JoinRoom(playerID, roomname, nick, password string) (string, bool) {
	if len(nick) == 0 {
		return "invalid nickname", false
	}
	if len(password) == 0 {
		return "invalid password", false
	}
	room, ok := cn.NameRooms[roomname]
	if !ok {
		return fmt.Sprintf("could not find room: %s", room.Name), false
	}

	if room.Password != password {
		return "invalid password", false
	}
	ok = room.Join(playerID, nick)
	if !ok {
		return "unable to join room", false
	}
	cn.PlayerRooms[playerID] = room
	log.Printf("player %s joined room %s", nick, roomname)
	return "joined the room", true

}

func (cn *CodeNames) LeaveRoom(playerID string) bool {
	room, ok := cn.PlayerRooms[playerID]
	if !ok {
		log.Printf("player: %s tried to leave when they are not in a room", playerID)
		return false
	}

	ok = room.Leave(playerID)
	if !ok {
		log.Printf("player %s was unable to leave room %s", playerID, room.Name)
		return false
	}
	delete(cn.PlayerRooms, playerID)

	// room gets deleted when all players leave
	// todo(voldy): fix this, maybe?
	if len(room.Players) == 0 {
		delete(cn.NameRooms, room.Name)
	}
	return true
}

func (cn *CodeNames) JoinTeam(playerID, team string) bool {
	room, ok := cn.PlayerRooms[playerID]
	if !ok {
		log.Printf("player %s tried to join a team but they are not in a room", playerID)
		return false
	}
	room.ChangeTeam(playerID, team)
	return true
}

func (cn *CodeNames) RandomizeTeams(playerID string) bool {
	room, ok := cn.PlayerRooms[playerID]
	if !ok {
		log.Printf("player %s tried to randomize but they are not in a room", playerID)
		return false
	}
	room.RandomizeTeams(playerID)
	return true
}

func (cn *CodeNames) NewGame(playerID string) bool {
	room, ok := cn.PlayerRooms[playerID]
	if !ok {
		log.Printf("player %s tried to start a new game but they are not in a room", playerID)
		return false
	}

	room.NewGame()

	return true
}

func (cn *CodeNames) SwitchRole(playerID, role string) (string, bool) {
	room, ok := cn.PlayerRooms[playerID]
	if !ok {
		log.Printf("player %s tried switch role to %s but they are not in a room", playerID, role)
		return "you can't switch roles when you are not part of a room", false
	}

	return room.SwitchRole(playerID, role)
}

func (cn *CodeNames) SwitchDifficulty(playerID, difficulty string) bool {
	room, ok := cn.PlayerRooms[playerID]
	if !ok {
		log.Printf("player %s tried to switch difficulty to %s but they are not in a room", playerID, difficulty)
		return false
	}

	room.ChangeDifficulty(playerID, difficulty)

	return true
}

func (cn *CodeNames) SwitchMode(playerID, roomName, mode, timerAmount string) bool {
	room, ok := cn.PlayerRooms[playerID]
	if !ok {
		log.Printf("player %s tried to switch mode to %s but they are not in a room", playerID, mode)
		return false
	}

	room.SwitchMode(playerID, mode, timerAmount)
	return true
}

func (cn *CodeNames) SwitchConsensus(playerID, roomname, consensus string) bool {
	room, ok := cn.PlayerRooms[playerID]
	if !ok {
		log.Printf("player %s tried to switch consensus to %s but they are not in a room", playerID, consensus)
		return false
	}

	room.SwitchConsensus(playerID, consensus)
	return true
}

func (cn *CodeNames) EndTurn(playerID string) bool {
	room, ok := cn.PlayerRooms[playerID]
	if !ok {
		log.Printf("player %s tried end turn but they are not in a room", playerID)
		return false
	}

	room.EndTurn(playerID)
	return true
}

func (cn *CodeNames) ClickTile(playerID string, i, j int) bool {
	room, ok := cn.PlayerRooms[playerID]
	if !ok {
		log.Printf("player %s tried click a tile but they are not in a room", playerID)
		return false
	}

	room.SelectTile(playerID, i, j)
	return true
}

func (cn *CodeNames) DeclareClue(playerID, word string, count int) bool {
	room, ok := cn.PlayerRooms[playerID]
	if !ok {
		log.Printf("player %s tried to declare a clue %s for %d but they are not in a room", playerID, word, count)
		return false
	}

	room.DeclareClue(playerID, word, count)
	return true
}

func (cn *CodeNames) ChangeCards(playerID, pack string) bool {
	room, ok := cn.PlayerRooms[playerID]
	if !ok {
		log.Printf("player %s tried to change pack to %s but they are not in a room", playerID, pack)
		return false
	}

	room.ChangeCards(playerID, pack)
	return true
}

func (cn *CodeNames) UpdateTimeSlider(playerID string, value int) bool {
	room, ok := cn.PlayerRooms[playerID]
	if !ok {
		log.Printf("player %s tried to update timer to value %d but they are not in a room", playerID, value)
		return false
	}

	room.ChangeTimer(playerID, value)
	return true
}

func (cn *CodeNames) GameState(playerID string) *gameState {
	room, ok := cn.PlayerRooms[playerID]
	if !ok {
		log.Printf("player %s tried to get game state but they are not in a room but they are not in a room", playerID)
		return nil
	}

	p, ok := room.Player(playerID)
	if !ok {
		log.Printf("player %s not found in the room", playerID)
		return nil
	}
	return &gameState{
		Room:       room.Name,
		Game:       room.Game,
		Difficulty: room.Difficulty,
		Consensus:  room.Consesus,
		Mode:       room.Mode,
		Players:    room.Players,
		Team:       p.Team,
	}
}

func (cn *CodeNames) Players() int {
	return len(cn.PlayerRooms)
}

func (cn *CodeNames) Rooms() int {
	return len(cn.NameRooms)
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
