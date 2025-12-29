package neuralnet

// ClassifyPosition determines the position class for evaluation.
// This is a port of gnubg's ClassifyPosition function from eval.c.
// For now, we only support standard backgammon (not hypergammon variants).
func ClassifyPosition(board Board) PositionClass {
	// Find the back checker for each side
	nOppBack := -1
	for i := 24; i >= 0; i-- {
		if board[0][i] > 0 {
			nOppBack = i
			break
		}
	}

	nBack := -1
	for i := 24; i >= 0; i-- {
		if board[1][i] > 0 {
			nBack = i
			break
		}
	}

	// Check if game is over (one side has no checkers on board)
	if nBack < 0 || nOppBack < 0 {
		return ClassOver
	}

	// Check if there's contact (back checkers haven't passed each other)
	if nBack+nOppBack > 22 {
		// Contact position - check if crashed
		const N = 6

		for side := 0; side < 2; side++ {
			// Count total checkers for this side
			tot := 0
			for i := 0; i < 25; i++ {
				tot += int(board[side][i])
			}

			if tot <= N {
				return ClassCrashed
			}

			// Check for crashed positions based on checker distribution
			bar := int(board[side][24])
			point1 := int(board[side][0])

			if bar > 1 {
				if tot <= N+bar {
					return ClassCrashed
				}
				if 1+tot-(bar+point1) <= N && point1 > 1 {
					return ClassCrashed
				}
			} else {
				if tot <= N+(point1-1) {
					return ClassCrashed
				}
			}
		}

		return ClassContact
	}

	// Race position - no contact
	// Check if this is a bearoff position (all checkers in home board)
	// First check for two-sided bearoff (6 points, 6 checkers max)
	if IsBearoff(board, 6, 6) {
		return ClassBearoffTS
	}

	// Then check for one-sided bearoff (6 points, 15 checkers max)
	if IsBearoff(board, 6, 15) {
		return ClassBearoff1
	}

	return ClassRace
}

// IsBearoff checks if a position can be evaluated using the bearoff database.
// nPoints is the number of points covered by the database (typically 6)
// nChequers is the maximum number of checkers (typically 15)
func IsBearoff(board Board, nPoints, nChequers int) bool {
	// Check both sides have all checkers in their home board (points 0 to nPoints-1)
	for side := 0; side < 2; side++ {
		// Check that no checkers are outside the home board
		for i := nPoints; i < 25; i++ {
			if board[side][i] > 0 {
				return false
			}
		}

		// Count checkers in home board
		total := 0
		for i := 0; i < nPoints; i++ {
			total += int(board[side][i])
		}

		// Must have at most nChequers
		if total > nChequers {
			return false
		}
	}

	return true
}

// GetBearoffBoard extracts the bearoff portion of the board (first 6 points)
func GetBearoffBoard(board Board) [2][6]uint8 {
	var result [2][6]uint8
	for side := 0; side < 2; side++ {
		for i := 0; i < 6; i++ {
			result[side][i] = board[side][i]
		}
	}
	return result
}

// anPoint maps checker count to blocked status (0 = empty, 1 = blocked/has checkers)
var anPoint = [16]int{0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}

// anEscapes and anEscapes1 are lookup tables for escape calculations
var anEscapes [0x1000]int
var anEscapes1 [0x1000]int
var escapesInitialized = false

// initEscapeTables initializes the escape lookup tables
func initEscapeTables() {
	if escapesInitialized {
		return
	}

	// ComputeTable0
	for i := 0; i < 0x1000; i++ {
		c := 0
		for n0 := 0; n0 <= 5; n0++ {
			for n1 := 0; n1 <= n0; n1++ {
				// Can escape if target not blocked AND at least one intermediate not blocked
				target := n0 + n1 + 1
				targetBlocked := (i & (1 << target)) != 0
				bothIntermediatesBlocked := (i&(1<<n0)) != 0 && (i&(1<<n1)) != 0
				if !targetBlocked && !bothIntermediatesBlocked {
					if n0 == n1 {
						c++
					} else {
						c += 2
					}
				}
			}
		}
		anEscapes[i] = c
	}

	// ComputeTable1
	anEscapes1[0] = 0
	for i := 1; i < 0x1000; i++ {
		c := 0
		low := 0
		for (i & (1 << low)) == 0 {
			low++
		}

		for n0 := 0; n0 <= 5; n0++ {
			for n1 := 0; n1 <= n0; n1++ {
				target := n0 + n1 + 1
				targetBlocked := (i & (1 << target)) != 0
				bothIntermediatesBlocked := (i&(1<<n0)) != 0 && (i&(1<<n1)) != 0
				if target > low && !targetBlocked && !bothIntermediatesBlocked {
					if n0 == n1 {
						c++
					} else {
						c += 2
					}
				}
			}
		}
		anEscapes1[i] = c
	}

	escapesInitialized = true
}

// Escapes calculates how many rolls let a checker escape from point n
func Escapes(board [25]uint8, n int) int {
	initEscapeTables()

	af := 0
	m := n
	if m > 12 {
		m = 12
	}

	for i := 0; i < m; i++ {
		af |= anPoint[board[24+i-n]] << i
	}

	return anEscapes[af]
}
