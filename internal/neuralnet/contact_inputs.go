package neuralnet

// Contact input indices (relative to the start of extra inputs for each side)
const (
	iOff1          = 0 // Men off encoding (3 values)
	iOff2          = 1
	iOff3          = 2
	iBreakContact  = 3  // Minimum pips to break contact
	iBackChequer   = 4  // Location of back checker
	iBackAnchor    = 5  // Location of back anchor
	iForwardAnchor = 6  // Forward anchor in opponent's home
	iPiploss       = 7  // Average pips opponent loses from hits
	iP1            = 8  // Probability of hitting at least one
	iP2            = 9  // Probability of hitting at least two
	iBackescapes   = 10 // How many rolls let back checker escape
	iAcontain      = 11 // Active containment
	iAcontain2     = 12 // Active containment squared
	iContain       = 13 // Containment
	iContain2      = 14 // Containment squared
	iMobility      = 15 // Mobility measure
	iMoment2       = 16 // One-sided moment
	iEnter         = 17 // Average pips lost when on bar
	iEnter2        = 18 // Probability of not entering from bar
	iTiming        = 19 // Timing measure
	iBackbone      = 20 // Backbone strength
	iBackg         = 21 // Back game indicator
	iBackg1        = 22 // Back game indicator (single anchor)
	iFreepip       = 23 // Free pips
	iBackrescapes  = 24 // Back rescapes
)

// ContactInputs calculates the full 250-float inputs for contact positions.
// This includes base inputs (200) plus heuristic features (50 = 25 per side).
func ContactInputs(board Board) []float32 {
	inputs := make([]float32, NumContactInputs)
	ContactInputsInto(board, inputs)
	return inputs
}

// ContactInputsInto calculates contact inputs into the provided slice
func ContactInputsInto(board Board, inputs []float32) {
	// First, calculate base inputs (200 floats)
	BaseInputsInto(board, inputs)

	// Calculate extra inputs for each side
	// Note: gnubg accidentally switched sides when training, so we follow that
	b0 := inputs[MinPPerPoint*25*2:]
	menOffNonCrashed(board[0], b0[iOff1:])
	calculateHalfInputs(board[1], board[0], b0)

	b1 := inputs[MinPPerPoint*25*2+MoreInputs:]
	menOffNonCrashed(board[1], b1[iOff1:])
	calculateHalfInputs(board[0], board[1], b1)
}

// CrashedInputs calculates inputs for crashed positions (one side nearly closed out)
func CrashedInputs(board Board) []float32 {
	inputs := make([]float32, NumContactInputs)
	CrashedInputsInto(board, inputs)
	return inputs
}

// CrashedInputsInto calculates crashed inputs into the provided slice
func CrashedInputsInto(board Board, inputs []float32) {
	// First, calculate base inputs (200 floats)
	BaseInputsInto(board, inputs)

	// Calculate extra inputs for each side
	b0 := inputs[MinPPerPoint*25*2:]
	menOffAll(board[1], b0[iOff1:])
	calculateHalfInputs(board[1], board[0], b0)

	b1 := inputs[MinPPerPoint*25*2+MoreInputs:]
	menOffAll(board[0], b1[iOff1:])
	calculateHalfInputs(board[0], board[1], b1)
}

// menOffNonCrashed encodes men off for non-crashed positions (max 8 off)
func menOffNonCrashed(anBoard [25]uint8, afInput []float32) {
	menOff := 15
	for i := 0; i < 25; i++ {
		menOff -= int(anBoard[i])
	}

	// Encode in 3 buckets: 0-2, 3-5, 6-8
	if menOff <= 2 {
		if menOff > 0 {
			afInput[0] = float32(menOff) / 3.0
		} else {
			afInput[0] = 0.0
		}
		afInput[1] = 0.0
		afInput[2] = 0.0
	} else if menOff <= 5 {
		afInput[0] = 1.0
		afInput[1] = float32(menOff-3) / 3.0
		afInput[2] = 0.0
	} else {
		afInput[0] = 1.0
		afInput[1] = 1.0
		afInput[2] = float32(menOff-6) / 3.0
	}
}

// menOffAll encodes men off for crashed positions (can have more off)
func menOffAll(anBoard [25]uint8, afInput []float32) {
	menOff := 15
	for i := 0; i < 25; i++ {
		menOff -= int(anBoard[i])
	}

	// Encode in 3 buckets: 0-5, 6-10, 11-15
	if menOff <= 5 {
		if menOff > 0 {
			afInput[0] = float32(menOff) / 5.0
		} else {
			afInput[0] = 0.0
		}
		afInput[1] = 0.0
		afInput[2] = 0.0
	} else if menOff <= 10 {
		afInput[0] = 1.0
		afInput[1] = float32(menOff-5) / 5.0
		afInput[2] = 0.0
	} else {
		afInput[0] = 1.0
		afInput[1] = 1.0
		afInput[2] = float32(menOff-10) / 5.0
	}
}

// calculateHalfInputs calculates heuristic inputs for one player.
// anBoard is the player's board, anBoardOpp is the opponent's board.
// afInput is the slice to write the heuristic inputs to.
func calculateHalfInputs(anBoard, anBoardOpp [25]uint8, afInput []float32) {
	initEscapeTables()

	// Find opponent's back checker
	nOppBack := -1
	for i := 24; i >= 0; i-- {
		if anBoardOpp[i] > 0 {
			nOppBack = i
			break
		}
	}
	nOppBack = 23 - nOppBack

	// Break contact calculation
	np := 0
	for i := nOppBack + 1; i < 25; i++ {
		if anBoard[i] > 0 {
			np += (i + 1 - nOppBack) * int(anBoard[i])
		}
	}
	afInput[iBreakContact] = float32(np) / (15.0 + 152.0)

	// Free pips
	p := 0
	for i := 0; i < nOppBack; i++ {
		if anBoard[i] > 0 {
			p += (i + 1) * int(anBoard[i])
		}
	}
	afInput[iFreepip] = float32(p) / 100.0

	// Timing calculation
	afInput[iTiming] = calculateTiming(anBoard, nOppBack)

	// Back chequer and anchor
	nBack := 24
	for nBack >= 0 {
		if anBoard[nBack] > 0 {
			break
		}
		nBack--
	}
	afInput[iBackChequer] = float32(nBack) / 24.0

	// Back anchor
	backAnchor := nBack
	if nBack == 24 {
		backAnchor = 23
	}
	for backAnchor >= 0 {
		if anBoard[backAnchor] >= 2 {
			break
		}
		backAnchor--
	}
	afInput[iBackAnchor] = float32(backAnchor) / 24.0

	// Forward anchor
	forwardAnchor := 0
	for j := 18; j <= backAnchor; j++ {
		if anBoard[j] >= 2 {
			forwardAnchor = 24 - j
			break
		}
	}
	if forwardAnchor == 0 {
		for j := 17; j >= 12; j-- {
			if anBoard[j] >= 2 {
				forwardAnchor = 24 - j
				break
			}
		}
	}
	if forwardAnchor == 0 {
		afInput[iForwardAnchor] = 2.0
	} else {
		afInput[iForwardAnchor] = float32(forwardAnchor) / 6.0
	}

	// Pip loss and hit probability calculations
	piploss, p1, p2 := calculateHitStats(anBoard, anBoardOpp, nOppBack)
	afInput[iPiploss] = piploss
	afInput[iP1] = p1
	afInput[iP2] = p2

	// Back escapes
	afInput[iBackescapes] = float32(Escapes(anBoard, 23-nOppBack)) / 36.0
	afInput[iBackrescapes] = float32(escapes1(anBoard, 23-nOppBack)) / 36.0

	// Containment
	n := 36
	for i := 15; i < 24-nOppBack; i++ {
		if j := Escapes(anBoard, i); j < n {
			n = j
		}
	}
	afInput[iAcontain] = float32(36-n) / 36.0
	afInput[iAcontain2] = afInput[iAcontain] * afInput[iAcontain]

	if nOppBack < 0 {
		n = 36
	}
	for i := 15; i < 24; i++ {
		if j := Escapes(anBoard, i); j < n {
			n = j
		}
	}
	afInput[iContain] = float32(36-n) / 36.0
	afInput[iContain2] = afInput[iContain] * afInput[iContain]

	// Mobility
	mobSum := 0
	for i := 6; i < 25; i++ {
		if anBoard[i] > 0 {
			mobSum += (i - 5) * int(anBoard[i]) * Escapes(anBoardOpp, i)
		}
	}
	afInput[iMobility] = float32(mobSum) / 3600.0

	// Moment calculation
	afInput[iMoment2] = calculateMoment(anBoard)

	// Bar entry stats
	afInput[iEnter], afInput[iEnter2] = calculateBarEntry(anBoard, anBoardOpp)

	// Backbone and back game indicators
	afInput[iBackbone] = calculateBackbone(anBoard)
	afInput[iBackg], afInput[iBackg1] = calculateBackGame(anBoard)
}
