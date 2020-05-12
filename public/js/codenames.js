let socket = io({path: window.location.pathname + 'socket.io'}) // Connect to server


// Sign In Page Elements
////////////////////////////////////////////////////////////////////////////
// Divs
let joinDiv = document.getElementById('join-game')
let joinErrorMessage = document.getElementById('error-message')
// Input Fields
let joinNickname = document.getElementById('join-nickname')
let joinRoom = document.getElementById('join-room')
let joinPassword = document.getElementById('join-password')
// Buttons
let joinEnter = document.getElementById('join-enter')
let joinCreate = document.getElementById('join-create')


// Game Page Elements
////////////////////////////////////////////////////////////////////////////
// Divs
let gameDiv = document.getElementById('game')
let clueEntryDiv = document.getElementById('clue-form')
let boardDiv = document.getElementById('board')
let aboutWindow = document.getElementById('about-window')
let afkWindow = document.getElementById('afk-window')
let serverMessageWindow = document.getElementById('server-message')
let serverMessage = document.getElementById('message')
let overlay = document.getElementById('overlay')
let logDiv = document.getElementById('log')
// Buttons
let leaveRoom = document.getElementById('leave-room')
let joinRed = document.getElementById('join-red')
let joinBlue = document.getElementById('join-blue')
let randomizeTeams = document.getElementById('randomize-teams')
let endTurn = document.getElementById('end-turn')
let newGame = document.getElementById('new-game')
let clueDeclareButton = document.getElementById('declare-clue')
let buttonRoleGuesser = document.getElementById('role-guesser')
let buttonRoleSpymaster = document.getElementById('role-spymaster')
let toggleDifficulty = document.getElementById('player-difficulty')
let buttonDifficultyNormal = document.getElementById('difficulty-normal')
let buttonDifficultyHard = document.getElementById('difficulty-hard')
let buttonModeCasual = document.getElementById('mode-casual')
let buttonModeTimed = document.getElementById('mode-timed')
let buttonConsensusSingle = document.getElementById('consensus-single')
let buttonConsensusConsensus = document.getElementById('consensus-consensus')
let buttonAbout = document.getElementById('about-button')
let buttonAfk = document.getElementById('not-afk')
let buttonServerMessageOkay = document.getElementById('server-message-okay')
let buttonBasecards = document.getElementById('base-pack')
let buttonDuetcards = document.getElementById('duet-pack')
let buttonUndercovercards = document.getElementById('undercover-pack')
let buttonCustomcards = document.getElementById('custom-pack')
let buttonNsfwcards = document.getElementById('nsfw-pack')// Clue entry
let clueWord = document.getElementById('clue-word')
let clueCount = document.getElementById('clue-count')
// Slider
let timerSlider = document.getElementById('timer-slider')
let timerSliderLabel = document.getElementById('timer-slider-label')
// Player Lists
let undefinedList = document.getElementById('undefined-list')
let redTeam = document.getElementById('red-team')
let blueTeam = document.getElementById('blue-team')
// UI Elements
let scoreRed = document.getElementById('score-red')
let scoreBlue = document.getElementById('score-blue')
let turnMessage = document.getElementById('status')
let timer = document.getElementById('timer')
let clueDisplay = document.getElementById('clue-display')


// init
////////////////////////////////////////////////////////////////////////////
// Default game settings
let playerRole = 'guesser'
let difficulty = 'normal'
let mode = 'casual'
let consensus = 'single'

// Show the proper toggle options
buttonModeCasual.disabled = true;
buttonModeTimed.disabled = false;
buttonRoleGuesser.disabled = true;
buttonRoleSpymaster.disabled = false;
buttonConsensusSingle.disabled = true;
buttonConsensusConsensus.disabled = false;

// Autofill room and password from query fragment
joinRoom.value = extractFromFragment(window.location.hash, 'room');
joinPassword.value = extractFromFragment(window.location.hash, 'password');


// UI Interaction with server
////////////////////////////////////////////////////////////////////////////
// User Joins Room
joinEnter.onclick = () => {       
  socket.emit('joinRoom', {
    nickname:joinNickname.value,
    room:joinRoom.value,
    password:joinPassword.value
  })
}
// User Creates Room
joinCreate.onclick = () => {      
  socket.emit('createRoom', {
    nickname:joinNickname.value,
    room:joinRoom.value,
    password:joinPassword.value
  })
}
// User Leaves Room
leaveRoom.onclick = () => {       
  socket.emit('leaveRoom', {})
}
// User Joins Red Team
joinRed.onclick = () => {         
  socket.emit('joinTeam', {
    team:'red'
  })
}
// User Joins Blue Team
joinBlue.onclick = () => {        
  socket.emit('joinTeam', {
    team:'blue'
  })
}
// User Randomizes Team
randomizeTeams.onclick = () => {  
  socket.emit('randomizeTeams', {})
}
// User Starts New Game
newGame.onclick = () => {         
  socket.emit('newGame', {})
}
clueDeclareButton.onclick = () => {
  socket.emit('declareClue', {word: clueWord.value, count: clueCount.value})
  clueWord.value = ''
  clueCount.value = 1
  return false
}
// User Picks spymaster Role
buttonRoleSpymaster.onclick = () => { 
  socket.emit('switchRole', {role:'spymaster'})
}
// User Picks guesser Role
buttonRoleGuesser.onclick = () => {   
  socket.emit('switchRole', {role:'guesser'})
}
// User Picks Hard Difficulty
buttonDifficultyHard.onclick = () => {
  socket.emit('switchDifficulty', {difficulty:'hard'})
}
// User Picks Normal Difficulty 
buttonDifficultyNormal.onclick = () => {
  socket.emit('switchDifficulty', {difficulty:'normal'})
}
// User Picks Timed Mode
buttonModeTimed.onclick = () => { 
  socket.emit('switchMode', {mode:'timed'})
}
// User Picks Casual Mode
buttonModeCasual.onclick = () => {
  socket.emit('switchMode', {mode:'casual'})
}
// User Picks Single Consensus Mode
buttonConsensusSingle.onclick = () => {
  socket.emit('switchConsensus', {consensus:'single'})
}
// User Picks Consensus Consensus Mode
buttonConsensusConsensus.onclick = () => {
  socket.emit('switchConsensus', {consensus:'consensus'})
}
// User Ends Turn
endTurn.onclick = () => {
  socket.emit('endTurn', {})
}
// User Clicks Tile
function tileClicked(i,j){
  socket.emit('clickTile', {i:i, j:j})
}
// User Clicks About
buttonAbout.onclick = () => {
  if (aboutWindow.style.display === 'none') {
    aboutWindow.style.display = 'block'
    overlay.style.display = 'block'
    buttonAbout.className = 'open above'
  } else {
    aboutWindow.style.display = 'none'
    overlay.style.display = 'none'
    buttonAbout.className = 'above'
  }
}
// User Clicks card pack
buttonBasecards.onclick = () => {
  socket.emit('changeCards', {pack:'base'})
}
// User Clicks card pack
buttonDuetcards.onclick = () => {
  socket.emit('changeCards', {pack:'duet'})
}
// User Clicks card pack
buttonUndercovercards.onclick = () => {
  socket.emit('changeCards', {pack:'undercover'})
}
/// User Clicks card pack
buttonCustomcards.onclick = () => {
  socket.emit('changeCards', {pack:'custom'})
}
// User Clicks card pack
buttonNsfwcards.onclick = () => {
  socket.emit('changeCards', {pack:'nsfw'})
}
// When the slider is changed
timerSlider.addEventListener("input", () =>{
  socket.emit('timerSlider', {value:timerSlider.value})
})

// User confirms theyre not afk
buttonAfk.onclick = () => {
  socket.emit('active')
  afkWindow.style.display = 'none'
  overlay.style.display = 'none'
}

// User confirms server message
buttonServerMessageOkay.onclick = () => {
  serverMessageWindow.style.display = 'none'
  overlay.style.display = 'none'
}

// Server Responses to this client
////////////////////////////////////////////////////////////////////////////
socket.on('serverStats', (data) => {        // Client gets server stats
  document.getElementById('server-stats').innerHTML = "Players: " + data.players + " | Rooms: " + data.rooms
})

socket.on('joinResponse', (data) =>{        // Response to joining room
  if(data.success){
    joinDiv.style.display = 'none'
    gameDiv.style.display = 'block'
    joinErrorMessage.innerText = ''
    updateFragment();
  } else joinErrorMessage.innerText = data.msg
})

socket.on('createResponse', (data) =>{      // Response to creating room
  if(data.success){
    joinDiv.style.display = 'none'
    gameDiv.style.display = 'block'
    joinErrorMessage.innerText = ''
    updateFragment();
  } else joinErrorMessage.innerText = data.msg
})

socket.on('leaveResponse', (data) =>{       // Response to leaving room
  if(data.success){
    joinDiv.style.display = 'block'
    gameDiv.style.display = 'none'
    wipeBoard();
  }
})

socket.on('timerUpdate', (data) => {        // Server update client timer
  timer.innerHTML = "[" + data.timer + "]"
})

socket.on('newGameResponse', (data) => {    // Response to New Game
  if (data.success){
    wipeBoard();
  }
})

socket.on('afkWarning', () => {    // Response to Afk Warning
  afkWindow.style.display = 'block'
  overlay.style.display = 'block'
})

socket.on('afkKicked', () => {    // Response to Afk Kick
  afkWindow.style.display = 'none'
  serverMessageWindow.style.display = 'block'
  serverMessage.innerHTML = 'You were kicked for being AFK'
  overlay.style.display = 'block'
})

socket.on('serverMessage', (data) => {    // Response to Server message
  serverMessage.innerHTML = data.msg
  serverMessageWindow.style.display = 'block'
  overlay.style.display = 'block'
})

socket.on('switchRoleResponse', (data) =>{  // Response to Switching Role
  if(data.success){
    playerRole = data.role;
    if (playerRole === 'guesser') {
      buttonRoleGuesser.disabled = true;
      buttonRoleSpymaster.disabled = false;
      toggleDifficulty.style.display = "none"
    } else {
      buttonRoleGuesser.disabled = false;
      buttonRoleSpymaster.disabled = true;
      toggleDifficulty.style.display = "block"
    }
    wipeBoard();
  }
})

socket.on('gameState', (data) =>{           // Response to gamestate update
  if (data.difficulty !== difficulty){  // Update the clients difficulty
    difficulty = data.difficulty
    wipeBoard();                        // Update the appearance of the tiles
  }
  mode = data.mode                      // Update the clients game mode
  consensus = data.consensus            // Update the clients consensus mode
  updateInfo(data.game, data.team)      // Update the games turn information
  updateTimerSlider(data.game, data.mode)          // Update the games timer slider
  updatePacks(data.game)                // Update the games pack information
  updatePlayerlist(data.players)        // Update the player list for the room

  let proposals = []
  for (let i in data.players){
    let guessProposal = data.players[i].guessProposal
    if (guessProposal !== null){
      proposals.push(guessProposal)
    }
  }

  // Update the board display
  updateBoard(data.game.board, proposals, data.game.over)
  updateLog(data.game.log)
})


// Utility Functions
////////////////////////////////////////////////////////////////////////////

// Wipe all of the descriptor tile classes from each tile
function wipeBoard(){
  for (let x = 0; x < 5; x++){
    let row = document.getElementById('row-' + (x+1))
    for (let y = 0; y < 5; y++){
      let button = row.children[y]
      button.className = 'tile'
    }
  }
}

// Update the game info displayed to the client
function updateInfo(game, team){
  scoreBlue.innerHTML = game.blue                         // Update the blue tiles left
  scoreRed.innerHTML = game.red                           // Update the red tiles left
  turnMessage.innerHTML = game.turn + "'s turn"           // Update the turn msg
  turnMessage.className = game.turn                       // Change color of turn msg
  if (game.over){                                         // Display winner
    turnMessage.innerHTML = game.winner + " wins!"
    turnMessage.className = game.winner
  }
  if (team !== game.turn) endTurn.disabled = true         // Disable end turn button for opposite team
  else endTurn.disabled = false
  if (playerRole === 'spymaster') {
    endTurn.disabled = true // Disable end turn button for spymasters
  }
  clueEntryDiv.style.display = playerRole === 'spymaster' && game.clue === null && team === game.turn ? '' : 'none'
  if (game.over || game.clue === null){
    clueDisplay.innerText = '___'
  }
  else {
    clueDisplay.innerText = game.clue.word + " (" + (game.clue.count === 'unlimited' ? '∞' : game.clue.count) + ")"
  }
}

// Update the clients timer slider
function updateTimerSlider(game, mode){
  let minutes = (game.timerAmount - 1) / 60
  timerSlider.value = minutes
  timerSliderLabel.innerHTML = "Timer Length : " + timerSlider.value + "min"

  // If the mode is not timed, dont show the slider
  if (mode === 'casual'){
    timerSlider.style.display = 'none'
    timerSliderLabel.style.display = 'none'
  } else {
    timerSlider.style.display = 'block'
    timerSliderLabel.style.display = 'block'
  }
}

// Update the pack toggle buttons
function updatePacks(game){
  if (game.base) buttonBasecards.className = 'enabled'
  else buttonBasecards.className = ''
  if (game.duet) buttonDuetcards.className = 'enabled'
  else buttonDuetcards.className = ''
  if (game.undercover) buttonUndercovercards.className = 'enabled'
  else buttonUndercovercards.className = ''
   if (game.custom) buttonCustomcards.className = 'enabled'
  else buttonCustomcards.className = ''
  if (game.nsfw) buttonNsfwcards.className = 'enabled'
  else buttonNsfwcards.className = ''  
  document.getElementById('word-pool').innerHTML = "Word Pool: " + game.words.length
}

// Update the board
function updateBoard(board, proposals, gameOver){
  // Add description classes to each tile depending on the tiles color
  for (let x = 0; x < 5; x++){
    let row = document.getElementById('row-' + (x+1))
    for (let y = 0; y < 5; y++){
      let button = row.children[y]
      button.innerHTML = board[x][y].word
      button.className = "tile"
      if (board[x][y].type === 'red') button.className += " r"    // Red tile
      if (board[x][y].type === 'blue') button.className += " b"   // Blue tile
      if (board[x][y].type === 'neutral') button.className += " n"// Neutral tile
      if (board[x][y].type === 'death') button.className += " d"  // Death tile
      if (board[x][y].flipped) button.className += " flipped"     // Flipped tile
      if (proposals.includes(board[x][y].word)) button.className += " proposed" // proposed guess
      if (playerRole === 'spymaster' || gameOver) button.className += " s"    // Flag all tiles if the client is a spy master
      if (difficulty === 'hard') button.className += " h"         // Flag all tiles if game is in hard mode
    }
  }
  // Show the proper toggle options for the game difficulty
  if (difficulty === 'normal') {
    buttonDifficultyNormal.disabled = true;
    buttonDifficultyHard.disabled = false;
  } else {
    buttonDifficultyNormal.disabled = false;
    buttonDifficultyHard.disabled = true;
  }
  // Show the proper toggle options for the game mode
  if (mode === 'casual') {
    buttonModeCasual.disabled = true;
    buttonModeTimed.disabled = false;
    timer.innerHTML = ""
  } else {
    buttonModeCasual.disabled = false;
    buttonModeTimed.disabled = true;
  }
  // Show the proper toggle options for the game consensus mode
  if (consensus === 'single') {
    buttonConsensusSingle.disabled = true;
    buttonConsensusConsensus.disabled = false;
  } else {
    buttonConsensusSingle.disabled = false;
    buttonConsensusConsensus.disabled = true;
  }
}

// Update the player list
function updatePlayerlist(players){
  undefinedList.innerHTML = ''
  redTeam.innerHTML = ''
  blueTeam.innerHTML = ''
  for (let i in players){
    // Create a li element for each player
    let li = document.createElement('li');
    li.innerText = players[i].nickname
    // If the player is a spymaster, put brackets around their name
    if (players[i].role === 'spymaster') li.innerText = "[" + players[i].nickname + "]"
    else if (players[i].guessProposal !== null){
      let guessProposal = document.createElement('span')
      guessProposal.classList.add('guess-proposal')
      guessProposal.innerText = players[i].guessProposal
      li.appendChild(guessProposal)
    }
    // Add the player to their teams ul
    if (players[i].team === 'undecided'){
      undefinedList.appendChild(li)
    } else if (players[i].team === 'red'){
      redTeam.appendChild(li)
    } else if (players[i].team === 'blue'){
      blueTeam.appendChild(li)
    }
  }
}

function updateLog(log){
  logDiv.innerHTML = ''
  log.forEach(logEntry => {
    let logSpan = document.createElement('span')
    logSpan.className = logEntry.event + " " + logEntry.team
    if (logEntry.event === 'flipTile'){
      logSpan.innerText = (logEntry.team + " team flipped " + logEntry.word
                           + " (" + logEntry.type + ")"
                           + (logEntry.type === 'death' ? " ending the game"
                              : logEntry.endedTurn ? " ending their turn"
                              : ""))
    }
    else if (logEntry.event === 'switchTurn'){
      logSpan.innerText = "Switched to " + logEntry.team + " team's turn"
    }
    else if (logEntry.event === 'declareClue'){
      logSpan.innerText = (logEntry.team + ' team was given the clue "'
                           + logEntry.clue.word + '" ('
                           + (logEntry.clue.count === 'unlimited'
                              ? '∞'
                              : logEntry.clue.count)
                           + ')')
    }
    else if (logEntry.event === 'endTurn'){
      logSpan.innerText = logEntry.team + " team ended their turn"
    }
    logDiv.prepend(logSpan)
  })
}

function extractFromFragment(fragment, toExtract) {
  let vars = fragment.substring(1).split('&');
  for (let i = 0; i < vars.length; i++) {
    var pair = vars[i].split('=');
    if (pair.length !== 2) {
      continue;
    }
    if (decodeURIComponent(pair[0]) === toExtract) {
        return decodeURIComponent(pair[1]);
    }
  }
  return '';
}

function updateFragment() {
  let room = joinRoom.value;
  let password = joinPassword.value;
  let fragment =  'room=' + encodeURIComponent(room) + '&password=' + encodeURIComponent(password);
  window.location.hash = fragment;
}



