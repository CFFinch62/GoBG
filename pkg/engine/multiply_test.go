package engine

import (
	"math"
	"path/filepath"
	"testing"
	"time"
)

// createTestEngine creates an engine with neural network weights loaded
func createTestEngine(t *testing.T) *Engine {
	t.Helper()

	// Load weights from local data directory
	weightsPath := filepath.Join("..", "..", "data", "gnubg.weights")
	bearoffPath := filepath.Join("..", "..", "data", "gnubg_os0.bd")
	metPath := filepath.Join("..", "..", "data", "g11.xml")

	engine, err := NewEngine(EngineOptions{
		WeightsFileText: weightsPath,
		BearoffFile:     bearoffPath,
		METFile:         metPath,
	})
	if err != nil {
		t.Skipf("Skipping test - could not load data files: %v", err)
	}
	return engine
}

func TestEvaluatePlied0Ply(t *testing.T) {
	engine := createTestEngine(t)

	state := StartingPosition()

	// 0-ply should be equivalent to Evaluate
	eval0, err := engine.EvaluatePlied(state, 0)
	if err != nil {
		t.Fatalf("EvaluatePlied(0) failed: %v", err)
	}

	evalBase, err := engine.Evaluate(state)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	t.Logf("Base eval: Win=%.4f, WinG=%.4f, Equity=%.4f", evalBase.WinProb, evalBase.WinG, evalBase.Equity)
	t.Logf("0-ply eval: Win=%.4f, WinG=%.4f, Equity=%.4f", eval0.WinProb, eval0.WinG, eval0.Equity)

	// Should be identical
	if math.Abs(eval0.Equity-evalBase.Equity) > 0.0001 {
		t.Errorf("0-ply equity mismatch: got %f, want %f", eval0.Equity, evalBase.Equity)
	}
}

func TestEvaluatePlied1Ply(t *testing.T) {
	engine := createTestEngine(t)

	state := StartingPosition()

	start := time.Now()
	eval, err := engine.EvaluatePlied(state, 1)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("EvaluatePlied(1) failed: %v", err)
	}

	t.Logf("1-ply evaluation completed in %v", elapsed)
	t.Logf("Equity: %.4f, WinProb: %.4f", eval.Equity, eval.WinProb)

	// Starting position should be close to even
	if math.Abs(eval.Equity) > 0.1 {
		t.Errorf("Starting position equity too far from 0: %.4f", eval.Equity)
	}

	// Win probability should be close to 50%
	if math.Abs(eval.WinProb-0.5) > 0.1 {
		t.Errorf("Win probability too far from 50%%: %.4f", eval.WinProb)
	}
}

func TestEvaluatePlied2Ply(t *testing.T) {
	engine := createTestEngine(t)

	state := StartingPosition()

	start := time.Now()
	eval, err := engine.EvaluatePlied(state, 2)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("EvaluatePlied(2) failed: %v", err)
	}

	t.Logf("2-ply evaluation completed in %v", elapsed)
	t.Logf("Equity: %.4f, WinProb: %.4f", eval.Equity, eval.WinProb)

	// Starting position should still be close to even
	if math.Abs(eval.Equity) > 0.15 {
		t.Errorf("Starting position equity too far from 0: %.4f", eval.Equity)
	}
}

func TestMultiPlyConsistency(t *testing.T) {
	engine := createTestEngine(t)

	state := StartingPosition()

	// Get evaluations at different plies
	eval0, _ := engine.EvaluatePlied(state, 0)
	eval1, _ := engine.EvaluatePlied(state, 1)

	t.Logf("0-ply: Equity=%.4f, Win=%.4f", eval0.Equity, eval0.WinProb)
	t.Logf("1-ply: Equity=%.4f, Win=%.4f", eval1.Equity, eval1.WinProb)

	// All should have reasonable values
	for i, eval := range []*Evaluation{eval0, eval1} {
		if eval.WinProb < 0 || eval.WinProb > 1 {
			t.Errorf("%d-ply: WinProb out of range: %.4f", i, eval.WinProb)
		}
		if eval.WinG < 0 || eval.WinG > eval.WinProb {
			t.Errorf("%d-ply: WinG invalid: %.4f (WinProb=%.4f)", i, eval.WinG, eval.WinProb)
		}
	}
}

func TestMultiPlyMoveGeneration(t *testing.T) {
	engine := createTestEngine(t)

	state := StartingPosition()

	// Test that move generation works
	ml := GenerateMoves(state.Board, 3, 1)
	t.Logf("Starting position with 3-1: %d moves generated", len(ml.Moves))

	if len(ml.Moves) == 0 {
		t.Error("Expected moves to be generated for starting position")
	}

	// Test that 1-ply evaluation processes all dice rolls
	eval1, err := engine.EvaluatePlied(state, 1)
	if err != nil {
		t.Fatalf("1-ply evaluation failed: %v", err)
	}

	t.Logf("1-ply result: Win=%.4f, Equity=%.4f", eval1.WinProb, eval1.Equity)
}

func BenchmarkEvaluate0Ply(b *testing.B) {
	weightsPath := filepath.Join("..", "..", "data", "gnubg.weights")
	engine, err := NewEngine(EngineOptions{WeightsFileText: weightsPath})
	if err != nil {
		b.Skipf("Skipping - could not load weights: %v", err)
	}

	state := StartingPosition()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Evaluate(state)
		if err != nil {
			b.Fatalf("Evaluate failed: %v", err)
		}
	}
}

func BenchmarkEvaluatePlied1(b *testing.B) {
	weightsPath := filepath.Join("..", "..", "data", "gnubg.weights")
	engine, err := NewEngine(EngineOptions{WeightsFileText: weightsPath})
	if err != nil {
		b.Skipf("Skipping - could not load weights: %v", err)
	}

	state := StartingPosition()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.EvaluatePlied(state, 1)
		if err != nil {
			b.Fatalf("EvaluatePlied failed: %v", err)
		}
	}
}

func BenchmarkEvaluatePlied2(b *testing.B) {
	weightsPath := filepath.Join("..", "..", "data", "gnubg.weights")
	engine, err := NewEngine(EngineOptions{WeightsFileText: weightsPath})
	if err != nil {
		b.Skipf("Skipping - could not load weights: %v", err)
	}

	state := StartingPosition()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.EvaluatePlied(state, 2)
		if err != nil {
			b.Fatalf("EvaluatePlied failed: %v", err)
		}
	}
}

func TestPruneMoves(t *testing.T) {
	engine := createTestEngine(t)

	state := StartingPosition()

	// Generate moves for 3-1 (should have many moves)
	ml := GenerateMoves(state.Board, 3, 1)
	t.Logf("Generated %d moves for 3-1", len(ml.Moves))

	if len(ml.Moves) <= MinPruneMoves {
		t.Skipf("Not enough moves to test pruning (need > %d, got %d)", MinPruneMoves, len(ml.Moves))
	}

	// Test pruning
	pruned := engine.pruneMoves(state, ml.Moves)
	t.Logf("Pruned to %d moves", len(pruned))

	// Should have reduced the number of moves
	if len(pruned) >= len(ml.Moves) {
		t.Errorf("Pruning should reduce moves: got %d, original %d", len(pruned), len(ml.Moves))
	}

	// Should keep at least MinPruneMoves
	if len(pruned) < MinPruneMoves {
		t.Errorf("Should keep at least %d moves, got %d", MinPruneMoves, len(pruned))
	}
}

func TestPruneVsNoPrune(t *testing.T) {
	engine := createTestEngine(t)

	state := StartingPosition()

	// Compare pruned vs non-pruned evaluation at 1-ply
	opts := EvalOptions{Plies: 1, UsePrune: true}
	start := time.Now()
	evalPruned, err := engine.EvaluatePliedWithOptions(state, opts)
	prunedTime := time.Since(start)
	if err != nil {
		t.Fatalf("Pruned evaluation failed: %v", err)
	}

	opts.UsePrune = false
	start = time.Now()
	evalNoPrune, err := engine.EvaluatePliedWithOptions(state, opts)
	noPruneTime := time.Since(start)
	if err != nil {
		t.Fatalf("Non-pruned evaluation failed: %v", err)
	}

	t.Logf("1-ply Pruned:   Equity=%.4f, Time=%v", evalPruned.Equity, prunedTime)
	t.Logf("1-ply NoPrune:  Equity=%.4f, Time=%v", evalNoPrune.Equity, noPruneTime)

	// Results should be similar (within 0.05 equity)
	diff := math.Abs(evalPruned.Equity - evalNoPrune.Equity)
	if diff > 0.05 {
		t.Errorf("Pruned and non-pruned results differ too much: %.4f", diff)
	}
}

func TestPruneVsNoPrune2Ply(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping 2-ply comparison in short mode")
	}

	engine := createTestEngine(t)

	state := StartingPosition()

	// Compare pruned vs non-pruned evaluation at 2-ply
	opts := EvalOptions{Plies: 2, UsePrune: true}
	start := time.Now()
	evalPruned, err := engine.EvaluatePliedWithOptions(state, opts)
	prunedTime := time.Since(start)
	if err != nil {
		t.Fatalf("Pruned evaluation failed: %v", err)
	}

	opts.UsePrune = false
	start = time.Now()
	evalNoPrune, err := engine.EvaluatePliedWithOptions(state, opts)
	noPruneTime := time.Since(start)
	if err != nil {
		t.Fatalf("Non-pruned evaluation failed: %v", err)
	}

	t.Logf("2-ply Pruned:   Equity=%.4f, Time=%v", evalPruned.Equity, prunedTime)
	t.Logf("2-ply NoPrune:  Equity=%.4f, Time=%v", evalNoPrune.Equity, noPruneTime)
	t.Logf("Speedup: %.2fx", float64(noPruneTime)/float64(prunedTime))

	// Results should be similar (within 0.05 equity)
	diff := math.Abs(evalPruned.Equity - evalNoPrune.Equity)
	if diff > 0.05 {
		t.Errorf("Pruned and non-pruned results differ too much: %.4f", diff)
	}
}

func TestEvalCache(t *testing.T) {
	engine := createTestEngine(t)

	// Check cache was created
	cache := engine.Cache()
	if cache == nil {
		t.Fatal("Cache should be created by default")
	}

	state := StartingPosition()

	// First evaluation - cache miss
	cache.Flush() // Reset stats
	_, err := engine.EvaluateCached(state, 0)
	if err != nil {
		t.Fatalf("EvaluateCached failed: %v", err)
	}

	lookups, hits, adds := cache.Stats()
	t.Logf("After first eval: lookups=%d, hits=%d, adds=%d", lookups, hits, adds)

	if lookups != 1 || hits != 0 || adds != 1 {
		t.Errorf("Expected 1 lookup, 0 hits, 1 add; got %d, %d, %d", lookups, hits, adds)
	}

	// Second evaluation of same position - cache hit
	_, err = engine.EvaluateCached(state, 0)
	if err != nil {
		t.Fatalf("EvaluateCached failed: %v", err)
	}

	lookups, hits, adds = cache.Stats()
	t.Logf("After second eval: lookups=%d, hits=%d, adds=%d", lookups, hits, adds)

	if lookups != 2 || hits != 1 || adds != 1 {
		t.Errorf("Expected 2 lookups, 1 hit, 1 add; got %d, %d, %d", lookups, hits, adds)
	}

	t.Logf("Cache hit rate: %.1f%%", cache.HitRate())
}

func TestCachePerformance2Ply(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	engine := createTestEngine(t)
	state := StartingPosition()

	// Run 2-ply eval and check cache stats
	cache := engine.Cache()
	cache.Flush()

	start := time.Now()
	_, err := engine.EvaluatePlied(state, 2)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("EvaluatePlied failed: %v", err)
	}

	lookups, hits, _ := cache.Stats()
	hitRate := cache.HitRate()

	t.Logf("2-ply with cache: %v", elapsed)
	t.Logf("Cache: %d lookups, %d hits (%.1f%% hit rate)", lookups, hits, hitRate)

	// We expect significant cache hits in 2-ply evaluation
	if hitRate < 10 {
		t.Logf("Warning: Low cache hit rate (%.1f%%), expected >10%%", hitRate)
	}
}
