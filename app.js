
////////////////////////////////////////////////////////////////////////////

// Express
let express = require('express')

// Create app
let app = express()
app.disable('x-powered-by');

//Set up server
let server = app.listen(process.env.PORT || 2000, listen);

let requireHttps = (process.env.REQUIRE_HTTPS == "true");

// Callback function confirming server start
function listen(){
  let host = server.address().address;
  let port = server.address().port;
  console.log('Codenames Server Started at http://' + host + ':' + port);

  if (requireHttps) {
    console.log("Https Required");
  }
}

// Force SSL
app.use((req, res, next) => {
  if (requireHttps && req.header('x-forwarded-proto') !== 'https') {
    res.redirect(`https://${req.header('host')}${req.url}`)
  } else {
    next();
  }
});

//Keep session alive
app.get("/ping", (request, response) => {
  console.log(Date.now() + " Ping Received");
  response.sendStatus(200);
});
const https = require('https');
setInterval(() => {
  https.get(`https://${process.env.PROJECT_DOMAIN}.glitch.me/ping`);
}, 280000);

// Files for client
app.use(express.static('public'))

// Websocket
let io = require('socket.io')(server)

// Catch wildcard socket events
var middleware = require('socketio-wildcard')()
io.use(middleware)

// Make API requests
const Heroku = require('heroku-client')
const heroku = new Heroku({ token:process.env.API_TOKEN})// DELETE requests

// Daily Server Restart time
// UTC 13:00:00 = 9AM EST
let restartHour = 11//13 original
let restartMinute = 0//0
let restartSecond = 5
// restart warning time
let restartWarningHour = 10//12 original
let restartWarningMinute = 50//50
let restartWarningSecond = 2

////////////////////////////////////////////////////////////////////////////

// Codenames Game
const Game = require('./server/game.js')

// Objects to keep track of sockets, rooms and players
let SOCKET_LIST = {}
let ROOM_LIST = {}
let PLAYER_LIST = {}

// Room class
// Live rooms will have a name and password and keep track of game options / players in room
class Room {
  constructor(name, pass){
    this.room = '' + name
    this.password = '' + pass
    this.players = {}
    this.game = new Game()
    this.difficulty = 'normal'
    this.mode = 'casual'
    this.consensus = 'single'

    // Add room to room list
    ROOM_LIST[this.room] = this
  }
}

// Player class
// When players log in, they give a nickname, have a socket and a room they're trying to connect to
class Player {
  constructor(nickname, room, socket){
    this.id = socket.id

    // If someone in the room has the same name, append (1) to their nickname
    let nameAvailable = false
    let nameExists = false;
    let tempName = nickname
    let counter = 0
    while (!nameAvailable){
      if (ROOM_LIST[room]){
        nameExists = false;
        for (let i in ROOM_LIST[room].players){
          if (ROOM_LIST[room].players[i].nickname === tempName) nameExists = true
        }
        if (nameExists) tempName = nickname + "(" + ++counter + ")"
        else nameAvailable = true
      }
    }
    this.nickname = tempName
    this.room = room
    this.team = 'undecided'
    this.role = 'guesser'
    this.guessProposal = null
    this.timeout = 10800         // # of seconds until kicked for afk (3 hrs)
    this.afktimer = this.timeout

    // Add player to player list and add their socket to the socket list
    PLAYER_LIST[this.id] = this
  }

  // When a player joins a room, evenly distribute them to a team
  joinTeam(){
    let numInRoom = Object.keys(ROOM_LIST[this.room].players).length
    if (numInRoom % 2 === 0) this.team = 'blue'
    else this.team = 'red'
  }
}


// Server logic
////////////////////////////////////////////////////////////////////////////
io.sockets.on('connection', function(socket){

  // Alert server of the socket connection
  SOCKET_LIST[socket.id] = socket
  logStats('CONNECT: ' + socket.id)

  // Pass server stats to client
  socket.emit('serverStats', {
    players: Object.keys(PLAYER_LIST).length,
    rooms: Object.keys(ROOM_LIST).length
  })

  // LOBBY STUFF
  ////////////////////////////////////////////////////////////////////////////

  // Room Creation. Called when client attempts to create a rooom
  // Data: player nickname, room name, room password
  socket.on('createRoom', (data) => {createRoom(socket, data)})

  // Room Joining. Called when client attempts to join a room
  // Data: player nickname, room name, room password
  socket.on('joinRoom', (data) => {joinRoom(socket, data)})

  // Room Leaving. Called when client leaves a room
  socket.on('leaveRoom', () =>{leaveRoom(socket)})

  // Client Disconnect
  socket.on('disconnect', () => {socketDisconnect(socket)})


  // GAME STUFF
  ////////////////////////////////////////////////////////////////////////////

  // Join Team. Called when client joins a team (red / blue)
  // Data: team color
  socket.on('joinTeam', (data) => {
    if (!PLAYER_LIST[socket.id]) return // Prevent Crash
    let player = PLAYER_LIST[socket.id];  // Get player who made request
    player.team = data.team               // Update their team
    gameUpdate(player.room)               // Update the game for everyone in their room
  })

  // Randomize Team. Called when client randomizes the teams
  socket.on('randomizeTeams', () => {randomizeTeams(socket)})

  // New Game. Called when client starts a new game
  socket.on('newGame', () =>{newGame(socket)})

  // Switch Role. Called when client switches to spymaster / guesser
  // Data: New role
  socket.on('switchRole', (data) => {switchRole(socket, data)})

  // Switch Difficulty. Called when spymaster switches to hard / normal
  // Data: New difficulty
  socket.on('switchDifficulty', (data) => {
    if (!PLAYER_LIST[socket.id]) return // Prevent Crash
    let room = PLAYER_LIST[socket.id].room        // Get room the client was in
    ROOM_LIST[room].difficulty = data.difficulty  // Update the rooms difficulty
    gameUpdate(room)                              // Update the game for everyone in this room
  })

  // Switch Mode. Called when client switches to casual / timed
  // Data: New mode
  socket.on('switchMode', (data) => {
    if (!PLAYER_LIST[socket.id]) return // Prevent Crash
    let room = PLAYER_LIST[socket.id].room  // Get the room the client was in
    ROOM_LIST[room].mode = data.mode;       // Update the rooms game mode
    ROOM_LIST[room].game.timer = ROOM_LIST[room].game.timerAmount;   // Reset the timer in the room's game
    gameUpdate(room)                        // Update the game for everyone in this room
  })

  // Switch Consensus Mode. Called when client switches to single / consensus
  // Data: New consensus mode
  socket.on('switchConsensus', (data) => {
    if (!PLAYER_LIST[socket.id]) return // Prevent Crash
    let room = PLAYER_LIST[socket.id].room  // Get the room the client was in
    clearGuessProsposals(room)
    ROOM_LIST[room].consensus = data.consensus;       // Update the rooms consensus mode
    gameUpdate(room)                        // Update the game for everyone in this room
  })

  // End Turn. Called when client ends teams turn
  socket.on('endTurn', () => {
    if (!PLAYER_LIST[socket.id]) return // Prevent Crash
    let room = PLAYER_LIST[socket.id].room  // Get the room the client was in
    ROOM_LIST[room].game.switchTurn(true)       // Switch the room's game's turn
    clearGuessProsposals(room)
    gameUpdate(room)                        // Update the game for everyone in this room
  })

  // Click Tile. Called when client clicks a tile
  // Data: x and y location of tile in grid
  socket.on('clickTile', (data) => {clickTile(socket, data)})

  socket.on('declareClue', (data) => {declareClue(socket, data)})

  // Active. Called whenever client interacts with the game, resets afk timer
  socket.on('*', () => {
    if (!PLAYER_LIST[socket.id]) return // Prevent Crash
    PLAYER_LIST[socket.id].afktimer = PLAYER_LIST[socket.id].timeout
  })

  // Change card packs
  socket.on('changeCards', (data) => {
    if (!PLAYER_LIST[socket.id]) return // Prevent Crash
    let room = PLAYER_LIST[socket.id].room  // Get the room the client was in
    let game = ROOM_LIST[room].game
    if(data.pack === 'base'){               // Toggle packs in the game
      game.base = !game.base
    } else if (data.pack === 'duet'){
      game.duet = !game.duet
    } else if (data.pack === 'undercover'){
      game.undercover = !game.undercover
    } else if (data.pack === 'custom'){
      game.custom = !game.custom
    } else if (data.pack === 'nsfw'){
      game.nsfw = !game.nsfw
    }
    // If all options are disabled, re-enable the base pack
    if (!game.base && !game.duet && !game.undercover && !game.custom && !game.nsfw) game.base = true

    game.updateWordPool()
    gameUpdate(room)

  })

  // Change timer slider
  socket.on('timerSlider', (data) => {
    if (!PLAYER_LIST[socket.id]) return // Prevent Crash
    let room = PLAYER_LIST[socket.id].room  // Get the room the client was in
    let game = ROOM_LIST[room].game
    let currentAmount = game.timerAmount - 1  // Current timer amount
    let seconds = (data.value * 60) + 1       // the new amount of the slider
    if (currentAmount !== seconds){           // if they dont line up, update clients
      game.timerAmount = seconds
      game.timer = game.timerAmount
      gameUpdate(room)
    }
  })
})

// Create room function
// Gets a room name and password and attempts to make a new room if one doesn't exist
// On creation, the client that created the room is created and added to the room
function createRoom(socket, data){
  let roomName = data.room.trim()     // Trim whitespace from room name
  let passName = data.password.trim() // Trim whitespace from password
  let userName = data.nickname.trim() // Trim whitespace from nickname

  if (ROOM_LIST[roomName]) {   // If the requested room name is taken
    // Tell the client the room arleady exists
    socket.emit('createResponse', {success:false, msg:'Room Already Exists'})
  } else {
    if (roomName === "") {
      // Tell the client they need a valid room name
      socket.emit('createResponse', {success:false, msg:'Enter A Valid Room Name'})
    } else {
      if (userName === ''){
        // Tell the client they need a valid nickname
        socket.emit('createResponse', {success:false, msg:'Enter A Valid Nickname'})
      } else {    // If the room name and nickname are both valid, proceed
        new Room(roomName, passName)                          // Create a new room
        let player = new Player(userName, roomName, socket)   // Create a new player
        ROOM_LIST[roomName].players[socket.id] = player       // Add player to room
        player.joinTeam()                                     // Distribute player to team
        socket.emit('createResponse', {success:true, msg: ""})// Tell client creation was successful
        gameUpdate(roomName)                                  // Update the game for everyone in this room
        logStats(socket.id + "(" + player.nickname + ") CREATED '" + ROOM_LIST[player.room].room + "'(" + Object.keys(ROOM_LIST[player.room].players).length + ")")
      }
    }
  }
}

// Join room function
// Gets a room name and poassword and attempts to join said room
// On joining, the client that joined the room is created and added to the room
function joinRoom(socket, data){
  let roomName = data.room.trim()     // Trim whitespace from room name
  let pass = data.password.trim()     // Trim whitespace from password
  let userName = data.nickname.trim() // Trim whitespace from nickname

  if (!ROOM_LIST[roomName]){
    // Tell client the room doesnt exist
    socket.emit('joinResponse', {success:false, msg:"Room Not Found"})
  } else {
    if (ROOM_LIST[roomName].password !== pass){
      // Tell client the password is incorrect
      socket.emit('joinResponse', {success:false, msg:"Incorrect Password"})
    } else {
      if (userName === ''){
        // Tell client they need a valid nickname
        socket.emit('joinResponse', {success:false, msg:'Enter A Valid Nickname'})
      } else {  // If the room exists and the password / nickname are valid, proceed
        let player = new Player(userName, roomName, socket)   // Create a new player
        ROOM_LIST[roomName].players[socket.id] = player       // Add player to room
        player.joinTeam()                                     // Distribute player to team
        socket.emit('joinResponse', {success:true, msg:""})   // Tell client join was successful
        gameUpdate(roomName)                                  // Update the game for everyone in this room
        // Server Log
        logStats(socket.id + "(" + player.nickname + ") JOINED '" + ROOM_LIST[player.room].room + "'(" + Object.keys(ROOM_LIST[player.room].players).length + ")")
      }
    }
  }
}

// Leave room function
// Gets the client that left the room and removes them from the room's player list
function leaveRoom(socket){
  if (!PLAYER_LIST[socket.id]) return // Prevent Crash
  let player = PLAYER_LIST[socket.id]              // Get the player that made the request
  delete PLAYER_LIST[player.id]                    // Delete the player from the player list
  delete ROOM_LIST[player.room].players[player.id] // Remove the player from their room
  gameUpdate(player.room)                          // Update everyone in the room
  // Server Log
  logStats(socket.id + "(" + player.nickname + ") LEFT '" + ROOM_LIST[player.room].room + "'(" + Object.keys(ROOM_LIST[player.room].players).length + ")")

  // If the number of players in the room is 0 at this point, delete the room entirely
  if (Object.keys(ROOM_LIST[player.room].players).length === 0) {
    delete ROOM_LIST[player.room]
    logStats("DELETE ROOM: '" + player.room + "'")
  }
  socket.emit('leaveResponse', {success:true})     // Tell the client the action was successful
}

// Disconnect function
// Called when a client closes the browser tab
function socketDisconnect(socket){
  let player = PLAYER_LIST[socket.id] // Get the player that made the request
  delete SOCKET_LIST[socket.id]       // Delete the client from the socket list
  delete PLAYER_LIST[socket.id]       // Delete the player from the player list

  if(player){   // If the player was in a room
    delete ROOM_LIST[player.room].players[socket.id] // Remove the player from their room
    gameUpdate(player.room)                          // Update everyone in the room
    // Server Log
    logStats(socket.id + "(" + player.nickname + ") LEFT '" + ROOM_LIST[player.room].room + "'(" + Object.keys(ROOM_LIST[player.room].players).length + ")")

    // If the number of players in the room is 0 at this point, delete the room entirely
    if (Object.keys(ROOM_LIST[player.room].players).length === 0) {
      delete ROOM_LIST[player.room]
      logStats("DELETE ROOM: '" + player.room + "'")
    }
  }
  // Server Log
  logStats('DISCONNECT: ' + socket.id)
}

// Randomize Teams function
// Will mix up the teams in the room that the client is in
function randomizeTeams(socket){
  if (!PLAYER_LIST[socket.id]) return // Prevent Crash
  let room = PLAYER_LIST[socket.id].room   // Get the room that the client called from
  let players = ROOM_LIST[room].players    // Get the players in the room

  let color = 0;    // Get a starting color
  if (Math.random() < 0.5) color = 1

  let keys = Object.keys(players) // Get a list of players in the room from the dictionary
  let placed = []                 // Init a temp array to keep track of who has already moved

  while (placed.length < keys.length){
    let selection = keys[Math.floor(Math.random() * keys.length)] // Select random player index
    if (!placed.includes(selection)) placed.push(selection) // If index hasn't moved, move them
  }

  // Place the players in alternating teams from the new random order
  for (let i = 0; i < placed.length; i++){
    let player = players[placed[i]]
    if (color === 0){
      player.team = 'red'
      color = 1
    } else {
      player.team = 'blue'
      color = 0
    }
  }
  gameUpdate(room) // Update everyone in the room
}

// New game function
// Gets client that requested the new game and instantiates a new game board for the room
function newGame(socket){
  if (!PLAYER_LIST[socket.id]) return // Prevent Crash
  let room = PLAYER_LIST[socket.id].room  // Get the room that the client called from
  ROOM_LIST[room].game.init();      // Make a new game for that room

  // Make everyone in the room a guesser and tell their client the game is new
  for(let player in ROOM_LIST[room].players){
    PLAYER_LIST[player].role = 'guesser';
    PLAYER_LIST[player].guessProposal = null;
    SOCKET_LIST[player].emit('switchRoleResponse', {success:true, role:'guesser'})
    SOCKET_LIST[player].emit('newGameResponse', {success:true})
  }
  gameUpdate(room) // Update everyone in the room
}

// Switch role function
// Gets clients requested role and switches it
function switchRole(socket, data){
  let currentPlayer = PLAYER_LIST[socket.id]
  if (!currentPlayer) return // Prevent Crash
  let room = currentPlayer.room // Get the room that the client called from

  if (currentPlayer.team === 'undecided'){
    // Dissallow the client a role switch if they're not on a team
    socket.emit('switchRoleResponse', {success:false})
    return
  }

  if (currentPlayer.role === 'spymaster'){
    // Dissallow the client a role switch if they're already spymaster
    //   so they've seen the answers.
    socket.emit('switchRoleResponse', {success:false})
    return
  }

  // Do not allow to switch to spymaster if there is already one in the team
  if (data.role === 'spymaster') {
    for (let player in ROOM_LIST[room].players) {
      const otherPlayer = PLAYER_LIST[player];
      if (otherPlayer !== currentPlayer && otherPlayer.team === currentPlayer.team && otherPlayer.role === 'spymaster') {
        socket.emit('switchRoleResponse', {success:false})
        return
      }
    }
  }

  currentPlayer.role = data.role; // Set the new role
  socket.emit('switchRoleResponse', {success:true, role:data.role}) // Alert client
  gameUpdate(room) // Update everyone in the room
}

// Click tile function
// Gets client and the tile they clicked and pushes that change to the rooms game
function clickTile(socket, data){
  if (!PLAYER_LIST[socket.id]) return // Prevent Crash
  let room = PLAYER_LIST[socket.id].room  // Get the room that the client called from

  if (PLAYER_LIST[socket.id].team === ROOM_LIST[room].game.turn){ // If it was this players turn
    if (!ROOM_LIST[room].game.over){  // If the game is not over
      if (PLAYER_LIST[socket.id].role !== 'spymaster'){ // If the client isnt spymaster
        var doFlip = true
        if (ROOM_LIST[room].consensus === 'consensus'){
          let guess = ROOM_LIST[room].game.board[data.i][data.j].word
          // If player already made this guess, then toggle to them not making any guess.
          if (PLAYER_LIST[socket.id].guessProposal === guess){
            PLAYER_LIST[socket.id].guessProposal = null
            gameUpdate(room)  // Update everyone in the room
            return
          }
          PLAYER_LIST[socket.id].guessProposal = guess
          var allAgree = true
          for (let player in ROOM_LIST[room].players){
            if (PLAYER_LIST[player].guessProposal !== guess && PLAYER_LIST[player].role !== 'spymaster' && PLAYER_LIST[player].team === ROOM_LIST[room].game.turn){
              doFlip = false
              break
            }
          }
        }
        if (doFlip){
          ROOM_LIST[room].game.flipTile(data.i, data.j) // Send the flipped tile info to the game
          clearGuessProsposals(room)
        }
        gameUpdate(room)  // Update everyone in the room
      }
    }
  }
}

// Declare clue function
// Gets client and the clue they gave and pushes that change to the rooms game
function declareClue(socket, data){
  if (!PLAYER_LIST[socket.id]) return // Prevent Crash
  let room = PLAYER_LIST[socket.id].room  // Get the room that the client called from
  let game = ROOM_LIST[room].game

  if (PLAYER_LIST[socket.id].team === game.turn){ // If it was this players turn
    if (!game.over){  // If the game is not over
      if (PLAYER_LIST[socket.id].role === 'spymaster'){ // If the client is spymaster
        if (game.declareClue(data)){
          gameUpdate(room)  // Update everyone in the room
        }
      }
    }
  }
}

function clearGuessProsposals(room){
  for (let player in ROOM_LIST[room].players){
    PLAYER_LIST[player].guessProposal = null
  }
}

// Update the gamestate for every client in the room that is passed to this function
function gameUpdate(room){
  // Create data package to send to the client
  let gameState = {
    room: room,
    players:ROOM_LIST[room].players,
    game:ROOM_LIST[room].game,
    difficulty:ROOM_LIST[room].difficulty,
    mode:ROOM_LIST[room].mode,
    consensus:ROOM_LIST[room].consensus
  }
  for (let player in ROOM_LIST[room].players){ // For everyone in the passed room
    gameState.team = PLAYER_LIST[player].team  // Add specific clients team info
    SOCKET_LIST[player].emit('gameState', gameState)  // Pass data to the client
  }
}

function logStats(addition){
  let inLobby = Object.keys(SOCKET_LIST).length - Object.keys(PLAYER_LIST).length
  let stats = '[R:' + Object.keys(ROOM_LIST).length + " P:" + Object.keys(PLAYER_LIST).length + " L:" + inLobby + "] "
  console.log(stats + addition)
}

// Restart Heroku Server
function herokuRestart(){
  // Let each socket know the server restarted and boot them to lobby
  for (let socket in SOCKET_LIST){
    SOCKET_LIST[socket].emit('serverMessage', {msg:"Server Successfully Restarted for Maintnence"})
    SOCKET_LIST[socket].emit('leaveResponse', {success:true})
  }
  heroku.delete('/apps/codenames-plus/dynos/').then(app => {})
}

// Warn users of restart
function herokuRestartWarning(){
  for (let player in PLAYER_LIST){
    SOCKET_LIST[player].emit('serverMessage', {msg:"Scheduled Server Restart in 10 Minutes"})
  }
}

// Every second, update the timer in the rooms that are on timed mode
setInterval(()=>{
  // Server Daily Restart Logic
  let time = new Date()
  // Warn clients of restart 10min in advance
  if (time.getHours() === restartWarningHour &&
      time.getMinutes() === restartWarningMinute &&
      time.getSeconds() < restartWarningSecond) herokuRestartWarning()
  // Restart server at specified time
  if (time.getHours() === restartHour &&
      time.getMinutes() === restartMinute &&
      time.getSeconds() < restartSecond) herokuRestart()

  // AFK Logic
  for (let player in PLAYER_LIST){
    PLAYER_LIST[player].afktimer--      // Count down every players afk timer
    // Give them a warning 5min before they get kicked
    if (PLAYER_LIST[player].afktimer < 300) SOCKET_LIST[player].emit('afkWarning')
    if (PLAYER_LIST[player].afktimer < 0) {   // Kick player if their timer runs out
      SOCKET_LIST[player].emit('afkKicked')
      logStats(player + "(" + PLAYER_LIST[player].nickname + ") AFK KICKED FROM '" + ROOM_LIST[PLAYER_LIST[player].room].room + "'(" + Object.keys(ROOM_LIST[PLAYER_LIST[player].room].players).length + ")")
      leaveRoom(SOCKET_LIST[player])
    }
  }
  // Game Timer Logic
  for (let room in ROOM_LIST){
    if (ROOM_LIST[room].mode === 'timed'
      && ROOM_LIST[room].game.over === false){
      ROOM_LIST[room].game.timer--          // If the room is in timed mode, count timer down

      if (ROOM_LIST[room].game.timer < 0){  // If timer runs out, switch that rooms turn
        ROOM_LIST[room].game.switchTurn()
        gameUpdate(room)   // Update everyone in the room
      }

      // Update the timer value to every client in the room
      for (let player in ROOM_LIST[room].players){
        SOCKET_LIST[player].emit('timerUpdate', {timer:ROOM_LIST[room].game.timer})
      }
    }
  }
}, 1000)
