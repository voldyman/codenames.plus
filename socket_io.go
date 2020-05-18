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

func socketServer(a *ActionRouter) *socketio.Server {
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

		a.CheckIfPlayerExists(playerID, func(players, rooms int, playerID string, isInRoom bool, gs *gameState) {
			s.Emit("serverStats", struct {
				Players          int        `json:"players"`
				Rooms            int        `json:"rooms"`
				SessionID        string     `json:"sessionId"`
				IsExistingPlayer bool       `json:"isExistingPlayer"`
				GameState        *gameState `json:"gameState"`
			}{
				Players:          players,
				Rooms:            rooms,
				SessionID:        playerID,
				IsExistingPlayer: isInRoom,
				GameState:        gs,
			})
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

		a.CreateRoom(ctx.PlayerID, req.Nickname, req.Room, req.Password, ResEmitFunc(func(msg string, success bool) {
			if success {
				s.Join(req.Room)
			}
			s.Emit("createResponse", createRoomResponse{
				Message: msg,
				Success: success,
			})

			a.RoomForPlayer(ctx.PlayerID, func(r *Room) {
				server.BroadcastToRoom("/", r.Name, "gameState", r.GameState())
			})

		}))
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

		if len(req.Nickname) == 0 {
			s.Emit("joinResponse", joinRoomResponse{
				Message: "invalid nickname",
				Success: false,
			})
		}

		if len(req.Password) == 0 {
			s.Emit("joinResponse", joinRoomResponse{
				Message: "invalid password",
				Success: false,
			})
		}

		ok = a.JoinRoom(ctx.PlayerID, req.Nickname, req.Room, req.Password, func(r *Room) {
			if r == nil {
				s.Emit("joinResponse", joinRoomResponse{
					Message: "cannot join room",
					Success: false,
				})
				return
			}

			s.Emit("joinResponse", joinRoomResponse{
				Message: "joined room",
				Success: true,
			})

			s.Join(r.Name)
			s.Emit("gameState", r.GameState())
			server.BroadcastToRoom("/", r.Name, "gameState", r.GameState())
		})
		if !ok {
			log.Warn("joining room failed")
			s.Emit("joinResponse", joinRoomResponse{
				Message: "room not found",
				Success: false,
			})
		}
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

		a.RoomForPlayer(ctx.PlayerID, func(r *Room) {
			r.Leave(ctx.PlayerID)

			s.Leave(r.Name)

			server.BroadcastToRoom("/", r.Name, "gameState", r.GameState())
		})
		s.Emit("reset")
		s.Emit("leaveResponse", leaveRoomResponse{
			Success: true,
		})
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
			"Team":      req.Team,
		}).Info("received join team request")

		ok = a.RoomForPlayer(ctx.PlayerID, func(r *Room) {
			r.ChangeTeam(ctx.PlayerID, req.Team)

			server.BroadcastToRoom("/", r.Name, "gameState", r.GameState())
			s.Emit("gameState", r.PlayerGameState(ctx.PlayerID))
		})
		if !ok {
			s.Emit("reset")
		}
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

		ok = a.RoomForPlayer(ctx.PlayerID, func(r *Room) {
			r.RandomizeTeams(ctx.PlayerID)

			server.BroadcastToRoom("/", r.Name, "gameState", r.GameState())
		})
		if !ok {
			s.Emit("reset")
		}
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

		ok = a.RoomForPlayer(ctx.PlayerID, func(r *Room) {
			r.NewGame()

			server.BroadcastToRoom("/", r.Name, "gameState", r.GameState())
			s.Emit("gameState", r.PlayerGameState(ctx.PlayerID))
		})
		if !ok {
			s.Emit("reset")
		}
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

		ok = a.RoomForPlayer(ctx.PlayerID, func(r *Room) {
			r.SwitchRole(ctx.PlayerID, req.Role)

			s.Emit("switchRoleResponse", switchRoleResponse{
				Role:    req.Role,
				Success: true,
			})

			server.BroadcastToRoom("/", r.Name, "gameState", r.GameState())
			s.Emit("gameState", r.PlayerGameState(ctx.PlayerID))
		})
		if !ok {
			s.Emit("reset")
		}
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

		ok = a.RoomForPlayer(ctx.PlayerID, func(r *Room) {
			r.ChangeDifficulty(ctx.PlayerID, req.Difficulty)

			server.BroadcastToRoom("/", r.Name, "gameState", r.GameState())
		})
		if !ok {
			s.Emit("reset")
		}
	})

	type switchModeRequest struct {
		Mode string `json:"mode"`
	}
	server.OnEvent("/", "switchMode", func(s socketio.Conn, req switchModeRequest) {
		ctx, ok := s.Context().(connContext)
		if !ok {
			log.Warn("connection context not set in joinTeam request")
			return
		}

		log.WithFields(logrus.Fields{
			"Operation": "switchMode",
			"PlayerID":  ctx.PlayerID,
			"Mode":      req.Mode,
		}).Info("received request to switch mode")

		ok = a.RoomForPlayer(ctx.PlayerID, func(r *Room) {
			r.SwitchMode(ctx.PlayerID, req.Mode)

			server.BroadcastToRoom("/", r.Name, "gameState", r.GameState())
		})
		if !ok {
			s.Emit("reset")
		}
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
		ok = a.RoomForPlayer(ctx.PlayerID, func(r *Room) {
			r.SwitchConsensus(ctx.PlayerID, req.Consensus)

			server.BroadcastToRoom("/", r.Name, "gameState", r.GameState())
		})
		if !ok {
			s.Emit("reset")
		}
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

		ok = a.RoomForPlayer(ctx.PlayerID, func(r *Room) {
			r.EndTurn(ctx.PlayerID)

			server.BroadcastToRoom("/", r.Name, "gameState", r.GameState())
		})
		if !ok {
			s.Emit("reset")
		}
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
		ok = a.RoomForPlayer(ctx.PlayerID, func(r *Room) {
			r.SelectTile(ctx.PlayerID, req.I, req.J)

			server.BroadcastToRoom("/", r.Name, "gameState", r.GameState())
		})
		if !ok {
			s.Emit("reset")
		}
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

		ok = a.RoomForPlayer(ctx.PlayerID, func(r *Room) {
			r.DeclareClue(ctx.PlayerID, req.Word, count)

			server.BroadcastToRoom("/", r.Name, "gameState", r.GameState())
		})
		if !ok {
			s.Emit("reset")
		}
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

		ok = a.RoomForPlayer(ctx.PlayerID, func(r *Room) {
			r.ChangeCards(ctx.PlayerID, req.Pack)

			server.BroadcastToRoom("/", r.Name, "gameState", r.GameState())
		})
		if !ok {
			s.Emit("reset")
		}
	})

	type timeSliderRequest struct {
		Value string `json:"value"`
	}
	server.OnEvent("/", "timerSlider", func(s socketio.Conn, req timeSliderRequest) {
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

		val, err := strconv.ParseFloat(req.Value, 64)
		if err != nil {
			log.Warn("unable to parse time, using default")
			val = 5
		}

		ok = a.RoomForPlayer(ctx.PlayerID, func(r *Room) {
			r.ChangeTimer(ctx.PlayerID, val)

			server.BroadcastToRoom("/", r.Name, "gameState", r.GameState())
		})
		if !ok {
			s.Emit("reset")
		}
	})

	server.OnError("/", func(s socketio.Conn, e error) {
		ctx, ok := s.Context().(connContext)
		if !ok {
			log.WithField("Error", err).Warn("client context not found while handling error")
			return
		}

		ok = a.LeaveRoom(ctx.PlayerID, func(r *Room) {
			log.WithFields(logrus.Fields{
				"PlayerID": ctx.PlayerID,
				"RoomName": r.Name,
			}).Warnf("error while handling socket.io request: %+v", e)

			s.Leave(r.Name)
			server.BroadcastToRoom("/", r.Name, "gameState", r.GameState())
		})
		if !ok {
			s.Emit("reset")
		}

	})
	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		ctx, ok := s.Context().(connContext)
		if !ok {
			log.WithField("Reason", reason).Warn("received a disconnect event without client context")
			return
		}

		ok = a.LeaveRoom(ctx.PlayerID, func(r *Room) {
			log.WithFields(logrus.Fields{
				"PlayerID": ctx.PlayerID,
				"RoomName": r.Name,
				"Reason":   reason,
			}).Info("closed connection")

			s.Leave(r.Name)
			server.BroadcastToRoom("/", r.Name, "gameState", r.GameState())
		})
		if !ok {
			s.Emit("reset")
		}

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
