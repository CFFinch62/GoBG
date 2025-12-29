package engine

import (
	"github.com/yourusername/bgengine/internal/positionid"
)

// EvalOptions controls evaluation behavior
type EvalOptions struct {
	Plies    int  // Number of plies to search (0 = neural net only)
	Cubeful  bool // Include cube equity (not yet implemented)
	UsePrune bool // Use pruning neural nets to filter moves
}

// DefaultEvalOptions returns sensible defaults for evaluation
func DefaultEvalOptions() EvalOptions {
	return EvalOptions{
		Plies:    0,
		Cubeful:  false,
		UsePrune: true,
	}
}

// EvaluatePliedWithOptions evaluates with explicit options
func (e *Engine) EvaluatePliedWithOptions(state *GameState, opts EvalOptions) (*Evaluation, error) {
	if opts.Plies <= 0 {
		return e.Evaluate(state)
	}
	return e.evaluateNPlyWithPrune(state, opts.Plies, opts.UsePrune)
}

// EvaluatePlied evaluates a position with n-ply lookahead
// plies=0 is equivalent to Evaluate (neural net only)
// plies=1 looks ahead one move (averages over opponent's dice)
// plies=2 looks ahead two moves, etc.
// Uses pruning by default for faster evaluation
func (e *Engine) EvaluatePlied(state *GameState, plies int) (*Evaluation, error) {
	if plies <= 0 {
		return e.Evaluate(state)
	}

	// For n-ply, we average over all possible dice rolls
	// and for each roll, find the best move, then evaluate recursively
	// Use pruning by default for performance
	return e.evaluateNPlyWithPrune(state, plies, true)
}

// evaluateNPlyWithPrune performs n-ply lookahead with optional move pruning
func (e *Engine) evaluateNPlyWithPrune(state *GameState, plies int, usePrune bool) (*Evaluation, error) {
	// Accumulate weighted probabilities
	var sumProbs [5]float64
	totalWeight := 0.0

	// Loop over all 21 possible dice combinations
	for d1 := 1; d1 <= 6; d1++ {
		for d2 := 1; d2 <= d1; d2++ {
			// Weight: doubles occur 1/36, non-doubles occur 2/36
			weight := 2.0
			if d1 == d2 {
				weight = 1.0
			}

			// Generate moves for this roll
			ml := GenerateMoves(state.Board, d1, d2)

			var eval *Evaluation
			var err error

			if len(ml.Moves) == 0 {
				// No legal moves - evaluate current position
				eval, err = e.evaluateAtPlyWithPrune(state, plies-1, usePrune)
			} else {
				// Apply pruning if enabled and we have enough moves
				moves := ml.Moves
				if usePrune && len(moves) > MinPruneMoves {
					moves = e.pruneMoves(state, moves)
				}
				// Find the best move and evaluate resulting position
				eval, err = e.findBestMoveEvalWithPrune(state, moves, plies-1, usePrune)
			}

			if err != nil {
				return nil, err
			}

			// Accumulate weighted probabilities
			sumProbs[0] += weight * eval.WinProb
			sumProbs[1] += weight * eval.WinG
			sumProbs[2] += weight * eval.WinBG
			sumProbs[3] += weight * eval.LoseG
			sumProbs[4] += weight * eval.LoseBG
			totalWeight += weight
		}
	}

	// Normalize (totalWeight = 36)
	result := &Evaluation{
		WinProb: sumProbs[0] / totalWeight,
		WinG:    sumProbs[1] / totalWeight,
		WinBG:   sumProbs[2] / totalWeight,
		LoseG:   sumProbs[3] / totalWeight,
		LoseBG:  sumProbs[4] / totalWeight,
	}

	// Calculate equity
	result.Equity = result.WinProb - (1 - result.WinProb) +
		result.WinG - result.LoseG +
		result.WinBG - result.LoseBG

	return result, nil
}

// evaluateAtPlyWithPrune evaluates position at specified ply depth with optional pruning
func (e *Engine) evaluateAtPlyWithPrune(state *GameState, plies int, usePrune bool) (*Evaluation, error) {
	if plies <= 0 {
		// Use cached evaluation for leaf nodes (most cache hits happen here)
		return e.EvaluateCached(state, 0)
	}
	return e.evaluateNPlyWithPrune(state, plies, usePrune)
}

// findBestMoveEvalWithPrune finds the best move and returns its evaluation
func (e *Engine) findBestMoveEvalWithPrune(state *GameState, moves []Move, plies int, usePrune bool) (*Evaluation, error) {
	var bestEval *Evaluation
	bestEquity := float64(-1000)

	for _, m := range moves {
		// Apply the move
		resultBoard := ApplyMove(state.Board, m)

		// Swap sides for opponent's perspective
		swappedBoard := swapBoardForMultiply(resultBoard)

		// Create state for evaluation from opponent's perspective
		evalState := &GameState{
			Board:       swappedBoard,
			Turn:        1 - state.Turn,
			CubeValue:   state.CubeValue,
			CubeOwner:   state.CubeOwner,
			MatchLength: state.MatchLength,
			Score:       state.Score,
			Crawford:    state.Crawford,
		}

		// Evaluate at specified ply
		eval, err := e.evaluateAtPlyWithPrune(evalState, plies, usePrune)
		if err != nil {
			continue
		}

		// Invert evaluation to get it from our perspective
		inverted := invertEvaluation(eval)

		if inverted.Equity > bestEquity {
			bestEquity = inverted.Equity
			bestEval = inverted
		}
	}

	if bestEval == nil {
		// Fallback to 0-ply evaluation
		return e.Evaluate(state)
	}

	return bestEval, nil
}

// swapBoardForMultiply swaps the board sides for multi-ply evaluation
func swapBoardForMultiply(board Board) Board {
	return Board(positionid.SwapSides(positionid.Board(board)))
}
