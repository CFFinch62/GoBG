package engine

import (
	"math"
	"path/filepath"
	"testing"

	"github.com/yourusername/bgengine/internal/positionid"
)

// GnubgReference contains known gnubg 0-ply evaluations for specific positions
// These values were captured from gnubg with "set evaluation plies 0"
type GnubgReference struct {
	Name       string
	PositionID string
	WinProb    float64
	WinG       float64
	WinBG      float64
	LoseG      float64
	LoseBG     float64
	Equity     float64
}

// Known gnubg 0-ply evaluations (captured from gnubg 1.08)
// These values match exactly what gnubg reports for 0-ply cubeless evaluation
var gnubgReferences = []GnubgReference{
	{
		Name:       "Starting Position",
		PositionID: "4HPwATDgc/ABMA",
		WinProb:    0.5242,
		WinG:       0.1494,
		WinBG:      0.0093,
		LoseG:      0.1439,
		LoseBG:     0.0086,
		Equity:     0.0788,
	},
}

func createAccuracyTestEngine(t *testing.T) *Engine {
	t.Helper()
	weightsPath := filepath.Join("..", "..", "data", "gnubg.weights")
	bearoffPath := filepath.Join("..", "..", "data", "gnubg_os0.bd")

	engine, err := NewEngine(EngineOptions{
		WeightsFileText: weightsPath,
		BearoffFile:     bearoffPath,
	})
	if err != nil {
		t.Skipf("Skipping accuracy test - could not load data files: %v", err)
	}
	return engine
}

func TestAccuracyVsGnubg0Ply(t *testing.T) {
	engine := createAccuracyTestEngine(t)

	maxEquityDiff := 0.0
	totalEquityDiff := 0.0
	count := 0

	for _, ref := range gnubgReferences {
		t.Run(ref.Name, func(t *testing.T) {
			// Decode position from ID
			board, err := positionid.BoardFromPositionID(ref.PositionID)
			if err != nil {
				t.Skipf("Could not decode position ID %s: %v", ref.PositionID, err)
				return
			}

			state := &GameState{
				Board:     Board(board),
				Turn:      0,
				CubeValue: 1,
				CubeOwner: -1,
			}

			eval, err := engine.Evaluate(state)
			if err != nil {
				t.Fatalf("Evaluate failed: %v", err)
			}

			// Compare values
			winDiff := math.Abs(eval.WinProb - ref.WinProb)
			equityDiff := math.Abs(eval.Equity - ref.Equity)

			t.Logf("Position: %s", ref.Name)
			t.Logf("  GoBG:   Win=%.4f, WinG=%.4f, Equity=%.4f",
				eval.WinProb, eval.WinG, eval.Equity)
			t.Logf("  gnubg:  Win=%.4f, WinG=%.4f, Equity=%.4f",
				ref.WinProb, ref.WinG, ref.Equity)
			t.Logf("  Diff:   Win=%.4f, Equity=%.4f", winDiff, equityDiff)

			// Track statistics
			if equityDiff > maxEquityDiff {
				maxEquityDiff = equityDiff
			}
			totalEquityDiff += equityDiff
			count++

			// Accuracy threshold: 0.02 equity (per success criteria)
			if equityDiff > 0.02 {
				t.Errorf("Equity difference %.4f exceeds threshold 0.02", equityDiff)
			}
		})
	}

	if count > 0 {
		avgDiff := totalEquityDiff / float64(count)
		t.Logf("\n=== Accuracy Summary ===")
		t.Logf("Positions tested: %d", count)
		t.Logf("Max equity diff:  %.4f", maxEquityDiff)
		t.Logf("Avg equity diff:  %.4f", avgDiff)
	}
}

func TestAccuracyStartingPosition(t *testing.T) {
	engine := createAccuracyTestEngine(t)

	state := StartingPosition()
	eval, err := engine.Evaluate(state)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	// gnubg 0-ply values for starting position
	expectedWin := 0.5242
	expectedEquity := 0.0788

	t.Logf("Starting Position Evaluation:")
	t.Logf("  Win Prob: %.4f (expected: %.4f, diff: %.4f)",
		eval.WinProb, expectedWin, math.Abs(eval.WinProb-expectedWin))
	t.Logf("  Equity:   %.4f (expected: %.4f, diff: %.4f)",
		eval.Equity, expectedEquity, math.Abs(eval.Equity-expectedEquity))

	if math.Abs(eval.Equity-expectedEquity) > 0.02 {
		t.Errorf("Equity differs by more than 0.02 from gnubg")
	}
}

func TestAccuracyStartingPosition1Ply(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping 1-ply test in short mode")
	}
	engine := createAccuracyTestEngine(t)

	state := StartingPosition()
	eval, err := engine.EvaluatePlied(state, 1)
	if err != nil {
		t.Fatalf("EvaluatePlied failed: %v", err)
	}

	// Our 1-ply: average best equity across all dice rolls from current position
	// This evaluates what happens when WE roll (player to move gets the roll)
	// Note: gnubg 1-ply may report differently depending on whose perspective
	t.Logf("Starting Position 1-ply Evaluation (GoBG):")
	t.Logf("  Win Prob: %.4f", eval.WinProb)
	t.Logf("  Equity:   %.4f", eval.Equity)

	// Basic sanity: 1-ply should still show roughly equal position
	if eval.WinProb < 0.45 || eval.WinProb > 0.60 {
		t.Errorf("1-ply WinProb %.4f outside expected range [0.45, 0.60]", eval.WinProb)
	}
	if eval.Equity < -0.10 || eval.Equity > 0.15 {
		t.Errorf("1-ply Equity %.4f outside expected range [-0.10, 0.15]", eval.Equity)
	}
}

func TestAccuracyEvaluationProbabilities(t *testing.T) {
	engine := createAccuracyTestEngine(t)

	state := StartingPosition()
	eval, err := engine.Evaluate(state)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	// Verify probability sanity checks
	t.Logf("Probability sanity checks:")
	t.Logf("  WinProb:  %.4f", eval.WinProb)
	t.Logf("  WinG:     %.4f", eval.WinG)
	t.Logf("  WinBG:    %.4f", eval.WinBG)
	t.Logf("  LoseG:    %.4f", eval.LoseG)
	t.Logf("  LoseBG:   %.4f", eval.LoseBG)

	// WinProb should be between 0 and 1
	if eval.WinProb < 0 || eval.WinProb > 1 {
		t.Errorf("WinProb out of range: %.4f", eval.WinProb)
	}

	// Gammon rates should be <= total win/lose rates
	if eval.WinG > eval.WinProb {
		t.Errorf("WinG (%.4f) > WinProb (%.4f)", eval.WinG, eval.WinProb)
	}
	if eval.WinBG > eval.WinG {
		t.Errorf("WinBG (%.4f) > WinG (%.4f)", eval.WinBG, eval.WinG)
	}

	loseProb := 1.0 - eval.WinProb
	if eval.LoseG > loseProb {
		t.Errorf("LoseG (%.4f) > LoseProb (%.4f)", eval.LoseG, loseProb)
	}
	if eval.LoseBG > eval.LoseG {
		t.Errorf("LoseBG (%.4f) > LoseG (%.4f)", eval.LoseBG, eval.LoseG)
	}
}
