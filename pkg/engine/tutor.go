// Package engine provides tutor mode for error detection and skill ratings.
package engine

import (
	"fmt"
)

// SkillType represents the skill rating of a move or cube decision.
type SkillType int

const (
	SkillVeryBad  SkillType = iota // Blunder: loses >= 0.12 equity
	SkillBad                       // Error: loses 0.06-0.12 equity
	SkillDoubtful                  // Doubtful: loses 0.03-0.06 equity
	SkillNone                      // Good or best move
)

// String returns the display name of the skill type.
func (s SkillType) String() string {
	return [...]string{"Very Bad", "Bad", "Doubtful", "None"}[s]
}

// Abbr returns the abbreviated notation (?, ??, ?!).
func (s SkillType) Abbr() string {
	return [...]string{"??", "?", "?!", ""}[s]
}

// LuckType represents the luck rating of a dice roll.
type LuckType int

const (
	LuckVeryBad  LuckType = iota // Lost >= 0.6 equity from roll
	LuckBad                      // Lost 0.3-0.6 equity
	LuckNone                     // Neutral
	LuckGood                     // Gained 0.3-0.6 equity
	LuckVeryGood                 // Gained >= 0.6 equity
)

// String returns the display name of the luck type.
func (l LuckType) String() string {
	return [...]string{"Very Unlucky", "Unlucky", "None", "Lucky", "Very Lucky"}[l]
}

// SkillThresholds are the equity loss thresholds for skill ratings.
// These match gnubg's default values.
var SkillThresholds = [4]float64{
	0.12, // SKILL_VERYBAD - blunder
	0.06, // SKILL_BAD - error
	0.03, // SKILL_DOUBTFUL - questionable
	0.00, // SKILL_NONE - good
}

// LuckThresholds are the equity swing thresholds for luck ratings.
// These match gnubg's default values.
var LuckThresholds = [5]float64{
	0.6, // LUCK_VERYBAD
	0.3, // LUCK_BAD
	0.0, // LUCK_NONE
	0.3, // LUCK_GOOD
	0.6, // LUCK_VERYGOOD
}

// RatingThresholds are error-per-move thresholds for overall player ratings.
// These match gnubg's arThrsRating values.
// Index corresponds to RatingType: 0=Undefined, 1=Awful, ..., 8=Supernatural
var RatingThresholds = [9]float64{
	1e38,  // Undefined (never used)
	0.035, // Awful (> 0.035)
	0.026, // Beginner (0.026-0.035)
	0.018, // Casual player (0.018-0.026)
	0.012, // Intermediate (0.012-0.018)
	0.008, // Advanced (0.008-0.012)
	0.005, // Expert (0.005-0.008)
	0.002, // World class (0.002-0.005)
	0.0,   // Supernatural (< 0.002)
}

// RatingType represents overall player rating level.
type RatingType int

const (
	RatingUndefined    RatingType = iota
	RatingAwful                   // > 0.035 EPM
	RatingBeginner                // 0.026-0.035 EPM
	RatingCasualPlayer            // 0.018-0.026 EPM
	RatingIntermediate            // 0.012-0.018 EPM
	RatingAdvanced                // 0.008-0.012 EPM
	RatingExpert                  // 0.005-0.008 EPM
	RatingWorldClass              // 0.002-0.005 EPM
	RatingSupernatural            // < 0.002 EPM
)

// String returns the display name of the rating.
func (r RatingType) String() string {
	return [...]string{
		"Undefined", "Awful", "Beginner", "Casual Player",
		"Intermediate", "Advanced", "Expert", "World Class", "Supernatural",
	}[r]
}

// ClassifySkill returns the skill rating based on equity loss.
// equityLoss should be positive for moves worse than best.
func ClassifySkill(equityLoss float64) SkillType {
	if equityLoss >= SkillThresholds[0] {
		return SkillVeryBad
	} else if equityLoss >= SkillThresholds[1] {
		return SkillBad
	} else if equityLoss >= SkillThresholds[2] {
		return SkillDoubtful
	}
	return SkillNone
}

// ClassifyLuck returns the luck rating based on equity swing.
// Positive values = good luck, negative = bad luck.
func ClassifyLuck(equitySwing float64) LuckType {
	if equitySwing > LuckThresholds[4] {
		return LuckVeryGood
	} else if equitySwing > LuckThresholds[3] {
		return LuckGood
	} else if equitySwing < -LuckThresholds[0] {
		return LuckVeryBad
	} else if equitySwing < -LuckThresholds[1] {
		return LuckBad
	}
	return LuckNone
}

// GetRating returns the player rating based on error per move.
// Lower EPM = better rating. Thresholds are upper bounds for each rating.
func GetRating(errorPerMove float64) RatingType {
	// Check from best (Supernatural) to worst (Awful)
	// Each threshold is the upper bound for that rating
	if errorPerMove < 0.002 {
		return RatingSupernatural
	} else if errorPerMove < 0.005 {
		return RatingWorldClass
	} else if errorPerMove < 0.008 {
		return RatingExpert
	} else if errorPerMove < 0.012 {
		return RatingAdvanced
	} else if errorPerMove < 0.018 {
		return RatingIntermediate
	} else if errorPerMove < 0.026 {
		return RatingCasualPlayer
	} else if errorPerMove < 0.035 {
		return RatingBeginner
	}
	return RatingAwful
}

// MoveSkillAnalysis contains the detailed analysis of a single move for tutoring.
type MoveSkillAnalysis struct {
	Move       Move           // The move that was played
	BestMove   Move           // The best move according to analysis
	Equity     float64        // Equity of the played move
	BestEquity float64        // Equity of the best move
	EquityLoss float64        // Best equity - played equity (positive = error)
	Skill      SkillType      // Skill rating
	IsForced   bool           // True if only one legal move
	TopMoves   []MoveWithEval // Top N moves for context
}

// CubeSkillAnalysis contains the detailed analysis of a cube decision for tutoring.
type CubeSkillAnalysis struct {
	Analysis    *CubeAnalysis // Full cube analysis from AnalyzeCube
	OptimalPlay CubeAction    // What should have been done
	ActualPlay  CubeAction    // What the player did
	EquityLoss  float64       // Cost of the error (if any)
	Skill       SkillType     // Skill rating
	IsClose     bool          // True if the decision was close
}

// AnalyzeMoveSkill evaluates a played move and returns skill analysis.
// playedMove is the move the player made, dice is the roll.
func (e *Engine) AnalyzeMoveSkill(state *GameState, playedMove Move, dice [2]int) (*MoveSkillAnalysis, error) {
	// Use AnalyzePosition which generates and evaluates all moves
	analysisResult, err := e.AnalyzePosition(state, dice)
	if err != nil {
		return nil, fmt.Errorf("analyzing position: %w", err)
	}

	analysis := &MoveSkillAnalysis{
		Move:     playedMove,
		IsForced: analysisResult.NumMoves <= 1,
	}

	if analysisResult.NumMoves == 0 {
		// No legal moves (dancer)
		analysis.Skill = SkillNone
		return analysis, nil
	}

	if analysisResult.NumMoves == 1 {
		// Forced move
		analysis.BestMove = analysisResult.Moves[0].Move
		analysis.Skill = SkillNone
		return analysis, nil
	}

	// Set best move info
	analysis.BestMove = analysisResult.BestMove
	analysis.BestEquity = analysisResult.BestEquity

	// Store top moves for context (up to 5)
	maxTop := 5
	if len(analysisResult.Moves) < maxTop {
		maxTop = len(analysisResult.Moves)
	}
	analysis.TopMoves = analysisResult.Moves[:maxTop]

	// Find the played move by comparing resulting positions
	playedResult := ApplyMove(state.Board, playedMove)
	foundMove := false

	for _, m := range analysisResult.Moves {
		resultBoard := ApplyMove(state.Board, m.Move)
		if EqualBoards(resultBoard, playedResult) {
			analysis.Equity = m.Equity
			foundMove = true
			break
		}
	}

	if !foundMove {
		// Move not found - this shouldn't happen for legal moves
		// Default to worst possible to flag the issue
		analysis.Equity = analysisResult.Moves[len(analysisResult.Moves)-1].Equity
	}

	// Calculate equity loss
	analysis.EquityLoss = analysis.BestEquity - analysis.Equity
	analysis.Skill = ClassifySkill(analysis.EquityLoss)

	return analysis, nil
}

// AnalyzeCubeSkill evaluates a cube decision and returns skill analysis.
// actualAction is what the player did (Double, Take, Pass, NoDouble).
func (e *Engine) AnalyzeCubeSkill(state *GameState, actualAction CubeAction) (*CubeSkillAnalysis, error) {
	cubeAnalysis, err := e.AnalyzeCube(state)
	if err != nil {
		return nil, fmt.Errorf("analyzing cube: %w", err)
	}

	analysis := &CubeSkillAnalysis{
		Analysis:   cubeAnalysis,
		ActualPlay: actualAction,
	}

	// Determine optimal play from the decision
	analysis.OptimalPlay = cubeAnalysis.Decision.Action

	// Check if this is a close decision
	analysis.IsClose = isCloseCubeDecisionAnalysis(cubeAnalysis)

	// Calculate equity loss based on what happened
	switch actualAction {
	case NoDouble:
		if cubeAnalysis.Decision.Action == Double {
			// Missed double
			analysis.EquityLoss = cubeAnalysis.DoubleTakeEq - cubeAnalysis.NoDoubleEquity
			if analysis.EquityLoss < 0 {
				// If opponent would pass
				analysis.EquityLoss = cubeAnalysis.DoublePassEq - cubeAnalysis.NoDoubleEquity
			}
		}
	case Double:
		if cubeAnalysis.Decision.Action == NoDouble {
			// Wrong double
			analysis.EquityLoss = cubeAnalysis.NoDoubleEquity - cubeAnalysis.DoubleTakeEq
		}
	case Take:
		if cubeAnalysis.DecisionType == DOUBLE_PASS || cubeAnalysis.DecisionType == REDOUBLE_PASS {
			// Wrong take (should have passed)
			analysis.EquityLoss = 1.0 + cubeAnalysis.DoubleTakeEq // Lost 1 instead of equity from taking
		}
	case Pass:
		if cubeAnalysis.DecisionType == DOUBLE_TAKE || cubeAnalysis.DecisionType == REDOUBLE_TAKE {
			// Wrong pass (should have taken)
			analysis.EquityLoss = 1.0 + cubeAnalysis.DoubleTakeEq // Gave up take equity for -1
		}
	}

	if analysis.EquityLoss < 0 {
		analysis.EquityLoss = 0
	}
	analysis.Skill = ClassifySkill(analysis.EquityLoss)

	return analysis, nil
}

// isCloseCubeDecisionAnalysis returns true if the cube decision is close.
// A decision is close if the difference between doubling and not doubling
// is less than 0.16 equity.
func isCloseCubeDecisionAnalysis(ca *CubeAnalysis) bool {
	const threshold = 0.16
	doubleEq := ca.DoubleTakeEq
	if ca.DoublePassEq > doubleEq {
		doubleEq = ca.DoublePassEq
	}
	return doubleEq-ca.NoDoubleEquity < threshold
}
