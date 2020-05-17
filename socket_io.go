package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"strconv"
	"strings"
	"time"

	socketio "github.com/googollee/go-socket.io"
)

func socketServer(cn *CodeNames) *socketio.Server {
	server, err := socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}

	type connContext struct {
		PlayerID string
	}

	server.OnConnect("/", func(s socketio.Conn) error {
		playerID := randID("player")
		vals, err := url.ParseQuery(s.URL().RawQuery)
		if err == nil {
			id := vals.Get("sessionId")
			if len(id) > 0 || id != "null" {
				log.Println("Reusing id for new player", id)
				playerID = id
			}
		}

		ctx := connContext{
			PlayerID: playerID,
		}

		s.SetContext(ctx)
		fmt.Println("connected:", s.ID(), "given ID:", ctx.PlayerID)
		s.Emit("serverStats", struct {
			Players          int        `json:"players"`
			Rooms            int        `json:"rooms"`
			SessionID        string     `json:"sessionId"`
			IsExistingPlayer bool       `json:"isExistingPlayer"`
			GameState        *gameState `json:"gameState"`
		}{
			Players:          cn.Players(),
			Rooms:            cn.Rooms(),
			SessionID:        ctx.PlayerID,
			IsExistingPlayer: false,
			GameState:        nil,
		})
		return nil
	})

	type createRoomRequest struct {
		Room     string `json:"room"`
		Nickname string `json:"nickname"`
		Password string `json:"password"`
	}

	type createRoomResponse struct {
		Message string `json:"message"`
		Success bool   `json:"success"`
	}

	server.OnEvent("/", "createRoom", func(s socketio.Conn, req createRoomRequest) {
		fmt.Println("createRoom:", req)
		// setup context?
		ctx := s.Context().(connContext)
		msg, success := cn.CreateRoom(ctx.PlayerID, req.Nickname, req.Room, req.Password)
		s.Emit("createResponse", createRoomResponse{
			Message: msg,
			Success: success,
		})
		gs := cn.GameState(ctx.PlayerID)

		if success {
			s.Join(req.Room)
			s.Emit("gameState", gs)
		}
		server.BroadcastToRoom("/", cn.PlayerRoomName(ctx.PlayerID), "gameState", gs)
	})

	type joinRoomRequest struct {
		Room     string `json:"room"`
		Nickname string `json:"nickname"`
		Password string `json:"password"`
	}
	type joinRoomResponse struct {
		Message string `json:"message"`
		Success bool   `json:"success"`
	}
	server.OnEvent("/", "joinRoom", func(s socketio.Conn, req joinRoomRequest) {
		fmt.Printf("joinRoom: %+v\n", req)
		ctx := s.Context().(connContext)

		msg, success := cn.JoinRoom(ctx.PlayerID, req.Room, req.Nickname, req.Password)
		s.Emit("joinResponse", joinRoomResponse{
			Message: msg,
			Success: success,
		})
		gs := cn.GameState(ctx.PlayerID)
		if success {
			s.Join(req.Room)
			s.Emit("gameState", gs)
		}
		server.BroadcastToRoom("/", cn.PlayerRoomName(ctx.PlayerID), "gameState", gs)
	})

	type leaveRoomResponse struct {
		Success bool `json:"success"`
	}
	server.OnEvent("/", "leaveRoom", func(s socketio.Conn, vals map[string]interface{}) {
		fmt.Println("leaveRoom:", vals)
		ctx := s.Context().(connContext)

		ok := cn.LeaveRoom(ctx.PlayerID)
		if !ok {
			s.Emit("reset")
			return
		}

		s.Emit("leaveResponse", leaveRoomResponse{
			Success: true,
		})

		roomName := cn.PlayerRoomName(ctx.PlayerID)
		s.Leave(roomName)
		server.BroadcastToRoom("/", roomName, "gameState", cn.GameState(ctx.PlayerID))
	})

	type joinTeamRequest struct {
		Team string
	}
	server.OnEvent("/", "joinTeam", func(s socketio.Conn, req joinTeamRequest) {
		fmt.Println("joinTeam:", req)
		ctx := s.Context().(connContext)

		ok := cn.JoinTeam(ctx.PlayerID, req.Team)
		if !ok {
			s.Emit("reset")
			return
		}

		roomName := cn.PlayerRoomName(ctx.PlayerID)
		server.BroadcastToRoom("/", roomName, "gameState", cn.GameState(ctx.PlayerID))
	})

	server.OnEvent("/", "randomizeTeams", func(s socketio.Conn, req map[string]interface{}) {
		fmt.Println("randomizeTeams:", req)

		ctx := s.Context().(connContext)

		ok := cn.RandomizeTeams(ctx.PlayerID)
		if !ok {
			s.Emit("reset")
			return
		}

		roomName := cn.PlayerRoomName(ctx.PlayerID)
		server.BroadcastToRoom("/", roomName, "gameState", cn.GameState(ctx.PlayerID))
	})
	server.OnEvent("/", "newGame", func(s socketio.Conn, vals map[string]interface{}) {
		fmt.Println("newGame:", vals)

		ctx := s.Context().(connContext)

		ok := cn.NewGame(ctx.PlayerID)
		if !ok {
			s.Emit("reset")
			return
		}

		roomName := cn.PlayerRoomName(ctx.PlayerID)
		server.BroadcastToRoom("/", roomName, "gameState", cn.GameState(ctx.PlayerID))
	})

	type switchRoleRequest struct {
		Role string `json:"role"`
	}
	type switchRoleResponse struct {
		Role    string `json:"role"`
		Success bool   `json:"success"`
	}
	server.OnEvent("/", "switchRole", func(s socketio.Conn, req switchRoleRequest) {
		fmt.Println("switchRole:", req)

		ctx := s.Context().(connContext)

		role, success := cn.SwitchRole(ctx.PlayerID, req.Role)

		s.Emit("switchRoleResponse", switchRoleResponse{
			Role:    role,
			Success: success,
		})

		roomName := cn.PlayerRoomName(ctx.PlayerID)
		server.BroadcastToRoom("/", roomName, "gameState", cn.GameState(ctx.PlayerID))
	})

	type switchDifficultyRequest struct {
		Difficulty string `json:"difficulty"`
	}

	server.OnEvent("/", "switchDifficulty", func(s socketio.Conn, req switchDifficultyRequest) {
		fmt.Println("switchDifficulty:", req)

		ctx := s.Context().(connContext)
		cn.SwitchDifficulty(ctx.PlayerID, req.Difficulty)

		roomName := cn.PlayerRoomName(ctx.PlayerID)
		server.BroadcastToRoom("/", roomName, "gameState", cn.GameState(ctx.PlayerID))
	})

	type switchModeRequest struct {
		Room        string `json:"room"`
		Mode        string `json:"mode"`
		TimerAmount string `json:"timer_amount"`
	}
	server.OnEvent("/", "switchMode", func(s socketio.Conn, req switchModeRequest) {
		fmt.Println("switchMode:", req)
		ctx := s.Context().(connContext)

		cn.SwitchMode(ctx.PlayerID, req.Room, req.Mode, req.TimerAmount)

		roomName := cn.PlayerRoomName(ctx.PlayerID)
		server.BroadcastToRoom("/", roomName, "gameState", cn.GameState(ctx.PlayerID))
	})

	type switchConsensusRequest struct {
		Room      string `json:"room"`
		Consensus string `json:"consensus"`
	}
	server.OnEvent("/", "switchConsensus", func(s socketio.Conn, req switchConsensusRequest) {
		fmt.Println("switchConsensus:", req)
		ctx := s.Context().(connContext)
		cn.SwitchConsensus(ctx.PlayerID, req.Room, req.Consensus)

		roomName := cn.PlayerRoomName(ctx.PlayerID)
		server.BroadcastToRoom("/", roomName, "gameState", cn.GameState(ctx.PlayerID))
	})
	server.OnEvent("/", "endTurn", func(s socketio.Conn) {
		fmt.Println("endTurn")
		ctx := s.Context().(connContext)

		cn.EndTurn(ctx.PlayerID)

		roomName := cn.PlayerRoomName(ctx.PlayerID)
		server.BroadcastToRoom("/", roomName, "gameState", cn.GameState(ctx.PlayerID))
	})

	type clickTileRequest struct {
		I int `json:"i"`
		J int `json:"j"`
	}
	server.OnEvent("/", "clickTile", func(s socketio.Conn, req clickTileRequest) {
		fmt.Println("clickTile:", req)
		ctx := s.Context().(connContext)

		cn.ClickTile(ctx.PlayerID, req.I, req.J)

		roomName := cn.PlayerRoomName(ctx.PlayerID)
		server.BroadcastToRoom("/", roomName, "gameState", cn.GameState(ctx.PlayerID))
	})

	type declareClueRequest struct {
		Word  string `json:"word"`
		Count string `json:"count"`
	}
	server.OnEvent("/", "declareClue", func(s socketio.Conn, req declareClueRequest) {
		fmt.Println("declareClue:", req)
		ctx := s.Context().(connContext)

		count, err := strconv.Atoi(req.Count)
		if err != nil {
			count = 1
		}
		cn.DeclareClue(ctx.PlayerID, req.Word, count)

		roomName := cn.PlayerRoomName(ctx.PlayerID)
		server.BroadcastToRoom("/", roomName, "gameState", cn.GameState(ctx.PlayerID))
	})

	type changeCardsRequest struct {
		Pack string `json:"pack"`
	}
	server.OnEvent("/", "changeCards", func(s socketio.Conn, req changeCardsRequest) {
		fmt.Println("changeCards:", req)
		ctx := s.Context().(connContext)

		cn.ChangeCards(ctx.PlayerID, req.Pack)

		roomName := cn.PlayerRoomName(ctx.PlayerID)
		server.BroadcastToRoom("/", roomName, "gameState", cn.GameState(ctx.PlayerID))
	})

	type timeSliderRequest struct {
		Value int `json:"value"`
	}
	server.OnEvent("/", "timeSlider", func(s socketio.Conn, req timeSliderRequest) {
		fmt.Println("timeSlider:", req)
		ctx := s.Context().(connContext)

		cn.UpdateTimeSlider(ctx.PlayerID, req.Value)

		roomName := cn.PlayerRoomName(ctx.PlayerID)
		server.BroadcastToRoom("/", roomName, "gameState", cn.GameState(ctx.PlayerID))
	})

	server.OnError("/", func(s socketio.Conn, e error) {
		ctx := s.Context().(connContext)

		roomName := cn.PlayerRoomName(ctx.PlayerID)

		if len(roomName) > 0 {
			s.Leave(roomName)
			cn.LeaveRoom(ctx.PlayerID)
		}
		server.BroadcastToRoom("/", roomName, "gameState", cn.GameState(ctx.PlayerID))

		fmt.Printf("meet error: %+v\n", e)
	})
	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		ctx := s.Context().(connContext)

		roomName := cn.PlayerRoomName(ctx.PlayerID)
		if len(roomName) > 0 {
			s.Leave(roomName)
			cn.LeaveRoom(ctx.PlayerID)
		}
		server.BroadcastToRoom("/", roomName, "gameState", cn.GameState(ctx.PlayerID))

		fmt.Println("closed", reason)
	})
	return server
}

func randID(typ string) string {
	rand.Seed(time.Now().UnixNano())
	chars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZÅÄÖ" + "abcdefghijklmnopqrstuvwxyz" + "0123456789")
	length := 16
	var b strings.Builder
	b.WriteString(typ)
	b.WriteByte(':')
	for i := 0; i < length; i++ {
		b.WriteRune(chars[rand.Intn(len(chars))])
	}
	return b.String()
}
