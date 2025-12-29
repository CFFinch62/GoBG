package bearoff

// Combination table for computing bearoff positions
// anCombination[n-1][r-1] = C(n, r)
var anCombination [40][25]int
var combinationInitialized bool

// initCombination initializes the combination table
func initCombination() {
	if combinationInitialized {
		return
	}

	// C(n, 1) = n
	for i := 0; i < 40; i++ {
		anCombination[i][0] = i + 1
	}

	// C(1, r) = 0 for r > 1
	for j := 1; j < 25; j++ {
		anCombination[0][j] = 0
	}

	// C(n, r) = C(n-1, r-1) + C(n-1, r)
	for i := 1; i < 40; i++ {
		for j := 1; j < 25; j++ {
			anCombination[i][j] = anCombination[i-1][j-1] + anCombination[i-1][j]
		}
	}

	combinationInitialized = true
}

// Combination returns C(n, r) - the number of ways to choose r items from n
func Combination(n, r int) int {
	if n <= 0 || r <= 0 || n > 40 || r > 25 {
		return 0
	}
	initCombination()
	return anCombination[n-1][r-1]
}

// positionF is a helper function for PositionBearoff
func positionF(fBits uint32, n, r int) int {
	if n == r {
		return 0
	}

	if fBits&(1<<(n-1)) != 0 {
		return Combination(n-1, r) + positionF(fBits, n-1, r-1)
	}
	return positionF(fBits, n-1, r)
}

// PositionBearoff converts a bearoff board position to a position ID
// anBoard is the number of checkers on each point (0-5 for points 1-6)
// nPoints is the number of points (typically 6)
// nChequers is the maximum number of checkers (typically 15)
func PositionBearoff(anBoard []uint8, nPoints, nChequers int) int {
	if nPoints == 0 {
		return 0
	}

	// Calculate the bit pattern
	j := nPoints - 1
	for i := 0; i < nPoints; i++ {
		j += int(anBoard[i])
	}

	fBits := uint32(1) << j

	for i := 0; i < nPoints-1; i++ {
		j -= int(anBoard[i]) + 1
		fBits |= uint32(1) << j
	}

	return positionF(fBits, nChequers+nPoints, nPoints)
}

// positionInv is the inverse of positionF
func positionInv(nID, n, r int) uint32 {
	if r == 0 {
		return 0
	}
	if n == r {
		return (1 << n) - 1
	}

	nC := Combination(n-1, r)
	if nID >= nC {
		return (1 << (n - 1)) | positionInv(nID-nC, n-1, r-1)
	}
	return positionInv(nID, n-1, r)
}

// PositionFromBearoff converts a position ID back to a board position
func PositionFromBearoff(usID, nPoints, nChequers int) [6]uint8 {
	var anBoard [6]uint8

	fBits := positionInv(usID, nChequers+nPoints, nPoints)

	j := nPoints - 1
	for i := 0; i < nChequers+nPoints; i++ {
		if fBits&(1<<i) != 0 {
			if j == 0 {
				break
			}
			j--
		} else {
			anBoard[j]++
		}
	}

	return anBoard
}

