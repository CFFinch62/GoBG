package engine

import (
	"math/bits"
	"sort"

	"github.com/yourusername/bgengine/internal/neuralnet"
	"github.com/yourusername/bgengine/internal/positionid"
)

// Move filtering/pruning constants matching gnubg
const (
	MinPruneMoves = 5  // Minimum moves to keep after pruning
	MaxPruneMoves = 16 // Maximum moves to keep (MinPruneMoves + 11)
)

// MoveFilter controls which moves to consider at each ply level
type MoveFilter struct {
	Accept    int     // Always accept this many moves (0 means don't use this filter)
	Extra     int     // Accept up to this many additional moves...
	Threshold float32 // ...if they are within this equity of best move
}

// NullFilter means no filtering
var NullFilter = MoveFilter{Accept: -1}

// DefaultFilters provides sensible default move filters (matches gnubg "Normal")
var DefaultFilters = [4][4]MoveFilter{
	{{0, 8, 0.16}, {0, 0, 0}, {0, 0, 0}, {0, 0, 0}},
	{{0, 8, 0.16}, {-1, 0, 0}, {0, 0, 0}, {0, 0, 0}},
	{{0, 8, 0.16}, {-1, 0, 0}, {0, 2, 0.04}, {0, 0, 0}},
	{{0, 8, 0.16}, {-1, 0, 0}, {0, 2, 0.04}, {-1, 0, 0}},
}

// scoredMove pairs a move with its quick evaluation score
type scoredMove struct {
	move  Move
	score float32
	index int
}

// pruneMoves uses the pruning neural nets to quickly score moves and return only the best candidates
// Returns the top moves that should be fully evaluated
func (e *Engine) pruneMoves(state *GameState, moves []Move) []Move {
	if len(moves) <= MinPruneMoves {
		return moves // No pruning needed
	}

	// Calculate how many moves to keep: min(len, MinPruneMoves + floor(log2(len)))
	numToKeep := MinPruneMoves + bits.Len(uint(len(moves))) - 1
	if numToKeep > MaxPruneMoves {
		numToKeep = MaxPruneMoves
	}
	if numToKeep >= len(moves) {
		return moves // Keep all
	}

	// Check if we have pruning nets
	if e.pContact == nil && e.pRace == nil && e.pCrashed == nil {
		// No pruning nets, fall back to filtering by move count only
		return moves[:numToKeep]
	}

	// Score all moves with pruning nets
	scored := make([]scoredMove, len(moves))
	for i, m := range moves {
		scored[i] = scoredMove{
			move:  m,
			score: e.scoreMoveForPruning(state, m),
			index: i,
		}
	}

	// Sort by score (higher is better)
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Return top moves
	result := make([]Move, numToKeep)
	for i := 0; i < numToKeep; i++ {
		result[i] = scored[i].move
	}
	return result
}

// scoreMoveForPruning quickly scores a move using the pruning neural net
func (e *Engine) scoreMoveForPruning(state *GameState, m Move) float32 {
	// Apply the move
	resultBoard := ApplyMove(state.Board, m)

	// Swap sides for opponent's perspective
	swappedBoard := positionid.SwapSides(positionid.Board(resultBoard))

	// Classify the position
	board := neuralnet.Board(swappedBoard)
	class := neuralnet.ClassifyPosition(board)

	// Select pruning net based on position class
	var pNet *neuralnet.NeuralNet
	switch class {
	case neuralnet.ClassRace:
		pNet = e.pRace
	case neuralnet.ClassCrashed:
		pNet = e.pCrashed
	default:
		pNet = e.pContact
	}

	if pNet == nil {
		return 0 // No pruning net available
	}

	// Calculate base inputs only (200 inputs)
	inputs := make([]float32, neuralnet.NumPruningInputs)
	neuralnet.BaseInputsInto(board, inputs)

	// Evaluate with pruning net
	outputs := pNet.Evaluate(inputs)

	// Return utility from opponent's perspective (we'll negate since we want best for us)
	// Win prob is outputs[0], higher is better for opponent, worse for us
	// So we return negative of opponent's equity
	winProb := float64(outputs[0])
	winG := float64(outputs[1])
	winBG := float64(outputs[2])
	loseG := float64(outputs[3])
	loseBG := float64(outputs[4])

	// Equity from opponent's perspective
	oppEquity := winProb - (1 - winProb) + winG - loseG + winBG - loseBG

	// Return negative (our perspective)
	return float32(-oppEquity)
}
