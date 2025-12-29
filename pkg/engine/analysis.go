package engine

import (
	"sort"

	"github.com/yourusername/bgengine/internal/positionid"
)

// MoveWithEval is a move together with its evaluation
type MoveWithEval struct {
	Move   Move
	Eval   *Evaluation
	Equity float64 // Cached for sorting
}

// AnalysisResult contains the result of move analysis
type AnalysisResult struct {
	Moves      []MoveWithEval // All moves ranked by equity
	BestMove   Move           // Best move
	BestEquity float64        // Best equity
	NumMoves   int            // Total number of legal moves
}

// AnalyzePosition generates all legal moves, evaluates them, and returns ranked results
// dice should be [2]int with values 1-6
func (e *Engine) AnalyzePosition(state *GameState, dice [2]int) (*AnalysisResult, error) {
	ml := GenerateMoves(state.Board, dice[0], dice[1])

	if len(ml.Moves) == 0 {
		return &AnalysisResult{
			Moves:    nil,
			NumMoves: 0,
		}, nil
	}

	result := &AnalysisResult{
		Moves:    make([]MoveWithEval, len(ml.Moves)),
		NumMoves: len(ml.Moves),
	}

	// Evaluate each move
	for i, m := range ml.Moves {
		// Apply the move to get the resulting board
		resultBoard := ApplyMove(state.Board, m)

		// Swap sides for evaluation (opponent's perspective after our move)
		swappedBoard := swapBoard(resultBoard)

		// Create state for evaluation
		evalState := &GameState{
			Board:       swappedBoard,
			Turn:        1 - state.Turn,
			CubeValue:   state.CubeValue,
			CubeOwner:   state.CubeOwner,
			MatchLength: state.MatchLength,
			Score:       state.Score,
			Crawford:    state.Crawford,
		}

		// Evaluate the position from opponent's perspective
		eval, err := e.Evaluate(evalState)
		if err != nil {
			// On error, use default values
			eval = &Evaluation{
				WinProb: 0.5,
				Equity:  0.0,
			}
		}

		// Invert the evaluation to get it from our perspective
		inverted := invertEvaluation(eval)

		result.Moves[i] = MoveWithEval{
			Move:   m,
			Eval:   inverted,
			Equity: inverted.Equity,
		}
	}

	// Sort by equity (best first)
	sort.Slice(result.Moves, func(i, j int) bool {
		return result.Moves[i].Equity > result.Moves[j].Equity
	})

	// Set best move
	if len(result.Moves) > 0 {
		result.BestMove = result.Moves[0].Move
		result.BestEquity = result.Moves[0].Equity
	}

	return result, nil
}

// BestMove finds the best move for a position with the given dice roll
// Returns the best move and its evaluation
func (e *Engine) BestMove(state *GameState, dice [2]int) (Move, *Evaluation, error) {
	analysis, err := e.AnalyzePosition(state, dice)
	if err != nil {
		return Move{}, nil, err
	}

	if analysis.NumMoves == 0 {
		return Move{}, nil, nil // No legal moves
	}

	return analysis.BestMove, analysis.Moves[0].Eval, nil
}

// RankMoves evaluates and ranks the top N moves
// If n <= 0, returns all moves ranked
func (e *Engine) RankMoves(state *GameState, dice [2]int, n int) ([]MoveWithEval, error) {
	analysis, err := e.AnalyzePosition(state, dice)
	if err != nil {
		return nil, err
	}

	if n <= 0 || n > len(analysis.Moves) {
		return analysis.Moves, nil
	}

	return analysis.Moves[:n], nil
}

// swapBoard swaps the board representation between players
func swapBoard(board Board) Board {
	return Board(positionid.SwapSides(positionid.Board(board)))
}

// invertEvaluation inverts an evaluation from opponent's perspective to ours
func invertEvaluation(eval *Evaluation) *Evaluation {
	return &Evaluation{
		WinProb: 1.0 - eval.WinProb,
		WinG:    eval.LoseG,
		WinBG:   eval.LoseBG,
		LoseG:   eval.WinG,
		LoseBG:  eval.WinBG,
		Equity:  -eval.Equity,
	}
}
