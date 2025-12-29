package api

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/yourusername/bgengine/internal/positionid"
	"github.com/yourusername/bgengine/pkg/engine"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins - configure properly in production
	},
}

// WSMessage is a generic WebSocket message.
type WSMessage struct {
	Type    string          `json:"type"`    // Message type: "evaluate", "move", "cube", "ping"
	ID      string          `json:"id"`      // Request ID for correlating responses
	Payload json.RawMessage `json:"payload"` // Type-specific payload
}

// WSResponse is a generic WebSocket response.
type WSResponse struct {
	Type    string      `json:"type"`              // Response type: "result", "error", "pong"
	ID      string      `json:"id,omitempty"`      // Request ID
	Payload interface{} `json:"payload,omitempty"` // Response data
	Error   string      `json:"error,omitempty"`   // Error message if any
}

// WSClient represents a connected WebSocket client.
type WSClient struct {
	conn     *websocket.Conn
	handlers *Handlers
	sendChan chan WSResponse
	mu       sync.Mutex
}

// WebSocket handles WebSocket connections for real-time game analysis.
func (h *Handlers) WebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	client := &WSClient{conn: conn, handlers: h, sendChan: make(chan WSResponse, 256)}
	go client.writePump()
	client.readPump()
}

func (c *WSClient) writePump() {
	defer c.conn.Close()
	for msg := range c.sendChan {
		if err := c.conn.WriteJSON(msg); err != nil {
			return
		}
	}
}

func (c *WSClient) readPump() {
	defer func() { close(c.sendChan); c.conn.Close() }()
	for {
		var msg WSMessage
		if err := c.conn.ReadJSON(&msg); err != nil {
			return
		}
		c.handleMessage(msg)
	}
}

func (c *WSClient) handleMessage(msg WSMessage) {
	switch msg.Type {
	case "evaluate":
		c.handleEvaluate(msg)
	case "move":
		c.handleMove(msg)
	case "cube":
		c.handleCube(msg)
	case "ping":
		c.sendChan <- WSResponse{Type: "pong", ID: msg.ID}
	default:
		c.sendChan <- WSResponse{Type: "error", ID: msg.ID, Error: "unknown message type"}
	}
}

func (c *WSClient) handleEvaluate(msg WSMessage) {
	var req EvaluateRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		c.sendChan <- WSResponse{Type: "error", ID: msg.ID, Error: "invalid payload"}
		return
	}
	board, err := positionid.BoardFromPositionID(req.Position)
	if err != nil {
		c.sendChan <- WSResponse{Type: "error", ID: msg.ID, Error: "invalid position"}
		return
	}
	gs := &engine.GameState{
		Board: engine.Board(board), Turn: 0, CubeValue: 1, CubeOwner: -1,
		MatchLength: req.MatchLength, Score: req.Score, Crawford: req.Crawford,
	}
	eval, err := c.handlers.engine.Evaluate(gs)
	if err != nil {
		c.sendChan <- WSResponse{Type: "error", ID: msg.ID, Error: "evaluation failed"}
		return
	}
	c.sendChan <- WSResponse{Type: "result", ID: msg.ID, Payload: EvalToResponse(eval, req.Ply, false)}
}

func (c *WSClient) handleMove(msg WSMessage) {
	var req MoveRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		c.sendChan <- WSResponse{Type: "error", ID: msg.ID, Error: "invalid payload"}
		return
	}
	if req.Dice[0] < 1 || req.Dice[0] > 6 || req.Dice[1] < 1 || req.Dice[1] > 6 {
		c.sendChan <- WSResponse{Type: "error", ID: msg.ID, Error: "invalid dice"}
		return
	}
	board, err := positionid.BoardFromPositionID(req.Position)
	if err != nil {
		c.sendChan <- WSResponse{Type: "error", ID: msg.ID, Error: "invalid position"}
		return
	}
	gs := &engine.GameState{
		Board: engine.Board(board), Turn: 0, CubeValue: 1, CubeOwner: -1,
		Dice: req.Dice, MatchLength: req.MatchLength, Score: req.Score, Crawford: req.Crawford,
	}
	analysis, err := c.handlers.engine.AnalyzePosition(gs, req.Dice)
	if err != nil {
		c.sendChan <- WSResponse{Type: "error", ID: msg.ID, Error: "analysis failed"}
		return
	}
	numMoves := req.NumMoves
	if numMoves <= 0 || numMoves > len(analysis.Moves) {
		numMoves = len(analysis.Moves)
	}
	moves := make([]MoveResponse, numMoves)
	for i := 0; i < numMoves; i++ {
		m := analysis.Moves[i]
		winProb, winG := 0.0, 0.0
		if m.Eval != nil {
			winProb = m.Eval.WinProb * 100
			winG = m.Eval.WinG * 100
		}
		moves[i] = MoveResponse{Move: formatMove(m.Move), Equity: m.Equity, Win: winProb, WinG: winG}
	}
	c.sendChan <- WSResponse{Type: "result", ID: msg.ID, Payload: MovesResponse{Moves: moves, NumLegal: analysis.NumMoves, Dice: req.Dice, Position: req.Position}}
}

func (c *WSClient) handleCube(msg WSMessage) {
	var req CubeRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		c.sendChan <- WSResponse{Type: "error", ID: msg.ID, Error: "invalid payload"}
		return
	}
	board, err := positionid.BoardFromPositionID(req.Position)
	if err != nil {
		c.sendChan <- WSResponse{Type: "error", ID: msg.ID, Error: "invalid position"}
		return
	}
	cubeValue := req.CubeValue
	if cubeValue <= 0 {
		cubeValue = 1
	}
	gs := &engine.GameState{
		Board: engine.Board(board), Turn: 0, CubeValue: cubeValue, CubeOwner: req.CubeOwner,
		MatchLength: req.MatchLength, Score: req.Score, Crawford: req.Crawford,
	}
	analysis, err := c.handlers.engine.AnalyzeCube(gs)
	if err != nil {
		c.sendChan <- WSResponse{Type: "error", ID: msg.ID, Error: "cube analysis failed"}
		return
	}
	action := "no_double"
	switch analysis.Decision.Action {
	case engine.Double:
		action = "double_take"
	case engine.Redouble:
		action = "redouble_take"
	case engine.Take:
		action = "take"
	case engine.Pass:
		action = "pass"
	}
	c.sendChan <- WSResponse{Type: "result", ID: msg.ID, Payload: CubeResponse{
		Action: action, DoubleEquity: analysis.Decision.DoubleEquity,
		NoDoubleEquity: analysis.Decision.NoDoubleEquity, TakeEquity: analysis.Decision.TakeEquity,
		DoubleDiff: analysis.Decision.DoubleEquity - analysis.Decision.NoDoubleEquity,
	}}
}
