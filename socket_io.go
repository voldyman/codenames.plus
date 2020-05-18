package main

import (
	"math/rand"
	"net/url"
	"strconv"
	"strings"
	"time"

	socketio "github.com/googollee/go-socket.io"
	"github.com/sirupsen/logrus"
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
			if len(id) > 0 && id != "null" {
				log.WithField("ID", id).Info("Reusing id for new player")
				playerID = id
			}
		}

		ctx := connContext{
			PlayerID: playerID,
		}

		s.SetContext(ctx)
		log.WithFields(logrus.Fields{
			"SocketIO.ID": s.ID(),
			"PlayerID":    playerID,
		}).Info("connected client")

		s.Emit("reset")

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
		ctx, ok := s.Context().(connContext)
		if !ok {
			log.Warn("context was not set")
			return
		}
		log.WithFields(logrus.Fields{
			"Operation": "createRoom",
			"PlayerID":  ctx.PlayerID,
			"Room":      req.Room,
			"NickName":  req.Nickname,
			"Password":  req.Password,
		}).Info("create room request received")

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
		Message string `json:"msg"`
		Success bool   `json:"success"`
	}
	server.OnEvent("/", "joinRoom", func(s socketio.Conn, req joinRoomRequest) {
		ctx, ok := s.Context().(connContext)
		if !ok {
			log.Warn("context was not set")
			return
		}
		log.WithFields(logrus.Fields{
			"Operation": "joinRoom",
			"PlayerID":  ctx.PlayerID,
			"Room":      req.Room,
			"NickName":  req.Nickname,
			"Password":  req.Password,
		}).Info("join room request received")

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
		ctx, ok := s.Context().(connContext)
		if !ok {
			log.Warn("connection context not set in leave room request")
			return
		}
		log.WithFields(logrus.Fields{
			"Operation": "leaveRoomRequest",
			"PlayerID":  ctx.PlayerID,
		}).Info("received leaveRoomRequest")

		ok = cn.LeaveRoom(ctx.PlayerID)
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
		ctx, ok := s.Context().(connContext)
		if !ok {
			log.Warn("connection context not set in joinTeam request")
			return
		}

		log.WithFields(logrus.Fields{
			"Operation": "joinTeam",
			"PlayerID":  ctx.PlayerID,
			"Team":      req.Team}).Info("received join team request")

		ok = cn.JoinTeam(ctx.PlayerID, req.Team)
		if !ok {
			s.Emit("reset")
			return
		}

		roomName := cn.PlayerRoomName(ctx.PlayerID)
		server.BroadcastToRoom("/", roomName, "gameState", cn.GameState(ctx.PlayerID))
	})

	server.OnEvent("/", "randomizeTeams", func(s socketio.Conn, req map[string]interface{}) {
		ctx, ok := s.Context().(connContext)
		if !ok {
			log.Warn("connection context not set in randomize teams request")
			return
		}

		log.WithFields(logrus.Fields{
			"Operation": "randomizeTeams",
			"PlayerID":  ctx.PlayerID,
		}).Info("randomize teams request")

		ok = cn.RandomizeTeams(ctx.PlayerID)
		if !ok {
			s.Emit("reset")
			return
		}

		roomName := cn.PlayerRoomName(ctx.PlayerID)
		server.BroadcastToRoom("/", roomName, "gameState", cn.GameState(ctx.PlayerID))
	})
	server.OnEvent("/", "newGame", func(s socketio.Conn, vals map[string]interface{}) {
		ctx, ok := s.Context().(connContext)
		if !ok {
			log.Warn("connection context not set in joinTeam request")
			return
		}

		log.WithFields(logrus.Fields{
			"Operation": "newGame",
			"PlayerID":  ctx.PlayerID,
		}).Info("received new game request")

		ok = cn.NewGame(ctx.PlayerID)
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
		ctx, ok := s.Context().(connContext)
		if !ok {
			log.Warn("connection context not set in joinTeam request")
			return
		}

		log.WithFields(logrus.Fields{
			"Operation": "switchRole",
			"PlayerID":  ctx.PlayerID,
			"Role":      req.Role,
		}).Info("received switch role request")

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
		ctx, ok := s.Context().(connContext)
		if !ok {
			log.Warn("connection context not set in joinTeam request")
			return
		}

		log.WithFields(logrus.Fields{
			"Operation":  "switchDifficulty",
			"PlayerID":   ctx.PlayerID,
			"Difficulty": req.Difficulty,
		}).Info("received request to switch difficulty")

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
		ctx, ok := s.Context().(connContext)
		if !ok {
			log.Warn("connection context not set in joinTeam request")
			return
		}

		log.WithFields(logrus.Fields{
			"Operation":   "switchMode",
			"PlayerID":    ctx.PlayerID,
			"Room":        req.Room,
			"Mode":        req.Mode,
			"TimerAmount": req.TimerAmount,
		}).Info("received request to switch mode")

		cn.SwitchMode(ctx.PlayerID, req.Room, req.Mode, req.TimerAmount)

		roomName := cn.PlayerRoomName(ctx.PlayerID)
		server.BroadcastToRoom("/", roomName, "gameState", cn.GameState(ctx.PlayerID))
	})

	type switchConsensusRequest struct {
		Room      string `json:"room"`
		Consensus string `json:"consensus"`
	}
	server.OnEvent("/", "switchConsensus", func(s socketio.Conn, req switchConsensusRequest) {
		ctx, ok := s.Context().(connContext)
		if !ok {
			log.Warn("connection context not set in joinTeam request")
			return
		}

		log.WithFields(logrus.Fields{
			"Operation": "switchConsensus",
			"PlayerID":  ctx.PlayerID,
			"Room":      req.Room,
			"Consensus": req.Consensus,
		}).Info("received request to switch consensus")

		cn.SwitchConsensus(ctx.PlayerID, req.Room, req.Consensus)

		roomName := cn.PlayerRoomName(ctx.PlayerID)
		server.BroadcastToRoom("/", roomName, "gameState", cn.GameState(ctx.PlayerID))
	})
	server.OnEvent("/", "endTurn", func(s socketio.Conn) {
		ctx, ok := s.Context().(connContext)
		if !ok {
			log.Warn("connection context not set in joinTeam request")
			return
		}

		log.WithFields(logrus.Fields{
			"Operation": "endTurn",
			"PlayerID":  ctx.PlayerID,
		}).Info("received request to end turn")

		cn.EndTurn(ctx.PlayerID)

		roomName := cn.PlayerRoomName(ctx.PlayerID)
		server.BroadcastToRoom("/", roomName, "gameState", cn.GameState(ctx.PlayerID))
	})

	type clickTileRequest struct {
		I int `json:"i"`
		J int `json:"j"`
	}
	server.OnEvent("/", "clickTile", func(s socketio.Conn, req clickTileRequest) {
		ctx, ok := s.Context().(connContext)
		if !ok {
			log.Warn("connection context not set in joinTeam request")
			return
		}

		log.WithFields(logrus.Fields{
			"Operation": "clickTile",
			"PlayerID":  ctx.PlayerID,
			"I":         req.I,
			"J":         req.J,
		}).Info("received click tile request")

		cn.ClickTile(ctx.PlayerID, req.I, req.J)

		roomName := cn.PlayerRoomName(ctx.PlayerID)
		server.BroadcastToRoom("/", roomName, "gameState", cn.GameState(ctx.PlayerID))
	})

	type declareClueRequest struct {
		Word  string `json:"word"`
		Count string `json:"count"`
	}
	server.OnEvent("/", "declareClue", func(s socketio.Conn, req declareClueRequest) {
		ctx, ok := s.Context().(connContext)
		if !ok {
			log.Warn("connection context not set in joinTeam request")
			return
		}

		log.WithFields(logrus.Fields{
			"Operation": "declareClue",
			"PlayerID":  ctx.PlayerID,
			"Word":      req.Word,
			"Count":     req.Count,
		}).Info("received declare clue request")

		count, err := strconv.Atoi(req.Count)
		if err != nil {
			log.WithField("Count", req.Count).Warn("could not convert count to integer, using 1 as default")
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
		ctx, ok := s.Context().(connContext)
		if !ok {
			log.Warn("connection context not set in joinTeam request")
			return
		}

		log.WithFields(logrus.Fields{
			"Operation": "changeCards",
			"PlayerID":  ctx.PlayerID,
			"Pack":      req.Pack,
		}).Info("received change cards pack request")

		cn.ChangeCards(ctx.PlayerID, req.Pack)

		roomName := cn.PlayerRoomName(ctx.PlayerID)
		server.BroadcastToRoom("/", roomName, "gameState", cn.GameState(ctx.PlayerID))
	})

	type timeSliderRequest struct {
		Value int `json:"value"`
	}
	server.OnEvent("/", "timeSlider", func(s socketio.Conn, req timeSliderRequest) {
		ctx, ok := s.Context().(connContext)
		if !ok {
			log.Warn("connection context not set in joinTeam request")
			return
		}

		log.WithFields(logrus.Fields{
			"Operation": "timeSlider",
			"PlayerID":  ctx.PlayerID,
			"Value":     req.Value,
		}).Info("received update slider request")

		cn.UpdateTimeSlider(ctx.PlayerID, req.Value)

		roomName := cn.PlayerRoomName(ctx.PlayerID)
		server.BroadcastToRoom("/", roomName, "gameState", cn.GameState(ctx.PlayerID))
	})

	server.OnError("/", func(s socketio.Conn, e error) {
		ctx, ok := s.Context().(connContext)
		if !ok {
			log.WithField("Error", err).Warn("client context not found while handling error")
			return
		}

		roomName := cn.PlayerRoomName(ctx.PlayerID)

		if len(roomName) > 0 {
			s.Leave(roomName)
			cn.LeaveRoom(ctx.PlayerID)
		}
		server.BroadcastToRoom("/", roomName, "gameState", cn.GameState(ctx.PlayerID))
		log.Warnf("error while handling socket.io request: %+v", e)
	})
	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		ctx, ok := s.Context().(connContext)
		if !ok {
			log.WithField("Reason", reason).Warn("received a disconnect event without client context")
			return
		}

		roomName := cn.PlayerRoomName(ctx.PlayerID)
		if len(roomName) > 0 {
			s.Leave(roomName)
			cn.LeaveRoom(ctx.PlayerID)
		}
		server.BroadcastToRoom("/", roomName, "gameState", cn.GameState(ctx.PlayerID))

		log.WithField("reason", reason).Info("closed connection")
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
