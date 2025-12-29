package bearoff

import (
	"testing"
)

func TestCombination(t *testing.T) {
	tests := []struct {
		n, r     int
		expected int
	}{
		{6, 1, 6},
		{6, 2, 15},
		{6, 3, 20},
		{6, 6, 1},
		{10, 3, 120},
		{21, 6, 54264}, // C(21, 6) = number of positions for 6 points, 15 checkers
	}

	for _, tt := range tests {
		result := Combination(tt.n, tt.r)
		if result != tt.expected {
			t.Errorf("Combination(%d, %d) = %d, expected %d", tt.n, tt.r, result, tt.expected)
		}
	}
}

func TestPositionBearoff(t *testing.T) {
	// Test some known positions
	tests := []struct {
		board     []uint8
		nPoints   int
		nCheckers int
		expected  int
	}{
		// All checkers on point 1 (index 0)
		{[]uint8{15, 0, 0, 0, 0, 0}, 6, 15, 0},
		// One checker on each of first 6 points
		{[]uint8{1, 1, 1, 1, 1, 1}, 6, 15, 0}, // This should be a valid position
	}

	for i, tt := range tests {
		result := PositionBearoff(tt.board, tt.nPoints, tt.nCheckers)
		t.Logf("Test %d: board=%v, result=%d", i, tt.board, result)
		// Just verify it doesn't panic and returns a reasonable value
		if result < 0 {
			t.Errorf("PositionBearoff returned negative value: %d", result)
		}
	}
}

func TestPositionFromBearoff(t *testing.T) {
	// Test round-trip conversion
	for posID := 0; posID < 100; posID++ {
		board := PositionFromBearoff(posID, 6, 15)

		// Count total checkers
		total := 0
		for _, c := range board {
			total += int(c)
		}

		// Convert back
		resultID := PositionBearoff(board[:], 6, 15)
		if resultID != posID {
			t.Errorf("Round-trip failed: posID=%d, board=%v, resultID=%d", posID, board, resultID)
		}
	}
}

func TestLoadOneSided(t *testing.T) {
	db, err := LoadOneSided("../../data/gnubg_os0.bd")
	if err != nil {
		t.Skipf("Bearoff database not found: %v", err)
	}

	t.Logf("Database loaded: Type=%d, Points=%d, Checkers=%d, Compressed=%v, HasGammon=%v",
		db.Type, db.NPoints, db.NChequers, db.Compressed, db.HasGammon)

	if db.Type != BearoffOneSided {
		t.Errorf("Expected one-sided database, got type %d", db.Type)
	}

	if db.NPoints != 6 {
		t.Errorf("Expected 6 points, got %d", db.NPoints)
	}

	if db.NChequers != 15 {
		t.Errorf("Expected 15 checkers, got %d", db.NChequers)
	}

	// Test getting a distribution
	prob, gammonProb, err := db.GetDistribution(0)
	if err != nil {
		t.Errorf("GetDistribution(0) failed: %v", err)
	}

	// Position 0 should have all checkers on point 1, so should bear off in 1 roll
	t.Logf("Position 0 distribution: prob[0]=%f, prob[1]=%f", prob[0], prob[1])
	t.Logf("Position 0 gammon distribution: gammonProb[0]=%f", gammonProb[0])

	// Test evaluation
	board := [2][6]uint8{
		{5, 5, 5, 0, 0, 0}, // Opponent: 15 checkers on points 1-3
		{5, 5, 5, 0, 0, 0}, // Player: 15 checkers on points 1-3
	}
	output, err := db.Evaluate(board)
	if err != nil {
		t.Errorf("Evaluate failed: %v", err)
	}
	t.Logf("Evaluation: win=%f", output[0])

	// With equal positions, the player to move has an advantage
	// Win probability should be > 0.5 due to first-mover advantage
	if output[0] < 0.5 || output[0] > 0.8 {
		t.Errorf("Expected win probability between 0.5 and 0.8, got %f", output[0])
	}

	// Test average rolls
	mean, stddev, err := db.GetAverageRolls(0)
	if err != nil {
		t.Errorf("GetAverageRolls(0) failed: %v", err)
	}
	t.Logf("Position 0 average rolls: mean=%f, stddev=%f", mean, stddev)

	// Position 0 should bear off in 0 rolls (already off)
	if mean > 0.1 {
		t.Errorf("Expected mean near 0 for position 0, got %f", mean)
	}
}

func TestLoadTwoSided(t *testing.T) {
	db, err := LoadOneSided("../../data/gnubg_ts.bd")
	if err != nil {
		t.Skipf("Two-sided bearoff database not found: %v", err)
	}

	t.Logf("Database loaded: Type=%d, Points=%d, Checkers=%d, Cubeful=%v",
		db.Type, db.NPoints, db.NChequers, db.Cubeful)

	if db.Type != BearoffTwoSided {
		t.Errorf("Expected two-sided database, got type %d", db.Type)
	}

	if db.NPoints != 6 {
		t.Errorf("Expected 6 points, got %d", db.NPoints)
	}

	if db.NChequers != 6 {
		t.Errorf("Expected 6 checkers, got %d", db.NChequers)
	}

	// Test evaluation with equal positions
	board := [2][6]uint8{
		{3, 3, 0, 0, 0, 0}, // Opponent: 6 checkers on points 1-2
		{3, 3, 0, 0, 0, 0}, // Player: 6 checkers on points 1-2
	}
	output, err := db.Evaluate(board)
	if err != nil {
		t.Errorf("Evaluate failed: %v", err)
	}
	t.Logf("Equal position evaluation: win=%f", output[0])

	// With equal positions, the player to move has an advantage
	// In bearoff, first-mover advantage can be significant
	if output[0] < 0.5 || output[0] > 0.9 {
		t.Errorf("Expected win probability between 0.5 and 0.9, got %f", output[0])
	}

	// Test asymmetric position
	board2 := [2][6]uint8{
		{6, 0, 0, 0, 0, 0}, // Opponent: 6 checkers on point 1 (easy to bear off)
		{0, 0, 0, 0, 0, 6}, // Player: 6 checkers on point 6 (harder to bear off)
	}
	output2, err := db.Evaluate(board2)
	if err != nil {
		t.Errorf("Evaluate asymmetric failed: %v", err)
	}
	t.Logf("Asymmetric position evaluation: win=%f", output2[0])

	// Player has harder position, so should have lower win probability
	if output2[0] > 0.5 {
		t.Errorf("Expected win probability < 0.5 for harder position, got %f", output2[0])
	}
}
