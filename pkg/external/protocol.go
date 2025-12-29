// Package external implements gnubg's external player protocol.
// This allows the engine to communicate with other backgammon programs
// via a TCP socket using FIBS board format.
//
// Protocol overview:
// - Server listens on a TCP port
// - Client connects and sends commands
// - Commands include: evaluation, fibsboard, version, exit
// - Positions are sent in FIBS board format
// - Responses include best move or evaluation results
package external

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/yourusername/bgengine/pkg/engine"
)

// Server implements the external player protocol server.
type Server struct {
	engine   *engine.Engine
	listener net.Listener
	mu       sync.Mutex
	running  bool
	options  ServerOptions
}

// ServerOptions configures the external player server.
type ServerOptions struct {
	Port          int  // TCP port to listen on
	Cubeful       bool // Use cubeful evaluation
	Plies         int  // Search depth
	Deterministic bool // Use deterministic evaluation
	JacobyRule    bool // Jacoby rule for money games
	CrawfordRule  bool // Crawford rule for match play
	AllowBeavers  bool // Allow beavers
	PromptEnabled bool // Send prompts after responses
}

// DefaultServerOptions returns sensible defaults.
func DefaultServerOptions() ServerOptions {
	return ServerOptions{
		Port:          1234,
		Cubeful:       true,
		Plies:         2,
		Deterministic: true,
		JacobyRule:    true,
		CrawfordRule:  true,
		AllowBeavers:  false,
		PromptEnabled: true,
	}
}

// NewServer creates a new external player server.
func NewServer(eng *engine.Engine, opts ServerOptions) *Server {
	return &Server{
		engine:  eng,
		options: opts,
	}
}

// Start begins listening for connections.
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("server already running")
	}

	addr := fmt.Sprintf(":%d", s.options.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	s.listener = listener
	s.running = true

	go s.acceptLoop()

	return nil
}

// Stop stops the server.
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.running = false
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

// acceptLoop accepts incoming connections.
func (s *Server) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			s.mu.Lock()
			running := s.running
			s.mu.Unlock()
			if !running {
				return // Server stopped
			}
			continue
		}

		go s.handleConnection(conn)
	}
}

// handleConnection handles a single client connection.
func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// Send initial prompt if enabled
	if s.options.PromptEnabled {
		conn.Write([]byte("> "))
	}

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				// Log error
			}
			return
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		response := s.processCommand(line)
		conn.Write([]byte(response))

		if s.options.PromptEnabled {
			conn.Write([]byte("> "))
		}

		// Check for exit command
		if strings.ToLower(line) == "exit" || strings.ToLower(line) == "quit" {
			return
		}
	}
}

// processCommand processes a single command and returns the response.
func (s *Server) processCommand(cmd string) string {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return "Error: empty command\n"
	}

	command := strings.ToLower(parts[0])

	switch command {
	case "version":
		return "bgengine external player protocol 1.0\n"

	case "help":
		return s.helpResponse()

	case "exit", "quit":
		return "Goodbye\n"

	case "set":
		return s.handleSet(parts[1:])

	case "evaluation", "eval":
		return s.handleEvaluation(cmd)

	case "fibsboard", "board":
		return s.handleFIBSBoard(cmd)

	default:
		// Try to parse as FIBS board directly
		if strings.HasPrefix(cmd, "board:") {
			return s.handleFIBSBoard(cmd)
		}
		return fmt.Sprintf("Error: unknown command '%s'\n", command)
	}
}

// helpResponse returns help text.
func (s *Server) helpResponse() string {
	return `Available commands:
  version     - Show version information
  help        - Show this help
  set <opt>   - Set option (plies, cubeful, jacoby, crawford)
  evaluation  - Evaluate a position (with FIBS board)
  fibsboard   - Get best move for a position
  exit        - Close connection
`
}

// handleSet handles the set command.
func (s *Server) handleSet(args []string) string {
	if len(args) < 2 {
		return "Error: set requires option and value\n"
	}

	option := strings.ToLower(args[0])
	value := args[1]

	switch option {
	case "plies":
		plies, err := strconv.Atoi(value)
		if err != nil || plies < 0 || plies > 4 {
			return "Error: plies must be 0-4\n"
		}
		s.options.Plies = plies
		return fmt.Sprintf("plies set to %d\n", plies)

	case "cubeful":
		s.options.Cubeful = value == "on" || value == "true" || value == "1"
		return fmt.Sprintf("cubeful set to %v\n", s.options.Cubeful)

	case "jacoby":
		s.options.JacobyRule = value == "on" || value == "true" || value == "1"
		return fmt.Sprintf("jacoby set to %v\n", s.options.JacobyRule)

	case "crawford":
		s.options.CrawfordRule = value == "on" || value == "true" || value == "1"
		return fmt.Sprintf("crawford set to %v\n", s.options.CrawfordRule)

	default:
		return fmt.Sprintf("Error: unknown option '%s'\n", option)
	}
}

// handleEvaluation handles the evaluation command.
// Returns equity and win probabilities for a position.
func (s *Server) handleEvaluation(cmd string) string {
	// Extract FIBS board from command
	boardStart := strings.Index(cmd, "board:")
	if boardStart < 0 {
		return "Error: no board specified\n"
	}

	fb, err := ParseFIBSBoard(cmd[boardStart:])
	if err != nil {
		return fmt.Sprintf("Error: %v\n", err)
	}

	state := fb.ToGameState()

	// Evaluate the position
	eval, err := s.engine.Evaluate(state)
	if err != nil {
		return fmt.Sprintf("Error: %v\n", err)
	}

	// Format response
	return fmt.Sprintf("%.6f %.6f %.6f %.6f %.6f %.6f\n",
		eval.Equity, eval.WinProb, eval.WinG, eval.WinBG, eval.LoseG, eval.LoseBG)
}

// handleFIBSBoard handles the fibsboard command.
// Returns the best move for a position.
func (s *Server) handleFIBSBoard(cmd string) string {
	// Extract FIBS board from command
	boardStart := strings.Index(cmd, "board:")
	if boardStart < 0 {
		// Maybe the whole command is the board
		if strings.Contains(cmd, ":") {
			boardStart = 0
		} else {
			return "Error: no board specified\n"
		}
	}

	fb, err := ParseFIBSBoard(cmd[boardStart:])
	if err != nil {
		return fmt.Sprintf("Error: %v\n", err)
	}

	state := fb.ToGameState()

	// Check if this is a cube decision
	if fb.Doubled {
		// Opponent has doubled - should we take?
		decision, err := s.engine.AnalyzeCube(state)
		if err != nil {
			return fmt.Sprintf("Error: %v\n", err)
		}

		if decision.Decision.Action == engine.Take {
			return "take\n"
		}
		return "drop\n"
	}

	// Check if we should double
	if fb.CanDouble && fb.Dice[0] == 0 {
		decision, err := s.engine.AnalyzeCube(state)
		if err != nil {
			return fmt.Sprintf("Error: %v\n", err)
		}

		if decision.Decision.Action == engine.Double || decision.Decision.Action == engine.Redouble {
			return "double\n"
		}
	}

	// Find best move
	if fb.Dice[0] == 0 || fb.Dice[1] == 0 {
		return "Error: no dice rolled\n"
	}

	analysis, err := s.engine.AnalyzePosition(state, fb.Dice)
	if err != nil {
		return fmt.Sprintf("Error: %v\n", err)
	}

	if analysis.NumMoves == 0 {
		return "cannot move\n"
	}

	// Format the best move
	moveStr := FormatMove(analysis.BestMove, fb.Direction)
	return moveStr + "\n"
}
