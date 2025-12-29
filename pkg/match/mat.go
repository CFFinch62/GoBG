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

// MAT format is the Jellyfish/gnubg match format.
// Example format:
//
//  ; [Site "GamesGrid"]
//  ; [Player 1 "name1"]
//  ; [Player 2 "name2"]
//  7 point match
//
//  Game 1
//  name1 : 0            name2 : 0
//  1) 31: 8/5 6/5       52: 24/22 13/8
//  2) 43: 24/20 13/10   ...

var (
	matchLengthRE = regexp.MustCompile(`(\d+)\s+point\s+match`)
	gameHeaderRE  = regexp.MustCompile(`Game\s+(\d+)`)
	scoreLineRE   = regexp.MustCompile(`^(.+?)\s*:\s*(\d+)\s+(.+?)\s*:\s*(\d+)`)
	moveLineRE    = regexp.MustCompile(`^\s*(\d+)\)`)
	tagRE         = regexp.MustCompile(`\[(\w+)\s+"([^"]+)"\]`)
)

// ImportMAT reads a match from MAT format.
func ImportMAT(r io.Reader) (*Match, error) {
	scanner := bufio.NewScanner(r)
	match := &Match{
		Games: make([]*Game, 0),
	}

	var currentGame *Game
	inGame := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines
		if line == "" {
			continue
		}

		// Parse metadata comments
		if strings.HasPrefix(line, ";") {
			if m := tagRE.FindStringSubmatch(line); m != nil {
				key := strings.ToLower(m[1])
				value := m[2]
				switch key {
				case "player1", "player 1":
					match.Player1 = value
				case "player2", "player 2":
					match.Player2 = value
				case "site", "place":
					match.Place = value
				case "event":
					match.Event = value
				case "date":
					match.Date = value
				case "annotator", "transcriber":
					match.Annotator = value
				}
			}
			continue
		}

		// Parse match length
		if m := matchLengthRE.FindStringSubmatch(line); m != nil {
			match.MatchLength, _ = strconv.Atoi(m[1])
			continue
		}

		// Parse game header
		if m := gameHeaderRE.FindStringSubmatch(line); m != nil {
			// Save previous game if exists
			if currentGame != nil {
				match.Games = append(match.Games, currentGame)
			}
			gameNum, _ := strconv.Atoi(m[1])
			currentGame = &Game{
				Number:    gameNum,
				Actions:   make([]Action, 0),
				CubeValue: 1,
				CubeOwner: -1,
				Winner:    -1,
				Result:    ResultInProgress,
			}
			inGame = true
			continue
		}

		// Parse score line (name : score   name : score)
		if inGame && currentGame != nil {
			if m := scoreLineRE.FindStringSubmatch(line); m != nil {
				if match.Player1 == "" {
					match.Player1 = strings.TrimSpace(m[1])
				}
				if match.Player2 == "" {
					match.Player2 = strings.TrimSpace(m[3])
				}
				currentGame.Score1, _ = strconv.Atoi(m[2])
				currentGame.Score2, _ = strconv.Atoi(m[4])
				continue
			}
		}

		// Parse move lines
		if inGame && currentGame != nil && moveLineRE.MatchString(line) {
			parseMoveLineMAT(line, currentGame)
		}
	}

	// Don't forget the last game
	if currentGame != nil {
		match.Games = append(match.Games, currentGame)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading MAT file: %w", err)
	}

	return match, nil
}

// parseMoveLineMAT parses a single move line in MAT format.
// Format: "1) 31: 8/5 6/5       52: 24/22 13/8"
func parseMoveLineMAT(line string, game *Game) {
	// Remove the move number prefix
	parts := strings.SplitN(line, ")", 2)
	if len(parts) < 2 {
		return
	}
	line = strings.TrimSpace(parts[1])

	// Split into player 1 and player 2 portions
	// This is tricky because whitespace separates them
	// Look for pattern like "   " (multiple spaces) as separator
	halves := regexp.MustCompile(`\s{3,}`).Split(line, 2)

	for playerIdx, half := range halves {
		parsePlayerMoveMAT(strings.TrimSpace(half), playerIdx, game)
	}
}

// parsePlayerMoveMAT parses a single player's roll and move.
// Format: "31: 8/5 6/5" or "Doubles => 2" or "Takes" or "Drops"
func parsePlayerMoveMAT(text string, player int, game *Game) {
	if text == "" {
		return
	}

	text = strings.TrimSpace(text)

	// Handle cube actions
	lowerText := strings.ToLower(text)
	if strings.HasPrefix(lowerText, "doubles") {
		// Parse "Doubles => 2" or "Doubles"
		game.AddDouble(player, game.CubeValue*2)
		return
	}
	if lowerText == "takes" {
		game.AddTake(player)
		game.CubeValue *= 2
		game.CubeOwner = player
		return
	}
	if lowerText == "drops" || lowerText == "passes" {
		game.AddPass(player)
		game.Winner = 1 - player
		game.Result = ResultDrop
		return
	}

	// Parse roll and move: "31: 8/5 6/5"
	colonIdx := strings.Index(text, ":")
	if colonIdx == -1 {
		return
	}

	diceStr := strings.TrimSpace(text[:colonIdx])
	moveStr := strings.TrimSpace(text[colonIdx+1:])

	// Parse dice (e.g., "31" or "66")
	if len(diceStr) >= 2 {
		die1, _ := strconv.Atoi(string(diceStr[0]))
		die2, _ := strconv.Atoi(string(diceStr[1]))
		if die1 >= 1 && die1 <= 6 && die2 >= 1 && die2 <= 6 {
			game.AddRoll(player, die1, die2)
		}
	}

	// Parse move (e.g., "8/5 6/5" or "24/22 13/8")
	if moveStr != "" && !strings.Contains(strings.ToLower(moveStr), "cannot") {
		move, ok := parseMoveNotation(moveStr, player)
		if ok {
			game.AddMove(player, move)
		}
	}
}

// parseMoveNotation parses backgammon move notation like "8/5 6/5" or "24/22(2)".
// Returns the move and true if parsing succeeded.
func parseMoveNotation(notation string, player int) (engine.Move, bool) {
	move := engine.Move{
		From: [4]int8{-1, -1, -1, -1},
		To:   [4]int8{-1, -1, -1, -1},
	}

	// Split by spaces to get individual checker moves
	parts := strings.Fields(notation)

	moveIdx := 0
	for _, part := range parts {
		if moveIdx >= 4 {
			break // Maximum 4 partial moves
		}

		// Handle notation like "8/5" or "24/22" or "bar/22" or "6/off"
		// Also handle "8/5(2)" meaning move 2 checkers
		count := 1
		if idx := strings.Index(part, "("); idx != -1 {
			endIdx := strings.Index(part, ")")
			if endIdx > idx {
				count, _ = strconv.Atoi(part[idx+1 : endIdx])
			}
			part = part[:idx]
		}

		// Split by "/" to get from/to
		fromTo := strings.Split(part, "/")
		if len(fromTo) != 2 {
			continue
		}

		from := parsePoint(fromTo[0], player)
		to := parsePoint(fromTo[1], player)

		for i := 0; i < count && moveIdx < 4; i++ {
			move.From[moveIdx] = int8(from)
			move.To[moveIdx] = int8(to)
			moveIdx++
		}
	}

	return move, moveIdx > 0
}

// parsePoint converts point notation to internal point number.
// "bar" = 25 (for player 0) or 0 (for player 1)
// "off" = 0 (for player 0) or 25 (for player 1)
// Numbers are from player's perspective
func parsePoint(s string, player int) int {
	s = strings.ToLower(strings.TrimSpace(s))

	if s == "bar" {
		if player == 0 {
			return 25
		}
		return 0
	}

	if s == "off" || s == "home" {
		if player == 0 {
			return 0
		}
		return 25
	}

	point, err := strconv.Atoi(s)
	if err != nil {
		return -1
	}

	// Convert from player's perspective to internal representation
	// Player 0 moves from 24 to 1 (high to low)
	// Player 1 moves from 1 to 24 (low to high)
	if player == 1 {
		point = 25 - point
	}

	return point
}

// ExportMAT writes a match in MAT format.
func ExportMAT(w io.Writer, match *Match) error {
	// Write metadata
	if match.Place != "" {
		fmt.Fprintf(w, " ; [Site \"%s\"]\n", match.Place)
	}
	if match.Event != "" {
		fmt.Fprintf(w, " ; [Event \"%s\"]\n", match.Event)
	}
	if match.Date != "" {
		fmt.Fprintf(w, " ; [Date \"%s\"]\n", match.Date)
	}
	fmt.Fprintf(w, " ; [Player 1 \"%s\"]\n", match.Player1)
	fmt.Fprintf(w, " ; [Player 2 \"%s\"]\n", match.Player2)
	if match.Annotator != "" {
		fmt.Fprintf(w, " ; [Annotator \"%s\"]\n", match.Annotator)
	}

	// Write match length
	if match.MatchLength > 0 {
		fmt.Fprintf(w, " %d point match\n\n", match.MatchLength)
	} else {
		fmt.Fprintf(w, " Unlimited match\n\n")
	}

	// Write each game
	for _, game := range match.Games {
		if err := exportGameMAT(w, match, game); err != nil {
			return err
		}
	}

	return nil
}

// exportGameMAT writes a single game in MAT format.
func exportGameMAT(w io.Writer, match *Match, game *Game) error {
	fmt.Fprintf(w, " Game %d\n", game.Number)
	fmt.Fprintf(w, " %s : %d                          %s : %d\n",
		match.Player1, game.Score1, match.Player2, game.Score2)

	moveNum := 0
	player := 0
	var currentRoll [2]int

	for _, action := range game.Actions {
		switch action.Type {
		case ActionRoll:
			if action.Player == 0 {
				moveNum++
				fmt.Fprintf(w, "%3d) ", moveNum)
			}
			currentRoll = action.Dice
			player = action.Player

		case ActionMove:
			moveStr := formatMoveMAT(action.Move, player)
			fmt.Fprintf(w, "%d%d: %s", currentRoll[0], currentRoll[1], moveStr)
			if player == 0 {
				fmt.Fprintf(w, "                    ")
			} else {
				fmt.Fprintf(w, "\n")
			}

		case ActionDouble:
			if player == 0 {
				fmt.Fprintf(w, "     Doubles => %d                    ", action.Value)
			} else {
				fmt.Fprintf(w, "Doubles => %d\n", action.Value)
			}

		case ActionTake:
			if player == 0 {
				fmt.Fprintf(w, "     Takes                    ")
			} else {
				fmt.Fprintf(w, "Takes\n")
			}

		case ActionPass:
			if player == 0 {
				fmt.Fprintf(w, "     Drops                    ")
			} else {
				fmt.Fprintf(w, "Drops\n")
			}
		}
	}

	fmt.Fprintf(w, "\n")
	return nil
}

// formatMoveMAT formats a move in MAT notation.
func formatMoveMAT(move engine.Move, player int) string {
	var parts []string

	for i := 0; i < 4; i++ {
		if move.From[i] < 0 {
			break
		}

		from := formatPointMAT(int(move.From[i]), player)
		to := formatPointMAT(int(move.To[i]), player)
		parts = append(parts, fmt.Sprintf("%s/%s", from, to))
	}

	return strings.Join(parts, " ")
}

// formatPointMAT formats a point number for MAT output.
func formatPointMAT(point int, player int) string {
	// Convert from internal to player's perspective
	if player == 1 {
		point = 25 - point
	}

	if point == 25 || point == 24 {
		return "bar"
	}
	if point <= 0 {
		return "off"
	}
	return strconv.Itoa(point)
}
