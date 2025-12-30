package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/yourusername/bgengine/internal/positionid"
	"github.com/yourusername/bgengine/pkg/engine"
)

// SSERolloutRequest is the query parameters for SSE rollout.
type SSERolloutRequest struct {
	Position string `json:"position"`
	Trials   int    `json:"trials"`
	Truncate int    `json:"truncate"`
	Workers  int    `json:"workers"`
}

// SSEEvent represents a Server-Sent Event.
type SSEEvent struct {
	Event string      `json:"event"` // Event type: "progress", "result", "error"
	Data  interface{} `json:"data"`  // Event data
}

// RolloutSSE handles Server-Sent Events for streaming rollout progress.
// GET /api/v1/rollout/stream?position=...&trials=...&truncate=...
func (h *Handlers) RolloutSSE(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Parse query parameters
	query := r.URL.Query()
	position := query.Get("position")
	if position == "" {
		writeSSEError(w, "position is required")
		return
	}

	board, err := positionid.BoardFromPositionID(position)
	if err != nil {
		writeSSEError(w, "invalid position: "+err.Error())
		return
	}

	trials := parseIntParam(query.Get("trials"), 1296)
	truncate := parseIntParam(query.Get("truncate"), 0)
	workers := parseIntParam(query.Get("workers"), 0)

	gs := &engine.GameState{
		Board:     engine.Board(board),
		Turn:      0,
		CubeValue: 1,
		CubeOwner: -1,
	}

	opts := engine.RolloutOptions{
		Trials:   trials,
		Truncate: truncate,
		Workers:  workers,
	}

	// Flush function for streaming
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeSSEError(w, "streaming not supported")
		return
	}

	// Progress callback sends SSE events
	callback := func(p engine.RolloutProgress) {
		writeSSEEvent(w, "progress", WSRolloutProgress{
			TrialsCompleted: p.TrialsCompleted,
			TrialsTotal:     p.TrialsTotal,
			Percent:         p.Percent,
			CurrentEquity:   p.CurrentEquity,
			CurrentCI:       p.CurrentCI,
		})
		flusher.Flush()
	}

	result, err := h.engine.RolloutWithProgress(gs, opts, callback)
	if err != nil {
		writeSSEError(w, "rollout failed: "+err.Error())
		return
	}

	// Send final result
	writeSSEEvent(w, "result", WSRolloutResult{
		Equity:          result.Equity,
		EquityCI:        result.EquityCI,
		WinProb:         result.WinProb * 100,
		WinG:            result.WinG * 100,
		WinBG:           result.WinBG * 100,
		LoseG:           result.LoseG * 100,
		LoseBG:          result.LoseBG * 100,
		TrialsCompleted: result.TrialsCompleted,
		GamesWon:        result.GamesWon,
		GamesLost:       result.GamesLost,
	})
	flusher.Flush()

	// Send done event to signal completion
	writeSSEEvent(w, "done", nil)
	flusher.Flush()
}

// writeSSEEvent writes a Server-Sent Event to the response.
func writeSSEEvent(w http.ResponseWriter, event string, data interface{}) {
	fmt.Fprintf(w, "event: %s\n", event)
	if data != nil {
		jsonData, _ := json.Marshal(data)
		fmt.Fprintf(w, "data: %s\n", jsonData)
	}
	fmt.Fprintf(w, "\n")
}

// writeSSEError writes an error event and closes the stream.
func writeSSEError(w http.ResponseWriter, message string) {
	writeSSEEvent(w, "error", map[string]string{"error": message})
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

// parseIntParam parses an integer from a string with a default value.
func parseIntParam(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	var val int
	if _, err := fmt.Sscanf(s, "%d", &val); err != nil {
		return defaultVal
	}
	return val
}

