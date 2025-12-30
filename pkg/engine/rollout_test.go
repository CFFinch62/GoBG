package engine

import (
	"runtime"
	"testing"
	"time"
)

func TestRolloutDefaultOptions(t *testing.T) {
	opts := DefaultRolloutOptions()
	if opts.Trials != 1296 {
		t.Errorf("Default Trials = %d, want 1296", opts.Trials)
	}
	if opts.Truncate != 0 {
		t.Errorf("Default Truncate = %d, want 0", opts.Truncate)
	}
}

func TestRolloutStartingPosition(t *testing.T) {
	engine, err := NewEngine(EngineOptions{})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	state := StartingPosition()
	opts := RolloutOptions{
		Trials:  324, // Larger for better accuracy
		Seed:    12345,
		Workers: 4,
	}

	start := time.Now()
	result, err := engine.Rollout(state, opts)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Rollout failed: %v", err)
	}

	t.Logf("Rollout completed in %v", elapsed)
	t.Logf("Trials: %d, Equity: %.4f ± %.4f (CI: %.4f)",
		result.TrialsCompleted, result.Equity, result.EquityStdDev, result.EquityCI)
	t.Logf("WinProb: %.4f, WinG: %.4f, LoseG: %.4f",
		result.WinProb, result.WinG, result.LoseG)

	// Verify all trials completed
	if result.TrialsCompleted != opts.Trials {
		t.Errorf("TrialsCompleted = %d, want %d", result.TrialsCompleted, opts.Trials)
	}

	// Win probability should be near 50% for starting position
	if result.WinProb < 0.3 || result.WinProb > 0.7 {
		t.Errorf("WinProb = %.4f, expected near 0.5", result.WinProb)
	}

	// Equity should be near 0 for starting position
	if result.Equity < -0.5 || result.Equity > 0.5 {
		t.Errorf("Equity = %.4f, expected near 0", result.Equity)
	}
}

func TestRolloutWithTruncation(t *testing.T) {
	engine, err := NewEngine(EngineOptions{})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	state := StartingPosition()
	opts := RolloutOptions{
		Trials:   36,
		Truncate: 10, // Truncate after 10 plies
		Seed:     54321,
		Workers:  2,
	}

	start := time.Now()
	result, err := engine.Rollout(state, opts)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Rollout failed: %v", err)
	}

	t.Logf("Truncated rollout completed in %v", elapsed)
	t.Logf("Equity: %.4f ± %.4f", result.Equity, result.EquityStdDev)

	// Should complete all trials
	if result.TrialsCompleted != opts.Trials {
		t.Errorf("TrialsCompleted = %d, want %d", result.TrialsCompleted, opts.Trials)
	}
}

func TestRolloutDeterministic(t *testing.T) {
	engine, err := NewEngine(EngineOptions{})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	state := StartingPosition()
	opts := RolloutOptions{
		Trials:  36,
		Seed:    99999,
		Workers: 1, // Single worker for determinism
	}

	result1, err := engine.Rollout(state, opts)
	if err != nil {
		t.Fatalf("First rollout failed: %v", err)
	}

	result2, err := engine.Rollout(state, opts)
	if err != nil {
		t.Fatalf("Second rollout failed: %v", err)
	}

	// With same seed and single worker, results should be identical
	if result1.Equity != result2.Equity {
		t.Errorf("Determinism check failed: equity1 = %.6f, equity2 = %.6f",
			result1.Equity, result2.Equity)
	}
	if result1.WinProb != result2.WinProb {
		t.Errorf("Determinism check failed: winProb1 = %.6f, winProb2 = %.6f",
			result1.WinProb, result2.WinProb)
	}
}

func TestGameStatus(t *testing.T) {
	engine, err := NewEngine(EngineOptions{})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Game in progress - starting position
	state := StartingPosition()
	status := engine.gameStatus(&state.Board)
	if status != 0 {
		t.Errorf("Starting position status = %d, want 0 (in progress)", status)
	}

	// Player 0 wins - all checkers borne off
	var winBoard Board
	// Player 0: no checkers (all borne off)
	// Player 1: 15 checkers on point 0 (gammon - none borne off)
	winBoard[1][0] = 15
	status = engine.gameStatus(&winBoard)
	if status <= 0 {
		t.Errorf("Player 0 win status = %d, want > 0", status)
	}
}

func TestRolloutSinglePly(t *testing.T) {
	// Test that a single ply of rollout produces reasonable results
	engine, err := NewEngine(EngineOptions{})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	state := StartingPosition()

	// Generate moves for player 0 with 3-1 roll
	moves := engine.generateMovesForBoard(&state.Board, 0, 3, 1)
	t.Logf("Generated %d moves for player 0 with 3-1", len(moves))

	if len(moves) == 0 {
		t.Fatal("No moves generated")
	}

	// Apply the first move
	copyBoard := state.Board
	engine.applyMoveToBoard(&copyBoard, 0, moves[0])

	// Check that something changed
	t.Logf("Move: %v", moves[0])
	t.Logf("Original board[0]: %v", state.Board[0])
	t.Logf("After move board[0]: %v", copyBoard[0])
	t.Logf("Original board[1]: %v", state.Board[1])
	t.Logf("After move board[1]: %v", copyBoard[1])

	// Count total checkers before and after
	p0Before, p0After := 0, 0
	p1Before, p1After := 0, 0
	for i := 0; i < 25; i++ {
		p0Before += int(state.Board[0][i])
		p0After += int(copyBoard[0][i])
		p1Before += int(state.Board[1][i])
		p1After += int(copyBoard[1][i])
	}
	t.Logf("P0 checkers: before=%d, after=%d", p0Before, p0After)
	t.Logf("P1 checkers: before=%d, after=%d", p1Before, p1After)

	if p0Before != p0After || p0Before != 15 {
		t.Errorf("P0 checker count changed: %d -> %d", p0Before, p0After)
	}
	if p1Before != p1After || p1Before != 15 {
		t.Errorf("P1 checker count changed: %d -> %d", p1Before, p1After)
	}
}

func BenchmarkRollout1000(b *testing.B) {
	engine, err := NewEngine(EngineOptions{})
	if err != nil {
		b.Fatalf("Failed to create engine: %v", err)
	}

	state := StartingPosition()
	opts := RolloutOptions{
		Trials:  1000,
		Workers: runtime.GOMAXPROCS(0),
		Seed:    12345,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Rollout(state, opts)
		if err != nil {
			b.Fatalf("Rollout failed: %v", err)
		}
	}
}

func BenchmarkRolloutTruncated(b *testing.B) {
	engine, err := NewEngine(EngineOptions{})
	if err != nil {
		b.Fatalf("Failed to create engine: %v", err)
	}

	state := StartingPosition()
	opts := RolloutOptions{
		Trials:   1000,
		Truncate: 10, // Truncate at 10 plies
		Workers:  runtime.GOMAXPROCS(0),
		Seed:     12345,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Rollout(state, opts)
		if err != nil {
			b.Fatalf("Rollout failed: %v", err)
		}
	}
}

func TestRolloutWithProgress(t *testing.T) {
	engine, err := NewEngine(EngineOptions{})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	state := StartingPosition()
	opts := RolloutOptions{
		Trials:  500,
		Seed:    12345,
		Workers: 4,
	}

	var progressCalls []RolloutProgress
	callback := func(p RolloutProgress) {
		progressCalls = append(progressCalls, p)
	}

	result, err := engine.RolloutWithProgress(state, opts, callback)
	if err != nil {
		t.Fatalf("RolloutWithProgress failed: %v", err)
	}

	// Should have received multiple progress updates
	if len(progressCalls) < 2 {
		t.Errorf("Expected multiple progress callbacks, got %d", len(progressCalls))
	}

	// First progress call should have low percentage
	if len(progressCalls) > 0 && progressCalls[0].Percent > 50 {
		t.Errorf("First progress call should be early, got %.1f%%", progressCalls[0].Percent)
	}

	// Last progress call should be at 100%
	if len(progressCalls) > 0 {
		last := progressCalls[len(progressCalls)-1]
		if last.Percent != 100.0 {
			t.Errorf("Last progress call should be 100%%, got %.1f%%", last.Percent)
		}
		if last.TrialsCompleted != opts.Trials {
			t.Errorf("Last progress call should have all trials, got %d", last.TrialsCompleted)
		}
	}

	// Result should match regular rollout
	if result.TrialsCompleted != opts.Trials {
		t.Errorf("TrialsCompleted = %d, want %d", result.TrialsCompleted, opts.Trials)
	}

	t.Logf("Received %d progress callbacks", len(progressCalls))
	for i, p := range progressCalls {
		t.Logf("  [%d] %.1f%% (%d/%d) equity=%.4f ±%.4f",
			i, p.Percent, p.TrialsCompleted, p.TrialsTotal, p.CurrentEquity, p.CurrentCI)
	}
}

func TestRolloutWithProgressNilCallback(t *testing.T) {
	engine, err := NewEngine(EngineOptions{})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	state := StartingPosition()
	opts := RolloutOptions{
		Trials:  100,
		Seed:    12345,
		Workers: 2,
	}

	// Should work with nil callback
	result, err := engine.RolloutWithProgress(state, opts, nil)
	if err != nil {
		t.Fatalf("RolloutWithProgress with nil callback failed: %v", err)
	}

	if result.TrialsCompleted != opts.Trials {
		t.Errorf("TrialsCompleted = %d, want %d", result.TrialsCompleted, opts.Trials)
	}
}

func TestRolloutProgressEquityConvergence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping convergence test in short mode")
	}

	engine, err := NewEngine(EngineOptions{})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	state := StartingPosition()
	opts := RolloutOptions{
		Trials:  1000,
		Seed:    12345,
		Workers: 4,
	}

	var progressCalls []RolloutProgress
	callback := func(p RolloutProgress) {
		progressCalls = append(progressCalls, p)
	}

	result, err := engine.RolloutWithProgress(state, opts, callback)
	if err != nil {
		t.Fatalf("RolloutWithProgress failed: %v", err)
	}

	// CI should decrease as trials increase (convergence)
	if len(progressCalls) >= 3 {
		firstCI := progressCalls[0].CurrentCI
		lastCI := progressCalls[len(progressCalls)-1].CurrentCI
		if lastCI >= firstCI {
			t.Logf("Warning: CI did not decrease (first=%.4f, last=%.4f)", firstCI, lastCI)
		}
	}

	t.Logf("Final: equity=%.4f ±%.4f", result.Equity, result.EquityCI)
}
