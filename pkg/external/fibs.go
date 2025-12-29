package external

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/yourusername/bgengine/pkg/engine"
)

// FIBSBoard represents a parsed FIBS board string.
// See: http://www.fibs.com/fibs_interface.html#board_state
type FIBSBoard struct {
	Player1      string  // Your name
	Player2      string  // Opponent's name
	MatchLength  int     // Match length (0 = unlimited)
	Score1       int     // Your score
	Score2       int     // Opponent's score
	Board        [26]int // Board positions (-n = opponent, +n = you)
	Turn         int     // Whose turn (1 = you, -1 = opponent)
	Dice         [2]int  // Your dice (0,0 if not rolled)
	OppDice      [2]int  // Opponent's dice
	Cube         int     // Cube value
	CanDouble    bool    // Can you double?
	OppCanDouble bool    // Can opponent double?
	Doubled      bool    // Has opponent doubled?
	Color        int     // Your color (1 or -1)
	Direction    int     // Your direction (1 or -1)
	Crawford     bool    // Is this Crawford game?
}

// ParseFIBSBoard parses a FIBS board string.
// Format: board:player1:player2:matchlen:score1:score2:board[26]:turn:dice[4]:cube:...
func ParseFIBSBoard(s string) (*FIBSBoard, error) {
	// Remove "board:" prefix if present
	if strings.HasPrefix(s, "board:") {
		s = s[6:]
	}

	parts := strings.Split(s, ":")
	if len(parts) < 32 {
		return nil, fmt.Errorf("invalid FIBS board: expected at least 32 fields, got %d", len(parts))
	}

	fb := &FIBSBoard{}

	fb.Player1 = parts[0]
	fb.Player2 = parts[1]
	fb.MatchLength, _ = strconv.Atoi(parts[2])
	fb.Score1, _ = strconv.Atoi(parts[3])
	fb.Score2, _ = strconv.Atoi(parts[4])

	// Parse board (26 positions)
	for i := 0; i < 26; i++ {
		fb.Board[i], _ = strconv.Atoi(parts[5+i])
	}

	fb.Turn, _ = strconv.Atoi(parts[31])

	// Parse dice
	if len(parts) > 35 {
		fb.Dice[0], _ = strconv.Atoi(parts[32])
		fb.Dice[1], _ = strconv.Atoi(parts[33])
		fb.OppDice[0], _ = strconv.Atoi(parts[34])
		fb.OppDice[1], _ = strconv.Atoi(parts[35])
	}

	// Parse cube info
	if len(parts) > 36 {
		fb.Cube, _ = strconv.Atoi(parts[36])
	}
	if len(parts) > 37 {
		fb.CanDouble = parts[37] == "1"
	}
	if len(parts) > 38 {
		fb.OppCanDouble = parts[38] == "1"
	}
	if len(parts) > 39 {
		fb.Doubled = parts[39] == "1"
	}
	if len(parts) > 40 {
		fb.Color, _ = strconv.Atoi(parts[40])
	}
	if len(parts) > 41 {
		fb.Direction, _ = strconv.Atoi(parts[41])
	}

	return fb, nil
}

// ToGameState converts a FIBS board to an engine GameState.
func (fb *FIBSBoard) ToGameState() *engine.GameState {
	state := &engine.GameState{
		CubeValue:   fb.Cube,
		MatchLength: fb.MatchLength,
		Score:       [2]int{fb.Score1, fb.Score2},
		Dice:        fb.Dice,
	}

	// Convert FIBS board to engine board
	// FIBS: positive = your checkers, negative = opponent's
	// Engine: Board[0] = player 0, Board[1] = player 1
	// Note: FIBS has 26 positions but engine Board is [2][25]
	for i := 0; i < 25; i++ {
		if fb.Board[i] > 0 {
			state.Board[0][i] = uint8(fb.Board[i])
		} else if fb.Board[i] < 0 {
			if 24-i >= 0 {
				state.Board[1][24-i] = uint8(-fb.Board[i])
			}
		}
	}

	// Determine turn
	if fb.Turn == 1 {
		state.Turn = 0
	} else {
		state.Turn = 1
	}

	// Determine cube owner
	if fb.CanDouble && fb.OppCanDouble {
		state.CubeOwner = -1 // Centered
	} else if fb.CanDouble {
		state.CubeOwner = 0
	} else {
		state.CubeOwner = 1
	}

	return state
}

// FormatMove formats a move in FIBS notation.
func FormatMove(move engine.Move, direction int) string {
	var parts []string

	for i := 0; i < 4; i++ {
		if move.From[i] < 0 {
			break
		}

		from := int(move.From[i])
		to := int(move.To[i])

		// Convert to FIBS notation
		fromStr := formatFIBSPoint(from, direction)
		toStr := formatFIBSPoint(to, direction)

		parts = append(parts, fmt.Sprintf("%s/%s", fromStr, toStr))
	}

	return strings.Join(parts, " ")
}

// formatFIBSPoint formats a point for FIBS output.
func formatFIBSPoint(point int, direction int) string {
	if point == 24 || point == 25 {
		return "bar"
	}
	if point < 0 {
		return "off"
	}
	return strconv.Itoa(point + 1)
}
