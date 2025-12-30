package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/yourusername/bgengine/internal/positionid"
	"github.com/yourusername/bgengine/pkg/engine"
)

// Handlers holds the HTTP handlers and engine reference.
type Handlers struct {
	engine  *engine.Engine
	version string
	pool    *WorkerPool
}

// NewHandlers creates a new Handlers instance without a worker pool.
func NewHandlers(e *engine.Engine, version string) *Handlers {
	return &Handlers{
		engine:  e,
		version: version,
		pool:    nil,
	}
}

// NewHandlersWithPool creates a new Handlers instance with a worker pool.
func NewHandlersWithPool(e *engine.Engine, version string, pool *WorkerPool) *Handlers {
	return &Handlers{
		engine:  e,
		version: version,
		pool:    pool,
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
	resp := HealthResponse{
		Status:  "ok",
		Version: h.version,
		Ready:   h.engine != nil,
	}

	// Include pool stats if available
	if h.pool != nil {
		stats := h.pool.Stats()
		resp.Pool = &stats
	}

	writeJSON(w, http.StatusOK, resp)
}

// Evaluate handles POST /api/evaluate
func (h *Handlers) Evaluate(w http.ResponseWriter, r *http.Request) {
	// Acquire fast worker slot if pool is configured
	if h.pool != nil {
		if err := h.pool.AcquireFast(r.Context()); err != nil {
			writeError(w, http.StatusServiceUnavailable, "server busy", "SERVER_BUSY")
			return
		}
		defer h.pool.ReleaseFast()
	}

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
	// Acquire fast worker slot if pool is configured
	if h.pool != nil {
		if err := h.pool.AcquireFast(r.Context()); err != nil {
			writeError(w, http.StatusServiceUnavailable, "server busy", "SERVER_BUSY")
			return
		}
		defer h.pool.ReleaseFast()
	}

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
	// Acquire fast worker slot if pool is configured
	if h.pool != nil {
		if err := h.pool.AcquireFast(r.Context()); err != nil {
			writeError(w, http.StatusServiceUnavailable, "server busy", "SERVER_BUSY")
			return
		}
		defer h.pool.ReleaseFast()
	}

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
	// Acquire slow worker slot if pool is configured (rollouts are CPU-intensive)
	if h.pool != nil {
		if err := h.pool.AcquireSlow(r.Context()); err != nil {
			writeError(w, http.StatusServiceUnavailable, "server busy", "SERVER_BUSY")
			return
		}
		defer h.pool.ReleaseSlow()
	}

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

// HandleTutorMove analyzes a played move and returns skill analysis.
func (h *Handlers) HandleTutorMove(w http.ResponseWriter, r *http.Request) {
	var req TutorMoveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON", "INVALID_JSON")
		return
	}

	if req.Position == "" {
		writeError(w, http.StatusBadRequest, "position is required", "MISSING_POSITION")
		return
	}

	if req.Move == "" {
		writeError(w, http.StatusBadRequest, "move is required", "MISSING_MOVE")
		return
	}

	// Parse the game state
	gs, err := parseGameStateFromTutor(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "INVALID_POSITION")
		return
	}

	// Parse the move
	playedMove, err := engine.ParseMove(req.Move)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid move notation: %v", err), "INVALID_MOVE")
		return
	}

	// Analyze the move
	analysis, err := h.engine.AnalyzeMoveSkill(gs, playedMove, req.Dice)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "ANALYSIS_ERROR")
		return
	}

	// Build response
	resp := TutorMoveResponse{
		Skill:        skillToString(analysis.Skill),
		SkillAbbr:    analysis.Skill.Abbr(),
		EquityLoss:   analysis.EquityLoss,
		BestMove:     formatMove(analysis.BestMove),
		BestEquity:   analysis.BestEquity,
		PlayedEquity: analysis.Equity,
		IsForced:     analysis.IsForced,
		Suggestion:   generateMoveSuggestion(analysis),
	}

	// Add top moves
	for _, m := range analysis.TopMoves {
		winProb := 0.0
		winG := 0.0
		if m.Eval != nil {
			winProb = m.Eval.WinProb
			winG = m.Eval.WinG
		}
		resp.TopMoves = append(resp.TopMoves, MoveResponse{
			Move:   formatMove(m.Move),
			Equity: m.Equity,
			Win:    winProb * 100,
			WinG:   winG * 100,
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

// HandleTutorCube analyzes a cube decision and returns skill analysis.
func (h *Handlers) HandleTutorCube(w http.ResponseWriter, r *http.Request) {
	var req TutorCubeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON", "INVALID_JSON")
		return
	}

	if req.Position == "" {
		writeError(w, http.StatusBadRequest, "position is required", "MISSING_POSITION")
		return
	}

	// Parse the cube action
	action, err := parseCubeAction(req.Action)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "INVALID_ACTION")
		return
	}

	// Parse the game state
	gs, err := parseGameStateFromCubeTutor(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "INVALID_POSITION")
		return
	}

	// Analyze the cube decision
	analysis, err := h.engine.AnalyzeCubeSkill(gs, action)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "ANALYSIS_ERROR")
		return
	}

	resp := TutorCubeResponse{
		Skill:      skillToString(analysis.Skill),
		SkillAbbr:  analysis.Skill.Abbr(),
		EquityLoss: analysis.EquityLoss,
		Optimal:    cubeActionToString(analysis.OptimalPlay),
		Played:     cubeActionToString(analysis.ActualPlay),
		IsClose:    analysis.IsClose,
		Suggestion: generateCubeSuggestion(analysis),
	}

	writeJSON(w, http.StatusOK, resp)
}

// HandleAnalyzeGame analyzes a complete game and returns statistics.
func (h *Handlers) HandleAnalyzeGame(w http.ResponseWriter, r *http.Request) {
	var req AnalyzeGameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON", "INVALID_JSON")
		return
	}

	if len(req.Positions) == 0 {
		writeError(w, http.StatusBadRequest, "positions array is required", "MISSING_POSITIONS")
		return
	}

	resp := GameAnalysisResponse{
		TotalMoves:  0,
		MoveErrors:  []MoveError{},
		CubeErrors:  []CubeError{},
		Suggestions: []string{},
	}

	// Analyze each position
	for i, pos := range req.Positions {
		moveNum := i + 1

		// Build game state
		gs, err := parseGameStateFromPosition(pos)
		if err != nil {
			continue // Skip invalid positions
		}

		// Analyze move if present
		if pos.Move != "" && pos.Dice != [2]int{0, 0} {
			playedMove, err := engine.ParseMove(pos.Move)
			if err != nil {
				continue
			}

			analysis, err := h.engine.AnalyzeMoveSkill(gs, playedMove, pos.Dice)
			if err != nil {
				continue
			}

			// Count stats
			if !analysis.IsForced {
				resp.Players[pos.Player].TotalMoves++
				resp.Players[pos.Player].TotalError += analysis.EquityLoss
				resp.TotalMoves++

				switch analysis.Skill {
				case engine.SkillVeryBad:
					resp.Players[pos.Player].Blunders++
				case engine.SkillBad:
					resp.Players[pos.Player].Errors++
				case engine.SkillDoubtful:
					resp.Players[pos.Player].Doubtful++
				}

				// Record errors
				if analysis.Skill != engine.SkillNone {
					resp.MoveErrors = append(resp.MoveErrors, MoveError{
						MoveNumber: moveNum,
						Player:     pos.Player,
						Position:   pos.Position,
						Dice:       pos.Dice,
						Played:     pos.Move,
						Best:       formatMove(analysis.BestMove),
						EquityLoss: analysis.EquityLoss,
						Skill:      skillToString(analysis.Skill),
					})
				}
			}
		}

		// Analyze cube decision if present
		if pos.CubeAction != "" {
			action, err := parseCubeAction(pos.CubeAction)
			if err != nil {
				continue
			}

			analysis, err := h.engine.AnalyzeCubeSkill(gs, action)
			if err != nil {
				continue
			}

			resp.Players[pos.Player].TotalCubeDecisions++
			resp.Players[pos.Player].TotalError += analysis.EquityLoss

			if analysis.Skill != engine.SkillNone {
				resp.CubeErrors = append(resp.CubeErrors, CubeError{
					MoveNumber: moveNum,
					Player:     pos.Player,
					Position:   pos.Position,
					Played:     cubeActionToString(analysis.ActualPlay),
					Optimal:    cubeActionToString(analysis.OptimalPlay),
					EquityLoss: analysis.EquityLoss,
					Skill:      skillToString(analysis.Skill),
				})
			}
		}
	}

	// Calculate error per move and ratings
	for i := 0; i < 2; i++ {
		totalDecisions := resp.Players[i].TotalMoves + resp.Players[i].TotalCubeDecisions
		if totalDecisions > 0 {
			resp.Players[i].ErrorPerMove = resp.Players[i].TotalError / float64(totalDecisions)
		}
		resp.Players[i].Rating = engine.GetRating(resp.Players[i].ErrorPerMove).String()
	}

	// Generate suggestions
	resp.Suggestions = generateGameSuggestions(&resp)

	writeJSON(w, http.StatusOK, resp)
}

// ============================================================================
// Tutor Helper Functions
// ============================================================================

// parseGameStateFromTutor creates a GameState from a TutorMoveRequest.
func parseGameStateFromTutor(req TutorMoveRequest) (*engine.GameState, error) {
	board, err := positionid.BoardFromPositionID(req.Position)
	if err != nil {
		return nil, fmt.Errorf("invalid position ID: %w", err)
	}

	gs := &engine.GameState{
		Board:       engine.Board(board),
		Turn:        0,
		CubeValue:   1,
		CubeOwner:   -1,
		MatchLength: req.MatchLength,
		Score:       req.Score,
		Crawford:    req.Crawford,
		Dice:        req.Dice,
	}

	if req.CubeValue > 0 {
		gs.CubeValue = req.CubeValue
	}
	gs.CubeOwner = req.CubeOwner

	return gs, nil
}

// parseGameStateFromCubeTutor creates a GameState from a TutorCubeRequest.
func parseGameStateFromCubeTutor(req TutorCubeRequest) (*engine.GameState, error) {
	board, err := positionid.BoardFromPositionID(req.Position)
	if err != nil {
		return nil, fmt.Errorf("invalid position ID: %w", err)
	}

	gs := &engine.GameState{
		Board:       engine.Board(board),
		Turn:        0,
		CubeValue:   1,
		CubeOwner:   -1,
		MatchLength: req.MatchLength,
		Score:       req.Score,
		Crawford:    req.Crawford,
	}

	if req.CubeValue > 0 {
		gs.CubeValue = req.CubeValue
	}
	gs.CubeOwner = req.CubeOwner

	return gs, nil
}

// parseGameStateFromPosition creates a GameState from a GamePosition.
func parseGameStateFromPosition(pos GamePosition) (*engine.GameState, error) {
	board, err := positionid.BoardFromPositionID(pos.Position)
	if err != nil {
		return nil, fmt.Errorf("invalid position ID: %w", err)
	}

	gs := &engine.GameState{
		Board:       engine.Board(board),
		Turn:        pos.Player,
		CubeValue:   1,
		CubeOwner:   -1,
		MatchLength: pos.MatchLength,
		Score:       pos.Score,
		Crawford:    pos.Crawford,
		Dice:        pos.Dice,
	}

	if pos.CubeValue > 0 {
		gs.CubeValue = pos.CubeValue
	}
	gs.CubeOwner = pos.CubeOwner

	return gs, nil
}

// parseCubeAction parses a cube action string.
func parseCubeAction(action string) (engine.CubeAction, error) {
	switch strings.ToLower(action) {
	case "double":
		return engine.Double, nil
	case "take":
		return engine.Take, nil
	case "pass":
		return engine.Pass, nil
	case "no_double", "nodouble":
		return engine.NoDouble, nil
	default:
		return engine.NoDouble, fmt.Errorf("invalid cube action: %s", action)
	}
}

// skillToString converts a SkillType to a string.
func skillToString(skill engine.SkillType) string {
	switch skill {
	case engine.SkillVeryBad:
		return "very_bad"
	case engine.SkillBad:
		return "bad"
	case engine.SkillDoubtful:
		return "doubtful"
	default:
		return "none"
	}
}

// cubeActionToString converts a CubeAction to a string.
func cubeActionToString(action engine.CubeAction) string {
	switch action {
	case engine.Double:
		return "double"
	case engine.Take:
		return "take"
	case engine.Pass:
		return "pass"
	case engine.NoDouble:
		return "no_double"
	default:
		return "unknown"
	}
}

// generateMoveSuggestion generates an improvement suggestion for a move error.
func generateMoveSuggestion(analysis *engine.MoveSkillAnalysis) string {
	if analysis.Skill == engine.SkillNone || analysis.IsForced {
		return ""
	}

	switch analysis.Skill {
	case engine.SkillVeryBad:
		return fmt.Sprintf("This was a blunder losing %.3f equity. The best move was %s.",
			analysis.EquityLoss, formatMove(analysis.BestMove))
	case engine.SkillBad:
		return fmt.Sprintf("This was an error losing %.3f equity. Consider %s instead.",
			analysis.EquityLoss, formatMove(analysis.BestMove))
	case engine.SkillDoubtful:
		return fmt.Sprintf("This move is questionable (%.3f equity loss). %s was slightly better.",
			analysis.EquityLoss, formatMove(analysis.BestMove))
	default:
		return ""
	}
}

// generateCubeSuggestion generates an improvement suggestion for a cube error.
func generateCubeSuggestion(analysis *engine.CubeSkillAnalysis) string {
	if analysis.Skill == engine.SkillNone {
		return ""
	}

	optimalStr := cubeActionToString(analysis.OptimalPlay)
	actualStr := cubeActionToString(analysis.ActualPlay)

	switch analysis.Skill {
	case engine.SkillVeryBad:
		return fmt.Sprintf("This was a cube blunder losing %.3f equity. You should have chosen %s instead of %s.",
			analysis.EquityLoss, optimalStr, actualStr)
	case engine.SkillBad:
		return fmt.Sprintf("This was a cube error losing %.3f equity. %s was correct.",
			analysis.EquityLoss, optimalStr)
	case engine.SkillDoubtful:
		return fmt.Sprintf("This cube decision is questionable (%.3f equity loss). %s was slightly better.",
			analysis.EquityLoss, optimalStr)
	default:
		return ""
	}
}

// generateGameSuggestions generates overall improvement suggestions for a game.
func generateGameSuggestions(resp *GameAnalysisResponse) []string {
	var suggestions []string

	for i := 0; i < 2; i++ {
		player := resp.Players[i]
		playerName := fmt.Sprintf("Player %d", i+1)

		if player.Blunders > 0 {
			suggestions = append(suggestions,
				fmt.Sprintf("%s had %d blunder(s). Review these positions carefully.", playerName, player.Blunders))
		}

		if player.ErrorPerMove > 0.02 {
			suggestions = append(suggestions,
				fmt.Sprintf("%s's error rate (%.3f) is high. Focus on checker play fundamentals.", playerName, player.ErrorPerMove))
		}

		if player.TotalCubeDecisions > 0 {
			cubeErrors := 0
			for _, ce := range resp.CubeErrors {
				if ce.Player == i {
					cubeErrors++
				}
			}
			if cubeErrors > 0 {
				suggestions = append(suggestions,
					fmt.Sprintf("%s made %d cube error(s). Study cube theory and match equity.", playerName, cubeErrors))
			}
		}
	}

	if len(suggestions) == 0 {
		suggestions = append(suggestions, "Good game! Both players played well.")
	}

	return suggestions
}
