package engine

import (
	"testing"
)

func TestAnalyzePositionList(t *testing.T) {
	engine, err := NewEngine(EngineOptions{
		WeightsFileText: "../../data/gnubg.weights",
		BearoffFile:     "../../data/gnubg_os0.bd",
	})
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	// Create a simple position list with a starting position and move
	state := StartingPosition()
	
	// Best move for 31
	bestMove, _, err := engine.BestMove(state, [2]int{3, 1})
	if err != nil {
		t.Fatalf("BestMove failed: %v", err)
	}
	
	positions := []AnalyzedPosition{
		{
			Board:      state.Board,
			Turn:       0,
			Dice:       [2]int{3, 1},
			CubeValue:  1,
			CubeOwner:  -1,
			Move:       &bestMove,
			GameNumber: 1,
			MoveNumber: 1,
			Player:     0,
		},
	}
	
	opts := DefaultMatchAnalysisOptions()
	opts.Player1Name = "Test Player 1"
	opts.Player2Name = "Test Player 2"
	
	result, err := engine.AnalyzePositionList(positions, opts)
	if err != nil {
		t.Fatalf("AnalyzePositionList failed: %v", err)
	}
	
	if result.TotalGames != 1 {
		t.Errorf("TotalGames = %d, want 1", result.TotalGames)
	}
	
	if result.TotalMoves != 1 {
		t.Errorf("TotalMoves = %d, want 1", result.TotalMoves)
	}
	
	// Playing best move should have no error
	if result.PlayerStats[0].TotalError > 0.001 {
		t.Errorf("TotalError = %f, want ~0 for best move", result.PlayerStats[0].TotalError)
	}
}

func TestAnalyzePositionListWithError(t *testing.T) {
	engine, err := NewEngine(EngineOptions{
		WeightsFileText: "../../data/gnubg.weights",
		BearoffFile:     "../../data/gnubg_os0.bd",
	})
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	state := StartingPosition()
	
	// Create a suboptimal move for 31 (not the standard 8/5 6/5)
	badMove := Move{
		From: [4]int8{23, 23, -1, -1},
		To:   [4]int8{20, 22, -1, -1},
	}
	
	positions := []AnalyzedPosition{
		{
			Board:      state.Board,
			Turn:       0,
			Dice:       [2]int{3, 1},
			CubeValue:  1,
			CubeOwner:  -1,
			Move:       &badMove,
			GameNumber: 1,
			MoveNumber: 1,
			Player:     0,
		},
	}
	
	opts := DefaultMatchAnalysisOptions()
	result, err := engine.AnalyzePositionList(positions, opts)
	if err != nil {
		t.Fatalf("AnalyzePositionList failed: %v", err)
	}
	
	// Playing suboptimal move should have some error
	if result.PlayerStats[0].TotalError < 0.01 {
		t.Errorf("Expected error for suboptimal move, got %f", result.PlayerStats[0].TotalError)
	}
	
	// Should have recorded an error
	if len(result.MoveErrors) == 0 {
		t.Error("Expected at least one move error recorded")
	}
}

func TestFormatMove(t *testing.T) {
	tests := []struct {
		name string
		move Move
		want string
	}{
		{
			name: "simple move",
			move: Move{From: [4]int8{7, 5, -1, -1}, To: [4]int8{4, 4, -1, -1}},
			want: "8/5 6/5",
		},
		{
			name: "bar move",
			move: Move{From: [4]int8{24, -1, -1, -1}, To: [4]int8{20, -1, -1, -1}},
			want: "bar/21",
		},
		{
			name: "empty move",
			move: Move{From: [4]int8{-1, -1, -1, -1}, To: [4]int8{-1, -1, -1, -1}},
			want: "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatMove(tt.move)
			if got != tt.want {
				t.Errorf("FormatMove() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEncodePositionID(t *testing.T) {
	state := StartingPosition()
	posID := EncodePositionID(state.Board)
	
	// Starting position should have a known ID
	if posID != "4HPwATDgc/ABMA" {
		t.Errorf("EncodePositionID() = %q, want 4HPwATDgc/ABMA", posID)
	}
}

