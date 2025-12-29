package engine

import (
	"testing"
)

// startingBoard returns the standard backgammon starting position
func startingBoard() Board {
	var board Board
	// Player 0 (from their perspective)
	board[0][5] = 5  // 5 on 6-point
	board[0][7] = 3  // 3 on 8-point
	board[0][12] = 5 // 5 on 13-point
	board[0][23] = 2 // 2 on 24-point (opponent's 1-point)

	// Player 1 (from their perspective)
	board[1][5] = 5  // 5 on 6-point
	board[1][7] = 3  // 3 on 8-point
	board[1][12] = 5 // 5 on 13-point
	board[1][23] = 2 // 2 on 24-point (opponent's 1-point)

	return board
}

func TestGenerateMovesStartingPosition31(t *testing.T) {
	board := startingBoard()
	ml := GenerateMoves(board, 3, 1)

	// From starting position with 3-1, there should be multiple legal moves
	if len(ml.Moves) == 0 {
		t.Error("Expected at least one legal move for 3-1 from starting position")
	}

	t.Logf("Generated %d moves for 3-1 from starting position", len(ml.Moves))

	// Verify all moves use both dice (2 sub-moves)
	for i, m := range ml.Moves {
		if m.From[0] < 0 || m.From[1] < 0 {
			t.Errorf("Move %d doesn't use both dice: From=%v To=%v", i, m.From, m.To)
		}
		if m.From[2] >= 0 {
			t.Errorf("Move %d uses more than 2 dice for non-doubles: From=%v", i, m.From)
		}
	}
}

func TestGenerateMovesStartingPosition66(t *testing.T) {
	board := startingBoard()
	ml := GenerateMoves(board, 6, 6)

	// From starting position with 6-6, there should be legal moves
	if len(ml.Moves) == 0 {
		t.Error("Expected at least one legal move for 6-6 from starting position")
	}

	t.Logf("Generated %d moves for 6-6 from starting position", len(ml.Moves))

	// Verify all moves use 4 dice (doubles)
	for i, m := range ml.Moves {
		if m.From[0] < 0 || m.From[1] < 0 || m.From[2] < 0 || m.From[3] < 0 {
			t.Errorf("Move %d doesn't use all 4 dice for doubles: From=%v To=%v", i, m.From, m.To)
		}
	}
}

func TestGenerateMovesBarEntry(t *testing.T) {
	var board Board
	// Player 1 has a checker on the bar
	board[1][24] = 1
	// Player 1 has other checkers
	board[1][5] = 5
	board[1][7] = 3
	board[1][12] = 5
	board[1][23] = 1

	// Player 0 has some checkers
	board[0][5] = 5
	board[0][7] = 3
	board[0][12] = 5
	board[0][23] = 2

	ml := GenerateMoves(board, 3, 1)

	// All moves must start from the bar (point 24)
	for i, m := range ml.Moves {
		if m.From[0] != 24 {
			t.Errorf("Move %d doesn't start from bar: From=%v", i, m.From)
		}
	}

	t.Logf("Generated %d moves with checker on bar", len(ml.Moves))
}

func TestGenerateMovesBlocked(t *testing.T) {
	var board Board
	// Player 1 has a checker on the bar
	board[1][24] = 1
	board[1][5] = 5

	// Player 0 blocks all entry points (1-6)
	for i := 0; i < 6; i++ {
		board[0][23-i] = 2 // Block points 1-6 from player 1's perspective
	}

	ml := GenerateMoves(board, 3, 1)

	// No legal moves - all entry points blocked
	if len(ml.Moves) != 0 {
		t.Errorf("Expected 0 moves when bar entry is blocked, got %d", len(ml.Moves))
	}
}

func TestGenerateMovesBearoff(t *testing.T) {
	var board Board
	// Player 1 has all checkers in home board (points 0-5)
	board[1][0] = 3
	board[1][1] = 3
	board[1][2] = 3
	board[1][3] = 3
	board[1][4] = 2
	board[1][5] = 1

	// Player 0 has borne off all checkers (empty board for opponent)

	ml := GenerateMoves(board, 6, 5)

	if len(ml.Moves) == 0 {
		t.Error("Expected legal bearoff moves")
	}

	t.Logf("Generated %d bearoff moves for 6-5", len(ml.Moves))
}

func TestApplyMove(t *testing.T) {
	board := startingBoard()
	ml := GenerateMoves(board, 3, 1)

	if len(ml.Moves) == 0 {
		t.Fatal("No moves generated")
	}

	// Apply the first move
	result := ApplyMove(board, ml.Moves[0])

	// Verify the board changed
	if EqualBoards(board, result) {
		t.Error("Board should have changed after applying move")
	}
}

func TestNoDuplicateMoves(t *testing.T) {
	board := startingBoard()
	ml := GenerateMoves(board, 3, 1)

	// Check for duplicate resulting positions
	seen := make(map[string]bool)
	for i, m := range ml.Moves {
		result := ApplyMove(board, m)
		key := boardToString(result)
		if seen[key] {
			t.Errorf("Duplicate move at index %d", i)
		}
		seen[key] = true
	}
}

func boardToString(b Board) string {
	// Simple string representation for comparison
	result := ""
	for i := 0; i < 2; i++ {
		for j := 0; j < 25; j++ {
			result += string(rune('0' + b[i][j]))
		}
	}
	return result
}

