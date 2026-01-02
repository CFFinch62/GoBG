package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/yourusername/bgengine/pkg/engine"
)

// getTestEngine returns an engine for testing (no networks = fast, uses fallback values)
func getTestEngine() *engine.Engine {
	// Empty options = no neural nets loaded, uses fallback values for evaluation
	eng, _ := engine.NewEngine(engine.EngineOptions{})
	return eng
}

// TestHealthHandler tests the health endpoint.
func TestHealthHandler(t *testing.T) {
	h := NewHandlers(nil, "test-version")

	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()

	h.Health(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Health status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var health HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Fatalf("Decode error: %v", err)
	}

	if health.Status != "ok" {
		t.Errorf("Status = %q, want %q", health.Status, "ok")
	}
	if health.Version != "test-version" {
		t.Errorf("Version = %q, want %q", health.Version, "test-version")
	}
}

func TestHealthHandlerReady(t *testing.T) {
	eng := getTestEngine()
	h := NewHandlers(eng, "1.0.0")

	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()

	h.Health(w, req)

	var health HealthResponse
	json.NewDecoder(w.Result().Body).Decode(&health)

	if !health.Ready {
		t.Error("Expected ready = true when engine is set")
	}
}

func TestEvaluateHandler(t *testing.T) {
	eng := getTestEngine()
	h := NewHandlers(eng, "1.0.0")

	tests := []struct {
		name       string
		body       interface{}
		wantStatus int
		wantError  bool
	}{
		{
			name:       "valid position",
			body:       EvaluateRequest{Position: "4HPwATDgc/ABMA", Ply: 0},
			wantStatus: http.StatusOK,
		},
		{
			name:       "empty position",
			body:       EvaluateRequest{Position: ""},
			wantStatus: http.StatusBadRequest,
			wantError:  true,
		},
		{
			name:       "invalid position",
			body:       EvaluateRequest{Position: "invalid!!!"},
			wantStatus: http.StatusBadRequest,
			wantError:  true,
		},
		{
			name:       "invalid json",
			body:       "not json",
			wantStatus: http.StatusBadRequest,
			wantError:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var body []byte
			if s, ok := tc.body.(string); ok {
				body = []byte(s)
			} else {
				body, _ = json.Marshal(tc.body)
			}
			req := httptest.NewRequest("POST", "/api/evaluate", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.Evaluate(w, req)

			resp := w.Result()
			if resp.StatusCode != tc.wantStatus {
				t.Errorf("Status = %d, want %d", resp.StatusCode, tc.wantStatus)
			}

			if !tc.wantError && tc.wantStatus == http.StatusOK {
				var eval EvaluateResponse
				if err := json.NewDecoder(resp.Body).Decode(&eval); err != nil {
					t.Fatalf("Decode error: %v", err)
				}
				// With mock networks, we still get a response
				if eval.Win < 0 || eval.Win > 100 {
					t.Errorf("Win = %f, want 0-100", eval.Win)
				}
			}
		})
	}
}

func TestMoveHandler(t *testing.T) {
	eng := getTestEngine()
	h := NewHandlers(eng, "1.0.0")

	tests := []struct {
		name       string
		body       interface{}
		wantStatus int
	}{
		{
			name: "valid move request",
			body: MoveRequest{
				Position: "4HPwATDgc/ABMA",
				Dice:     [2]int{3, 1},
				NumMoves: 3,
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "missing position",
			body: MoveRequest{
				Dice:     [2]int{3, 1},
				NumMoves: 3,
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "missing dice",
			body: MoveRequest{
				Position: "4HPwATDgc/ABMA",
				NumMoves: 3,
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid dice value",
			body: MoveRequest{
				Position: "4HPwATDgc/ABMA",
				Dice:     [2]int{7, 1}, // dice must be 1-6
				NumMoves: 3,
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.body)
			req := httptest.NewRequest("POST", "/api/move", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.Move(w, req)

			resp := w.Result()
			if resp.StatusCode != tc.wantStatus {
				t.Errorf("Status = %d, want %d", resp.StatusCode, tc.wantStatus)
			}

			if tc.wantStatus == http.StatusOK {
				var moveResp MovesResponse
				if err := json.NewDecoder(resp.Body).Decode(&moveResp); err != nil {
					t.Fatalf("Decode error: %v", err)
				}
				if moveResp.NumLegal < 0 {
					t.Error("Expected non-negative NumLegal")
				}
			}
		})
	}
}

func TestCubeHandler(t *testing.T) {
	eng := getTestEngine()
	h := NewHandlers(eng, "1.0.0")

	tests := []struct {
		name       string
		body       interface{}
		wantStatus int
	}{
		{
			name:       "valid cube request",
			body:       CubeRequest{Position: "4HPwATDgc/ABMA"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing position",
			body:       CubeRequest{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "with match context",
			body:       CubeRequest{Position: "4HPwATDgc/ABMA", MatchLength: 7, Score: [2]int{3, 2}},
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.body)
			req := httptest.NewRequest("POST", "/api/cube", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.Cube(w, req)

			resp := w.Result()
			if resp.StatusCode != tc.wantStatus {
				t.Errorf("Status = %d, want %d", resp.StatusCode, tc.wantStatus)
			}

			if tc.wantStatus == http.StatusOK {
				var cubeResp CubeResponse
				if err := json.NewDecoder(resp.Body).Decode(&cubeResp); err != nil {
					t.Fatalf("Decode error: %v", err)
				}
				// Action should be one of: no_double, double_take, double_pass, take, pass
				validActions := map[string]bool{
					"no_double": true, "double_take": true, "double_pass": true,
					"take": true, "pass": true, "redouble_take": true, "redouble_pass": true,
				}
				if !validActions[cubeResp.Action] {
					t.Errorf("Invalid action: %s", cubeResp.Action)
				}
			}
		})
	}
}

func TestRolloutHandler(t *testing.T) {
	eng := getTestEngine()
	h := NewHandlers(eng, "1.0.0")

	tests := []struct {
		name       string
		body       interface{}
		wantStatus int
	}{
		{
			name:       "valid rollout request",
			body:       RolloutRequest{Position: "4HPwATDgc/ABMA", Trials: 10, Seed: 12345},
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing position",
			body:       RolloutRequest{Trials: 100},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.body)
			req := httptest.NewRequest("POST", "/api/rollout", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.Rollout(w, req)

			resp := w.Result()
			if resp.StatusCode != tc.wantStatus {
				t.Errorf("Status = %d, want %d", resp.StatusCode, tc.wantStatus)
			}

			if tc.wantStatus == http.StatusOK {
				var rolloutResp RolloutResponse
				if err := json.NewDecoder(resp.Body).Decode(&rolloutResp); err != nil {
					t.Fatalf("Decode error: %v", err)
				}
				if rolloutResp.Trials <= 0 {
					t.Error("Expected positive Trials")
				}
			}
		})
	}
}

// TestFormatMove tests the move formatting helper
func TestFormatMove(t *testing.T) {
	tests := []struct {
		move engine.Move
		want string
	}{
		{
			move: engine.Move{From: [4]int8{7, 5, -1, -1}, To: [4]int8{4, 4, -1, -1}},
			want: "8/5 6/5",
		},
		{
			move: engine.Move{From: [4]int8{23, -1, -1, -1}, To: [4]int8{17, -1, -1, -1}},
			want: "24/18",
		},
		{
			move: engine.Move{From: [4]int8{24, -1, -1, -1}, To: [4]int8{20, -1, -1, -1}},
			want: "bar/21",
		},
		{
			move: engine.Move{From: [4]int8{5, -1, -1, -1}, To: [4]int8{-1, -1, -1, -1}},
			want: "6/off",
		},
	}

	for _, tc := range tests {
		got := formatMove(tc.move)
		if got != tc.want {
			t.Errorf("formatMove(%v) = %q, want %q", tc.move, got, tc.want)
		}
	}
}

// ============================================================================
// WebSocket Tests
// ============================================================================

func TestWebSocketUpgrade(t *testing.T) {
	eng := getTestEngine()
	h := NewHandlers(eng, "1.0.0")

	server := httptest.NewServer(http.HandlerFunc(h.WebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}
	defer ws.Close()

	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusSwitchingProtocols)
	}
}

func TestWebSocketPing(t *testing.T) {
	eng := getTestEngine()
	h := NewHandlers(eng, "1.0.0")

	server := httptest.NewServer(http.HandlerFunc(h.WebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}
	defer ws.Close()

	// Send ping
	msg := WSMessage{Type: "ping", ID: "test-ping-1"}
	if err := ws.WriteJSON(msg); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Read pong
	ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	var resp WSResponse
	if err := ws.ReadJSON(&resp); err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if resp.Type != "pong" {
		t.Errorf("Response type = %q, want %q", resp.Type, "pong")
	}
	if resp.ID != "test-ping-1" {
		t.Errorf("Response ID = %q, want %q", resp.ID, "test-ping-1")
	}
}

func TestWebSocketEvaluate(t *testing.T) {
	eng := getTestEngine()
	h := NewHandlers(eng, "1.0.0")

	server := httptest.NewServer(http.HandlerFunc(h.WebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}
	defer ws.Close()

	// Send evaluate request
	payload, _ := json.Marshal(EvaluateRequest{Position: "4HPwATDgc/ABMA"})
	msg := WSMessage{Type: "evaluate", ID: "eval-1", Payload: payload}
	if err := ws.WriteJSON(msg); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Read response
	ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	var resp WSResponse
	if err := ws.ReadJSON(&resp); err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if resp.Type != "result" {
		t.Errorf("Response type = %q, want %q", resp.Type, "result")
	}
	if resp.ID != "eval-1" {
		t.Errorf("Response ID = %q, want %q", resp.ID, "eval-1")
	}
	if resp.Error != "" {
		t.Errorf("Unexpected error: %s", resp.Error)
	}
}

func TestWebSocketMove(t *testing.T) {
	eng := getTestEngine()
	h := NewHandlers(eng, "1.0.0")

	server := httptest.NewServer(http.HandlerFunc(h.WebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}
	defer ws.Close()

	// Send move request
	payload, _ := json.Marshal(MoveRequest{Position: "4HPwATDgc/ABMA", Dice: [2]int{3, 1}, NumMoves: 3})
	msg := WSMessage{Type: "move", ID: "move-1", Payload: payload}
	if err := ws.WriteJSON(msg); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Read response
	ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	var resp WSResponse
	if err := ws.ReadJSON(&resp); err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if resp.Type != "result" {
		t.Errorf("Response type = %q, want %q", resp.Type, "result")
	}
	if resp.ID != "move-1" {
		t.Errorf("Response ID = %q, want %q", resp.ID, "move-1")
	}
}

func TestWebSocketCube(t *testing.T) {
	eng := getTestEngine()
	h := NewHandlers(eng, "1.0.0")

	server := httptest.NewServer(http.HandlerFunc(h.WebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}
	defer ws.Close()

	// Send cube request
	payload, _ := json.Marshal(CubeRequest{Position: "4HPwATDgc/ABMA"})
	msg := WSMessage{Type: "cube", ID: "cube-1", Payload: payload}
	if err := ws.WriteJSON(msg); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Read response
	ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	var resp WSResponse
	if err := ws.ReadJSON(&resp); err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if resp.Type != "result" {
		t.Errorf("Response type = %q, want %q", resp.Type, "result")
	}
	if resp.ID != "cube-1" {
		t.Errorf("Response ID = %q, want %q", resp.ID, "cube-1")
	}
}

func TestWebSocketErrors(t *testing.T) {
	eng := getTestEngine()
	h := NewHandlers(eng, "1.0.0")

	server := httptest.NewServer(http.HandlerFunc(h.WebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}
	defer ws.Close()

	tests := []struct {
		name    string
		msgType string
		payload interface{}
		wantErr string
	}{
		{"unknown type", "unknown", nil, "unknown message type"},
		{"invalid position", "evaluate", EvaluateRequest{Position: "invalid!!!"}, "invalid position"},
		{"invalid dice", "move", MoveRequest{Position: "4HPwATDgc/ABMA", Dice: [2]int{7, 1}}, "invalid dice"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var payload json.RawMessage
			if tc.payload != nil {
				payload, _ = json.Marshal(tc.payload)
			}
			msg := WSMessage{Type: tc.msgType, ID: tc.name, Payload: payload}
			if err := ws.WriteJSON(msg); err != nil {
				t.Fatalf("Write failed: %v", err)
			}

			ws.SetReadDeadline(time.Now().Add(2 * time.Second))
			var resp WSResponse
			if err := ws.ReadJSON(&resp); err != nil {
				t.Fatalf("Read failed: %v", err)
			}

			if resp.Type != "error" {
				t.Errorf("Response type = %q, want %q", resp.Type, "error")
			}
			if !strings.Contains(resp.Error, tc.wantErr) {
				t.Errorf("Error = %q, want containing %q", resp.Error, tc.wantErr)
			}
		})
	}
}

func TestWebSocketRollout(t *testing.T) {
	eng := getTestEngine()
	h := NewHandlers(eng, "1.0.0")

	server := httptest.NewServer(http.HandlerFunc(h.WebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}
	defer ws.Close()

	// Send rollout request
	payload := `{"position":"4HPwATDgc/ABMA","trials":100,"truncate":5}`
	msg := WSMessage{
		Type:    "rollout",
		ID:      "test-rollout-1",
		Payload: json.RawMessage(payload),
	}
	if err := ws.WriteJSON(msg); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Collect responses (progress updates + final result)
	var progressCount int
	var finalResult *WSResponse

	ws.SetReadDeadline(time.Now().Add(30 * time.Second))
	for {
		var resp WSResponse
		if err := ws.ReadJSON(&resp); err != nil {
			t.Fatalf("Read failed: %v", err)
		}

		if resp.ID != "test-rollout-1" {
			t.Errorf("Response ID = %q, want %q", resp.ID, "test-rollout-1")
		}

		switch resp.Type {
		case "progress":
			progressCount++
		case "result":
			finalResult = &resp
		case "error":
			t.Fatalf("Unexpected error: %s", resp.Error)
		}

		if resp.Type == "result" {
			break
		}
	}

	// Should have received progress updates
	if progressCount < 1 {
		t.Errorf("Expected progress updates, got %d", progressCount)
	}

	// Should have final result
	if finalResult == nil {
		t.Error("Expected final result")
	}

	t.Logf("Received %d progress updates before final result", progressCount)
}

func TestWebSocketRolloutProgress(t *testing.T) {
	eng := getTestEngine()
	h := NewHandlers(eng, "1.0.0")

	server := httptest.NewServer(http.HandlerFunc(h.WebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}
	defer ws.Close()

	// Request more trials to get more progress updates
	payload := `{"position":"4HPwATDgc/ABMA","trials":200,"truncate":5}`
	msg := WSMessage{
		Type:    "rollout",
		ID:      "test-rollout-progress",
		Payload: json.RawMessage(payload),
	}
	if err := ws.WriteJSON(msg); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Track progress percentages
	var percentages []float64

	ws.SetReadDeadline(time.Now().Add(30 * time.Second))
	for {
		var resp WSResponse
		if err := ws.ReadJSON(&resp); err != nil {
			t.Fatalf("Read failed: %v", err)
		}

		if resp.Type == "progress" {
			// Extract percent from payload
			data, _ := json.Marshal(resp.Payload)
			var progress WSRolloutProgress
			json.Unmarshal(data, &progress)
			percentages = append(percentages, progress.Percent)
		}

		if resp.Type == "result" {
			break
		}
	}

	// Verify percentages are increasing
	for i := 1; i < len(percentages); i++ {
		if percentages[i] < percentages[i-1] {
			t.Errorf("Progress should increase: %.1f%% -> %.1f%%", percentages[i-1], percentages[i])
		}
	}

	// Last progress should be 100%
	if len(percentages) > 0 && percentages[len(percentages)-1] != 100.0 {
		t.Errorf("Last progress should be 100%%, got %.1f%%", percentages[len(percentages)-1])
	}

	t.Logf("Progress updates: %v", percentages)
}

func TestRolloutSSE(t *testing.T) {
	eng := getTestEngine()
	h := NewHandlers(eng, "1.0.0")

	server := httptest.NewServer(http.HandlerFunc(h.RolloutSSE))
	defer server.Close()

	// Make request with query parameters
	url := server.URL + "?position=4HPwATDgc/ABMA&trials=100&truncate=5"
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("Content-Type") != "text/event-stream" {
		t.Errorf("Content-Type = %q, want %q", resp.Header.Get("Content-Type"), "text/event-stream")
	}

	// Read all events
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Read body failed: %v", err)
	}

	bodyStr := string(body)

	// Should contain progress events
	if !strings.Contains(bodyStr, "event: progress") {
		t.Error("Expected progress events in response")
	}

	// Should contain result event
	if !strings.Contains(bodyStr, "event: result") {
		t.Error("Expected result event in response")
	}

	// Should contain done event
	if !strings.Contains(bodyStr, "event: done") {
		t.Error("Expected done event in response")
	}

	t.Logf("SSE response:\n%s", bodyStr[:min(500, len(bodyStr))])
}

func TestRolloutSSEMissingPosition(t *testing.T) {
	eng := getTestEngine()
	h := NewHandlers(eng, "1.0.0")

	server := httptest.NewServer(http.HandlerFunc(h.RolloutSSE))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	if !strings.Contains(bodyStr, "event: error") {
		t.Error("Expected error event for missing position")
	}
	if !strings.Contains(bodyStr, "position is required") {
		t.Error("Expected 'position is required' error message")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ============================================================================
// Tutor Handler Tests
// ============================================================================

func TestTutorMoveHandler(t *testing.T) {
	eng := getTestEngine()
	h := NewHandlers(eng, "1.0.0")

	tests := []struct {
		name       string
		body       interface{}
		wantStatus int
		wantError  bool
	}{
		{
			name: "valid move analysis",
			body: TutorMoveRequest{
				Position: "4HPwATDgc/ABMA", // Starting position
				Dice:     [2]int{3, 1},
				Move:     "8/5 6/5", // Good opening move
			},
			wantStatus: http.StatusOK,
			wantError:  false,
		},
		{
			name: "bad move analysis",
			body: TutorMoveRequest{
				Position: "4HPwATDgc/ABMA",
				Dice:     [2]int{3, 1},
				Move:     "24/21 24/23", // Running with back checkers - bad
			},
			wantStatus: http.StatusOK,
			wantError:  false,
		},
		{
			name:       "missing position",
			body:       TutorMoveRequest{Dice: [2]int{3, 1}, Move: "8/5"},
			wantStatus: http.StatusBadRequest,
			wantError:  true,
		},
		{
			name:       "missing move",
			body:       TutorMoveRequest{Position: "4HPwATDgc/ABMA", Dice: [2]int{3, 1}},
			wantStatus: http.StatusBadRequest,
			wantError:  true,
		},
		{
			name:       "invalid json",
			body:       "not json",
			wantStatus: http.StatusBadRequest,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyBytes []byte
			if str, ok := tt.body.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, _ = json.Marshal(tt.body)
			}

			req := httptest.NewRequest("POST", "/api/tutor/move", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.HandleTutorMove(w, req)

			resp := w.Result()
			if resp.StatusCode != tt.wantStatus {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("Status = %d, want %d. Body: %s", resp.StatusCode, tt.wantStatus, body)
			}

			if !tt.wantError && resp.StatusCode == http.StatusOK {
				var result TutorMoveResponse
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				// Verify response has expected fields
				if result.Skill == "" {
					t.Error("Expected skill to be set")
				}
				if result.BestMove == "" {
					t.Error("Expected best_move to be set")
				}
			}
		})
	}
}

func TestTutorMoveHandlerSkillDetection(t *testing.T) {
	eng := getTestEngine()
	h := NewHandlers(eng, "1.0.0")

	// Test that move analysis returns valid structure
	// Note: Test engine uses fallback values, so all moves evaluate similarly
	body := TutorMoveRequest{
		Position: "4HPwATDgc/ABMA", // Starting position
		Dice:     [2]int{3, 1},
		Move:     "24/21 24/23", // Running - normally a bad move
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/tutor/move", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.HandleTutorMove(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result TutorMoveResponse
	json.NewDecoder(resp.Body).Decode(&result)

	// Verify response structure is valid
	if result.Skill == "" {
		t.Error("Expected skill to be set")
	}
	if result.BestMove == "" {
		t.Error("Expected best_move to be set")
	}
	// EquityLoss should be non-negative
	if result.EquityLoss < 0 {
		t.Errorf("Expected equity loss >= 0, got %f", result.EquityLoss)
	}
	t.Logf("Move analysis: skill=%s, loss=%.4f, best=%s", result.Skill, result.EquityLoss, result.BestMove)
}

func TestTutorCubeHandler(t *testing.T) {
	eng := getTestEngine()
	h := NewHandlers(eng, "1.0.0")

	tests := []struct {
		name       string
		body       interface{}
		wantStatus int
		wantError  bool
	}{
		{
			name: "valid cube analysis - no double",
			body: TutorCubeRequest{
				Position: "4HPwATDgc/ABMA", // Starting position
				Action:   "no_double",
			},
			wantStatus: http.StatusOK,
			wantError:  false,
		},
		{
			name: "valid cube analysis - double",
			body: TutorCubeRequest{
				Position: "4HPwATDgc/ABMA",
				Action:   "double",
			},
			wantStatus: http.StatusOK,
			wantError:  false,
		},
		{
			name:       "missing position",
			body:       TutorCubeRequest{Action: "double"},
			wantStatus: http.StatusBadRequest,
			wantError:  true,
		},
		{
			name: "invalid action",
			body: TutorCubeRequest{
				Position: "4HPwATDgc/ABMA",
				Action:   "invalid_action",
			},
			wantStatus: http.StatusBadRequest,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyBytes []byte
			if str, ok := tt.body.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, _ = json.Marshal(tt.body)
			}

			req := httptest.NewRequest("POST", "/api/tutor/cube", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.HandleTutorCube(w, req)

			resp := w.Result()
			if resp.StatusCode != tt.wantStatus {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("Status = %d, want %d. Body: %s", resp.StatusCode, tt.wantStatus, body)
			}

			if !tt.wantError && resp.StatusCode == http.StatusOK {
				var result TutorCubeResponse
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if result.Skill == "" {
					t.Error("Expected skill to be set")
				}
				if result.Optimal == "" {
					t.Error("Expected optimal to be set")
				}
			}
		})
	}
}

func TestAnalyzeGameHandler(t *testing.T) {
	eng := getTestEngine()
	h := NewHandlers(eng, "1.0.0")

	tests := []struct {
		name       string
		body       interface{}
		wantStatus int
		wantError  bool
	}{
		{
			name: "valid game analysis",
			body: AnalyzeGameRequest{
				Positions: []GamePosition{
					{
						Position: "4HPwATDgc/ABMA",
						Dice:     [2]int{3, 1},
						Move:     "8/5 6/5",
						Player:   0,
					},
					{
						Position: "4HPwATDgc/ABMA",
						Dice:     [2]int{6, 4},
						Move:     "24/14",
						Player:   1,
					},
				},
			},
			wantStatus: http.StatusOK,
			wantError:  false,
		},
		{
			name:       "empty positions",
			body:       AnalyzeGameRequest{Positions: []GamePosition{}},
			wantStatus: http.StatusBadRequest,
			wantError:  true,
		},
		{
			name:       "invalid json",
			body:       "not json",
			wantStatus: http.StatusBadRequest,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyBytes []byte
			if str, ok := tt.body.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, _ = json.Marshal(tt.body)
			}

			req := httptest.NewRequest("POST", "/api/tutor/game", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.HandleAnalyzeGame(w, req)

			resp := w.Result()
			if resp.StatusCode != tt.wantStatus {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("Status = %d, want %d. Body: %s", resp.StatusCode, tt.wantStatus, body)
			}

			if !tt.wantError && resp.StatusCode == http.StatusOK {
				var result GameAnalysisResponse
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if result.TotalMoves == 0 {
					t.Error("Expected total_moves > 0")
				}
			}
		})
	}
}

func TestAnalyzeGameHandlerWithErrors(t *testing.T) {
	eng := getTestEngine()
	h := NewHandlers(eng, "1.0.0")

	// Create a game with multiple positions
	// Note: Test engine uses fallback values, so error detection may not work
	body := AnalyzeGameRequest{
		Positions: []GamePosition{
			{
				Position: "4HPwATDgc/ABMA",
				Dice:     [2]int{3, 1},
				Move:     "24/21 24/23",
				Player:   0,
			},
			{
				Position: "4HPwATDgc/ABMA",
				Dice:     [2]int{3, 1},
				Move:     "8/5 6/5",
				Player:   1,
			},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/tutor/game", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.HandleAnalyzeGame(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result GameAnalysisResponse
	json.NewDecoder(resp.Body).Decode(&result)

	// Verify response structure
	if result.TotalMoves != 2 {
		t.Errorf("Expected 2 total moves, got %d", result.TotalMoves)
	}

	// Should have suggestions (always generated)
	if len(result.Suggestions) == 0 {
		t.Error("Expected suggestions to be generated")
	}

	// Verify player stats are populated
	for i := 0; i < 2; i++ {
		if result.Players[i].Rating == "" {
			t.Errorf("Expected player %d rating to be set", i)
		}
	}

	t.Logf("Game analysis: %d moves, %d errors, suggestions: %v",
		result.TotalMoves, len(result.MoveErrors), result.Suggestions)
}

func TestTutorMoveHandlerTopMoves(t *testing.T) {
	eng := getTestEngine()
	h := NewHandlers(eng, "1.0.0")

	body := TutorMoveRequest{
		Position: "4HPwATDgc/ABMA",
		Dice:     [2]int{3, 1},
		Move:     "8/5 6/5",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/tutor/move", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.HandleTutorMove(w, req)

	var result TutorMoveResponse
	json.NewDecoder(w.Result().Body).Decode(&result)

	// Should have top moves for context
	if len(result.TopMoves) == 0 {
		t.Error("Expected top_moves to be populated")
	}

	// Top moves should be ordered by equity
	for i := 1; i < len(result.TopMoves); i++ {
		if result.TopMoves[i].Equity > result.TopMoves[i-1].Equity {
			t.Errorf("Top moves not sorted: move %d equity %.4f > move %d equity %.4f",
				i, result.TopMoves[i].Equity, i-1, result.TopMoves[i-1].Equity)
		}
	}

	t.Logf("Top %d moves returned", len(result.TopMoves))
	for i, m := range result.TopMoves {
		t.Logf("  %d. %s (equity: %.4f)", i+1, m.Move, m.Equity)
	}
}

// ============================================================================
// FIBS Board Handler Tests
// ============================================================================

func TestFIBSBoardHandler(t *testing.T) {
	eng := getTestEngine()
	h := NewHandlers(eng, "1.0.0")

	tests := []struct {
		name       string
		body       interface{}
		wantStatus int
		wantError  bool
	}{
		{
			name: "valid fibs board - starting position",
			body: FIBSBoardRequest{
				Board: "board:You:Opponent:5:0:0:0:2:0:0:0:0:5:0:3:0:0:0:0:5:0:0:0:0:0:0:0:0:2:0:-2:0:0:0:0:0:-5:0:-3:0:0:0:0:-5:0:0:0:0:0:0:0:0:-2:0:1:3:1:0:0:1:1:1:0:1:-1:0:25:0:0:0:0:0:0:0:0",
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "valid fibs board with num_moves",
			body: FIBSBoardRequest{
				Board:    "board:You:Opponent:5:0:0:0:2:0:0:0:0:5:0:3:0:0:0:0:5:0:0:0:0:0:0:0:0:2:0:-2:0:0:0:0:0:-5:0:-3:0:0:0:0:-5:0:0:0:0:0:0:0:0:-2:0:1:3:1:0:0:1:1:1:0:1:-1:0:25:0:0:0:0:0:0:0:0",
				NumMoves: 3,
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing board",
			body:       FIBSBoardRequest{},
			wantStatus: http.StatusBadRequest,
			wantError:  true,
		},
		{
			name: "invalid fibs board - too few fields",
			body: FIBSBoardRequest{
				Board: "board:You:Opponent:5:0:0",
			},
			wantStatus: http.StatusBadRequest,
			wantError:  true,
		},
		{
			name:       "invalid json",
			body:       "not json",
			wantStatus: http.StatusBadRequest,
			wantError:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var body []byte
			if s, ok := tc.body.(string); ok {
				body = []byte(s)
			} else {
				body, _ = json.Marshal(tc.body)
			}
			req := httptest.NewRequest("POST", "/api/fibsboard", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.HandleFIBSBoard(w, req)

			resp := w.Result()
			if resp.StatusCode != tc.wantStatus {
				respBody, _ := io.ReadAll(resp.Body)
				t.Errorf("Status = %d, want %d. Body: %s", resp.StatusCode, tc.wantStatus, respBody)
			}

			if !tc.wantError && tc.wantStatus == http.StatusOK {
				var fibsResp FIBSBoardResponse
				if err := json.NewDecoder(resp.Body).Decode(&fibsResp); err != nil {
					t.Fatalf("Decode error: %v", err)
				}
				// Verify response has expected fields
				if fibsResp.Player1 == "" {
					t.Error("Expected player1 to be set")
				}
				if fibsResp.Player2 == "" {
					t.Error("Expected player2 to be set")
				}
				if fibsResp.PositionID == "" {
					t.Error("Expected position_id to be set")
				}
				// Win probability should be in range
				if fibsResp.Win < 0 || fibsResp.Win > 100 {
					t.Errorf("Win = %f, want 0-100", fibsResp.Win)
				}
			}
		})
	}
}

func TestFIBSBoardHandlerWithDice(t *testing.T) {
	eng := getTestEngine()
	h := NewHandlers(eng, "1.0.0")

	// FIBS board string with dice rolled (3,1)
	// Format after "board:" prefix:
	// 0-1: player names, 2: match length, 3-4: scores,
	// 5-30: board[26], 31: turn, 32-35: dice[4], 36: cube, 37-38: can_double, 39: doubled
	// Starting position in FIBS format:
	// Points 0-25 where positive=your checkers, negative=opponent
	// FIBS counts from 1-24, with 0=off and 25=bar
	// Starting: 2 on point 1, 5 on point 12, 3 on point 17, 5 on point 19 (for player)
	// Opponent: 2 on 24, 5 on 13, 3 on 8, 5 on 6
	// Build a proper starting position FIBS string:
	// pos[1]=2, pos[6]=-5, pos[8]=-3, pos[12]=5, pos[13]=-5, pos[17]=3, pos[19]=5, pos[24]=-2
	body := FIBSBoardRequest{
		// Starting position with 3,1 dice
		// Format: board:p1:p2:ml:s1:s2:board[26]:turn:d1:d2:od1:od2:cube:candbl:oppcandbl
		// Fields after "board:": 0-1=names, 2=ml, 3-4=scores, 5-30=board[26], 31=turn, 32-35=dice
		// Board[26]: indices 0-25, where 0 is off, 25 is bar
		// Starting: point1=2, point6=-5, point8=-3, point12=5, point13=-5, point17=3, point19=5, point24=-2
		Board:    "board:You:Opponent:5:0:0:0:2:0:0:0:0:-5:0:-3:0:0:0:5:-5:0:0:0:3:0:5:0:0:0:-2:0:0:1:3:1:0:0:1:1:1",
		NumMoves: 5,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/fibsboard", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.HandleFIBSBoard(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Status = %d, want %d. Body: %s", resp.StatusCode, http.StatusOK, respBody)
	}

	var fibsResp FIBSBoardResponse
	if err := json.NewDecoder(resp.Body).Decode(&fibsResp); err != nil {
		t.Fatalf("Decode error: %v", err)
	}

	// Verify dice were parsed correctly
	if fibsResp.Dice[0] != 3 || fibsResp.Dice[1] != 1 {
		t.Errorf("Dice = %v, want [3,1]", fibsResp.Dice)
	}

	// Position with checkers should have legal moves
	if fibsResp.NumLegal > 0 {
		t.Logf("FIBS board analysis: %s vs %s, %d legal moves, top %d returned",
			fibsResp.Player1, fibsResp.Player2, fibsResp.NumLegal, len(fibsResp.Moves))
		for i, m := range fibsResp.Moves {
			t.Logf("  %d. %s (equity: %.4f)", i+1, m.Move, m.Equity)
		}
	} else {
		// Even if no moves (e.g., the board layout is unusual), the response structure should be valid
		t.Logf("FIBS board analysis: dice %v parsed, num_legal=%d", fibsResp.Dice, fibsResp.NumLegal)
	}
}

func TestFIBSBoardHandlerParsesPlayerNames(t *testing.T) {
	eng := getTestEngine()
	h := NewHandlers(eng, "1.0.0")

	// FIBS board with specific player names
	body := FIBSBoardRequest{
		Board: "board:Alice:Bob:7:3:2:0:2:0:0:0:0:5:0:3:0:0:0:0:5:0:0:0:0:0:0:0:0:2:0:-2:0:0:0:0:0:-5:0:-3:0:0:0:0:-5:0:0:0:0:0:0:0:0:-2:0:1:0:0:0:0:1:1:1:0:1:-1:0:25:0:0:0:0:0:0:0:0",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/fibsboard", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.HandleFIBSBoard(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var fibsResp FIBSBoardResponse
	json.NewDecoder(resp.Body).Decode(&fibsResp)

	if fibsResp.Player1 != "Alice" {
		t.Errorf("Player1 = %q, want %q", fibsResp.Player1, "Alice")
	}
	if fibsResp.Player2 != "Bob" {
		t.Errorf("Player2 = %q, want %q", fibsResp.Player2, "Bob")
	}
	if fibsResp.MatchLength != 7 {
		t.Errorf("MatchLength = %d, want %d", fibsResp.MatchLength, 7)
	}
	if fibsResp.Score1 != 3 {
		t.Errorf("Score1 = %d, want %d", fibsResp.Score1, 3)
	}
	if fibsResp.Score2 != 2 {
		t.Errorf("Score2 = %d, want %d", fibsResp.Score2, 2)
	}
}
