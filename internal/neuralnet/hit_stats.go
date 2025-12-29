package neuralnet

// hit_stats.go contains the complex hit probability calculations
// This is a port of gnubg's piploss calculation from eval.c

// aanCombination[n] - How many ways to hit from a distance of n pips.
// Each number is an index into aIntermediate.
var aanCombination = [24][5]int{
	{0, -1, -1, -1, -1},  //  1
	{1, 2, -1, -1, -1},   //  2
	{3, 4, 5, -1, -1},    //  3
	{6, 7, 8, 9, -1},     //  4
	{10, 11, 12, -1, -1}, //  5
	{13, 14, 15, 16, 17}, //  6
	{18, 19, 20, -1, -1}, //  7
	{21, 22, 23, 24, -1}, //  8
	{25, 26, 27, -1, -1}, //  9
	{28, 29, -1, -1, -1}, // 10
	{30, -1, -1, -1, -1}, // 11
	{31, 32, 33, -1, -1}, // 12
	{-1, -1, -1, -1, -1}, // 13
	{-1, -1, -1, -1, -1}, // 14
	{34, -1, -1, -1, -1}, // 15
	{35, -1, -1, -1, -1}, // 16
	{-1, -1, -1, -1, -1}, // 17
	{36, -1, -1, -1, -1}, // 18
	{-1, -1, -1, -1, -1}, // 19
	{37, -1, -1, -1, -1}, // 20
	{-1, -1, -1, -1, -1}, // 21
	{-1, -1, -1, -1, -1}, // 22
	{-1, -1, -1, -1, -1}, // 23
	{38, -1, -1, -1, -1}, // 24
}

// intermediate describes one way to hit
type intermediate struct {
	fAll           bool   // if true, all intermediate points required
	anIntermediate [3]int // intermediate points required
	nFaces         int    // number of dice faces used (1-4)
	nPips          int    // number of pips to hit
}

// aIntermediate describes all 39 ways to hit
var aIntermediate = [39]intermediate{
	{true, [3]int{0, 0, 0}, 1, 1},    //  0: 1x hits 1
	{true, [3]int{0, 0, 0}, 1, 2},    //  1: 2x hits 2
	{true, [3]int{1, 0, 0}, 2, 2},    //  2: 11 hits 2
	{true, [3]int{0, 0, 0}, 1, 3},    //  3: 3x hits 3
	{false, [3]int{1, 2, 0}, 2, 3},   //  4: 21 hits 3
	{true, [3]int{1, 2, 0}, 3, 3},    //  5: 11 hits 3
	{true, [3]int{0, 0, 0}, 1, 4},    //  6: 4x hits 4
	{false, [3]int{1, 3, 0}, 2, 4},   //  7: 31 hits 4
	{true, [3]int{2, 0, 0}, 2, 4},    //  8: 22 hits 4
	{true, [3]int{1, 2, 3}, 4, 4},    //  9: 11 hits 4
	{true, [3]int{0, 0, 0}, 1, 5},    // 10: 5x hits 5
	{false, [3]int{1, 4, 0}, 2, 5},   // 11: 41 hits 5
	{false, [3]int{2, 3, 0}, 2, 5},   // 12: 32 hits 5
	{true, [3]int{0, 0, 0}, 1, 6},    // 13: 6x hits 6
	{false, [3]int{1, 5, 0}, 2, 6},   // 14: 51 hits 6
	{false, [3]int{2, 4, 0}, 2, 6},   // 15: 42 hits 6
	{true, [3]int{3, 0, 0}, 2, 6},    // 16: 33 hits 6
	{true, [3]int{2, 4, 0}, 3, 6},    // 17: 22 hits 6
	{false, [3]int{1, 6, 0}, 2, 7},   // 18: 61 hits 7
	{false, [3]int{2, 5, 0}, 2, 7},   // 19: 52 hits 7
	{false, [3]int{3, 4, 0}, 2, 7},   // 20: 43 hits 7
	{false, [3]int{2, 6, 0}, 2, 8},   // 21: 62 hits 8
	{false, [3]int{3, 5, 0}, 2, 8},   // 22: 53 hits 8
	{true, [3]int{4, 0, 0}, 2, 8},    // 23: 44 hits 8
	{true, [3]int{2, 4, 6}, 4, 8},    // 24: 22 hits 8
	{false, [3]int{3, 6, 0}, 2, 9},   // 25: 63 hits 9
	{false, [3]int{4, 5, 0}, 2, 9},   // 26: 54 hits 9
	{true, [3]int{3, 6, 0}, 3, 9},    // 27: 33 hits 9
	{false, [3]int{4, 6, 0}, 2, 10},  // 28: 64 hits 10
	{true, [3]int{5, 0, 0}, 2, 10},   // 29: 55 hits 10
	{false, [3]int{5, 6, 0}, 2, 11},  // 30: 65 hits 11
	{true, [3]int{6, 0, 0}, 2, 12},   // 31: 66 hits 12
	{true, [3]int{4, 8, 0}, 3, 12},   // 32: 44 hits 12
	{true, [3]int{3, 6, 9}, 4, 12},   // 33: 33 hits 12
	{true, [3]int{5, 10, 0}, 3, 15},  // 34: 55 hits 15
	{true, [3]int{4, 8, 12}, 4, 16},  // 35: 44 hits 16
	{true, [3]int{6, 12, 0}, 3, 18},  // 36: 66 hits 18
	{true, [3]int{5, 10, 15}, 4, 20}, // 37: 55 hits 20
	{true, [3]int{6, 12, 18}, 4, 24}, // 38: 66 hits 24
}

// aaRoll[n] - All ways to hit with the n'th roll
// Each entry is an index into aIntermediate
var aaRoll = [21][4]int{
	{0, 2, 5, 9},     // 11
	{1, 8, 17, 24},   // 22
	{3, 16, 27, 33},  // 33
	{6, 23, 32, 35},  // 44
	{10, 29, 34, 37}, // 55
	{13, 31, 36, 38}, // 66
	{0, 1, 4, -1},    // 21
	{0, 3, 7, -1},    // 31
	{1, 3, 12, -1},   // 32
	{0, 6, 11, -1},   // 41
	{1, 6, 15, -1},   // 42
	{3, 6, 20, -1},   // 43
	{0, 10, 14, -1},  // 51
	{1, 10, 19, -1},  // 52
	{3, 10, 22, -1},  // 53
	{6, 10, 26, -1},  // 54
	{0, 13, 18, -1},  // 61
	{1, 13, 21, -1},  // 62
	{3, 13, 25, -1},  // 63
	{6, 13, 28, -1},  // 64
	{10, 13, 30, -1}, // 65
}

// msb32 returns the position of the most significant bit (0-based from right)
func msb32(x int) int {
	if x == 0 {
		return -1
	}
	pos := 0
	if x >= 1<<16 {
		x >>= 16
		pos += 16
	}
	if x >= 1<<8 {
		x >>= 8
		pos += 8
	}
	if x >= 1<<4 {
		x >>= 4
		pos += 4
	}
	if x >= 1<<2 {
		x >>= 2
		pos += 2
	}
	if x >= 1<<1 {
		pos++
	}
	return pos
}

// calculateHitStats calculates pip loss and hit probability statistics
func calculateHitStats(anBoard, anBoardOpp [25]uint8, nOppBack int) (piploss, p1, p2 float32) {
	// Count how many inner points we have made
	nBoard := 0
	for i := 0; i < 6; i++ {
		if anBoard[i] >= 2 {
			nBoard++
		}
	}

	// Track which hitting combinations are available
	var aHit [39]int

	// For every point we'd consider hitting a blot on
	maxPoint := 21
	if nBoard > 2 {
		maxPoint = 23
	}

	for i := maxPoint; i >= 0; i-- {
		// If there's a blot there
		if anBoardOpp[i] != 1 {
			continue
		}

		// For every point beyond
		for j := 24 - i; j < 25; j++ {
			// If we have a hitter and are willing to hit
			if anBoard[j] == 0 {
				continue
			}
			// Don't break inner points (2 checkers) to hit
			if j < 6 && anBoard[j] == 2 {
				continue
			}

			// For every roll that can hit from that point
			dist := j - 24 + i
			for n := 0; n < 5; n++ {
				combIdx := aanCombination[dist][n]
				if combIdx == -1 {
					break
				}

				pi := &aIntermediate[combIdx]

				// Check intermediate points
				canHit := true
				if pi.fAll {
					if pi.nFaces > 1 {
						for k := 0; k < 3 && pi.anIntermediate[k] > 0; k++ {
							if anBoardOpp[i-pi.anIntermediate[k]] > 1 {
								canHit = false
								break
							}
						}
					}
				} else {
					// Either of two points required
					if anBoardOpp[i-pi.anIntermediate[0]] > 1 &&
						anBoardOpp[i-pi.anIntermediate[1]] > 1 {
						canHit = false
					}
				}

				if canHit {
					aHit[combIdx] |= 1 << j
				}
			}
		}
	}

	// Calculate roll statistics
	var aRoll [21]rollStat

	// Process based on bar status
	if anBoard[24] == 0 {
		// Not on bar
		calculateHitsNotOnBar(anBoard, aHit[:], aRoll[:])
	} else if anBoard[24] == 1 {
		// One checker on bar
		calculateHitsOneOnBar(anBoard, anBoardOpp, aHit[:], aRoll[:])
	} else {
		// Multiple checkers on bar
		calculateHitsMultipleOnBar(aHit[:], aRoll[:])
	}

	// Sum up statistics
	npSum := 0
	n1 := 0
	n2 := 0

	for i := 0; i < 6; i++ {
		npSum += aRoll[i].nPips
		if aRoll[i].nChequers > 0 {
			n1++
			if aRoll[i].nChequers > 1 {
				n2++
			}
		}
	}

	for i := 6; i < 21; i++ {
		npSum += aRoll[i].nPips * 2
		if aRoll[i].nChequers > 0 {
			n1 += 2
			if aRoll[i].nChequers > 1 {
				n2 += 2
			}
		}
	}

	piploss = float32(npSum) / (12.0 * 36.0)
	p1 = float32(n1) / 36.0
	p2 = float32(n2) / 36.0

	return piploss, p1, p2
}

// rollStat tracks hitting statistics for a single roll
type rollStat struct {
	nChequers int
	nPips     int
}

// calculateHitsNotOnBar processes hits when not on bar
func calculateHitsNotOnBar(anBoard [25]uint8, aHit []int, aRoll []rollStat) {
	for i := 0; i < 21; i++ {
		n := -1 // hitter used

		for j := 0; j < 4; j++ {
			r := aaRoll[i][j]
			if r < 0 {
				break
			}
			if aHit[r] == 0 {
				continue
			}

			pi := &aIntermediate[r]

			if pi.nFaces == 1 {
				// Direct shot
				k := msb32(aHit[r])
				if n != k || anBoard[k] > 1 {
					aRoll[i].nChequers++
				}
				n = k

				if pips := k - pi.nPips + 1; pips > aRoll[i].nPips {
					aRoll[i].nPips = pips
				}

				// Check for multiple direct shots with doubles
				if aaRoll[i][3] >= 0 && (aHit[r] & ^(1<<k)) != 0 {
					aRoll[i].nChequers++
				}
			} else {
				// Indirect shot
				if aRoll[i].nChequers == 0 {
					aRoll[i].nChequers = 1
				}

				k := msb32(aHit[r])
				if pips := k - pi.nPips + 1; pips > aRoll[i].nPips {
					aRoll[i].nPips = pips
				}
			}
		}
	}
}

// calculateHitsOneOnBar processes hits with one checker on bar
func calculateHitsOneOnBar(anBoard, anBoardOpp [25]uint8, aHit []int, aRoll []rollStat) {
	for i := 0; i < 21; i++ {
		n := 0 // free to use either die to enter

		for j := 0; j < 4; j++ {
			r := aaRoll[i][j]
			if r < 0 {
				break
			}
			if aHit[r] == 0 {
				continue
			}

			pi := &aIntermediate[r]

			if pi.nFaces == 1 {
				// Direct shot
				for k := msb32(aHit[r]); k > 0; k-- {
					if aHit[r]&(1<<k) == 0 {
						continue
					}

					// If we need this die to enter, we can't hit elsewhere
					if n != 0 && k != 24 {
						break
					}

					// If not from bar, other die must enter
					if k != 24 {
						npip := aIntermediate[aaRoll[i][1-j]].nPips
						if anBoardOpp[npip-1] > 1 {
							break
						}
						n = 1
					}

					aRoll[i].nChequers++
					if pips := k - pi.nPips + 1; pips > aRoll[i].nPips {
						aRoll[i].nPips = pips
					}
				}
			} else {
				// Indirect shot - only from bar
				if aHit[r]&(1<<24) == 0 {
					continue
				}

				if aRoll[i].nChequers == 0 {
					aRoll[i].nChequers = 1
				}

				if pips := 25 - pi.nPips; pips > aRoll[i].nPips {
					aRoll[i].nPips = pips
				}
			}
		}
	}
}

// calculateHitsMultipleOnBar processes hits with multiple checkers on bar
func calculateHitsMultipleOnBar(aHit []int, aRoll []rollStat) {
	for i := 0; i < 21; i++ {
		// Only count direct shots from point 24
		for j := 0; j < 2; j++ {
			r := aaRoll[i][j]
			if aHit[r]&(1<<24) == 0 {
				continue
			}

			pi := &aIntermediate[r]
			if pi.nFaces != 1 {
				continue
			}

			aRoll[i].nChequers++
			if pips := 25 - pi.nPips; pips > aRoll[i].nPips {
				aRoll[i].nPips = pips
			}
		}
	}
}
