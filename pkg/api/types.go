// Package api provides HTTP/JSON REST API for the backgammon engine.
package api

import "github.com/yourusername/bgengine/pkg/engine"

// ============================================================================
// Request Types
// ============================================================================

// EvaluateRequest is the request body for position evaluation.
type EvaluateRequest struct {
	Position    string `json:"position"`               // Position ID (gnubg format)
	MatchLength int    `json:"match_length,omitempty"` // 0 = money game
	Score       [2]int `json:"score,omitempty"`        // Match score [player, opponent]
	CubeValue   int    `json:"cube_value,omitempty"`   // Cube value (default 1)
	CubeOwner   int    `json:"cube_owner,omitempty"`   // -1=centered, 0=player, 1=opponent
	Crawford    bool   `json:"crawford,omitempty"`     // Crawford game
	Ply         int    `json:"ply,omitempty"`          // Evaluation depth (0, 1, or 2)
}

// MoveRequest is the request body for finding best moves.
type MoveRequest struct {
	Position    string `json:"position"`               // Position ID (gnubg format)
	Dice        [2]int `json:"dice"`                   // Dice roll [die1, die2]
	MatchLength int    `json:"match_length,omitempty"` // 0 = money game
	Score       [2]int `json:"score,omitempty"`        // Match score
	CubeValue   int    `json:"cube_value,omitempty"`   // Cube value
	CubeOwner   int    `json:"cube_owner,omitempty"`   // Cube owner
	Crawford    bool   `json:"crawford,omitempty"`     // Crawford game
	NumMoves    int    `json:"num_moves,omitempty"`    // Max moves to return (default 5)
	Ply         int    `json:"ply,omitempty"`          // Evaluation depth
}

// CubeRequest is the request body for cube decision analysis.
type CubeRequest struct {
	Position    string `json:"position"`               // Position ID
	MatchLength int    `json:"match_length,omitempty"` // 0 = money game
	Score       [2]int `json:"score,omitempty"`        // Match score
	CubeValue   int    `json:"cube_value,omitempty"`   // Current cube value
	CubeOwner   int    `json:"cube_owner,omitempty"`   // Current cube owner
	Crawford    bool   `json:"crawford,omitempty"`     // Crawford game
}

// RolloutRequest is the request body for Monte Carlo rollouts.
type RolloutRequest struct {
	Position    string `json:"position"`               // Position ID
	Trials      int    `json:"trials,omitempty"`       // Number of trials (default 1296)
	Truncate    int    `json:"truncate,omitempty"`     // Truncate at N plies (0 = full)
	MatchLength int    `json:"match_length,omitempty"` // 0 = money game
	Score       [2]int `json:"score,omitempty"`        // Match score
	CubeValue   int    `json:"cube_value,omitempty"`   // Cube value
	CubeOwner   int    `json:"cube_owner,omitempty"`   // Cube owner
	Crawford    bool   `json:"crawford,omitempty"`     // Crawford game
	Seed        int64  `json:"seed,omitempty"`         // Random seed (0 = random)
}

// TutorMoveRequest is the request for analyzing a played move.
type TutorMoveRequest struct {
	Position    string `json:"position"`               // Position ID before the move
	Dice        [2]int `json:"dice"`                   // Dice rolled
	Move        string `json:"move"`                   // Move played (e.g., "8/5 6/5")
	MatchLength int    `json:"match_length,omitempty"` // 0 = money game
	Score       [2]int `json:"score,omitempty"`        // Match score
	CubeValue   int    `json:"cube_value,omitempty"`   // Cube value
	CubeOwner   int    `json:"cube_owner,omitempty"`   // Cube owner
	Crawford    bool   `json:"crawford,omitempty"`     // Crawford game
	Ply         int    `json:"ply,omitempty"`          // Evaluation depth
}

// TutorCubeRequest is the request for analyzing a cube decision.
type TutorCubeRequest struct {
	Position    string `json:"position"`               // Position ID
	Action      string `json:"action"`                 // "double", "take", "pass", "no_double"
	MatchLength int    `json:"match_length,omitempty"` // 0 = money game
	Score       [2]int `json:"score,omitempty"`        // Match score
	CubeValue   int    `json:"cube_value,omitempty"`   // Current cube value
	CubeOwner   int    `json:"cube_owner,omitempty"`   // Current cube owner
	Crawford    bool   `json:"crawford,omitempty"`     // Crawford game
}

// AnalyzeGameRequest is the request for analyzing a complete game.
type AnalyzeGameRequest struct {
	Positions []GamePosition `json:"positions"`            // List of positions with actions
	MatchPlay bool           `json:"match_play,omitempty"` // True for match, false for money
}

// GamePosition represents a single position in a game to analyze.
type GamePosition struct {
	Position    string `json:"position"`               // Position ID
	Dice        [2]int `json:"dice,omitempty"`         // Dice if checker play
	Move        string `json:"move,omitempty"`         // Move played (if checker play)
	CubeAction  string `json:"cube_action,omitempty"`  // Cube action taken
	Player      int    `json:"player"`                 // 0 or 1
	MatchLength int    `json:"match_length,omitempty"` // Match length
	Score       [2]int `json:"score,omitempty"`        // Score at this position
	CubeValue   int    `json:"cube_value,omitempty"`   // Cube value
	CubeOwner   int    `json:"cube_owner,omitempty"`   // Cube owner
	Crawford    bool   `json:"crawford,omitempty"`     // Crawford game
}

// ============================================================================
// Response Types
// ============================================================================

// EvaluateResponse is the response for position evaluation.
type EvaluateResponse struct {
	Equity  float64 `json:"equity"`  // Expected value
	Win     float64 `json:"win"`     // P(win) as percentage
	WinG    float64 `json:"win_g"`   // P(win gammon) as percentage
	WinBG   float64 `json:"win_bg"`  // P(win backgammon) as percentage
	LoseG   float64 `json:"lose_g"`  // P(lose gammon) as percentage
	LoseBG  float64 `json:"lose_bg"` // P(lose backgammon) as percentage
	Ply     int     `json:"ply"`     // Ply used for evaluation
	Cubeful bool    `json:"cubeful"` // Whether cubeful evaluation was used
}

// MoveResponse is a single move in the response.
type MoveResponse struct {
	Move   string  `json:"move"`   // Human-readable move notation (e.g., "8/5 6/5")
	Equity float64 `json:"equity"` // Expected value after this move
	Win    float64 `json:"win"`    // P(win) as percentage
	WinG   float64 `json:"win_g"`  // P(win gammon) as percentage
}

// MovesResponse is the response for best moves.
type MovesResponse struct {
	Moves    []MoveResponse `json:"moves"`     // Ranked moves (best first)
	NumLegal int            `json:"num_legal"` // Total number of legal moves
	Dice     [2]int         `json:"dice"`      // Dice used
	Position string         `json:"position"`  // Position evaluated
}

// CubeResponse is the response for cube decisions.
type CubeResponse struct {
	Action         string  `json:"action"`           // "no_double", "double_take", "double_pass", "too_good"
	DoubleEquity   float64 `json:"double_equity"`    // Equity if doubled
	NoDoubleEquity float64 `json:"no_double_equity"` // Equity if not doubled
	TakeEquity     float64 `json:"take_equity"`      // Opponent's equity if they take
	DoubleDiff     float64 `json:"double_diff"`      // Difference (double - no double)
}

// RolloutResponse is the response for rollouts.
type RolloutResponse struct {
	Equity      float64 `json:"equity"`       // Mean equity
	StdDev      float64 `json:"std_dev"`      // Standard deviation
	CI95        float64 `json:"ci_95"`        // 95% confidence interval (+/-)
	Win         float64 `json:"win"`          // P(win) as percentage
	WinG        float64 `json:"win_g"`        // P(win gammon) as percentage
	WinBG       float64 `json:"win_bg"`       // P(win backgammon) as percentage
	LoseG       float64 `json:"lose_g"`       // P(lose gammon) as percentage
	LoseBG      float64 `json:"lose_bg"`      // P(lose backgammon) as percentage
	Trials      int     `json:"trials"`       // Number of trials completed
	Truncated   bool    `json:"truncated"`    // Whether games were truncated
	TruncatePly int     `json:"truncate_ply"` // Ply at which truncation occurred
}

// ErrorResponse is returned when an error occurs.
type ErrorResponse struct {
	Error   string `json:"error"`             // Error message
	Code    string `json:"code,omitempty"`    // Error code
	Details string `json:"details,omitempty"` // Additional details
}

// HealthResponse is the response for health check.
type HealthResponse struct {
	Status  string     `json:"status"`         // "ok" or "error"
	Version string     `json:"version"`        // Engine version
	Ready   bool       `json:"ready"`          // Whether engine is fully loaded
	Pool    *PoolStats `json:"pool,omitempty"` // Worker pool statistics
}

// TutorMoveResponse is the response for move skill analysis.
type TutorMoveResponse struct {
	Skill        string         `json:"skill"`         // "none", "doubtful", "bad", "very_bad"
	SkillAbbr    string         `json:"skill_abbr"`    // "", "?!", "?", "??"
	EquityLoss   float64        `json:"equity_loss"`   // Equity lost by this move
	BestMove     string         `json:"best_move"`     // Best move notation
	BestEquity   float64        `json:"best_equity"`   // Equity of best move
	PlayedEquity float64        `json:"played_equity"` // Equity of played move
	IsForced     bool           `json:"is_forced"`     // True if only one legal move
	TopMoves     []MoveResponse `json:"top_moves"`     // Top 5 moves for context
	Suggestion   string         `json:"suggestion"`    // Improvement suggestion
}

// TutorCubeResponse is the response for cube decision skill analysis.
type TutorCubeResponse struct {
	Skill      string  `json:"skill"`       // "none", "doubtful", "bad", "very_bad"
	SkillAbbr  string  `json:"skill_abbr"`  // "", "?!", "?", "??"
	EquityLoss float64 `json:"equity_loss"` // Equity lost by this decision
	Optimal    string  `json:"optimal"`     // Optimal action
	Played     string  `json:"played"`      // Played action
	IsClose    bool    `json:"is_close"`    // True if decision was close
	Suggestion string  `json:"suggestion"`  // Improvement suggestion
}

// GameAnalysisResponse is the response for complete game analysis.
type GameAnalysisResponse struct {
	Players     [2]PlayerStats `json:"players"`     // Stats for each player
	TotalMoves  int            `json:"total_moves"` // Total moves analyzed
	MoveErrors  []MoveError    `json:"move_errors"` // All errors found
	CubeErrors  []CubeError    `json:"cube_errors"` // All cube errors
	LuckStats   [2]float64     `json:"luck_stats"`  // Luck for each player
	Suggestions []string       `json:"suggestions"` // Overall improvement suggestions
}

// PlayerStats contains analysis stats for one player.
type PlayerStats struct {
	TotalMoves         int     `json:"total_moves"`    // Number of unforced moves
	TotalCubeDecisions int     `json:"total_cube"`     // Number of cube decisions
	TotalError         float64 `json:"total_error"`    // Sum of equity lost
	ErrorPerMove       float64 `json:"error_per_move"` // Average equity lost per move
	Rating             string  `json:"rating"`         // Skill rating
	Blunders           int     `json:"blunders"`       // Very bad moves
	Errors             int     `json:"errors"`         // Bad moves
	Doubtful           int     `json:"doubtful"`       // Doubtful moves
	LuckAdjusted       float64 `json:"luck_adjusted"`  // Luck-adjusted error rate
}

// MoveError represents a single move error in a game.
type MoveError struct {
	MoveNumber int     `json:"move_number"` // 1-indexed move number
	Player     int     `json:"player"`      // 0 or 1
	Position   string  `json:"position"`    // Position ID
	Dice       [2]int  `json:"dice"`        // Dice rolled
	Played     string  `json:"played"`      // Move played
	Best       string  `json:"best"`        // Best move
	EquityLoss float64 `json:"equity_loss"` // Equity lost
	Skill      string  `json:"skill"`       // Skill rating
}

// CubeError represents a single cube error in a game.
type CubeError struct {
	MoveNumber int     `json:"move_number"` // 1-indexed move number
	Player     int     `json:"player"`      // 0 or 1
	Position   string  `json:"position"`    // Position ID
	Played     string  `json:"played"`      // Action taken
	Optimal    string  `json:"optimal"`     // Correct action
	EquityLoss float64 `json:"equity_loss"` // Equity lost
	Skill      string  `json:"skill"`       // Skill rating
}

// ============================================================================
// Helper Functions
// ============================================================================

// EvalToResponse converts an engine Evaluation to an API response.
func EvalToResponse(eval *engine.Evaluation, ply int, cubeful bool) *EvaluateResponse {
	return &EvaluateResponse{
		Equity:  eval.Equity,
		Win:     eval.WinProb * 100,
		WinG:    eval.WinG * 100,
		WinBG:   eval.WinBG * 100,
		LoseG:   eval.LoseG * 100,
		LoseBG:  eval.LoseBG * 100,
		Ply:     ply,
		Cubeful: cubeful,
	}
}
