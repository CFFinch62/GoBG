package neuralnet

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestLoadWeightsText(t *testing.T) {
	// Load the gnubg weights file from local data directory
	weightsPath := filepath.Join("..", "..", "data", "gnubg.weights")
	w, err := LoadWeightsText(weightsPath)
	if err != nil {
		t.Fatalf("Failed to load weights: %v", err)
	}

	t.Logf("Loaded weights:\n%s", w.String())

	// Verify dimensions
	if w.Contact.CInput != 250 {
		t.Errorf("Contact net has %d inputs, expected 250", w.Contact.CInput)
	}
	if w.Contact.COutput != 5 {
		t.Errorf("Contact net has %d outputs, expected 5", w.Contact.COutput)
	}

	if w.Race.COutput != 5 {
		t.Errorf("Race net has %d outputs, expected 5", w.Race.COutput)
	}

	if w.PContact.CInput != 200 {
		t.Errorf("Pruning contact net has %d inputs, expected 200", w.PContact.CInput)
	}
}

func TestNeuralNetEvaluate(t *testing.T) {
	// Load weights from local data directory
	weightsPath := filepath.Join("..", "..", "data", "gnubg.weights")
	w, err := LoadWeightsText(weightsPath)
	if err != nil {
		t.Fatalf("Failed to load weights: %v", err)
	}

	// Create a simple test input (all zeros)
	input := make([]float32, w.PContact.CInput)

	// Evaluate
	output := w.PContact.Evaluate(input)

	t.Logf("Output for zero input: %v", output)

	// Outputs should be valid probabilities (between 0 and 1)
	for i, v := range output {
		if v < 0 || v > 1 {
			t.Errorf("Output %d = %f is not a valid probability", i, v)
		}
	}
}

func TestBaseInputs(t *testing.T) {
	// Starting position
	var board Board
	board[0][5] = 5
	board[0][7] = 3
	board[0][12] = 5
	board[0][23] = 2

	board[1][5] = 5
	board[1][7] = 3
	board[1][12] = 5
	board[1][23] = 2

	inputs := BaseInputs(board)

	if len(inputs) != 200 {
		t.Errorf("Expected 200 inputs, got %d", len(inputs))
	}

	// Check that point 5 (with 5 checkers) has correct encoding
	// inpvec[5] = {0.0, 0.0, 1.0, 1.0}
	idx := 5 * 4
	if inputs[idx] != 0.0 || inputs[idx+1] != 0.0 || inputs[idx+2] != 1.0 || inputs[idx+3] != 1.0 {
		t.Errorf("Point 5 encoding incorrect: got %v", inputs[idx:idx+4])
	}
}

func TestRaceInputs(t *testing.T) {
	// Simple race position - all checkers in home board
	var board Board
	board[0][0] = 5
	board[0][1] = 5
	board[0][2] = 5

	board[1][0] = 5
	board[1][1] = 5
	board[1][2] = 5

	inputs := RaceInputs(board)

	if len(inputs) != NumRaceInputs {
		t.Errorf("Expected %d inputs, got %d", NumRaceInputs, len(inputs))
	}
}

func TestEvaluateStartingPosition(t *testing.T) {
	// Load weights from local data directory
	weightsPath := filepath.Join("..", "..", "data", "gnubg.weights")
	w, err := LoadWeightsText(weightsPath)
	if err != nil {
		t.Fatalf("Failed to load weights: %v", err)
	}

	// Starting position
	var board Board
	board[0][5] = 5
	board[0][7] = 3
	board[0][12] = 5
	board[0][23] = 2

	board[1][5] = 5
	board[1][7] = 3
	board[1][12] = 5
	board[1][23] = 2

	// Use pruning net with base inputs
	inputs := BaseInputs(board)
	output := w.PContact.Evaluate(inputs)

	t.Logf("Starting position evaluation: %v", output)

	// Win probability should be around 0.5 for starting position
	winProb := output[0]
	if winProb < 0.4 || winProb > 0.6 {
		t.Logf("Warning: Win probability %f seems off for starting position", winProb)
	}
}

func TestClassifyPosition(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() Board
		expected PositionClass
	}{
		{
			name: "starting position is contact",
			setup: func() Board {
				var board Board
				board[0][5] = 5
				board[0][7] = 3
				board[0][12] = 5
				board[0][23] = 2
				board[1][5] = 5
				board[1][7] = 3
				board[1][12] = 5
				board[1][23] = 2
				return board
			},
			expected: ClassContact,
		},
		{
			name: "bearoff position - all checkers in home board",
			setup: func() Board {
				var board Board
				// Player 0 in their home board
				board[0][0] = 5
				board[0][1] = 5
				board[0][2] = 5
				// Player 1 in their home board (far from player 0)
				board[1][0] = 5
				board[1][1] = 5
				board[1][2] = 5
				return board
			},
			expected: ClassBearoff1,
		},
		{
			name: "pure race position - not bearoff",
			setup: func() Board {
				var board Board
				// Player 0 has checkers outside home board
				board[0][0] = 5
				board[0][1] = 5
				board[0][6] = 5 // Outside home board
				// Player 1 in their home board
				board[1][0] = 5
				board[1][1] = 5
				board[1][2] = 5
				return board
			},
			expected: ClassRace,
		},
		{
			name: "game over - player 0 has no checkers",
			setup: func() Board {
				var board Board
				// Only player 1 has checkers
				board[1][0] = 15
				return board
			},
			expected: ClassOver,
		},
		{
			name: "game over - player 1 has no checkers",
			setup: func() Board {
				var board Board
				// Only player 0 has checkers
				board[0][0] = 15
				return board
			},
			expected: ClassOver,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			board := tt.setup()
			class := ClassifyPosition(board)
			if class != tt.expected {
				t.Errorf("ClassifyPosition() = %v, expected %v", class, tt.expected)
			}
		})
	}
}

func TestContactInputs(t *testing.T) {
	// Starting position
	var board Board
	board[0][5] = 5
	board[0][7] = 3
	board[0][12] = 5
	board[0][23] = 2

	board[1][5] = 5
	board[1][7] = 3
	board[1][12] = 5
	board[1][23] = 2

	inputs := ContactInputs(board)

	if len(inputs) != NumContactInputs {
		t.Errorf("Expected %d inputs, got %d", NumContactInputs, len(inputs))
	}

	// Check base inputs are populated (first 200)
	baseSum := float32(0)
	for i := 0; i < 200; i++ {
		baseSum += inputs[i]
	}
	if baseSum == 0 {
		t.Error("Base inputs are all zeros")
	}

	// Check heuristic inputs are populated (200-250)
	heuristicSum := float32(0)
	for i := 200; i < 250; i++ {
		heuristicSum += inputs[i]
	}
	if heuristicSum == 0 {
		t.Error("Heuristic inputs are all zeros")
	}

	// All inputs should be finite
	for i, v := range inputs {
		if v != v { // NaN check
			t.Errorf("Input %d is NaN", i)
		}
	}
}

func TestCrashedInputs(t *testing.T) {
	// Crashed position - one side nearly closed out
	var board Board
	// Player 0 has a strong board
	board[0][0] = 2
	board[0][1] = 2
	board[0][2] = 2
	board[0][3] = 2
	board[0][4] = 2
	board[0][5] = 2
	board[0][6] = 3 // 15 total

	// Player 1 has most checkers on bar or trapped
	board[1][24] = 5 // on bar
	board[1][23] = 2 // trapped behind
	board[1][0] = 8  // in home

	inputs := CrashedInputs(board)

	if len(inputs) != NumContactInputs {
		t.Errorf("Expected %d inputs, got %d", NumContactInputs, len(inputs))
	}

	// All inputs should be finite
	for i, v := range inputs {
		if v != v { // NaN check
			t.Errorf("Input %d is NaN", i)
		}
	}
}

func TestContactEvaluation(t *testing.T) {
	// Load weights from local data directory
	weightsPath := filepath.Join("..", "..", "data", "gnubg.weights")
	w, err := LoadWeightsText(weightsPath)
	if err != nil {
		t.Fatalf("Failed to load weights: %v", err)
	}

	// Starting position
	var board Board
	board[0][5] = 5
	board[0][7] = 3
	board[0][12] = 5
	board[0][23] = 2

	board[1][5] = 5
	board[1][7] = 3
	board[1][12] = 5
	board[1][23] = 2

	// Use full contact net with contact inputs
	inputs := ContactInputs(board)
	output := w.Contact.Evaluate(inputs)

	t.Logf("Contact net evaluation of starting position: %v", output)

	// Win probability should be around 0.5 for starting position
	winProb := output[0]
	if winProb < 0.45 || winProb > 0.55 {
		t.Logf("Warning: Win probability %f seems off for starting position", winProb)
	}

	// All outputs should be valid probabilities
	for i, v := range output {
		if v < 0 || v > 1 {
			t.Errorf("Output %d = %f is not a valid probability", i, v)
		}
	}
}

func TestEscapes(t *testing.T) {
	// Test escape calculation with various board configurations
	var board [25]uint8

	// Empty board - should have maximum escapes (36 rolls escape)
	escapes := Escapes(board, 6)
	if escapes == 0 {
		t.Error("Expected some escapes from empty board")
	}
	if escapes != 36 {
		t.Errorf("Empty board should have 36 escapes, got %d", escapes)
	}

	// Blocked points should reduce escapes
	// For n=6, Escapes reads board[24+i-6] for i=0..5, i.e. board[18..23]
	// These are the points 1-6 ahead of the checker
	board[19] = 2 // Block 2 pips ahead
	board[20] = 2 // Block 3 pips ahead
	escapesBlocked := Escapes(board, 6)
	if escapesBlocked >= escapes {
		t.Errorf("Blocked points should reduce escapes: was %d, now %d", escapes, escapesBlocked)
	}
}

// TestMenOffNonCrashed verifies the menOffNonCrashed encoding matches gnubg
func TestMenOffNonCrashed(t *testing.T) {
	tests := []struct {
		menOff   int
		expected [3]float32
	}{
		{0, [3]float32{0.0, 0.0, 0.0}},
		{1, [3]float32{1.0 / 3.0, 0.0, 0.0}},
		{2, [3]float32{2.0 / 3.0, 0.0, 0.0}},
		{3, [3]float32{1.0, 0.0 / 3.0, 0.0}},
		{4, [3]float32{1.0, 1.0 / 3.0, 0.0}},
		{5, [3]float32{1.0, 2.0 / 3.0, 0.0}},
		{6, [3]float32{1.0, 1.0, 0.0 / 3.0}},
		{7, [3]float32{1.0, 1.0, 1.0 / 3.0}},
		{8, [3]float32{1.0, 1.0, 2.0 / 3.0}},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("menOff=%d", tt.menOff), func(t *testing.T) {
			// Create a board with the right number of checkers off
			var board [25]uint8
			checkersOn := 15 - tt.menOff
			board[0] = uint8(checkersOn)

			afInput := make([]float32, 3)
			menOffNonCrashed(board, afInput)

			for i := 0; i < 3; i++ {
				if abs32(afInput[i]-tt.expected[i]) > 0.0001 {
					t.Errorf("afInput[%d] = %f, expected %f", i, afInput[i], tt.expected[i])
				}
			}
		})
	}
}

// TestMenOffAll verifies the menOffAll encoding matches gnubg
func TestMenOffAll(t *testing.T) {
	tests := []struct {
		menOff   int
		expected [3]float32
	}{
		{0, [3]float32{0.0, 0.0, 0.0}},
		{1, [3]float32{1.0 / 5.0, 0.0, 0.0}},
		{5, [3]float32{1.0, 0.0, 0.0}},
		{6, [3]float32{1.0, 1.0 / 5.0, 0.0}},
		{10, [3]float32{1.0, 1.0, 0.0}},
		{11, [3]float32{1.0, 1.0, 1.0 / 5.0}},
		{15, [3]float32{1.0, 1.0, 1.0}},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("menOff=%d", tt.menOff), func(t *testing.T) {
			var board [25]uint8
			checkersOn := 15 - tt.menOff
			if checkersOn > 0 {
				board[0] = uint8(checkersOn)
			}

			afInput := make([]float32, 3)
			menOffAll(board, afInput)

			for i := 0; i < 3; i++ {
				if abs32(afInput[i]-tt.expected[i]) > 0.0001 {
					t.Errorf("afInput[%d] = %f, expected %f", i, afInput[i], tt.expected[i])
				}
			}
		})
	}
}

// TestContactInputsStartingPosition verifies contact inputs for starting position
func TestContactInputsStartingPosition(t *testing.T) {
	// Starting position
	var board Board
	board[0][5] = 5
	board[0][7] = 3
	board[0][12] = 5
	board[0][23] = 2

	board[1][5] = 5
	board[1][7] = 3
	board[1][12] = 5
	board[1][23] = 2

	inputs := ContactInputs(board)

	// Verify length
	if len(inputs) != NumContactInputs {
		t.Fatalf("Expected %d inputs, got %d", NumContactInputs, len(inputs))
	}

	// Verify base inputs for point 5 (5 checkers) - should be {0, 0, 1, 1}
	idx := 5 * 4
	expected := []float32{0.0, 0.0, 1.0, 1.0}
	for i := 0; i < 4; i++ {
		if abs32(inputs[idx+i]-expected[i]) > 0.0001 {
			t.Errorf("Point 5 input[%d] = %f, expected %f", i, inputs[idx+i], expected[i])
		}
	}

	// Verify base inputs for point 7 (3 checkers) - should be {0, 0, 1, 0}
	idx = 7 * 4
	expected = []float32{0.0, 0.0, 1.0, 0.0}
	for i := 0; i < 4; i++ {
		if abs32(inputs[idx+i]-expected[i]) > 0.0001 {
			t.Errorf("Point 7 input[%d] = %f, expected %f", i, inputs[idx+i], expected[i])
		}
	}

	// Verify base inputs for point 12 (5 checkers) - should be {0, 0, 1, 1}
	idx = 12 * 4
	expected = []float32{0.0, 0.0, 1.0, 1.0}
	for i := 0; i < 4; i++ {
		if abs32(inputs[idx+i]-expected[i]) > 0.0001 {
			t.Errorf("Point 12 input[%d] = %f, expected %f", i, inputs[idx+i], expected[i])
		}
	}

	// Verify base inputs for point 23 (2 checkers) - should be {0, 1, 0, 0}
	idx = 23 * 4
	expected = []float32{0.0, 1.0, 0.0, 0.0}
	for i := 0; i < 4; i++ {
		if abs32(inputs[idx+i]-expected[i]) > 0.0001 {
			t.Errorf("Point 23 input[%d] = %f, expected %f", i, inputs[idx+i], expected[i])
		}
	}

	// Verify men off encoding (no checkers off in starting position)
	// First side's men off starts at index 200
	menOffIdx := MinPPerPoint * 25 * 2
	if inputs[menOffIdx] != 0.0 || inputs[menOffIdx+1] != 0.0 || inputs[menOffIdx+2] != 0.0 {
		t.Errorf("Men off should be 0 for starting position, got %v", inputs[menOffIdx:menOffIdx+3])
	}
}

// TestHeuristicInputsRanges verifies heuristic inputs are in expected ranges
func TestHeuristicInputsRanges(t *testing.T) {
	// Starting position
	var board Board
	board[0][5] = 5
	board[0][7] = 3
	board[0][12] = 5
	board[0][23] = 2

	board[1][5] = 5
	board[1][7] = 3
	board[1][12] = 5
	board[1][23] = 2

	inputs := ContactInputs(board)

	// Check heuristic inputs for first side (indices 200-224)
	heuristicStart := MinPPerPoint * 25 * 2

	// iBreakContact should be in [0, 1]
	bc := inputs[heuristicStart+iBreakContact]
	if bc < 0 || bc > 1 {
		t.Errorf("iBreakContact = %f, expected in [0, 1]", bc)
	}

	// iBackChequer should be in [0, 1]
	backCheq := inputs[heuristicStart+iBackChequer]
	if backCheq < 0 || backCheq > 1 {
		t.Errorf("iBackChequer = %f, expected in [0, 1]", backCheq)
	}

	// For starting position, back chequer is at point 23
	expectedBackCheq := float32(23) / 24.0
	if abs32(backCheq-expectedBackCheq) > 0.0001 {
		t.Errorf("iBackChequer = %f, expected %f", backCheq, expectedBackCheq)
	}

	// iBackAnchor should be in [0, 1]
	backAnch := inputs[heuristicStart+iBackAnchor]
	if backAnch < 0 || backAnch > 1 {
		t.Errorf("iBackAnchor = %f, expected in [0, 1]", backAnch)
	}

	// For starting position, back anchor is at point 23 (2 checkers)
	expectedBackAnch := float32(23) / 24.0
	if abs32(backAnch-expectedBackAnch) > 0.0001 {
		t.Errorf("iBackAnchor = %f, expected %f", backAnch, expectedBackAnch)
	}

	// iPiploss, iP1, iP2 should be in [0, 1]
	piploss := inputs[heuristicStart+iPiploss]
	p1 := inputs[heuristicStart+iP1]
	p2 := inputs[heuristicStart+iP2]
	if piploss < 0 || piploss > 1 {
		t.Errorf("iPiploss = %f, expected in [0, 1]", piploss)
	}
	if p1 < 0 || p1 > 1 {
		t.Errorf("iP1 = %f, expected in [0, 1]", p1)
	}
	if p2 < 0 || p2 > 1 {
		t.Errorf("iP2 = %f, expected in [0, 1]", p2)
	}

	// iBackescapes should be in [0, 1]
	backEsc := inputs[heuristicStart+iBackescapes]
	if backEsc < 0 || backEsc > 1 {
		t.Errorf("iBackescapes = %f, expected in [0, 1]", backEsc)
	}

	// iContain, iContain2 should be in [0, 1]
	contain := inputs[heuristicStart+iContain]
	contain2 := inputs[heuristicStart+iContain2]
	if contain < 0 || contain > 1 {
		t.Errorf("iContain = %f, expected in [0, 1]", contain)
	}
	if contain2 < 0 || contain2 > 1 {
		t.Errorf("iContain2 = %f, expected in [0, 1]", contain2)
	}

	// iEnter2 should be in [0, 1]
	enter2 := inputs[heuristicStart+iEnter2]
	if enter2 < 0 || enter2 > 1 {
		t.Errorf("iEnter2 = %f, expected in [0, 1]", enter2)
	}

	// iBackbone should be in [0, 1]
	backbone := inputs[heuristicStart+iBackbone]
	if backbone < 0 || backbone > 1 {
		t.Errorf("iBackbone = %f, expected in [0, 1]", backbone)
	}
}

// TestCalculateTiming verifies timing calculation
func TestCalculateTiming(t *testing.T) {
	// Test case: all checkers on bar
	// The timing calculation is complex - it accounts for home board gaps
	var board [25]uint8
	board[24] = 15

	timing := calculateTiming(board, 0)
	// With all 15 checkers on bar (point 24):
	// t = 24 * 15 = 360, no = 15
	// Then home board loop subtracts for empty points (need 2 checkers each)
	// Points 5,4,3,2,1,0 each subtract 2*i from t and 2 from no
	// Final t = 360 - 10 - 8 - 6 - 4 - 2 - 0 = 330
	expected := float32(330) / 100.0
	if abs32(timing-expected) > 0.0001 {
		t.Errorf("Timing = %f, expected %f", timing, expected)
	}

	// Test with checkers in home board
	var board2 [25]uint8
	board2[0] = 2
	board2[1] = 2
	board2[2] = 2
	board2[3] = 2
	board2[4] = 2
	board2[5] = 5 // 15 total

	timing2 := calculateTiming(board2, 0)
	// With proper home board, timing should be lower
	if timing2 >= timing {
		t.Errorf("Timing with home board (%f) should be less than all on bar (%f)", timing2, timing)
	}
}

// TestCalculateMoment verifies moment calculation
func TestCalculateMoment(t *testing.T) {
	// All checkers on point 0
	var board [25]uint8
	board[0] = 15

	moment := calculateMoment(board)
	// With all checkers at point 0, mean is 0, so moment should be 0
	if moment != 0.0 {
		t.Errorf("Moment = %f, expected 0.0", moment)
	}

	// Spread checkers
	board[0] = 5
	board[12] = 5
	board[24] = 5

	moment = calculateMoment(board)
	// Should be positive since checkers are spread
	if moment <= 0 {
		t.Errorf("Moment = %f, expected > 0", moment)
	}
}

// TestCalculateBackbone verifies backbone calculation
func TestCalculateBackbone(t *testing.T) {
	// No anchors
	var board [25]uint8
	board[0] = 15

	backbone := calculateBackbone(board)
	// With only one point, backbone should be 0
	if backbone != 0.0 {
		t.Errorf("Backbone = %f, expected 0.0", backbone)
	}

	// Two adjacent anchors
	board[0] = 0
	board[20] = 2
	board[21] = 2
	board[22] = 2
	board[23] = 2
	board[24] = 7

	backbone = calculateBackbone(board)
	// Should be in [0, 1]
	if backbone < 0 || backbone > 1 {
		t.Errorf("Backbone = %f, expected in [0, 1]", backbone)
	}
}

// TestCalculateBackGame verifies back game calculation
func TestCalculateBackGame(t *testing.T) {
	// No anchors in opponent's home
	var board [25]uint8
	board[0] = 15

	backg, backg1 := calculateBackGame(board)
	if backg != 0.0 || backg1 != 0.0 {
		t.Errorf("BackGame = (%f, %f), expected (0, 0)", backg, backg1)
	}

	// Single anchor in opponent's home (point 18-23)
	board[0] = 13
	board[20] = 2

	backg, backg1 = calculateBackGame(board)
	// With one anchor, backg1 should be set
	if backg != 0.0 {
		t.Errorf("BackGame backg = %f, expected 0", backg)
	}
	if backg1 <= 0 {
		t.Errorf("BackGame backg1 = %f, expected > 0", backg1)
	}

	// Two anchors in opponent's home
	board[0] = 11
	board[20] = 2
	board[22] = 2

	backg, backg1 = calculateBackGame(board)
	// With two anchors, backg should be set
	if backg <= 0 {
		t.Errorf("BackGame backg = %f, expected > 0", backg)
	}
	if backg1 != 0.0 {
		t.Errorf("BackGame backg1 = %f, expected 0", backg1)
	}
}

// TestCalculateBarEntry verifies bar entry calculation
func TestCalculateBarEntry(t *testing.T) {
	// Not on bar
	var board [25]uint8
	board[0] = 15
	var boardOpp [25]uint8
	boardOpp[0] = 15

	enter, enter2 := calculateBarEntry(board, boardOpp)
	if enter != 0.0 {
		t.Errorf("Enter = %f, expected 0 when not on bar", enter)
	}
	// enter2 = (36 - (n-6)^2) / 36 where n = points made
	// With 1 point made (point 0), n=1, enter2 = (36 - 25) / 36 = 11/36
	expectedEnter2 := float32(36-25) / 36.0
	if abs32(enter2-expectedEnter2) > 0.0001 {
		t.Errorf("Enter2 = %f, expected %f", enter2, expectedEnter2)
	}

	// On bar with opponent having only point 0 made
	board[0] = 14
	board[24] = 1

	enter, enter2 = calculateBarEntry(board, boardOpp)
	// With one point made, there's still some loss from doubles
	// loss = 4 * 1 = 4 (11 loses), enter = 4 / (36 * 49/6) ≈ 0.0136
	expectedEnter := float32(4) / (36.0 * (49.0 / 6.0))
	if abs32(enter-expectedEnter) > 0.0001 {
		t.Errorf("Enter = %f, expected %f", enter, expectedEnter)
	}

	// On bar with closed opponent home board (6 points made)
	boardOpp[0] = 2
	boardOpp[1] = 2
	boardOpp[2] = 2
	boardOpp[3] = 2
	boardOpp[4] = 2
	boardOpp[5] = 5

	enter, enter2 = calculateBarEntry(board, boardOpp)
	// With closed home board, enter should be positive
	if enter <= 0 {
		t.Errorf("Enter = %f, expected > 0 with closed home board", enter)
	}
	// enter2 = (36 - (6-6)^2) / 36 = 1.0 with 6 points made
	if abs32(enter2-1.0) > 0.0001 {
		t.Errorf("Enter2 = %f, expected 1.0 with closed home board", enter2)
	}
}

func abs32(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}
