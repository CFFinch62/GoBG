package met

import (
	"testing"
)

func TestLoadXML(t *testing.T) {
	table, err := LoadXML("../../data/g11.xml")
	if err != nil {
		t.Skipf("MET file not found: %v", err)
	}

	t.Logf("Loaded MET: %s", table.Name)
	t.Logf("Description: %s", table.Description)
	t.Logf("Length: %d", table.Length)

	// Check some known values from g11
	// At 0-0 in a match, equity should be 0.5
	equity := table.GetME(0, 0, 11, 0, false)
	if equity < 0.49 || equity > 0.51 {
		t.Errorf("Expected equity ~0.5 at 0-0, got %f", equity)
	}

	// Player 0 ahead should have > 0.5 equity
	equity = table.GetME(5, 0, 11, 0, false)
	if equity < 0.5 {
		t.Errorf("Expected equity > 0.5 when leading, got %f", equity)
	}

	// Player 0 behind should have < 0.5 equity
	equity = table.GetME(0, 5, 11, 0, false)
	if equity > 0.5 {
		t.Errorf("Expected equity < 0.5 when trailing, got %f", equity)
	}
}

func TestGetME(t *testing.T) {
	table := Default()

	tests := []struct {
		name     string
		score0   int
		score1   int
		matchTo  int
		player   int
		crawford bool
		wantMin  float32
		wantMax  float32
	}{
		{
			name:    "0-0 in 11pt match, player 0",
			score0:  0, score1: 0, matchTo: 11, player: 0,
			wantMin: 0.4, wantMax: 0.6,
		},
		{
			name:    "0-0 in 11pt match, player 1",
			score0:  0, score1: 0, matchTo: 11, player: 1,
			wantMin: 0.4, wantMax: 0.6,
		},
		{
			name:    "match won by player 0",
			score0:  11, score1: 5, matchTo: 11, player: 0,
			wantMin: 1.0, wantMax: 1.0,
		},
		{
			name:    "match won by player 1",
			score0:  5, score1: 11, matchTo: 11, player: 1,
			wantMin: 1.0, wantMax: 1.0,
		},
		{
			name:    "money game",
			score0:  0, score1: 0, matchTo: 0, player: 0,
			wantMin: 0.5, wantMax: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			equity := table.GetME(tt.score0, tt.score1, tt.matchTo, tt.player, tt.crawford)
			if equity < tt.wantMin || equity > tt.wantMax {
				t.Errorf("GetME() = %f, want between %f and %f", equity, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestGetMEAfterResult(t *testing.T) {
	table := Default()

	// If player 0 wins a gammon from 0-0 in 5pt match, they're at 2-0
	// Their equity should be > 0.5
	equity := table.GetMEAfterResult(0, 0, 5, 0, 2, 0, false)
	if equity <= 0.5 {
		t.Errorf("Expected equity > 0.5 after winning gammon, got %f", equity)
	}

	// If player 1 wins a backgammon from 0-0 in 5pt match, they're at 0-3
	// Player 0's equity should be < 0.5
	equity = table.GetMEAfterResult(0, 0, 5, 0, 3, 1, false)
	if equity >= 0.5 {
		t.Errorf("Expected equity < 0.5 after opponent wins backgammon, got %f", equity)
	}
}

func TestSymmetry(t *testing.T) {
	table := Default()

	// Equity should be symmetric: P0 at score (a,b) = 1 - P1 at score (a,b)
	for i := 0; i < 10; i++ {
		for j := 0; j < 10; j++ {
			eq0 := table.GetME(i, j, 11, 0, false)
			eq1 := table.GetME(i, j, 11, 1, false)
			sum := eq0 + eq1
			if sum < 0.99 || sum > 1.01 {
				t.Errorf("Symmetry violation at (%d,%d): %f + %f = %f", i, j, eq0, eq1, sum)
			}
		}
	}
}

