package match

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/yourusername/bgengine/pkg/engine"
)

// SGF (Smart Game Format) is a standard format for recording games.
// See: https://www.red-bean.com/sgf/backgammon.html
//
// Example SGF:
// (;FF[4]GM[6]AP[gnubg:1.06.002]
//  PW[Player1]PB[Player2]
//  MI[length:7][game:0][ws:0][bs:0]
//  ;B[31]
//  ;W[52]
//  ...)

var (
	sgfPropertyRE = regexp.MustCompile(`([A-Z]+)\[([^\]]*)\]`)
)

// ImportSGF reads a match from SGF format.
func ImportSGF(r io.Reader) (*Match, error) {
	scanner := bufio.NewScanner(r)
	var content strings.Builder

	for scanner.Scan() {
		content.WriteString(scanner.Text())
		content.WriteString("\n")
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading SGF file: %w", err)
	}

	return parseSGF(content.String())
}

// parseSGF parses SGF content into a Match.
func parseSGF(content string) (*Match, error) {
	match := &Match{
		Games: make([]*Game, 0),
	}

	// Find all game trees (each starts with '(')
	games := splitSGFGames(content)

	for i, gameContent := range games {
		game, err := parseSGFGame(gameContent, i+1)
		if err != nil {
			return nil, fmt.Errorf("parsing game %d: %w", i+1, err)
		}

		// Extract match info from first game
		if i == 0 {
			extractMatchInfo(gameContent, match)
		}

		match.Games = append(match.Games, game)
	}

	return match, nil
}

// splitSGFGames splits SGF content into individual game trees.
func splitSGFGames(content string) []string {
	var games []string
	depth := 0
	start := -1

	for i, ch := range content {
		if ch == '(' {
			if depth == 0 {
				start = i
			}
			depth++
		} else if ch == ')' {
			depth--
			if depth == 0 && start >= 0 {
				games = append(games, content[start:i+1])
				start = -1
			}
		}
	}

	return games
}

// extractMatchInfo extracts match-level info from SGF properties.
func extractMatchInfo(content string, match *Match) {
	props := parseSGFProperties(content)

	if pw, ok := props["PW"]; ok {
		match.Player1 = pw
	}
	if pb, ok := props["PB"]; ok {
		match.Player2 = pb
	}
	if dt, ok := props["DT"]; ok {
		match.Date = dt
	}
	if ev, ok := props["EV"]; ok {
		match.Event = ev
	}
	if pc, ok := props["PC"]; ok {
		match.Place = pc
	}
	if an, ok := props["AN"]; ok {
		match.Annotator = an
	}

	// Parse MI (Match Information) property
	if mi, ok := props["MI"]; ok {
		miParts := strings.Split(mi, "][")
		for _, part := range miParts {
			kv := strings.SplitN(part, ":", 2)
			if len(kv) == 2 {
				switch kv[0] {
				case "length":
					match.MatchLength, _ = strconv.Atoi(kv[1])
				}
			}
		}
	}
}

// parseSGFProperties extracts all properties from SGF content.
func parseSGFProperties(content string) map[string]string {
	props := make(map[string]string)

	matches := sgfPropertyRE.FindAllStringSubmatch(content, -1)
	for _, m := range matches {
		if len(m) >= 3 {
			props[m[1]] = m[2]
		}
	}

	return props
}

// parseSGFGame parses a single SGF game tree.
func parseSGFGame(content string, gameNum int) (*Game, error) {
	game := NewGame(gameNum, 0, 0, false)

	// Split into nodes (separated by ';')
	nodes := strings.Split(content, ";")

	for _, node := range nodes[1:] { // Skip first empty element
		if err := parseSGFNode(node, game); err != nil {
			return nil, err
		}
	}

	return game, nil
}

// parseSGFNode parses a single SGF node and adds actions to the game.
func parseSGFNode(node string, game *Game) error {
	props := parseSGFProperties(node)

	// Parse dice roll (B or W property with dice)
	// Format: B[31] means Black rolled 3-1
	if dice, ok := props["B"]; ok && len(dice) >= 2 {
		die1, _ := strconv.Atoi(string(dice[0]))
		die2, _ := strconv.Atoi(string(dice[1]))
		if die1 >= 1 && die1 <= 6 && die2 >= 1 && die2 <= 6 {
			game.AddRoll(1, die1, die2) // Black = player 1
		}
		// Parse move if present (rest of the value)
		if len(dice) > 2 {
			move := parseSGFMove(dice[2:], 1)
			game.AddMove(1, move)
		}
	}

	if dice, ok := props["W"]; ok && len(dice) >= 2 {
		die1, _ := strconv.Atoi(string(dice[0]))
		die2, _ := strconv.Atoi(string(dice[1]))
		if die1 >= 1 && die1 <= 6 && die2 >= 1 && die2 <= 6 {
			game.AddRoll(0, die1, die2) // White = player 0
		}
		if len(dice) > 2 {
			move := parseSGFMove(dice[2:], 0)
			game.AddMove(0, move)
		}
	}

	// Parse cube actions
	if _, ok := props["D"]; ok {
		// Double offered
		game.AddDouble(game.CubeOwner, game.CubeValue*2)
	}
	if _, ok := props["T"]; ok {
		// Take
		game.AddTake(1 - game.CubeOwner)
	}
	if _, ok := props["P"]; ok {
		// Pass (drop)
		game.AddPass(1 - game.CubeOwner)
	}

	return nil
}

// parseSGFMove parses SGF move notation.
// SGF uses letters for points: a=1, b=2, ..., x=24, y=bar, z=off
func parseSGFMove(moveStr string, player int) engine.Move {
	move := engine.Move{
		From: [4]int8{-1, -1, -1, -1},
		To:   [4]int8{-1, -1, -1, -1},
	}

	// Each move is 2 characters (from, to)
	moveIdx := 0
	for i := 0; i+1 < len(moveStr) && moveIdx < 4; i += 2 {
		from := sgfPointToInt(moveStr[i], player)
		to := sgfPointToInt(moveStr[i+1], player)
		if from >= 0 && to >= -1 {
			move.From[moveIdx] = int8(from)
			move.To[moveIdx] = int8(to)
			moveIdx++
		}
	}

	return move
}

// sgfPointToInt converts SGF point notation to internal point number.
func sgfPointToInt(ch byte, player int) int {
	if ch == 'y' {
		// Bar
		return 24
	}
	if ch == 'z' {
		// Bear off
		return -1
	}
	if ch >= 'a' && ch <= 'x' {
		point := int(ch - 'a' + 1)
		// Convert to internal representation
		if player == 0 {
			return 25 - point
		}
		return point - 1
	}
	return -2 // Invalid
}

// ExportSGF writes a match in SGF format.
func ExportSGF(w io.Writer, match *Match) error {
	for _, game := range match.Games {
		if err := exportGameSGF(w, match, game); err != nil {
			return err
		}
	}
	return nil
}

// exportGameSGF writes a single game in SGF format.
func exportGameSGF(w io.Writer, match *Match, game *Game) error {
	// Write game tree header
	fmt.Fprintf(w, "(;FF[4]GM[6]AP[bgengine:1.0]\n")

	// Write player names
	fmt.Fprintf(w, "PW[%s]PB[%s]\n", match.Player1, match.Player2)

	// Write match info
	fmt.Fprintf(w, "MI[length:%d][game:%d][ws:%d][bs:%d]\n",
		match.MatchLength, game.Number-1, game.Score1, game.Score2)

	// Write date if available
	if match.Date != "" {
		fmt.Fprintf(w, "DT[%s]\n", match.Date)
	}
	if match.Event != "" {
		fmt.Fprintf(w, "EV[%s]\n", match.Event)
	}
	if match.Place != "" {
		fmt.Fprintf(w, "PC[%s]\n", match.Place)
	}

	// Write game actions
	var currentRoll [2]int
	var currentPlayer int

	for _, action := range game.Actions {
		switch action.Type {
		case ActionRoll:
			currentRoll = action.Dice
			currentPlayer = action.Player

		case ActionMove:
			playerChar := "W"
			if currentPlayer == 1 {
				playerChar = "B"
			}
			moveStr := formatMoveSGF(action.Move, currentPlayer)
			fmt.Fprintf(w, ";%s[%d%d%s]\n", playerChar, currentRoll[0], currentRoll[1], moveStr)

		case ActionDouble:
			fmt.Fprintf(w, ";D[]\n")

		case ActionTake:
			fmt.Fprintf(w, ";T[]\n")

		case ActionPass:
			fmt.Fprintf(w, ";P[]\n")
		}
	}

	// Close game tree
	fmt.Fprintf(w, ")\n")
	return nil
}

// formatMoveSGF formats a move in SGF notation.
func formatMoveSGF(move engine.Move, player int) string {
	var result strings.Builder

	for i := 0; i < 4; i++ {
		if move.From[i] < 0 {
			break
		}
		result.WriteByte(intToSGFPoint(int(move.From[i]), player))
		result.WriteByte(intToSGFPoint(int(move.To[i]), player))
	}

	return result.String()
}

// intToSGFPoint converts internal point number to SGF notation.
func intToSGFPoint(point int, player int) byte {
	if point == 24 || point == 25 {
		return 'y' // Bar
	}
	if point < 0 {
		return 'z' // Bear off
	}

	// Convert from internal to SGF (a=1, b=2, ..., x=24)
	var sgfPoint int
	if player == 0 {
		sgfPoint = 25 - point
	} else {
		sgfPoint = point + 1
	}

	if sgfPoint >= 1 && sgfPoint <= 24 {
		return byte('a' + sgfPoint - 1)
	}
	return 'z'
}
