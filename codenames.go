package main

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
)

type CodeNames struct {
	sync.RWMutex
	PlayerRooms map[string]*Room
	NameRooms   map[string]*Room
}

func NewCodeNames() *CodeNames {
	return &CodeNames{
		PlayerRooms: map[string]*Room{},
		NameRooms:   map[string]*Room{},
	}
}

func (cn *CodeNames) PlayerRoom(playerID string) *Room {
	cn.RLock()
	defer cn.RUnlock()
	room := cn.PlayerRooms[playerID]
	if room == nil {
		return nil
	}
	return room
}

func (cn *CodeNames) PlayerRoomName(playerID string) string {
	room := cn.PlayerRoom(playerID)
	if room == nil {
		return ""
	}
	return room.Name
}

func (cn *CodeNames) RoomName(roomName string) *Room {
	cn.RLock()
	defer cn.RUnlock()
	room, ok := cn.NameRooms[roomName]
	if !ok {
		return nil
	}
	return room
}

func (cn *CodeNames) SetNameRoom(roomName string, room *Room) {
	cn.Lock()
	cn.NameRooms[roomName] = room
	cn.Unlock()

}

func (cn *CodeNames) CleanRoom(room *Room) {
	cn.Lock()
	if len(room.Players) == 0 {
		delete(cn.NameRooms, room.Name)
	}
	cn.Unlock()
}

func (cn *CodeNames) DeletePlayer(playerID string) {
	cn.Lock()
	delete(cn.PlayerRooms, playerID)
	cn.Unlock()
}

func (cn *CodeNames) CreateRoom(playerID, nick, room, password string) (string, bool) {
	if oldRoom := cn.PlayerRoom(playerID); oldRoom != nil {
		oldRoom.Leave(playerID)
	}

	if r := cn.RoomName(room); r != nil {
		log.WithFields(logrus.Fields{
			"PlayerID": playerID,
			"Nick":     nick,
			"Password": password,
			"RoomName": room,
		}).Info("player tried to create room but already exists")
		return fmt.Sprintf("room %s already exists.", room), false
	}
	cn.SetNameRoom(room, NewRoom(room, password))
	return cn.JoinRoom(playerID, room, nick, password)
}

func (cn *CodeNames) JoinRoom(playerID, roomname, nick, password string) (string, bool) {
	if len(nick) == 0 {
		return "invalid nickname", false
	}
	if len(password) == 0 {
		return "invalid password", false
	}

	room := cn.RoomName(roomname)
	if room == nil {
		log.WithFields(logrus.Fields{
			"PlayerID": playerID,
			"Nick":     nick,
			"Password": password,
			"RoomName": room,
		}).Info("player tried to join a room but the room does not exist")

		return fmt.Sprintf("could not find room: %s", room.Name), false
	}

	if room.Password != password {
		return "invalid password", false
	}
	ok := room.Join(playerID, nick)
	if !ok {
		return "unable to join room", false
	}

	cn.Lock()
	cn.PlayerRooms[playerID] = room
	cn.Unlock()

	log.WithFields(logrus.Fields{
		"PlayerID":   playerID,
		"PlayerNick": nick,
		"RoomName":   room.Name,
	}).Info("player joined room")

	return "joined the room", true

}

func (cn *CodeNames) LeaveRoom(playerID string) bool {
	room := cn.PlayerRoom(playerID)
	if room == nil {
		log.WithFields(logrus.Fields{
			"PlayerID": playerID,
		}).Warn("player tried to leave the roo but they are not in any room")
		return false
	}

	ok := room.Leave(playerID)
	if !ok {
		log.WithFields(logrus.Fields{
			"PlayerID": playerID,
			"RoomName": room.Name,
		}).Warn("player was unable to leave the room")

		return false
	}

	cn.DeletePlayer(playerID)

	// room gets deleted when all players leave
	// todo(voldy): fix this, maybe?
	cn.CleanRoom(room)

	return true
}

func (cn *CodeNames) JoinTeam(playerID, team string) bool {
	room := cn.PlayerRoom(playerID)
	if room == nil {
		log.WithFields(logrus.Fields{
			"PlayerID": playerID,
			"RoomName": room.Name,
			"Team":     team,
		}).Info("player tried to join team but they are not in any room")
		return false
	}
	room.ChangeTeam(playerID, team)
	return true
}

func (cn *CodeNames) RandomizeTeams(playerID string) bool {
	room := cn.PlayerRoom(playerID)
	if room == nil {
		log.WithFields(logrus.Fields{
			"PlayerID": playerID,
		}).Warn("player tried to randomize teams but they are not in any room")
		return false
	}
	room.RandomizeTeams(playerID)
	return true
}

func (cn *CodeNames) NewGame(playerID string) bool {
	room := cn.PlayerRoom(playerID)
	if room == nil {
		log.WithFields(logrus.Fields{
			"PlayerID": playerID,
		}).Warn("player tried to start a new game but they are not in any room")
		return false
	}

	room.NewGame()

	return true
}

func (cn *CodeNames) SwitchRole(playerID, role string) (string, bool) {
	room := cn.PlayerRoom(playerID)
	if room == nil {
		log.WithFields(logrus.Fields{
			"PlayerID": playerID,
			"Role":     role,
		}).Warn("player tried to switch role but they are not in any room")
		return "you can't switch roles when you are not part of a room", false
	}

	return room.SwitchRole(playerID, role)
}

func (cn *CodeNames) SwitchDifficulty(playerID, difficulty string) bool {
	room := cn.PlayerRoom(playerID)
	if room == nil {
		log.Printf("player %s tried to switch difficulty to %s but they are not in a room", playerID, difficulty)
		return false
	}

	room.ChangeDifficulty(playerID, difficulty)

	return true
}

func (cn *CodeNames) SwitchMode(playerID, mode string) bool {
	room := cn.PlayerRoom(playerID)
	if room == nil {
		log.WithFields(logrus.Fields{
			"PlayerID": playerID,
			"Mode":     mode,
		}).Info("switch mode failed, player was not found in a room")
		return false
	}

	room.SwitchMode(playerID, mode)
	return true
}

func (cn *CodeNames) SwitchConsensus(playerID, roomname, consensus string) bool {
	room := cn.PlayerRoom(playerID)
	if room == nil {
		log.Printf("player %s tried to switch consensus to %s but they are not in a room", playerID, consensus)
		return false
	}

	room.SwitchConsensus(playerID, consensus)
	return true
}

func (cn *CodeNames) EndTurn(playerID string) bool {
	room := cn.PlayerRoom(playerID)
	if room == nil {
		log.Printf("player %s tried end turn but they are not in a room", playerID)
		return false
	}

	room.EndTurn(playerID)
	return true
}

func (cn *CodeNames) ClickTile(playerID string, i, j int) bool {
	room := cn.PlayerRoom(playerID)
	if room == nil {
		log.Printf("player %s tried click a tile but they are not in a room", playerID)
		return false
	}

	room.SelectTile(playerID, i, j)
	return true
}

func (cn *CodeNames) DeclareClue(playerID, word string, count int) bool {
	room := cn.PlayerRoom(playerID)
	if room == nil {
		log.Printf("player %s tried to declare a clue %s for %d but they are not in a room", playerID, word, count)
		return false
	}

	room.DeclareClue(playerID, word, count)
	return true
}

func (cn *CodeNames) ChangeCards(playerID, pack string) bool {
	room := cn.PlayerRoom(playerID)
	if room == nil {
		log.Printf("player %s tried to change pack to %s but they are not in a room", playerID, pack)
		return false
	}

	room.ChangeCards(playerID, pack)
	return true
}

func (cn *CodeNames) UpdateTimeSlider(playerID string, value float64) bool {
	room := cn.PlayerRoom(playerID)
	if room == nil {
		log.Printf("player %s tried to update timer to value %d but they are not in a room", playerID, value)
		return false
	}

	room.ChangeTimer(playerID, value)
	return true
}

func (cn *CodeNames) PlayerGameState(playerID string) *gameState {
	room := cn.PlayerRoom(playerID)
	if room == nil {
		log.WithField("PlayerID", playerID).Info("room not found for player to retrieve game state")
		return nil
	}

	p, ok := room.Player(playerID)
	if !ok {
		logrus.WithFields(logrus.Fields{
			"PlayerID": playerID,
			"RoomName": room.Name,
		}).Warn("player was mapped to a room but they were not in the room")

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

func (cn *CodeNames) RoomNameGameState(roomName string) *gameState {
	r := cn.RoomName(roomName)
	if r == nil {
		log.WithFields(logrus.Fields{
			"RoomName": roomName,
		}).Warn("room not found for looking up state")
		return nil
	}
	return cn.roomGameState(r)
}

func (cn *CodeNames) roomGameState(room *Room) *gameState {
	return &gameState{
		Room:       room.Name,
		Game:       room.Game,
		Difficulty: room.Difficulty,
		Consensus:  room.Consesus,
		Mode:       room.Mode,
		Players:    room.Players,
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
