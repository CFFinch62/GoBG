package api

import (
	"bytes"
	"encoding/json"
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
