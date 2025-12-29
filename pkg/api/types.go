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

// ============================================================================
// Response Types
// ============================================================================

// EvaluateResponse is the response for position evaluation.
type EvaluateResponse struct {
	Equity  float64 `json:"equity"`   // Expected value
	Win     float64 `json:"win"`      // P(win) as percentage
	WinG    float64 `json:"win_g"`    // P(win gammon) as percentage
	WinBG   float64 `json:"win_bg"`   // P(win backgammon) as percentage
	LoseG   float64 `json:"lose_g"`   // P(lose gammon) as percentage
	LoseBG  float64 `json:"lose_bg"`  // P(lose backgammon) as percentage
	Ply     int     `json:"ply"`      // Ply used for evaluation
	Cubeful bool    `json:"cubeful"`  // Whether cubeful evaluation was used
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
	Moves     []MoveResponse `json:"moves"`      // Ranked moves (best first)
	NumLegal  int            `json:"num_legal"`  // Total number of legal moves
	Dice      [2]int         `json:"dice"`       // Dice used
	Position  string         `json:"position"`   // Position evaluated
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
	Status  string `json:"status"`  // "ok" or "error"
	Version string `json:"version"` // Engine version
	Ready   bool   `json:"ready"`   // Whether engine is fully loaded
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

