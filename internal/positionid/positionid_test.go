package positionid

import (
	"testing"
)

// Standard starting position for backgammon
// In gnubg's TanBoard representation:
// - Points are numbered 0-23 from each player's perspective
// - Point 24 is the bar
// - Each player has their own view of the board
//
// Standard starting position (from gnubg):
// Player 0: 2 on point 23, 5 on point 12, 3 on point 7, 5 on point 5
// Player 1: 2 on point 23, 5 on point 12, 3 on point 7, 5 on point 5
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

// Known position ID for starting position from gnubg
const startingPositionID = "4HPwATDgc/ABMA"

func TestPositionIDStartingPosition(t *testing.T) {
	board := startingBoard()
	posID := PositionID(board)

	if posID != startingPositionID {
		t.Errorf("PositionID mismatch: got %s, want %s", posID, startingPositionID)
	}
}

func TestPositionKeyRoundTrip(t *testing.T) {
	board := startingBoard()

	// Create position key
	key := MakePositionKey(board)

	// Convert back to board
	board2 := BoardFromKey(key)

	// Compare
	if !EqualBoards(board, board2) {
		t.Errorf("PositionKey round-trip failed")
		t.Errorf("Original: %v", board)
		t.Errorf("Result:   %v", board2)
	}
}

func TestOldPositionKeyRoundTrip(t *testing.T) {
	board := startingBoard()

	// Create old position key
	key := MakeOldPositionKey(board)

	// Convert back to board
	board2 := BoardFromOldKey(key)

	// Compare
	if !EqualBoards(board, board2) {
		t.Errorf("OldPositionKey round-trip failed")
		t.Errorf("Original: %v", board)
		t.Errorf("Result:   %v", board2)
	}
}

func TestPositionIDRoundTrip(t *testing.T) {
	board := startingBoard()

	// Generate position ID
	posID := PositionID(board)

	// Decode back to board
	board2, err := BoardFromPositionID(posID)
	if err != nil {
		t.Fatalf("BoardFromPositionID failed: %v", err)
	}

	// Compare
	if !EqualBoards(board, board2) {
		t.Errorf("PositionID round-trip failed")
		t.Errorf("Original: %v", board)
		t.Errorf("Result:   %v", board2)
	}
}

func TestBoardFromPositionID(t *testing.T) {
	// Test decoding the known starting position ID
	board, err := BoardFromPositionID(startingPositionID)
	if err != nil {
		t.Fatalf("BoardFromPositionID failed: %v", err)
	}

	expected := startingBoard()
	if !EqualBoards(board, expected) {
		t.Errorf("BoardFromPositionID mismatch")
		t.Errorf("Got:      %v", board)
		t.Errorf("Expected: %v", expected)
	}
}

func TestCheckPosition(t *testing.T) {
	// Valid starting position
	board := startingBoard()
	if !CheckPosition(board) {
		t.Error("CheckPosition should return true for starting position")
	}

	// Invalid: too many checkers
	var invalid Board
	for i := 0; i < 25; i++ {
		invalid[0][i] = 1
	}
	if CheckPosition(invalid) {
		t.Error("CheckPosition should return false for >15 checkers")
	}

	// Invalid: both players on same point
	var overlap Board
	overlap[0][5] = 2
	overlap[1][18] = 2 // 23 - 5 = 18
	if CheckPosition(overlap) {
		t.Error("CheckPosition should return false for overlapping checkers")
	}
}

func TestCombination(t *testing.T) {
	// Test known combinations
	tests := []struct {
		n, r     uint32
		expected uint32
	}{
		{5, 2, 10},   // C(5,2) = 10
		{6, 3, 20},   // C(6,3) = 20
		{10, 5, 252}, // C(10,5) = 252
	}

	for _, tc := range tests {
		result := Combination(tc.n, tc.r)
		if result != tc.expected {
			t.Errorf("Combination(%d, %d) = %d, want %d", tc.n, tc.r, result, tc.expected)
		}
	}
}

func TestPositionBearoffRoundTrip(t *testing.T) {
	// Test a simple bearoff position
	board := []uint8{3, 2, 1, 0, 0, 0} // 6 checkers on first 3 points
	nPoints := uint32(6)
	nChequers := uint32(6)

	// Get bearoff index
	idx := PositionBearoff(board, nPoints, nChequers)

	// Convert back
	board2 := PositionFromBearoff(idx, nPoints, nChequers)

	// Compare
	for i := 0; i < int(nPoints); i++ {
		if board[i] != board2[i] {
			t.Errorf("PositionBearoff round-trip failed at point %d: got %d, want %d",
				i, board2[i], board[i])
		}
	}
}
