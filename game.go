package main

import (
	"fmt"
	"log"
	"time"
)

const (
	delayBetweenSimStep = (250 * time.Millisecond)
)

type GamePlayerCtrl chan *PlayerAction
type GameState int
type GamePlayerState int

var (
	// Game state
	GameStateRunning = GameState(0)
	GameStatePaused  = GameState(1)
	GameStateStopped = GameState(2)
	// Game Player State 
	GamePlayerStateAdded   = GamePlayerState(0)
	GamePlayerStatePresent = GamePlayerState(1)
	GamePlayerStateUpdated = GamePlayerState(2)
	GamePlayerStateRemoved = GamePlayerState(3)
)

// Definition of the game object
type Game struct {
	id         uint64
	sim        *Simulation
	board      *Board
	state      GameState
	players    map[*Player]*GamePlayerInfo
	playerCtrl GamePlayerCtrl
	AddPlayer  chan *Player
	RmPlayer   chan *Player
}

type GamePlayerInfo struct {
	PlayerId uint64
	State    GamePlayerState
	Name     string
	Score    int
}

// Initalization of the game object.game  It s being done in the package's
// global scope so the network event handler will have access to it when
// receiving new player connections.
func NewGame(id uint64) *Game {
	g := &Game{
		id:         id,
		state:      GameStateStopped,
		players:    make(map[*Player]*GamePlayerInfo),
		playerCtrl: make(GamePlayerCtrl),
		AddPlayer:  make(chan *Player),
		RmPlayer:   make(chan *Player),
	}
	return g
}

// Returns the game's id
func (g Game) GetId() uint64 {
	return g.id
}

// Returns if the game has reached its limit of players
func (g *Game) IsFull() bool {
	return false // TODO use a load balancer for this in the world
}

// Event receiver to processing messages between the simulation and
// the players.  If players are connected to the game the simulation
// will be started, but as soon as the last player drops out the
// simulation will be terminated.
func (g *Game) Run() {
	ticker := time.NewTicker(delayBetweenSimStep)
	defer func() {
		log.Println("Game ", g.id, " event loop terminating")
		ticker.Stop()
	}()
	for {
		select {
		case <-ticker.C:
			if g.state != GameStateRunning {
				continue
			}
			g.simulate()

		case p := <-g.AddPlayer:
			log.Printf("Adding player %d to game %d", p.GetId(), g.id)
			g.addPlayer(p)

		case p := <-g.RmPlayer:
			log.Printf("Removing player %d from game %d", p.GetId(), g.id)
			g.removePlayer(p)

		case ctrl := <-g.playerCtrl:
			var pInfo *GamePlayerInfo
			if pInfo = g.players[ctrl.Player]; pInfo == nil {
				// Ignore players we don't know about, TODO should we disconnect them?
				continue
			}
			g.procPlayerCtrl(ctrl, pInfo)
		}
	}
}
func (g *Game) simulate() {
	toA := g.sim.Step()
	if toA != nil {
		msg := MsgCreateGameUpdate()
		msg.AddEntityUpdates(toA)
		g.broadcastUpdate(msg)
	}
}

// Adds a new player to the game, and starting the game if needed.
func (g *Game) addPlayer(p *Player) {
	pInfo := &GamePlayerInfo{
		State:    GamePlayerStateAdded,
		PlayerId: p.GetId(),
		Score:    0,
		Name:     fmt.Sprintf("Player %d", p.GetId()),
	}
	if g.state != GameStateRunning {
		g.startGame()
	}

	// Update the current player with the current state of the game
	toP := g.board.GetEntityArray()
	if toP != nil {
		msg := MsgCreateGameUpdate()
		infos := make([]*GamePlayerInfo, len(g.players))
		i := 0
		for _, info := range g.players {
			infos[i] = info
			i++
		}
		msg.AddPlayerGameInfos(infos)
		msg.AddEntityUpdates(toP)
		g.playerUpdate(p, msg)
	}
	g.players[p] = pInfo
	p.SetGameCtrl(&g.playerCtrl)

	// Let all players now about the new player
	msg := MsgCreateGameUpdate()
	msg.AddPlayerGameInfo(pInfo, -1)
	g.broadcastUpdate(msg)
}

// Removes the passed in player from the game, and stops the game
// if that is the last player to be removed.
func (g *Game) removePlayer(p *Player) {
	if pInfo := g.players[p]; pInfo != nil {
		delete(g.players, p)
		p.SetGameCtrl(nil)

		pInfo.State = GamePlayerStateRemoved
		msg := MsgCreateGameUpdate()
		msg.AddPlayerGameInfo(pInfo, -1)
		g.broadcastUpdate(msg)
	}
	if len(g.players) == 0 {
		g.stopGame()
	}
}

// Processes the player's control in relation to the game.
func (g *Game) procPlayerCtrl(ctrl *PlayerAction, pInfo *GamePlayerInfo) {
	if ctrl.Game.Command == PlayerCmdGameSelectEntity {
		// TODO do matching based on what the player selected previously
		pInfo.State = GamePlayerStateUpdated
		e := g.board.GetEntityById(ctrl.Game.EntityId)
		e.state = EntityStateSelected

		msg := MsgCreateGameUpdate()
		msg.AddPlayerGameInfo(pInfo, -1)
		msg.AddEntityUpdate(e, -1)
		g.broadcastUpdate(msg)

		pInfo.State = GamePlayerStatePresent
		e.state = EntityStatePresent
	}
}

// Processes an update from a player
func (g *Game) playerUpdate(p *Player, update interface{}) {
	err := p.SendToPlayer(update)
	if err != nil {
		g.RmPlayer <- p
	}
}

// Sends out an update to all players
func (g *Game) broadcastUpdate(update interface{}) {
	for p, _ := range g.players {
		g.playerUpdate(p, update)
	}
}

// Create the simulator, and start it running
func (g *Game) startGame() {
	g.board = NewBoard()
	g.sim = NewSimulation(g.board)
	g.state = GameStateRunning
}

// Terminate the simulator, and remove its instance
func (g *Game) stopGame() {
	g.state = GameStateStopped
	g.sim = nil
	g.board = nil
}

// Returns the current state of the game
func (g Game) getState() GameState {
	return g.state
}
