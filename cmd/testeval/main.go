// Command testeval tests the evaluation engine with gnubg data files
package main

import (
	"fmt"
	"os"

	"github.com/yourusername/bgengine/internal/bearoff"
	"github.com/yourusername/bgengine/internal/met"
	"github.com/yourusername/bgengine/internal/neuralnet"
	"github.com/yourusername/bgengine/pkg/engine"
)

func main() {
	fmt.Println("=== GoBG Evaluation Engine Test ===")
	fmt.Println()

	// Define file paths - prefer local data directory, fall back to system
	weightsFileText := "data/gnubg.weights"
	weightsFileBinary := "/usr/share/gnubg/gnubg.wd"
	bearoffFile := "data/gnubg_os0.bd"
	bearoffFileFallback := "/usr/share/gnubg/gnubg_os0.bd"
	metFile := "data/g11.xml"

	// Test 1: Neural Network Weights
	fmt.Println("1. Testing Neural Network Weights...")
	var weights *neuralnet.Weights
	var weightsFile string
	var err error

	// Try local text weights first
	if _, err = os.Stat(weightsFileText); err == nil {
		weights, err = neuralnet.LoadWeightsText(weightsFileText)
		weightsFile = weightsFileText
	} else if _, err = os.Stat(weightsFileBinary); err == nil {
		weights, err = neuralnet.LoadWeightsBinary(weightsFileBinary)
		weightsFile = weightsFileBinary
	}

	if weights != nil && err == nil {
		fmt.Printf("   OK: Loaded weights from %s\n", weightsFile)
		fmt.Printf("       Contact net: loaded\n")
		fmt.Printf("       Race net: loaded\n")
		if weights.Crashed != nil {
			fmt.Printf("       Crashed net: loaded\n")
		}
	} else if err != nil {
		fmt.Printf("   FAIL: %v\n", err)
	} else {
		fmt.Printf("   SKIP: No weights file found\n")
	}
	fmt.Println()

	// Test 2: One-Sided Bearoff Database
	fmt.Println("2. Testing One-Sided Bearoff Database...")
	if _, err := os.Stat(bearoffFile); err != nil {
		bearoffFile = bearoffFileFallback
	}
	if _, err := os.Stat(bearoffFile); err == nil {
		db, err := bearoff.LoadOneSided(bearoffFile)
		if err != nil {
			fmt.Printf("   FAIL: %v\n", err)
		} else {
			fmt.Printf("   OK: Loaded one-sided bearoff database from %s\n", bearoffFile)
			fmt.Printf("       Points: %d, Checkers: %d\n", db.NPoints, db.NChequers)

			// Test a simple bearoff position
			var testBoard [2][6]uint8
			testBoard[0] = [6]uint8{2, 2, 2, 2, 2, 5} // 15 checkers total
			testBoard[1] = [6]uint8{2, 2, 2, 2, 2, 5}
			output, err := db.Evaluate(testBoard)
			if err != nil {
				fmt.Printf("   Bearoff eval: %v\n", err)
			} else {
				fmt.Printf("       Test position: Win=%.4f, WinG=%.4f\n", output[0], output[1])
			}
		}
	} else {
		fmt.Printf("   SKIP: %s not found\n", bearoffFile)
	}
	fmt.Println()

	// Test 2b: Two-Sided Bearoff Database
	bearoffTSFile := "data/gnubg_ts.bd"
	fmt.Println("2b. Testing Two-Sided Bearoff Database...")
	if _, err := os.Stat(bearoffTSFile); err == nil {
		db, err := bearoff.LoadOneSided(bearoffTSFile)
		if err != nil {
			fmt.Printf("   FAIL: %v\n", err)
		} else {
			fmt.Printf("   OK: Loaded two-sided bearoff database from %s\n", bearoffTSFile)
			fmt.Printf("       Type: %d, Points: %d, Checkers: %d, Cubeful: %v\n",
				db.Type, db.NPoints, db.NChequers, db.Cubeful)

			// Test a simple bearoff position (6 checkers max)
			var testBoard [2][6]uint8
			testBoard[0] = [6]uint8{3, 3, 0, 0, 0, 0} // 6 checkers
			testBoard[1] = [6]uint8{3, 3, 0, 0, 0, 0}
			output, err := db.Evaluate(testBoard)
			if err != nil {
				fmt.Printf("   Bearoff eval: %v\n", err)
			} else {
				fmt.Printf("       Equal position: Win=%.4f\n", output[0])
			}

			// Test asymmetric position
			testBoard[0] = [6]uint8{6, 0, 0, 0, 0, 0} // Easy to bear off
			testBoard[1] = [6]uint8{0, 0, 0, 0, 0, 6} // Harder to bear off
			output, err = db.Evaluate(testBoard)
			if err != nil {
				fmt.Printf("   Bearoff eval: %v\n", err)
			} else {
				fmt.Printf("       Asymmetric position: Win=%.4f\n", output[0])
			}
		}
	} else {
		fmt.Printf("   SKIP: %s not found\n", bearoffTSFile)
	}
	fmt.Println()

	// Test 3: Match Equity Table
	fmt.Println("3. Testing Match Equity Table...")
	if _, err := os.Stat(metFile); err == nil {
		table, err := met.LoadXML(metFile)
		if err != nil {
			fmt.Printf("   FAIL: %v\n", err)
		} else {
			fmt.Printf("   OK: Loaded MET from %s\n", metFile)
			fmt.Printf("       Name: %s\n", table.Name)
			fmt.Printf("       Length: %d\n", table.Length)

			// Test some match equities
			fmt.Printf("       0-0 in 11pt: %.4f\n", table.GetME(0, 0, 11, 0, false))
			fmt.Printf("       5-0 in 11pt: %.4f\n", table.GetME(5, 0, 11, 0, false))
			fmt.Printf("       0-5 in 11pt: %.4f\n", table.GetME(0, 5, 11, 0, false))
			fmt.Printf("       10-10 in 11pt: %.4f\n", table.GetME(10, 10, 11, 0, false))
		}
	} else {
		fmt.Printf("   SKIP: %s not found\n", metFile)
	}
	fmt.Println()

	// Test 4: Position Classification
	fmt.Println("4. Testing Position Classification...")
	startPos := engine.StartingPosition()
	board := neuralnet.Board(startPos.Board)
	class := neuralnet.ClassifyPosition(board)
	fmt.Printf("   Starting position: %s\n", className(class))

	// Create a race position
	var raceBoard neuralnet.Board
	raceBoard[0][0] = 5
	raceBoard[0][1] = 5
	raceBoard[0][2] = 5
	raceBoard[1][23] = 5
	raceBoard[1][22] = 5
	raceBoard[1][21] = 5
	class = neuralnet.ClassifyPosition(raceBoard)
	fmt.Printf("   Race position: %s\n", className(class))

	// Create a bearoff position
	var bearoffBoard neuralnet.Board
	bearoffBoard[0][0] = 5
	bearoffBoard[0][1] = 5
	bearoffBoard[0][2] = 5
	bearoffBoard[1][0] = 5
	bearoffBoard[1][1] = 5
	bearoffBoard[1][2] = 5
	class = neuralnet.ClassifyPosition(bearoffBoard)
	fmt.Printf("   Bearoff position: %s\n", className(class))
	fmt.Println()

	// Test 5: Full Engine Evaluation
	fmt.Println("5. Testing Full Engine Evaluation...")
	eng, err := engine.NewEngine(engine.EngineOptions{
		WeightsFileText: weightsFileText,
		BearoffFile:     bearoffFile,
		METFile:         metFile,
	})
	if err != nil {
		fmt.Printf("   Engine creation: %v\n", err)
		fmt.Println("   Testing with default engine (no weights)...")
		eng, _ = engine.NewEngine(engine.EngineOptions{})
	}

	eval, err := eng.Evaluate(startPos)
	if err != nil {
		fmt.Printf("   FAIL: %v\n", err)
	} else {
		fmt.Printf("   Starting position evaluation:\n")
		fmt.Printf("       Win:    %.4f\n", eval.WinProb)
		fmt.Printf("       WinG:   %.4f\n", eval.WinG)
		fmt.Printf("       WinBG:  %.4f\n", eval.WinBG)
		fmt.Printf("       LoseG:  %.4f\n", eval.LoseG)
		fmt.Printf("       LoseBG: %.4f\n", eval.LoseBG)
		fmt.Printf("       Equity: %.4f\n", eval.Equity)
	}
	fmt.Println()

	// 6. Test Move Analysis
	fmt.Println("6. Testing Move Analysis...")
	dice := [2]int{3, 1}
	analysis, err := eng.AnalyzePosition(startPos, dice)
	if err != nil {
		fmt.Printf("   FAIL: %v\n", err)
	} else {
		fmt.Printf("   Dice: %d-%d\n", dice[0], dice[1])
		fmt.Printf("   Legal moves: %d\n", analysis.NumMoves)
		if analysis.NumMoves > 0 {
			fmt.Printf("   Best move: %s (Equity: %.4f)\n",
				formatMove(analysis.BestMove), analysis.BestEquity)
			// Show top 3 moves
			n := 3
			if n > len(analysis.Moves) {
				n = len(analysis.Moves)
			}
			fmt.Println("   Top moves:")
			for i := 0; i < n; i++ {
				m := analysis.Moves[i]
				fmt.Printf("     %d. %s Eq: %.4f (Win: %.1f%%)\n",
					i+1, formatMove(m.Move), m.Equity, m.Eval.WinProb*100)
			}
		}
	}
	fmt.Println()

	fmt.Println("=== Test Complete ===")
}

func className(c neuralnet.PositionClass) string {
	names := []string{
		"Over",      // 0
		"Bearoff2",  // 1
		"BearoffTS", // 2
		"Bearoff1",  // 3
		"BearoffOS", // 4
		"Race",      // 5
		"Crashed",   // 6
		"Contact",   // 7
	}
	if int(c) < len(names) {
		return names[c]
	}
	return fmt.Sprintf("Unknown(%d)", c)
}

func formatMove(m engine.Move) string {
	result := ""
	for i := 0; i < 4; i++ {
		if m.From[i] < 0 {
			break
		}
		if i > 0 {
			result += " "
		}
		from := int(m.From[i]) + 1
		to := int(m.To[i]) + 1
		if m.From[i] == 24 {
			from = 25 // Bar
		}
		if m.To[i] < 0 {
			result += fmt.Sprintf("%d/off", from)
		} else {
			result += fmt.Sprintf("%d/%d", from, to)
		}
	}
	return result
}
