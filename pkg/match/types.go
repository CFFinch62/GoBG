// Package match provides match file import/export for backgammon games.
// Supports SGF (Smart Game Format) and MAT (Jellyfish Match) formats.
package match

import (
	"github.com/yourusername/bgengine/pkg/engine"
)

// Match represents a complete backgammon match.
type Match struct {
	// Match metadata
	Player1     string   // Name of player 1 (X)
	Player2     string   // Name of player 2 (O)
	MatchLength int      // Match length (0 = money game)
	Date        string   // Match date (YYYY-MM-DD format)
	Event       string   // Event name
	Round       string   // Round number
	Place       string   // Location
	Annotator   string   // Who analyzed the match
	Comment     string   // General match comments
	Games       []*Game  // List of games in the match
}

// Game represents a single game within a match.
type Game struct {
	Number       int           // Game number (1-indexed)
	Score1       int           // Player 1 score at start of game
	Score2       int           // Player 2 score at start of game
	Crawford     bool          // True if this is the Crawford game
	InitialBoard engine.Board  // Starting position (usually standard)
	CubeValue    int           // Initial cube value (usually 1)
	CubeOwner    int           // Initial cube owner (-1 = centered)
	Actions      []Action      // Sequence of game actions
	Winner       int           // 0 = player 1, 1 = player 2, -1 = not finished
	Points       int           // Points won (1, 2, or 3)
	Result       GameResult    // How the game ended
}

// ActionType represents the type of game action.
type ActionType int

const (
	ActionRoll       ActionType = iota // Dice roll
	ActionMove                         // Checker move
	ActionDouble                       // Cube double
	ActionTake                         // Take the cube
	ActionPass                         // Pass (decline the cube)
	ActionResign                       // Resignation
	ActionAcceptResign                 // Accept resignation
	ActionRejectResign                 // Reject resignation
)

// Action represents a single game action (roll, move, cube action).
type Action struct {
	Type   ActionType  // Type of action
	Player int         // 0 = player 1, 1 = player 2
	Dice   [2]int      // Dice values (for ActionRoll)
	Move   engine.Move // Move made (for ActionMove)
	Value  int         // Cube value (for ActionDouble) or resign level (for ActionResign)
}

// GameResult indicates how a game ended.
type GameResult int

const (
	ResultSingle       GameResult = iota // Normal win
	ResultGammon                         // Gammon
	ResultBackgammon                     // Backgammon
	ResultResignSingle                   // Resigned for single
	ResultResignGammon                   // Resigned for gammon
	ResultResignBG                       // Resigned for backgammon
	ResultDrop                           // Passed a double
	ResultInProgress                     // Game not finished
)

// NewMatch creates a new empty match.
func NewMatch(player1, player2 string, matchLength int) *Match {
	return &Match{
		Player1:     player1,
		Player2:     player2,
		MatchLength: matchLength,
		Games:       make([]*Game, 0),
	}
}

// NewGame creates a new game with standard starting position.
func NewGame(number, score1, score2 int, crawford bool) *Game {
	startBoard := engine.StartingPosition().Board
	return &Game{
		Number:       number,
		Score1:       score1,
		Score2:       score2,
		Crawford:     crawford,
		InitialBoard: startBoard,
		CubeValue:    1,
		CubeOwner:    -1,
		Actions:      make([]Action, 0),
		Winner:       -1,
		Result:       ResultInProgress,
	}
}

// AddRoll adds a dice roll action to the game.
func (g *Game) AddRoll(player int, die1, die2 int) {
	g.Actions = append(g.Actions, Action{
		Type:   ActionRoll,
		Player: player,
		Dice:   [2]int{die1, die2},
	})
}

// AddMove adds a move action to the game.
func (g *Game) AddMove(player int, move engine.Move) {
	g.Actions = append(g.Actions, Action{
		Type:   ActionMove,
		Player: player,
		Move:   move,
	})
}

// AddDouble adds a cube double action to the game.
func (g *Game) AddDouble(player int, value int) {
	g.Actions = append(g.Actions, Action{
		Type:   ActionDouble,
		Player: player,
		Value:  value,
	})
}

// AddTake adds a take action to the game.
func (g *Game) AddTake(player int) {
	g.Actions = append(g.Actions, Action{
		Type:   ActionTake,
		Player: player,
	})
}

// AddPass adds a pass (drop) action to the game.
func (g *Game) AddPass(player int) {
	g.Actions = append(g.Actions, Action{
		Type:   ActionPass,
		Player: player,
	})
}

