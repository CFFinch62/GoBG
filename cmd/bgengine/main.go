// bgengine - A high-performance backgammon analysis engine
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/yourusername/bgengine/internal/positionid"
	"github.com/yourusername/bgengine/pkg/engine"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "eval":
		cmdEval(args)
	case "move":
		cmdMove(args)
	case "cube":
		cmdCube(args)
	case "rollout":
		cmdRollout(args)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`bgengine - Backgammon Analysis Engine

Usage: bgengine <command> [options]

Commands:
  eval      Evaluate a position
  move      Find the best move for a dice roll
  cube      Analyze cube decisions
  rollout   Monte Carlo rollout

Use "bgengine <command> -h" for command-specific help.

Position ID Format:
  The position is specified using gnubg's position ID format.
  Example: "4HPwATDgc/ABMA:cIkqAAAAAAAA" (position:match)
  Only the position part (before :) is required.`)
}

func parsePosition(posStr string) (*engine.GameState, error) {
	// Handle gnubg format "positionID:matchID" - we only need the position part
	if idx := strings.Index(posStr, ":"); idx >= 0 {
		posStr = posStr[:idx]
	}

	board, err := positionid.BoardFromPositionID(posStr)
	if err != nil {
		return nil, fmt.Errorf("invalid position ID: %w", err)
	}

	return &engine.GameState{
		Board:     engine.Board(board),
		Turn:      0,
		CubeValue: 1,
		CubeOwner: -1,
	}, nil
}

func parseDice(diceStr string) ([2]int, error) {
	parts := strings.Split(diceStr, ",")
	if len(parts) != 2 {
		parts = strings.Split(diceStr, "-")
	}
	if len(parts) != 2 {
		return [2]int{}, fmt.Errorf("dice should be in format '3,1' or '3-1'")
	}

	d1, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	d2, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil || d1 < 1 || d1 > 6 || d2 < 1 || d2 > 6 {
		return [2]int{}, fmt.Errorf("dice values must be 1-6")
	}

	return [2]int{d1, d2}, nil
}

func createEngine() (*engine.Engine, error) {
	e, err := engine.NewEngine(engine.EngineOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create engine: %w", err)
	}
	return e, nil
}

func cmdEval(args []string) {
	fs := flag.NewFlagSet("eval", flag.ExitOnError)
	posFlag := fs.String("position", "", "Position ID (gnubg format)")
	posShort := fs.String("p", "", "Position ID (short form)")
	fs.Parse(args)

	pos := *posFlag
	if pos == "" {
		pos = *posShort
	}
	if pos == "" {
		fmt.Fprintln(os.Stderr, "Error: position required")
		fmt.Fprintln(os.Stderr, "Usage: bgengine eval -position <positionID>")
		os.Exit(1)
	}

	state, err := parsePosition(pos)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	e, err := createEngine()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	eval, err := e.Evaluate(state)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error evaluating position: %v\n", err)
		os.Exit(1)
	}

	printEvaluation(eval)
}

func printEvaluation(eval *engine.Evaluation) {
	fmt.Printf("Equity: %+.3f\n", eval.Equity)
	fmt.Printf("  Win:    %.1f%% (G: %.1f%%, BG: %.1f%%)\n",
		eval.WinProb*100, eval.WinG*100, eval.WinBG*100)
	fmt.Printf("  Lose:   %.1f%% (G: %.1f%%, BG: %.1f%%)\n",
		(1-eval.WinProb)*100, eval.LoseG*100, eval.LoseBG*100)
}

func cmdMove(args []string) {
	fs := flag.NewFlagSet("move", flag.ExitOnError)
	posFlag := fs.String("position", "", "Position ID (gnubg format)")
	posShort := fs.String("p", "", "Position ID (short form)")
	diceFlag := fs.String("dice", "", "Dice roll (e.g., 3,1 or 3-1)")
	diceShort := fs.String("d", "", "Dice roll (short form)")
	numMoves := fs.Int("n", 5, "Number of moves to show")
	fs.Parse(args)

	pos := *posFlag
	if pos == "" {
		pos = *posShort
	}
	dice := *diceFlag
	if dice == "" {
		dice = *diceShort
	}

	if pos == "" || dice == "" {
		fmt.Fprintln(os.Stderr, "Error: position and dice required")
		fmt.Fprintln(os.Stderr, "Usage: bgengine move -position <positionID> -dice <roll>")
		os.Exit(1)
	}

	state, err := parsePosition(pos)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	diceRoll, err := parseDice(dice)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	e, err := createEngine()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	moves, err := e.RankMoves(state, diceRoll, *numMoves)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error analyzing moves: %v\n", err)
		os.Exit(1)
	}

	if len(moves) == 0 {
		fmt.Println("No legal moves (forced to pass)")
		return
	}

	fmt.Printf("Best moves for roll %d-%d:\n", diceRoll[0], diceRoll[1])
	for i, m := range moves {
		moveStr := formatMove(m.Move)
		fmt.Printf("  %d. %-20s  Eq: %+.3f\n", i+1, moveStr, m.Equity)
	}
}

func formatMove(m engine.Move) string {
	var parts []string
	for i := 0; i < 4; i++ {
		if m.From[i] < 0 {
			break
		}
		from := m.From[i] + 1 // Convert to 1-based
		to := m.To[i] + 1
		if m.To[i] < 0 {
			parts = append(parts, fmt.Sprintf("%d/off", from))
		} else {
			parts = append(parts, fmt.Sprintf("%d/%d", from, to))
		}
	}
	return strings.Join(parts, " ")
}

func cmdCube(args []string) {
	fs := flag.NewFlagSet("cube", flag.ExitOnError)
	posFlag := fs.String("position", "", "Position ID (gnubg format)")
	posShort := fs.String("p", "", "Position ID (short form)")
	fs.Parse(args)

	pos := *posFlag
	if pos == "" {
		pos = *posShort
	}
	if pos == "" {
		fmt.Fprintln(os.Stderr, "Error: position required")
		fmt.Fprintln(os.Stderr, "Usage: bgengine cube -position <positionID>")
		os.Exit(1)
	}

	state, err := parsePosition(pos)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	e, err := createEngine()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	analysis, err := e.AnalyzeCube(state)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error analyzing cube: %v\n", err)
		os.Exit(1)
	}

	decisionStr := ""
	switch analysis.DecisionType {
	case engine.DOUBLE_TAKE, engine.REDOUBLE_TAKE:
		decisionStr = "Double, Take"
	case engine.DOUBLE_PASS, engine.REDOUBLE_PASS:
		decisionStr = "Double, Pass"
	case engine.NODOUBLE_TAKE, engine.NODOUBLE_BEAVER:
		decisionStr = "No Double"
	case engine.TOOGOOD_TAKE, engine.TOOGOOD_PASS, engine.TOOGOODRE_TAKE, engine.TOOGOODRE_PASS:
		decisionStr = "Too Good to Double"
	case engine.NOT_AVAILABLE:
		decisionStr = "Cube Not Available"
	default:
		decisionStr = "No Double"
	}

	fmt.Printf("Cube Decision: %s\n", decisionStr)
	fmt.Printf("  No double equity:  %+.3f\n", analysis.NoDoubleEquity)
	fmt.Printf("  Double/Take equity: %+.3f\n", analysis.DoubleTakeEq)
	fmt.Printf("  Double/Pass equity: %+.3f\n", analysis.DoublePassEq)
}

func cmdRollout(args []string) {
	fs := flag.NewFlagSet("rollout", flag.ExitOnError)
	posFlag := fs.String("position", "", "Position ID (gnubg format)")
	posShort := fs.String("p", "", "Position ID (short form)")
	trials := fs.Int("trials", 1296, "Number of games to simulate")
	workers := fs.Int("workers", 0, "Number of worker goroutines (0 = auto)")
	truncate := fs.Int("truncate", 0, "Truncate rollout at N plies (0 = play to end)")
	seed := fs.Int64("seed", 0, "Random seed (0 = random)")
	fs.Parse(args)

	pos := *posFlag
	if pos == "" {
		pos = *posShort
	}
	if pos == "" {
		fmt.Fprintln(os.Stderr, "Error: position required")
		fmt.Fprintln(os.Stderr, "Usage: bgengine rollout -position <positionID> [-trials N] [-workers N]")
		os.Exit(1)
	}

	state, err := parsePosition(pos)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	e, err := createEngine()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	opts := engine.RolloutOptions{
		Trials:   *trials,
		Workers:  *workers,
		Truncate: *truncate,
		Seed:     *seed,
	}

	start := time.Now()
	result, err := e.Rollout(state, opts)
	elapsed := time.Since(start)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error during rollout: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Rollout (%d trials, %.1fs):\n", result.TrialsCompleted, elapsed.Seconds())
	fmt.Printf("  Equity: %+.3f ± %.3f (95%% CI: ±%.3f)\n",
		result.Equity, result.EquityStdDev, result.EquityCI)
	fmt.Printf("  Win:    %.1f%% (G: %.1f%%, BG: %.1f%%)\n",
		result.WinProb*100, result.WinG*100, result.WinBG*100)
	fmt.Printf("  Lose:   %.1f%% (G: %.1f%%, BG: %.1f%%)\n",
		(1-result.WinProb)*100, result.LoseG*100, result.LoseBG*100)
}
