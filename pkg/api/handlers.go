package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/yourusername/bgengine/internal/positionid"
	"github.com/yourusername/bgengine/pkg/engine"
)

// Handlers holds the HTTP handlers and engine reference.
type Handlers struct {
	engine  *engine.Engine
	version string
}

// NewHandlers creates a new Handlers instance.
func NewHandlers(e *engine.Engine, version string) *Handlers {
	return &Handlers{
		engine:  e,
		version: version,
	}
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// writeError writes an error response.
func writeError(w http.ResponseWriter, status int, msg string, code string) {
	writeJSON(w, status, ErrorResponse{
		Error: msg,
		Code:  code,
	})
}

// parseGameState creates a GameState from request parameters.
func parseGameState(posID string, req interface{}) (*engine.GameState, error) {
	board, err := positionid.BoardFromPositionID(posID)
	if err != nil {
		return nil, fmt.Errorf("invalid position ID: %w", err)
	}

	gs := &engine.GameState{
		Board:     engine.Board(board),
		Turn:      0,
		CubeValue: 1,
		CubeOwner: -1,
	}

	// Apply optional parameters based on request type
	switch r := req.(type) {
	case *EvaluateRequest:
		gs.MatchLength = r.MatchLength
		gs.Score = r.Score
		if r.CubeValue > 0 {
			gs.CubeValue = r.CubeValue
		}
		gs.CubeOwner = r.CubeOwner
		gs.Crawford = r.Crawford
	case *MoveRequest:
		gs.MatchLength = r.MatchLength
		gs.Score = r.Score
		if r.CubeValue > 0 {
			gs.CubeValue = r.CubeValue
		}
		gs.CubeOwner = r.CubeOwner
		gs.Crawford = r.Crawford
		gs.Dice = r.Dice
	case *CubeRequest:
		gs.MatchLength = r.MatchLength
		gs.Score = r.Score
		if r.CubeValue > 0 {
			gs.CubeValue = r.CubeValue
		}
		gs.CubeOwner = r.CubeOwner
		gs.Crawford = r.Crawford
	case *RolloutRequest:
		gs.MatchLength = r.MatchLength
		gs.Score = r.Score
		if r.CubeValue > 0 {
			gs.CubeValue = r.CubeValue
		}
		gs.CubeOwner = r.CubeOwner
		gs.Crawford = r.Crawford
	}

	return gs, nil
}

// Health handles GET /api/health
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, HealthResponse{
		Status:  "ok",
		Version: h.version,
		Ready:   h.engine != nil,
	})
}

// Evaluate handles POST /api/evaluate
func (h *Handlers) Evaluate(w http.ResponseWriter, r *http.Request) {
	var req EvaluateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON", "INVALID_JSON")
		return
	}

	if req.Position == "" {
		writeError(w, http.StatusBadRequest, "position is required", "MISSING_POSITION")
		return
	}

	gs, err := parseGameState(req.Position, &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "INVALID_POSITION")
		return
	}

	eval, err := h.engine.Evaluate(gs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "EVAL_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, EvalToResponse(eval, req.Ply, false))
}

// formatMove converts a Move to human-readable notation.
func formatMove(m engine.Move) string {
	result := ""
	for i := 0; i < 4; i++ {
		if m.From[i] < 0 {
			break
		}
		if i > 0 {
			result += " "
		}
		from := int(m.From[i]) + 1
		if m.From[i] == 24 {
			result += "bar"
		} else {
			result += fmt.Sprintf("%d", from)
		}
		result += "/"
		if m.To[i] < 0 {
			result += "off"
		} else {
			result += fmt.Sprintf("%d", int(m.To[i])+1)
		}
	}
	return result
}

// Move handles POST /api/move
func (h *Handlers) Move(w http.ResponseWriter, r *http.Request) {
	var req MoveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON", "INVALID_JSON")
		return
	}

	if req.Position == "" {
		writeError(w, http.StatusBadRequest, "position is required", "MISSING_POSITION")
		return
	}

	if req.Dice[0] < 1 || req.Dice[0] > 6 || req.Dice[1] < 1 || req.Dice[1] > 6 {
		writeError(w, http.StatusBadRequest, "dice must be 1-6", "INVALID_DICE")
		return
	}

	gs, err := parseGameState(req.Position, &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "INVALID_POSITION")
		return
	}

	analysis, err := h.engine.AnalyzePosition(gs, req.Dice)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "ANALYSIS_ERROR")
		return
	}

	numMoves := req.NumMoves
	if numMoves <= 0 {
		numMoves = 5
	}
	if numMoves > len(analysis.Moves) {
		numMoves = len(analysis.Moves)
	}

	moves := make([]MoveResponse, numMoves)
	for i := 0; i < numMoves; i++ {
		m := analysis.Moves[i]
		moves[i] = MoveResponse{
			Move:   formatMove(m.Move),
			Equity: m.Equity,
			Win:    m.Eval.WinProb * 100,
			WinG:   m.Eval.WinG * 100,
		}
	}

	writeJSON(w, http.StatusOK, MovesResponse{
		Moves:    moves,
		NumLegal: analysis.NumMoves,
		Dice:     req.Dice,
		Position: req.Position,
	})
}

// Cube handles POST /api/cube
func (h *Handlers) Cube(w http.ResponseWriter, r *http.Request) {
	var req CubeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON", "INVALID_JSON")
		return
	}

	if req.Position == "" {
		writeError(w, http.StatusBadRequest, "position is required", "MISSING_POSITION")
		return
	}

	gs, err := parseGameState(req.Position, &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "INVALID_POSITION")
		return
	}

	decision, err := h.engine.AnalyzeCube(gs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "CUBE_ERROR")
		return
	}

	action := "no_double"
	diff := decision.DoubleTakeEq - decision.NoDoubleEquity
	if diff > 0 {
		if decision.DoublePassEq > decision.DoubleTakeEq {
			action = "double_pass"
		} else {
			action = "double_take"
		}
	}

	writeJSON(w, http.StatusOK, CubeResponse{
		Action:         action,
		DoubleEquity:   decision.DoubleTakeEq,
		NoDoubleEquity: decision.NoDoubleEquity,
		TakeEquity:     decision.DoubleTakeEq, // From opponent's perspective this is their take equity
		DoubleDiff:     diff,
	})
}

// Rollout handles POST /api/rollout
func (h *Handlers) Rollout(w http.ResponseWriter, r *http.Request) {
	var req RolloutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON", "INVALID_JSON")
		return
	}

	if req.Position == "" {
		writeError(w, http.StatusBadRequest, "position is required", "MISSING_POSITION")
		return
	}

	gs, err := parseGameState(req.Position, &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "INVALID_POSITION")
		return
	}

	trials := req.Trials
	if trials <= 0 {
		trials = 1296
	}

	opts := engine.RolloutOptions{
		Trials:   trials,
		Truncate: req.Truncate,
		Seed:     req.Seed,
	}

	result, err := h.engine.Rollout(gs, opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "ROLLOUT_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, RolloutResponse{
		Equity:      result.Equity,
		StdDev:      result.EquityStdDev,
		CI95:        result.EquityCI,
		Win:         result.WinProb * 100,
		WinG:        result.WinG * 100,
		WinBG:       result.WinBG * 100,
		LoseG:       result.LoseG * 100,
		LoseBG:      result.LoseBG * 100,
		Trials:      result.TrialsCompleted,
		Truncated:   req.Truncate > 0,
		TruncatePly: req.Truncate,
	})
}
