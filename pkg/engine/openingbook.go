package engine

// OpeningBook provides pre-computed best moves for opening rolls.
// These are the universally accepted best plays from the starting position.
// Based on extensive rollout analysis matching gnubg's recommendations.

// OpeningEntry represents a pre-computed opening move
type OpeningEntry struct {
	Move Move   // The best move
	Alt  *Move  // Alternative move (if close, nil otherwise)
	Note string // Brief explanation
}

// openingBook maps dice roll keys to best moves
// Key format: die1*10 + die2 where die1 > die2 (non-doubles only for opening)
// Board indices: 0-23 where 23 is the 24-point (back checkers), 0 is the 1-point
// From/To use point numbers: 24=back, 13=midpoint, 8=8pt, 6=6pt, etc.
// Index = point - 1 (so point 24 = index 23, point 6 = index 5)
var openingBook = map[int]OpeningEntry{
	// 6-x rolls: Run with back checker or make a point
	65: {
		// 24/13: Move from point 24 (idx 23) by 6+5=11 to point 13 (idx 12)
		Move: Move{From: [4]int8{23, -1, -1, -1}, To: [4]int8{12, -1, -1, -1}},
		Note: "24/13 - Run to safety",
	},
	64: {
		// 24/14: Move from point 24 (idx 23) by 6+4=10 to point 14 (idx 13)
		Move: Move{From: [4]int8{23, -1, -1, -1}, To: [4]int8{13, -1, -1, -1}},
		Note: "24/14 - Run to outfield",
	},
	63: {
		// 24/15: Move from point 24 (idx 23) by 6+3=9 to point 15 (idx 14)
		Move: Move{From: [4]int8{23, -1, -1, -1}, To: [4]int8{14, -1, -1, -1}},
		Note: "24/15 - Run to outfield",
	},
	62: {
		// 24/18 13/11: Split the back checkers
		Move: Move{From: [4]int8{23, 12, -1, -1}, To: [4]int8{17, 10, -1, -1}},
		Note: "24/18 13/11 - Diversify builders",
	},
	61: {
		// 13/7 8/7: Make the bar point
		Move: Move{From: [4]int8{12, 7, -1, -1}, To: [4]int8{6, 6, -1, -1}},
		Note: "13/7 8/7 - Make bar point",
	},
	// 5-x rolls
	54: {
		// 13/8 13/9: Bring two builders down
		Move: Move{From: [4]int8{12, 12, -1, -1}, To: [4]int8{7, 8, -1, -1}},
		Note: "13/8 13/9 - Bring builders down",
	},
	53: {
		// 8/3 6/3: Make the 3-point
		Move: Move{From: [4]int8{7, 5, -1, -1}, To: [4]int8{2, 2, -1, -1}},
		Note: "8/3 6/3 - Make 3-point",
	},
	52: {
		// 13/11 13/8: Bring builders down
		Move: Move{From: [4]int8{12, 12, -1, -1}, To: [4]int8{10, 7, -1, -1}},
		Note: "13/11 13/8 - Bring builders down",
	},
	51: {
		// 13/8 24/23: Slot and split
		Move: Move{From: [4]int8{12, 23, -1, -1}, To: [4]int8{7, 22, -1, -1}},
		Note: "13/8 24/23 - Slot and split",
	},
	// 4-x rolls
	43: {
		// 13/10 13/9: Bring builders down
		Move: Move{From: [4]int8{12, 12, -1, -1}, To: [4]int8{8, 9, -1, -1}},
		Note: "13/10 13/9 - Bring builders down",
	},
	42: {
		// 8/4 6/4: Make the 4-point
		Move: Move{From: [4]int8{7, 5, -1, -1}, To: [4]int8{3, 3, -1, -1}},
		Note: "8/4 6/4 - Make 4-point",
	},
	41: {
		// 13/9 24/23: Bring builder and split
		Move: Move{From: [4]int8{12, 23, -1, -1}, To: [4]int8{8, 22, -1, -1}},
		Note: "13/9 24/23 - Bring builder and split",
	},
	// 3-x rolls
	32: {
		// 13/11 13/10: Bring builders down
		Move: Move{From: [4]int8{12, 12, -1, -1}, To: [4]int8{10, 9, -1, -1}},
		Note: "13/11 13/10 - Bring builders down",
	},
	31: {
		// 8/5 6/5: Make the 5-point (golden point)
		Move: Move{From: [4]int8{7, 5, -1, -1}, To: [4]int8{4, 4, -1, -1}},
		Note: "8/5 6/5 - Make 5-point (golden point)",
	},
	// 2-1 roll
	21: {
		// 13/11 24/23: Slot and split
		Move: Move{From: [4]int8{12, 23, -1, -1}, To: [4]int8{10, 22, -1, -1}},
		Note: "13/11 24/23 - Slot and split",
	},
}

// LookupOpening checks if the position is the starting position and returns
// the best opening move if available
func (e *Engine) LookupOpening(state *GameState, dice [2]int) (*OpeningEntry, bool) {
	// Only applies to starting position
	if !isStartingPosition(state) {
		return nil, false
	}

	// Doubles are not opening rolls (player who rolls higher goes first)
	if dice[0] == dice[1] {
		return nil, false
	}

	// Normalize dice order (higher die first)
	d1, d2 := dice[0], dice[1]
	if d2 > d1 {
		d1, d2 = d2, d1
	}

	key := d1*10 + d2
	entry, ok := openingBook[key]
	if !ok {
		return nil, false
	}

	return &entry, true
}

// isStartingPosition checks if the game state is the standard starting position
func isStartingPosition(state *GameState) bool {
	start := StartingPosition()
	return EqualBoards(state.Board, start.Board)
}

// OpeningMoveWithEval returns the opening move with evaluation attached
func (e *Engine) OpeningMoveWithEval(state *GameState, dice [2]int) (Move, *Evaluation, error) {
	entry, found := e.LookupOpening(state, dice)
	if !found {
		// Fall back to regular analysis
		return e.BestMove(state, dice)
	}

	// Apply the opening move and evaluate the resulting position
	resultBoard := ApplyMove(state.Board, entry.Move)
	resultState := &GameState{
		Board:     resultBoard,
		Turn:      1 - state.Turn,
		CubeValue: state.CubeValue,
		CubeOwner: state.CubeOwner,
	}

	eval, err := e.Evaluate(resultState)
	if err != nil {
		return entry.Move, nil, err
	}

	// Invert evaluation since we evaluated opponent's perspective
	invertedEval := &Evaluation{
		WinProb: 1 - eval.WinProb,
		WinG:    eval.LoseG,
		WinBG:   eval.LoseBG,
		LoseG:   eval.WinG,
		LoseBG:  eval.WinBG,
		Equity:  -eval.Equity,
	}

	move := entry.Move
	move.Eval = invertedEval

	return move, invertedEval, nil
}
