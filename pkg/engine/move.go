package engine

import (
	"github.com/yourusername/bgengine/internal/positionid"
)

// MaxMoves is the maximum number of legal moves for any position
const MaxMoves = 3060

// MoveList contains all legal moves for a position
type MoveList struct {
	Moves      []Move
	MaxMoves   int                      // Maximum number of moves used (for partial moves)
	MaxPips    int                      // Maximum pips used
	OrigBoard  Board                    // Original board for duplicate detection
	ResultKeys []positionid.PositionKey // Keys of resulting positions
}

// GenerateMoves generates all legal moves for a position given a dice roll.
// n0 and n1 are the two dice values (1-6).
// Returns a MoveList containing all legal moves.
func GenerateMoves(board Board, n0, n1 int) *MoveList {
	ml := &MoveList{
		Moves:      make([]Move, 0, 32), // Pre-allocate for typical case
		ResultKeys: make([]positionid.PositionKey, 0, 32),
		OrigBoard:  board,
	}

	// Set up the roll array (4 elements for doubles)
	anRoll := [4]int{n0, n1, 0, 0}
	if n0 == n1 {
		anRoll[2] = n0
		anRoll[3] = n0
	}

	anMoves := [8]int{-1, -1, -1, -1, -1, -1, -1, -1}

	// Generate moves with dice in original order
	generateMovesSub(ml, anRoll[:], 0, 23, 0, board, anMoves[:], false)

	// If not doubles, also try with dice swapped
	if n0 != n1 {
		anRoll[0], anRoll[1] = anRoll[1], anRoll[0]
		anMoves = [8]int{-1, -1, -1, -1, -1, -1, -1, -1}
		generateMovesSub(ml, anRoll[:], 0, 23, 0, board, anMoves[:], false)
	}

	return ml
}

// generateMovesSub is the recursive move generation function
func generateMovesSub(ml *MoveList, anRoll []int, nMoveDepth int,
	iPip int, cPip int, board Board, anMoves []int, fPartial bool) bool {

	if nMoveDepth > 3 || anRoll[nMoveDepth] == 0 {
		return true
	}

	fUsed := false

	// Check if player has checkers on the bar
	if board[1][24] > 0 {
		// Must enter from bar first
		entryPoint := anRoll[nMoveDepth] - 1
		if board[0][23-entryPoint] >= 2 {
			// Blocked - can't enter
			return !fUsed || fPartial
		}

		anMoves[nMoveDepth*2] = 24
		anMoves[nMoveDepth*2+1] = 24 - anRoll[nMoveDepth]

		// Apply the move
		boardNew := board
		applySubMove(&boardNew, 24, anRoll[nMoveDepth])

		if generateMovesSub(ml, anRoll, nMoveDepth+1, 23, cPip+anRoll[nMoveDepth],
			boardNew, anMoves, fPartial) {
			saveMoves(ml, nMoveDepth+1, cPip+anRoll[nMoveDepth], anMoves, boardNew, fPartial)
		}

		return fPartial
	}

	// Not on bar - try all legal moves
	for i := iPip; i >= 0; i-- {
		if board[1][i] > 0 && legalMove(board, i, anRoll[nMoveDepth]) {
			anMoves[nMoveDepth*2] = i
			anMoves[nMoveDepth*2+1] = i - anRoll[nMoveDepth]

			// Apply the move
			boardNew := board
			applySubMove(&boardNew, i, anRoll[nMoveDepth])

			// For doubles, continue from same point; otherwise from point 23
			nextIPip := 23
			if anRoll[0] == anRoll[1] {
				nextIPip = i
			}

			if generateMovesSub(ml, anRoll, nMoveDepth+1, nextIPip,
				cPip+anRoll[nMoveDepth], boardNew, anMoves, fPartial) {
				saveMoves(ml, nMoveDepth+1, cPip+anRoll[nMoveDepth], anMoves, boardNew, fPartial)
			}

			fUsed = true
		}
	}

	return !fUsed || fPartial
}

// legalMove checks if a move from iSrc with nPips is legal
func legalMove(board Board, iSrc, nPips int) bool {
	iDest := iSrc - nPips

	if iDest >= 0 {
		// Normal move - check if destination is blocked
		return board[0][23-iDest] < 2
	}

	// Bearing off - must have all checkers in home board
	nBack := 24
	for nBack > 0 && board[1][nBack] == 0 {
		nBack--
	}

	// Can bear off if: all checkers in home board (nBack <= 5)
	// AND either exact roll or bearing off from highest point
	return nBack <= 5 && (iSrc == nBack || iDest == -1)
}

// applySubMove applies a single checker move to the board
func applySubMove(board *Board, iSrc, nRoll int) {
	iDest := iSrc - nRoll

	// Remove checker from source
	board[1][iSrc]--

	if iDest < 0 {
		// Bearing off - checker is removed from board
		return
	}

	// Check for hit
	if board[0][23-iDest] == 1 {
		// Hit opponent's blot
		board[0][23-iDest] = 0
		board[0][24]++ // Put on bar
	}

	// Place checker on destination
	board[1][iDest]++
}

// saveMoves saves a completed move to the move list
func saveMoves(ml *MoveList, cMoves int, cPip int, anMoves []int, board Board, fPartial bool) {
	if fPartial {
		// Save all moves, even incomplete ones
		if cMoves > ml.MaxMoves {
			ml.MaxMoves = cMoves
		}
		if cPip > ml.MaxPips {
			ml.MaxPips = cPip
		}
	} else {
		// Only save complete moves (using maximum dice)
		if cMoves < ml.MaxMoves {
			return
		}
		if cMoves > ml.MaxMoves {
			// New maximum - clear previous moves
			ml.Moves = ml.Moves[:0]
			ml.ResultKeys = ml.ResultKeys[:0]
			ml.MaxMoves = cMoves
			ml.MaxPips = cPip
		} else if cPip < ml.MaxPips {
			return
		} else if cPip > ml.MaxPips {
			ml.Moves = ml.Moves[:0]
			ml.ResultKeys = ml.ResultKeys[:0]
			ml.MaxPips = cPip
		}
	}

	// Create the move
	move := Move{
		From: [4]int8{-1, -1, -1, -1},
		To:   [4]int8{-1, -1, -1, -1},
		Hits: 0,
	}

	for i := 0; i < cMoves; i++ {
		move.From[i] = int8(anMoves[i*2])
		move.To[i] = int8(anMoves[i*2+1])
	}

	// Check for duplicate moves (same resulting position)
	key := positionid.MakePositionKey(positionid.Board(board))
	for _, existingKey := range ml.ResultKeys {
		if positionid.EqualKeys(key, existingKey) {
			return // Duplicate - don't add
		}
	}

	ml.Moves = append(ml.Moves, move)
	ml.ResultKeys = append(ml.ResultKeys, key)
}

// ApplyMove applies a move to a board and returns the resulting board
func ApplyMove(board Board, m Move) Board {
	result := board
	for i := 0; i < 4; i++ {
		if m.From[i] < 0 {
			break
		}
		nRoll := int(m.From[i] - m.To[i])
		applySubMove(&result, int(m.From[i]), nRoll)
	}
	return result
}

// CountHits counts the number of hits in a move
func CountHits(board Board, m Move) int8 {
	hits := int8(0)
	tempBoard := board
	for i := 0; i < 4; i++ {
		if m.From[i] < 0 {
			break
		}
		iDest := int(m.To[i])
		if iDest >= 0 && tempBoard[0][23-iDest] == 1 {
			hits++
		}
		nRoll := int(m.From[i] - m.To[i])
		applySubMove(&tempBoard, int(m.From[i]), nRoll)
	}
	return hits
}

// ParseMove parses a move notation string like "8/5 6/5" into a Move.
// Points are 1-indexed (1-24), "bar" for bar, "off" for bearing off.
func ParseMove(notation string) (Move, error) {
	move := Move{
		From: [4]int8{-1, -1, -1, -1},
		To:   [4]int8{-1, -1, -1, -1},
	}

	// Split by spaces to get individual checker moves
	parts := splitFields(notation)
	if len(parts) == 0 {
		return move, nil // Empty move (dancer)
	}

	moveIdx := 0
	for _, part := range parts {
		if moveIdx >= 4 {
			break
		}

		// Handle notation like "8/5(2)" meaning move 2 checkers
		count := 1
		if idx := indexByte(part, '('); idx != -1 {
			endIdx := indexByte(part, ')')
			if endIdx > idx {
				count = atoi(part[idx+1 : endIdx])
			}
			part = part[:idx]
		}

		// Split by "/" to get from/to
		slashIdx := indexByte(part, '/')
		if slashIdx == -1 {
			continue
		}
		fromStr := part[:slashIdx]
		toStr := part[slashIdx+1:]

		from := parsePointNotation(fromStr)
		to := parsePointNotation(toStr)

		for i := 0; i < count && moveIdx < 4; i++ {
			move.From[moveIdx] = int8(from)
			move.To[moveIdx] = int8(to)
			moveIdx++
		}
	}

	return move, nil
}

// splitFields splits a string by whitespace.
func splitFields(s string) []string {
	var result []string
	start := -1
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' || s[i] == '\t' {
			if start >= 0 {
				result = append(result, s[start:i])
				start = -1
			}
		} else {
			if start < 0 {
				start = i
			}
		}
	}
	if start >= 0 {
		result = append(result, s[start:])
	}
	return result
}

// indexByte returns the index of the first occurrence of c in s, or -1.
func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// atoi converts a string to an int, returning 0 on error.
func atoi(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			n = n*10 + int(s[i]-'0')
		}
	}
	return n
}

// parsePointNotation converts point notation to internal index.
// "bar" = 24 (index), "off" = -1, numbers are 1-indexed points.
func parsePointNotation(s string) int {
	// Lowercase comparison
	if len(s) == 3 && (s[0] == 'b' || s[0] == 'B') && (s[1] == 'a' || s[1] == 'A') && (s[2] == 'r' || s[2] == 'R') {
		return 24 // Bar is index 24
	}
	if len(s) == 3 && (s[0] == 'o' || s[0] == 'O') && (s[1] == 'f' || s[1] == 'F') && (s[2] == 'f' || s[2] == 'F') {
		return -1 // Off is -1
	}

	// Parse as number (1-indexed point)
	point := atoi(s)
	if point >= 1 && point <= 24 {
		return point - 1 // Convert to 0-indexed
	}
	return -1
}
