package engine

import (
	"testing"
)

func TestNewEngine(t *testing.T) {
	// Test creating engine with no options (should use defaults)
	e, err := NewEngine(EngineOptions{})
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}
	if e == nil {
		t.Fatal("NewEngine returned nil")
	}
	if e.met == nil {
		t.Error("Expected default MET to be loaded")
	}
}

func TestEvaluateStartingPosition(t *testing.T) {
	e, err := NewEngine(EngineOptions{})
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	state := StartingPosition()
	eval, err := e.Evaluate(state)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	// Starting position should be roughly equal
	if eval.WinProb < 0.3 || eval.WinProb > 0.7 {
		t.Errorf("Expected WinProb ~0.5, got %f", eval.WinProb)
	}
}

func TestEvaluateGameOver(t *testing.T) {
	e, err := NewEngine(EngineOptions{})
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	// Create a position where player 1 has borne off all checkers
	state := &GameState{
		Turn:      0,
		CubeValue: 1,
		CubeOwner: -1,
	}
	// Player 0 still has checkers
	state.Board[0][5] = 5
	state.Board[0][7] = 3
	state.Board[0][12] = 5
	state.Board[0][23] = 2
	// Player 1 has no checkers (all borne off)

	eval, err := e.Evaluate(state)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	// Player 1 has won
	if eval.WinProb != 1.0 {
		t.Errorf("Expected WinProb 1.0 for game over, got %f", eval.WinProb)
	}
}

func TestEvaluateGammon(t *testing.T) {
	e, err := NewEngine(EngineOptions{})
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	// Create a position where player 1 has won a gammon
	// Player 0 has checkers but none borne off, none in home board
	// and NOT in opponent's home board (points 18-23) or on bar
	state := &GameState{
		Turn:      0,
		CubeValue: 1,
		CubeOwner: -1,
	}
	// Player 0 has checkers in outer board (not home board, not opponent's home)
	state.Board[0][10] = 10 // 10 checkers on 11-point
	state.Board[0][12] = 5  // 5 checkers on 13-point
	// Player 1 has no checkers (all borne off)

	eval, err := e.Evaluate(state)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	// Player 1 has won a gammon (not backgammon since no checkers in opponent's home)
	if eval.WinProb != 1.0 {
		t.Errorf("Expected WinProb 1.0, got %f", eval.WinProb)
	}
	if eval.WinG != 1.0 {
		t.Errorf("Expected WinG 1.0 for gammon, got %f", eval.WinG)
	}
	if eval.WinBG != 0.0 {
		t.Errorf("Expected WinBG 0.0 (not backgammon), got %f", eval.WinBG)
	}
	if eval.Equity != 2.0 {
		t.Errorf("Expected Equity 2.0 for gammon, got %f", eval.Equity)
	}
}

func TestEvaluateBackgammon(t *testing.T) {
	e, err := NewEngine(EngineOptions{})
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	// Create a position where player 1 has won a backgammon
	// Player 0 has checkers in opponent's home board
	state := &GameState{
		Turn:      0,
		CubeValue: 1,
		CubeOwner: -1,
	}
	// Player 0 has checkers in opponent's home board (points 18-23)
	state.Board[0][18] = 10 // 10 checkers on 19-point (opponent's home)
	state.Board[0][23] = 5  // 5 checkers on 24-point (opponent's home)
	// Player 1 has no checkers (all borne off)

	eval, err := e.Evaluate(state)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	// Player 1 has won a backgammon
	if eval.WinProb != 1.0 {
		t.Errorf("Expected WinProb 1.0, got %f", eval.WinProb)
	}
	if eval.WinBG != 1.0 {
		t.Errorf("Expected WinBG 1.0 for backgammon, got %f", eval.WinBG)
	}
	if eval.Equity != 3.0 {
		t.Errorf("Expected Equity 3.0 for backgammon, got %f", eval.Equity)
	}
}

func TestGetMatchEquity(t *testing.T) {
	e, err := NewEngine(EngineOptions{})
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	state := &GameState{
		MatchLength: 11,
		Score:       [2]int{0, 0},
		Crawford:    false,
	}

	// At 0-0 in 11pt match, equity should be ~0.5
	eq := e.GetMatchEquity(state, 0)
	if eq < 0.4 || eq > 0.6 {
		t.Errorf("Expected match equity ~0.5 at 0-0, got %f", eq)
	}
}

// Global to prevent compiler optimizations
var benchEval *Evaluation

func BenchmarkEvaluateEngine(b *testing.B) {
	e, err := NewEngine(EngineOptions{
		WeightsFileText: "../../data/gnubg.weights",
	})
	if err != nil {
		b.Fatalf("NewEngine failed: %v", err)
	}

	state := StartingPosition()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchEval, _ = e.Evaluate(state)
	}
}

func BenchmarkEvaluateEngineContact(b *testing.B) {
	e, err := NewEngine(EngineOptions{
		WeightsFileText: "../../data/gnubg.weights",
	})
	if err != nil {
		b.Fatalf("NewEngine failed: %v", err)
	}

	// A contact position
	state := &GameState{
		Turn:      0,
		CubeValue: 1,
		CubeOwner: -1,
	}
	state.Board[0][5] = 5
	state.Board[0][7] = 3
	state.Board[0][12] = 5
	state.Board[0][23] = 2

	state.Board[1][5] = 5
	state.Board[1][7] = 3
	state.Board[1][12] = 5
	state.Board[1][23] = 2

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchEval, _ = e.Evaluate(state)
	}
}
