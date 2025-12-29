package engine

import (
	"testing"
)

func TestClassifySkill(t *testing.T) {
	tests := []struct {
		loss float64
		want SkillType
	}{
		{0.0, SkillNone},
		{0.02, SkillNone},
		{0.03, SkillDoubtful},
		{0.05, SkillDoubtful},
		{0.06, SkillBad},
		{0.10, SkillBad},
		{0.12, SkillVeryBad},
		{0.20, SkillVeryBad},
	}

	for _, tc := range tests {
		got := ClassifySkill(tc.loss)
		if got != tc.want {
			t.Errorf("ClassifySkill(%f) = %v, want %v", tc.loss, got, tc.want)
		}
	}
}

func TestClassifyLuck(t *testing.T) {
	tests := []struct {
		swing float64
		want  LuckType
	}{
		{0.0, LuckNone},
		{0.2, LuckNone},
		{0.31, LuckGood},
		{0.5, LuckGood},
		{0.61, LuckVeryGood},
		{-0.2, LuckNone},
		{-0.31, LuckBad},
		{-0.5, LuckBad},
		{-0.61, LuckVeryBad},
	}

	for _, tc := range tests {
		got := ClassifyLuck(tc.swing)
		if got != tc.want {
			t.Errorf("ClassifyLuck(%f) = %v, want %v", tc.swing, got, tc.want)
		}
	}
}

func TestGetRating(t *testing.T) {
	tests := []struct {
		epm  float64
		want RatingType
	}{
		{0.001, RatingSupernatural},
		{0.003, RatingWorldClass},
		{0.006, RatingExpert},
		{0.009, RatingAdvanced},
		{0.015, RatingIntermediate},
		{0.020, RatingCasualPlayer},
		{0.030, RatingBeginner},
		{0.040, RatingAwful},
	}

	for _, tc := range tests {
		got := GetRating(tc.epm)
		if got != tc.want {
			t.Errorf("GetRating(%f) = %v, want %v", tc.epm, got, tc.want)
		}
	}
}

func TestSkillTypeString(t *testing.T) {
	if SkillVeryBad.String() != "Very Bad" {
		t.Errorf("SkillVeryBad.String() = %q, want %q", SkillVeryBad.String(), "Very Bad")
	}
	if SkillBad.String() != "Bad" {
		t.Errorf("SkillBad.String() = %q, want %q", SkillBad.String(), "Bad")
	}
	if SkillDoubtful.String() != "Doubtful" {
		t.Errorf("SkillDoubtful.String() = %q, want %q", SkillDoubtful.String(), "Doubtful")
	}
	if SkillNone.String() != "None" {
		t.Errorf("SkillNone.String() = %q, want %q", SkillNone.String(), "None")
	}
}

func TestSkillTypeAbbr(t *testing.T) {
	if SkillVeryBad.Abbr() != "??" {
		t.Errorf("SkillVeryBad.Abbr() = %q, want %q", SkillVeryBad.Abbr(), "??")
	}
	if SkillBad.Abbr() != "?" {
		t.Errorf("SkillBad.Abbr() = %q, want %q", SkillBad.Abbr(), "?")
	}
	if SkillDoubtful.Abbr() != "?!" {
		t.Errorf("SkillDoubtful.Abbr() = %q, want %q", SkillDoubtful.Abbr(), "?!")
	}
	if SkillNone.Abbr() != "" {
		t.Errorf("SkillNone.Abbr() = %q, want %q", SkillNone.Abbr(), "")
	}
}

func TestAnalyzeMoveSkill(t *testing.T) {
	engine, err := NewEngine(EngineOptions{
		WeightsFileText: "../../data/gnubg.weights",
		BearoffFile:     "../../data/gnubg_os0.bd",
	})
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	state := StartingPosition()
	dice := [2]int{3, 1}

	// Get the best move
	bestMove, _, err := engine.BestMove(state, dice)
	if err != nil {
		t.Fatalf("BestMove failed: %v", err)
	}

	// Analyze playing the best move - should be SkillNone
	analysis, err := engine.AnalyzeMoveSkill(state, bestMove, dice)
	if err != nil {
		t.Fatalf("AnalyzeMoveSkill failed: %v", err)
	}

	if analysis.Skill != SkillNone {
		t.Errorf("Playing best move: Skill = %v, want SkillNone", analysis.Skill)
	}
	if analysis.EquityLoss > 0.001 {
		t.Errorf("Playing best move: EquityLoss = %f, want ~0", analysis.EquityLoss)
	}
	if analysis.IsForced {
		t.Errorf("Opening 3-1 should not be forced")
	}

	// Analyze playing a suboptimal move
	// 24/21 24/23 is a weak play with 3-1 (using 0-indexed points: 23/20 23/22)
	badMove := Move{
		From: [4]int8{23, 23, -1, -1},
		To:   [4]int8{20, 22, -1, -1},
	}
	badAnalysis, err := engine.AnalyzeMoveSkill(state, badMove, dice)
	if err != nil {
		t.Fatalf("AnalyzeMoveSkill for bad move failed: %v", err)
	}

	// The best 3-1 is 8/5 6/5 making the 5-point
	// Running with both checkers from the 24-point is weaker
	t.Logf("Best move: From=%v To=%v Equity=%.4f",
		analysis.BestMove.From, analysis.BestMove.To, analysis.BestEquity)
	t.Logf("Played (24/21 24/23): Equity=%.4f, Loss=%.4f, Skill=%v",
		badAnalysis.Equity, badAnalysis.EquityLoss, badAnalysis.Skill)

	// This move should be weaker than the best move (8/5 6/5)
	if badAnalysis.EquityLoss <= 0 {
		t.Errorf("Bad move should have positive equity loss, got %f", badAnalysis.EquityLoss)
	}
}
