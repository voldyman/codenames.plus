package main

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
)

type ResponseEmitter interface {
	Emit(msg string, success bool)
}
type resEmitFunc func(msg string, success bool)

func (r resEmitFunc) Emit(msg string, success bool) {
	r(msg, success)
}
func ResEmitFunc(fn func(msg string, success bool)) resEmitFunc {
	return resEmitFunc(fn)
}

type RoomAction func(r *Room)

type RoomActionReceiver chan<- RoomAction

type ActionRouter struct {
	sync.RWMutex
	playerRooms map[string]RoomActionReceiver
	nameRooms   map[string]RoomActionReceiver
}

func NewActionRouter() *ActionRouter {
	return &ActionRouter{
		playerRooms: map[string]RoomActionReceiver{},
		nameRooms:   map[string]RoomActionReceiver{},
	}
}

func (a *ActionRouter) PlayerRoomReceiver(playerID string) RoomActionReceiver {
	a.RLock()
	defer a.RUnlock()
	if rr, ok := a.playerRooms[playerID]; ok {
		return rr
	}
	return nil
}

func (a *ActionRouter) RoomNameReceiver(roomName string) RoomActionReceiver {
	a.RLock()
	defer a.RUnlock()
	if rr, ok := a.nameRooms[roomName]; ok {
		return rr
	}
	return nil

}

func (a *ActionRouter) CreateRoom(playerID, nick, room, password string, res ResponseEmitter) {
	if len(nick) == 0 {
		res.Emit("invalid nickname", false)
		return
	}

	if len(password) == 0 {
		res.Emit("invalid password", false)
	}

	// this actions needs to be atomic
	a.Lock()

	// check if room exists
	if _, ok := a.nameRooms[room]; ok {
		a.Unlock()
		res.Emit(fmt.Sprintf("room %s already exists.", room), false)
		return
	}

	// check if player is an another room
	if rr, ok := a.playerRooms[playerID]; ok {
		rr <- func(r *Room) {
			r.Leave(playerID)
		}
	}

	r := NewRoom(room, password)
	rr := startRoomRouter(r)

	rr <- func(r *Room) {
		r.Join(playerID, nick)
	}

	a.playerRooms[playerID] = rr
	a.nameRooms[room] = rr

	a.Unlock()

	res.Emit("created the room", true)
}

func (a *ActionRouter) JoinRoom(playerID, nick, roomName, password string, action RoomAction) bool {
	rr := a.RoomNameReceiver(roomName)
	if rr == nil {
		log.WithFields(logrus.Fields{
			"PlayerID": playerID,
			"RoomName": roomName,
			"Nick":     nick,
		}).Info("player tried to join a nonexistant room")
		return false
	}

	rr <- func(r *Room) {
		if r.Password != password {
			action(nil)
			return
		}

		r.Join(playerID, nick)
		a.Lock()
		defer a.Unlock()
		a.playerRooms[playerID] = rr

		action(r)
	}
	return true
}

func (a *ActionRouter) RoomForPlayer(playerID string, action RoomAction) bool {
	if rr := a.PlayerRoomReceiver(playerID); rr != nil {
		rr <- action
		return true
	}
	log.WithField("PlayerID", playerID).Warn("player not in any room, dropping action")
	return false
}

func (a *ActionRouter) RoomByName(roomName string, action RoomAction) bool {
	if rr := a.RoomNameReceiver(roomName); rr != nil {
		rr <- action
		return true
	}
	log.WithField("RoomName", roomName).Warn("room not found, dropping action")
	return false
}

func startRoomRouter(r *Room) RoomActionReceiver {
	actionChan := make(chan RoomAction)
	go func() {
		for {
			select {
			case action := <-actionChan:
				action(r)
			}
		}
	}()
	return actionChan
}

func (a *ActionRouter) CheckIfPlayerExists(playerID string, res func(players, rooms int, playerID string, isInRoom bool, gs *gameState)) {
	rr := a.PlayerRoomReceiver(playerID)
	if rr == nil {
		res(a.Players(), a.Rooms(), playerID, false, nil)
		return
	}

	rr <- func(r *Room) {
		res(a.Players(), a.Rooms(), playerID, true, r.GameState())
	}
}

func (a *ActionRouter) LeaveRoom(playerID string, action RoomAction) bool {
	if rr := a.PlayerRoomReceiver(playerID); rr != nil {
		rr <- action
		rr <- func(r *Room) {
			a.Lock()
			defer a.Unlock()

			r.Leave(playerID)
			delete(a.playerRooms, playerID)

			// todo(voldy): delay this for later?
			if len(r.Players) == 0 {
				delete(a.nameRooms, r.Name)
			}
		}

		return true
	}
	return false
}

func (a *ActionRouter) Players() int {
	a.RLock()
	defer a.RUnlock()
	return len(a.playerRooms)
}

func (a *ActionRouter) Rooms() int {
	a.RLock()
	defer a.RUnlock()
	return len(a.nameRooms)
}
