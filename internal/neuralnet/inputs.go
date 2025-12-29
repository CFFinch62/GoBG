package neuralnet

// Input encoding for neural network evaluation
// This is a port of gnubg's inputs.c and eval.c input calculation

// PositionClass indicates the type of position for evaluation
type PositionClass int

const (
	ClassOver      PositionClass = iota // Game already finished
	ClassBearoff2                       // Two-sided bearoff database
	ClassBearoffTS                      // Two-sided bearoff database (on disk)
	ClassBearoff1                       // One-sided bearoff database
	ClassBearoffOS                      // One-sided bearoff database (on disk)
	ClassRace                           // Race neural network
	ClassCrashed                        // Contact, one side has less than 7 active checkers
	ClassContact                        // Contact neural network
)

// Input counts for different network types
const (
	MinPPerPoint     = 4
	MoreInputs       = 25                                 // Number of extra inputs per side (heuristics)
	NumContactInputs = (25*MinPPerPoint + MoreInputs) * 2 // 250
	NumPruningInputs = 25 * MinPPerPoint * 2              // 200
)

// Input vector lookup tables for encoding checker counts
// inpvec[n] gives the 4 input values for n checkers on a point
var inpvec = [16][4]float32{
	{0.0, 0.0, 0.0, 0.0}, // 0
	{1.0, 0.0, 0.0, 0.0}, // 1
	{0.0, 1.0, 0.0, 0.0}, // 2
	{0.0, 0.0, 1.0, 0.0}, // 3
	{0.0, 0.0, 1.0, 0.5}, // 4
	{0.0, 0.0, 1.0, 1.0}, // 5
	{0.0, 0.0, 1.0, 1.5}, // 6
	{0.0, 0.0, 1.0, 2.0}, // 7
	{0.0, 0.0, 1.0, 2.5}, // 8
	{0.0, 0.0, 1.0, 3.0}, // 9
	{0.0, 0.0, 1.0, 3.5}, // 10
	{0.0, 0.0, 1.0, 4.0}, // 11
	{0.0, 0.0, 1.0, 4.5}, // 12
	{0.0, 0.0, 1.0, 5.0}, // 13
	{0.0, 0.0, 1.0, 5.5}, // 14
	{0.0, 0.0, 1.0, 6.0}, // 15
}

// inpvecb is for bar encoding (cumulative)
var inpvecb = [16][4]float32{
	{0.0, 0.0, 0.0, 0.0}, // 0
	{1.0, 0.0, 0.0, 0.0}, // 1
	{1.0, 1.0, 0.0, 0.0}, // 2
	{1.0, 1.0, 1.0, 0.0}, // 3
	{1.0, 1.0, 1.0, 0.5}, // 4
	{1.0, 1.0, 1.0, 1.0}, // 5
	{1.0, 1.0, 1.0, 1.5}, // 6
	{1.0, 1.0, 1.0, 2.0}, // 7
	{1.0, 1.0, 1.0, 2.5}, // 8
	{1.0, 1.0, 1.0, 3.0}, // 9
	{1.0, 1.0, 1.0, 3.5}, // 10
	{1.0, 1.0, 1.0, 4.0}, // 11
	{1.0, 1.0, 1.0, 4.5}, // 12
	{1.0, 1.0, 1.0, 5.0}, // 13
	{1.0, 1.0, 1.0, 5.5}, // 14
	{1.0, 1.0, 1.0, 6.0}, // 15
}

// Board is the board representation: [2][25]uint8
// [0] = player 0's checkers from their perspective
// [1] = player 1's checkers from their perspective
// Index 0-23 = points 1-24, index 24 = bar
type Board [2][25]uint8

// BaseInputs calculates the base neural network inputs from a board position.
// This encodes each point with 4 values based on checker count.
// Returns 200 floats (25 points * 4 values * 2 players)
func BaseInputs(board Board) []float32 {
	inputs := make([]float32, 200)
	BaseInputsInto(board, inputs)
	return inputs
}

// BaseInputsInto calculates base inputs into the provided slice
func BaseInputsInto(board Board, inputs []float32) {
	for side := 0; side < 2; side++ {
		offset := side * 25 * 4

		// Points 0-23
		for i := 0; i < 24; i++ {
			nc := board[side][i]
			if nc > 15 {
				nc = 15
			}
			inputs[offset+i*4+0] = inpvec[nc][0]
			inputs[offset+i*4+1] = inpvec[nc][1]
			inputs[offset+i*4+2] = inpvec[nc][2]
			inputs[offset+i*4+3] = inpvec[nc][3]
		}

		// Bar (point 24)
		nc := board[side][24]
		if nc > 15 {
			nc = 15
		}
		inputs[offset+24*4+0] = inpvecb[nc][0]
		inputs[offset+24*4+1] = inpvecb[nc][1]
		inputs[offset+24*4+2] = inpvecb[nc][2]
		inputs[offset+24*4+3] = inpvecb[nc][3]
	}
}

// Race input constants
const (
	RIoff          = 92      // Offset for "men off" inputs
	RIncross       = 92 + 14 // Offset for cross-over inputs
	HalfRaceInputs = RIncross + 1
	NumRaceInputs  = HalfRaceInputs * 2
)

// RaceInputs calculates neural network inputs for a race position.
// Race positions have no contact - all checkers have passed each other.
func RaceInputs(board Board) []float32 {
	inputs := make([]float32, NumRaceInputs)
	RaceInputsInto(board, inputs)
	return inputs
}

// RaceInputsInto calculates race inputs into the provided slice
func RaceInputsInto(board Board, inputs []float32) {
	for side := 0; side < 2; side++ {
		offset := side * HalfRaceInputs
		menOff := uint8(15)

		// Points 0-22 (in race, points 23 and 24 are always empty)
		for i := 0; i < 23; i++ {
			nc := board[side][i]
			menOff -= nc

			k := i * 4
			if nc == 1 {
				inputs[offset+k] = 1.0
			} else {
				inputs[offset+k] = 0.0
			}
			if nc == 2 {
				inputs[offset+k+1] = 1.0
			} else {
				inputs[offset+k+1] = 0.0
			}
			if nc >= 3 {
				inputs[offset+k+2] = 1.0
			} else {
				inputs[offset+k+2] = 0.0
			}
			if nc > 3 {
				inputs[offset+k+3] = float32(nc-3) / 2.0
			} else {
				inputs[offset+k+3] = 0.0
			}
		}

		// Men off (14 one-hot encoded values)
		for k := 0; k < 14; k++ {
			if menOff == uint8(k+1) {
				inputs[offset+RIoff+k] = 1.0
			} else {
				inputs[offset+RIoff+k] = 0.0
			}
		}

		// Cross-overs
		nCross := uint32(0)
		for k := 1; k < 4; k++ {
			for i := 6 * k; i < 6*k+6; i++ {
				nc := board[side][i]
				if nc > 0 {
					nCross += uint32(nc) * uint32(k)
				}
			}
		}
		inputs[offset+RIncross] = float32(nCross) / 10.0
	}
}
