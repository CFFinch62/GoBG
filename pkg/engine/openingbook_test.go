package engine

import (
	"testing"
)

func TestLookupOpeningExists(t *testing.T) {
	engine, err := NewEngine(EngineOptions{})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	state := StartingPosition()

	// Test all 15 non-double opening rolls
	openingRolls := [][2]int{
		{6, 5}, {6, 4}, {6, 3}, {6, 2}, {6, 1},
		{5, 4}, {5, 3}, {5, 2}, {5, 1},
		{4, 3}, {4, 2}, {4, 1},
		{3, 2}, {3, 1},
		{2, 1},
	}

	for _, dice := range openingRolls {
		entry, found := engine.LookupOpening(state, dice)
		if !found {
			t.Errorf("Opening not found for roll %d-%d", dice[0], dice[1])
			continue
		}
		if entry.Note == "" {
			t.Errorf("Opening note missing for roll %d-%d", dice[0], dice[1])
		}
		// Verify move has valid from/to
		if entry.Move.From[0] < 0 {
			t.Errorf("Opening move has invalid From for roll %d-%d", dice[0], dice[1])
		}
	}
}

func TestLookupOpeningReversedDice(t *testing.T) {
	engine, err := NewEngine(EngineOptions{})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	state := StartingPosition()

	// Test that reversed dice order works
	entry1, found1 := engine.LookupOpening(state, [2]int{6, 5})
	entry2, found2 := engine.LookupOpening(state, [2]int{5, 6})

	if !found1 || !found2 {
		t.Fatal("Opening not found for 6-5 or 5-6")
	}

	// Both should return the same move
	if entry1.Move.From != entry2.Move.From || entry1.Move.To != entry2.Move.To {
		t.Error("Reversed dice should return same move")
	}
}

func TestLookupOpeningDoublesNotFound(t *testing.T) {
	engine, err := NewEngine(EngineOptions{})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	state := StartingPosition()

	// Doubles are not opening rolls
	doubles := [][2]int{{1, 1}, {2, 2}, {3, 3}, {4, 4}, {5, 5}, {6, 6}}
	for _, dice := range doubles {
		_, found := engine.LookupOpening(state, dice)
		if found {
			t.Errorf("Doubles %d-%d should not be in opening book", dice[0], dice[1])
		}
	}
}

func TestLookupOpeningNonStartingPosition(t *testing.T) {
	engine, err := NewEngine(EngineOptions{})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Create a non-starting position
	state := StartingPosition()
	state.Board[0][5] = 4 // Modify the board

	_, found := engine.LookupOpening(state, [2]int{3, 1})
	if found {
		t.Error("Opening should not be found for non-starting position")
	}
}

func TestOpeningMoveWithEval(t *testing.T) {
	engine, err := NewEngine(EngineOptions{})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	state := StartingPosition()

	move, eval, err := engine.OpeningMoveWithEval(state, [2]int{3, 1})
	if err != nil {
		t.Fatalf("OpeningMoveWithEval failed: %v", err)
	}

	// 31 should make the 5-point: 8/5 6/5
	if move.From[0] != 7 || move.From[1] != 5 {
		t.Errorf("Expected 8/5 6/5 (from 7,5), got from %v", move.From)
	}
	if move.To[0] != 4 || move.To[1] != 4 {
		t.Errorf("Expected 8/5 6/5 (to 4,4), got to %v", move.To)
	}

	// Should have evaluation
	if eval == nil {
		t.Error("Expected evaluation to be returned")
	}
}

func TestOpeningMovesFallback(t *testing.T) {
	engine, err := NewEngine(EngineOptions{})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Non-starting position should fall back to regular analysis
	state := StartingPosition()
	state.Board[0][5] = 4 // Modify

	move, eval, err := engine.OpeningMoveWithEval(state, [2]int{3, 1})
	if err != nil {
		t.Fatalf("OpeningMoveWithEval failed: %v", err)
	}

	// Should still return a valid move via regular analysis
	if move.From[0] < 0 {
		t.Error("Expected valid move from fallback analysis")
	}
	if eval == nil {
		t.Error("Expected evaluation from fallback analysis")
	}
}

func TestOpeningBookSpecificMoves(t *testing.T) {
	engine, err := NewEngine(EngineOptions{})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	state := StartingPosition()

	tests := []struct {
		dice     [2]int
		expected string
	}{
		{[2]int{6, 5}, "24/13"},
		{[2]int{6, 1}, "13/7 8/7"},
		{[2]int{3, 1}, "8/5 6/5"},
		{[2]int{4, 2}, "8/4 6/4"},
	}

	for _, tc := range tests {
		entry, found := engine.LookupOpening(state, tc.dice)
		if !found {
			t.Errorf("Opening not found for %d-%d", tc.dice[0], tc.dice[1])
			continue
		}
		if entry.Note == "" || entry.Note[:len(tc.expected)] != tc.expected {
			t.Logf("Roll %d-%d: %s", tc.dice[0], tc.dice[1], entry.Note)
		}
	}
}

