// Package engine provides the public API for the backgammon engine.
package engine

// Board represents checker positions for both players.
// Index 0-24 represents points (0 = bar for opponent's checkers, 1-24 = board points)
// In gnubg's TanBoard: [2][25] where [player][point]
// Point 24 is the bar, points 0-23 are the board points
type Board [2][25]uint8

// GameState represents the full state needed for evaluation
type GameState struct {
	Board       Board  // Checker positions
	Turn        int    // 0 or 1 - who makes the next decision
	Dice        [2]int // Current roll (0,0 if not rolled)
	CubeValue   int    // 1, 2, 4, 8, 16, 32, 64, ...
	CubeOwner   int    // -1=centered, 0=player0, 1=player1
	MatchLength int    // 0 = money game
	Score       [2]int // Match score
	Crawford    bool   // Crawford game flag
}

// Evaluation contains equity estimates from position evaluation
type Evaluation struct {
	Equity  float64 // Expected value
	WinProb float64 // P(win)
	WinG    float64 // P(win gammon)
	WinBG   float64 // P(win backgammon)
	LoseG   float64 // P(lose gammon)
	LoseBG  float64 // P(lose backgammon)
}

// Move represents a sequence of checker moves
type Move struct {
	From [4]int8     // Starting points (-1 for unused)
	To   [4]int8     // Ending points (-1 for unused)
	Hits int8        // Number of hits
	Eval *Evaluation // Evaluation of this move (optional)
}

// CubeAction represents the possible cube actions
type CubeAction int

const (
	NoDouble CubeAction = iota
	Double
	Redouble // Double when we already own the cube
	Take
	Pass
	Beaver // Take and immediately redouble (optional rule)
)

// CubeDecision contains cube action recommendation
type CubeDecision struct {
	Action         CubeAction // Recommended action
	DoubleEquity   float64    // Equity if doubled
	NoDoubleEquity float64    // Equity if not doubled
	TakeEquity     float64    // Equity if opponent takes
}

// PositionKey is a compact binary representation of a board position
// Uses 7 uint32s to encode the position (4 bits per point)
type PositionKey struct {
	Data [7]uint32
}

// OldPositionKey is the legacy position key format (80 bits)
type OldPositionKey struct {
	Data [10]uint8
}

// StartingPosition returns the standard backgammon starting position
// In gnubg's TanBoard representation:
// - Points are numbered 0-23 from each player's perspective
// - Point 24 is the bar
// - Each player has their own view of the board
func StartingPosition() *GameState {
	gs := &GameState{
		Turn:        0,
		Dice:        [2]int{0, 0},
		CubeValue:   1,
		CubeOwner:   -1,
		MatchLength: 0,
		Score:       [2]int{0, 0},
		Crawford:    false,
	}

	// Set up standard starting position (gnubg format)
	// Player 0 (from their perspective)
	gs.Board[0][5] = 5  // 5 on 6-point
	gs.Board[0][7] = 3  // 3 on 8-point
	gs.Board[0][12] = 5 // 5 on 13-point
	gs.Board[0][23] = 2 // 2 on 24-point (opponent's 1-point)

	// Player 1 (from their perspective)
	gs.Board[1][5] = 5  // 5 on 6-point
	gs.Board[1][7] = 3  // 3 on 8-point
	gs.Board[1][12] = 5 // 5 on 13-point
	gs.Board[1][23] = 2 // 2 on 24-point (opponent's 1-point)

	return gs
}

// EqualBoards returns true if two boards are identical
func EqualBoards(b1, b2 Board) bool {
	for i := 0; i < 2; i++ {
		for j := 0; j < 25; j++ {
			if b1[i][j] != b2[i][j] {
				return false
			}
		}
	}
	return true
}

// EqualKeys returns true if two position keys are identical
func EqualKeys(k1, k2 PositionKey) bool {
	for i := 0; i < 7; i++ {
		if k1.Data[i] != k2.Data[i] {
			return false
		}
	}
	return true
}

// CopyKey copies a position key
func CopyKey(src PositionKey) PositionKey {
	var dst PositionKey
	copy(dst.Data[:], src.Data[:])
	return dst
}
