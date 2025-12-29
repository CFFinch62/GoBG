package engine

import (
	"math"
)

// CubeDecisionType represents the detailed cube decision (matching gnubg's cubedecision enum)
type CubeDecisionType int

const (
	DOUBLE_TAKE CubeDecisionType = iota
	DOUBLE_PASS
	NODOUBLE_TAKE
	TOOGOOD_TAKE
	TOOGOOD_PASS
	DOUBLE_BEAVER
	NODOUBLE_BEAVER
	REDOUBLE_TAKE
	REDOUBLE_PASS
	NO_REDOUBLE_TAKE
	TOOGOODRE_TAKE
	TOOGOODRE_PASS
	NO_REDOUBLE_BEAVER
	NODOUBLE_DEADCUBE    // cube is dead (match play only)
	NO_REDOUBLE_DEADCUBE // cube is dead (match play only)
	NOT_AVAILABLE        // Cube not available
	OPTIONAL_DOUBLE_TAKE
	OPTIONAL_REDOUBLE_TAKE
	OPTIONAL_DOUBLE_BEAVER
	OPTIONAL_DOUBLE_PASS
	OPTIONAL_REDOUBLE_PASS
)

// DoubleType represents the type of double (matching gnubg's doubletype enum)
type DoubleType int

const (
	DT_NORMAL DoubleType = iota
	DT_BEAVER
	DT_RACCOON
	NUM_DOUBLE_TYPES
)

// CubeInfo contains the information necessary for cube evaluation
// This matches gnubg's cubeinfo struct
type CubeInfo struct {
	NCube       int        // Current value of the cube
	FCubeOwner  int        // Owner of the cube: -1=centered, 0=player0, 1=player1
	FMove       int        // Player for which we calculate equity
	NMatchTo    int        // Match length (0 = money game)
	AnScore     [2]int     // Current score
	FCrawford   bool       // Crawford game flag
	FJacoby     bool       // Jacoby rule in effect
	FBeavers    bool       // Beavers allowed
	GammonPrice [4]float32 // Gammon prices: [0]=gammon for p0, [1]=gammon for p1, [2]=bg for p0, [3]=bg for p1
}

// Output indices for cube decision arrays (matching gnubg)
const (
	OUTPUT_OPTIMAL  = 0
	OUTPUT_NODOUBLE = 1
	OUTPUT_TAKE     = 2
	OUTPUT_DROP     = 3
)

// CubeAnalysis contains detailed cube analysis
type CubeAnalysis struct {
	Decision       CubeDecision     // Simple decision for public API
	DecisionType   CubeDecisionType // Detailed decision type
	ArDouble       [4]float64       // Equities: [0]=optimal, [1]=no double, [2]=double/take, [3]=double/pass
	NoDoubleEquity float64          // Equity if player doesn't double
	DoubleTakeEq   float64          // Equity if player doubles and opponent takes
	DoublePassEq   float64          // Equity if player doubles and opponent passes
	TakePoint      float64          // Win probability needed to take
	DoublePoint    float64          // Win probability needed to double
	TooGoodPoint   float64          // Win probability above which double is wrong (too good)
}

// SetCubeInfoMoney initializes CubeInfo for money game (matching gnubg)
func SetCubeInfoMoney(nCube, fCubeOwner, fMove int, fJacoby, fBeavers bool) *CubeInfo {
	pci := &CubeInfo{
		NCube:      nCube,
		FCubeOwner: fCubeOwner,
		FMove:      fMove,
		FJacoby:    fJacoby,
		FBeavers:   fBeavers,
		NMatchTo:   0,
	}

	// Set gammon prices for money game
	var gammonPrice float32 = 1.0
	if fJacoby && fCubeOwner == -1 {
		gammonPrice = 0.0 // No gammon value with centered cube and Jacoby
	}
	pci.GammonPrice = [4]float32{gammonPrice, gammonPrice, gammonPrice, gammonPrice}

	return pci
}

// SetCubeInfoMatch initializes CubeInfo for match play (matching gnubg)
func (e *Engine) SetCubeInfoMatch(nCube, fCubeOwner, fMove, nMatchTo int,
	anScore [2]int, fCrawford bool) *CubeInfo {
	pci := &CubeInfo{
		NCube:      nCube,
		FCubeOwner: fCubeOwner,
		FMove:      fMove,
		NMatchTo:   nMatchTo,
		AnScore:    anScore,
		FCrawford:  fCrawford,
		FJacoby:    false,
		FBeavers:   false,
	}

	// Calculate gammon prices from MET
	// This is simplified - gnubg uses precomputed tables
	e.calculateGammonPrices(pci)

	return pci
}

// calculateGammonPrices calculates gammon price for match play
func (e *Engine) calculateGammonPrices(pci *CubeInfo) {
	if e.met == nil || pci.NMatchTo == 0 {
		pci.GammonPrice = [4]float32{1.0, 1.0, 1.0, 1.0}
		return
	}

	// Gammon price = (MWC(+2) - MWC(+1)) / (MWC(+1) - MWC(-1))
	// where MWC is match winning chance after winning/losing games
	for player := 0; player < 2; player++ {
		// Normal win
		scoreWin1 := [2]int{pci.AnScore[0], pci.AnScore[1]}
		scoreWin1[player] += pci.NCube
		mwcWin1 := e.getMWCForScore(scoreWin1, pci.NMatchTo, player, pci.FCrawford)

		// Gammon win
		scoreWin2 := [2]int{pci.AnScore[0], pci.AnScore[1]}
		scoreWin2[player] += 2 * pci.NCube
		mwcWin2 := e.getMWCForScore(scoreWin2, pci.NMatchTo, player, pci.FCrawford)

		// Backgammon win
		scoreWin3 := [2]int{pci.AnScore[0], pci.AnScore[1]}
		scoreWin3[player] += 3 * pci.NCube
		mwcWin3 := e.getMWCForScore(scoreWin3, pci.NMatchTo, player, pci.FCrawford)

		// Lose
		scoreLose := [2]int{pci.AnScore[0], pci.AnScore[1]}
		scoreLose[1-player] += pci.NCube
		mwcLose := e.getMWCForScore(scoreLose, pci.NMatchTo, player, pci.FCrawford)

		// Calculate gammon price
		denom := mwcWin1 - mwcLose
		if denom > 0.0001 {
			pci.GammonPrice[player] = float32((mwcWin2 - mwcWin1) / denom)
			pci.GammonPrice[player+2] = float32((mwcWin3 - mwcWin2) / denom)
		} else {
			pci.GammonPrice[player] = 0.0
			pci.GammonPrice[player+2] = 0.0
		}
	}
}

// getMWCForScore returns match winning chance for a given score
func (e *Engine) getMWCForScore(score [2]int, matchTo, player int, crawford bool) float64 {
	if score[player] >= matchTo {
		return 1.0
	}
	if score[1-player] >= matchTo {
		return 0.0
	}
	if e.met == nil {
		return 0.5
	}
	return float64(e.met.GetME(score[0], score[1], matchTo, player, crawford))
}

// GetDPEq returns the double/pass equity and whether cube is available
// This matches gnubg's GetDPEq function
func (e *Engine) GetDPEq(pci *CubeInfo) (fCube bool, dpEq float64) {
	if pci.NMatchTo == 0 {
		// Money game: Double, pass equity is 1.0 points (normalized to 1-cube)
		// Cube available if centered or owned by player to move
		dpEq = 1.0
		fCube = (pci.FCubeOwner == -1) || (pci.FCubeOwner == pci.FMove)
		return
	}

	// Match play: Equity comes from MET
	// Cube available if:
	// - Not Crawford game
	// - Cube is not dead (winning cube points wouldn't exceed match)
	// - Post-Crawford: only trailer can double
	// - Player has access to cube

	fPostCrawford := !pci.FCrawford &&
		(pci.AnScore[0] == pci.NMatchTo-1 || pci.AnScore[1] == pci.NMatchTo-1)

	fCube = (!pci.FCrawford) &&
		(pci.AnScore[pci.FMove]+pci.NCube < pci.NMatchTo) &&
		(!(fPostCrawford && (pci.AnScore[pci.FMove] == pci.NMatchTo-1))) &&
		((pci.FCubeOwner == -1) || (pci.FCubeOwner == pci.FMove))

	// Get double/pass equity from MET
	if e.met != nil {
		dpEq = float64(e.met.GetME(pci.AnScore[0], pci.AnScore[1],
			pci.NMatchTo, pci.FMove, pci.FCrawford))
	} else {
		dpEq = 1.0
	}

	return
}

// MoneyLive calculates the live cube equity for money games
// This matches gnubg's MoneyLive function exactly
func MoneyLive(rW, rL, p float64, pci *CubeInfo) float64 {
	if pci.FCubeOwner == -1 {
		// Centered cube
		rTP := (rL - 0.5) / (rW + rL + 0.5)
		rCP := (rL + 1.0) / (rW + rL + 0.5)

		if p < rTP {
			// Linear interpolation between (0,-rL) and (rTP,-1)
			if pci.FJacoby {
				return -1.0
			}
			return -rL + (-1.0+rL)*p/rTP
		} else if p < rCP {
			// Linear interpolation between (rTP,-1) and (rCP,+1)
			return -1.0 + 2.0*(p-rTP)/(rCP-rTP)
		} else {
			// Linear interpolation between (rCP,+1) and (1,+rW)
			if pci.FJacoby {
				return 1.0
			}
			return 1.0 + (rW-1.0)*(p-rCP)/(1.0-rCP)
		}
	} else if pci.FCubeOwner == pci.FMove {
		// Owned cube - player owns it
		rCP := (rL + 1.0) / (rW + rL + 0.5)

		if p < rCP {
			// Linear interpolation between (0,-rL) and (rCP,+1)
			return -rL + (1.0+rL)*p/rCP
		} else {
			// Linear interpolation between (rCP,+1) and (1,+rW)
			return 1.0 + (rW-1.0)*(p-rCP)/(1.0-rCP)
		}
	} else {
		// Unavailable cube - opponent owns it
		rTP := (rL - 0.5) / (rW + rL + 0.5)

		if p < rTP {
			// Linear interpolation between (0,-rL) and (rTP,-1)
			return -rL + (-1.0+rL)*p/rTP
		} else {
			// Linear interpolation between (rTP,-1) and (1,rW)
			return -1.0 + (rW+1.0)*(p-rTP)/(1.0-rTP)
		}
	}
}

// Cl2CfMoney transforms cubeless equity to cubeful equity for money games
// This matches gnubg's Cl2CfMoney function
func (e *Engine) Cl2CfMoney(arOutput []float64, pci *CubeInfo, rCubeX float64) float64 {
	const epsilon = 0.0000001
	const omepsilon = 0.9999999

	var rW, rL float64

	// Calculate average win and loss W and L
	if arOutput[0] > epsilon { // OUTPUT_WIN
		rW = 1.0 + (arOutput[1]+arOutput[2])/arOutput[0] // WINGAMMON + WINBACKGAMMON
	} else {
		// Basically a dead cube
		return e.Utility(arOutput, pci)
	}

	if arOutput[0] < omepsilon {
		rL = 1.0 + (arOutput[3]+arOutput[4])/(1.0-arOutput[0]) // LOSEGAMMON + LOSEBACKGAMMON
	} else {
		// Basically a dead cube
		return e.Utility(arOutput, pci)
	}

	rEqDead := e.Utility(arOutput, pci)
	rEqLive := MoneyLive(rW, rL, arOutput[0], pci)

	return rEqDead*(1.0-rCubeX) + rEqLive*rCubeX
}

// Utility calculates the cubeless equity from output probabilities
// This matches gnubg's Utility function
func (e *Engine) Utility(arOutput []float64, pci *CubeInfo) float64 {
	// arOutput: [0]=Win, [1]=WinGammon, [2]=WinBackgammon, [3]=LoseGammon, [4]=LoseBackgammon

	// For money games with Jacoby rule and centered cube, gammons don't count
	if pci.NMatchTo == 0 && pci.FJacoby && pci.FCubeOwner == -1 {
		return 2.0*arOutput[0] - 1.0
	}

	// Standard utility calculation
	rUtility := 2.0*arOutput[0] - 1.0 + // Win probability contribution
		arOutput[1] + arOutput[2] - // Gammon and backgammon wins
		arOutput[3] - arOutput[4] // Gammon and backgammon losses

	return rUtility
}

// isOptional checks if two equities are close enough to be optional
func isOptional(r1, r2 float64) bool {
	const epsilon = 1.0e-5
	return math.Abs(r1-r2) <= epsilon
}

// winGammon checks if there's any chance of winning a gammon
func winGammon(arOutput []float64) bool {
	return arOutput[1] > 0.0 // OUTPUT_WINGAMMON
}

// winAny checks if there's any chance of winning
func winAny(arOutput []float64) bool {
	return arOutput[0] > 0.0 // OUTPUT_WIN
}

// AnalyzeCube analyzes the cube decision for the player on roll
func (e *Engine) AnalyzeCube(state *GameState) (*CubeAnalysis, error) {
	// First, get the evaluation of the current position
	eval, err := e.Evaluate(state)
	if err != nil {
		return nil, err
	}

	analysis := &CubeAnalysis{}

	// Build CubeInfo from GameState
	var pci *CubeInfo
	if state.MatchLength == 0 {
		pci = SetCubeInfoMoney(state.CubeValue, state.CubeOwner, state.Turn, false, false)
	} else {
		pci = e.SetCubeInfoMatch(state.CubeValue, state.CubeOwner, state.Turn,
			state.MatchLength, state.Score, state.Crawford)
	}

	// Check if cube is available
	fCube, dpEq := e.GetDPEq(pci)

	if !fCube {
		// Cube not available, return cubeless evaluation
		analysis.NoDoubleEquity = eval.Equity
		analysis.DecisionType = NOT_AVAILABLE
		analysis.Decision = CubeDecision{
			Action:         NoDouble,
			NoDoubleEquity: eval.Equity,
		}
		return analysis, nil
	}

	// Calculate the three key equities: no double, double/take, double/pass
	arDouble := [4]float64{}
	arDouble[OUTPUT_DROP] = dpEq

	// Build output arrays for gnubg-style decision
	arOutput := []float64{eval.WinProb, eval.WinG, eval.WinBG, eval.LoseG, eval.LoseBG}
	aarOutput := [2][]float64{arOutput, arOutput}

	// Calculate no-double equity (cubeful)
	if state.MatchLength == 0 {
		// Money game: use Janowski's formula
		rCubeX := 0.68 // Default cube efficiency
		analysis.NoDoubleEquity = e.Cl2CfMoney(arOutput, pci, rCubeX)
		arDouble[OUTPUT_NODOUBLE] = analysis.NoDoubleEquity

		// Double/take equity: evaluate with opponent owning doubled cube
		pciDT := SetCubeInfoMoney(pci.NCube*2, 1-pci.FMove, pci.FMove, pci.FJacoby, pci.FBeavers)
		analysis.DoubleTakeEq = 2.0 * e.Cl2CfMoney(arOutput, pciDT, rCubeX)
		arDouble[OUTPUT_TAKE] = analysis.DoubleTakeEq
	} else {
		// Match play: use MET-based calculation
		player := state.Turn
		cubeValue := state.CubeValue

		// MWC for various outcomes
		mwcWin := e.getMWCAfterWin(state, player, cubeValue)
		mwcLose := e.getMWCAfterLoss(state, player, cubeValue)
		p := eval.WinProb

		// No double MWC
		mwcNoDouble := p*mwcWin + (1-p)*mwcLose
		analysis.NoDoubleEquity = e.Mwc2Eq(float32(mwcNoDouble), pci)
		arDouble[OUTPUT_NODOUBLE] = analysis.NoDoubleEquity

		// Double/take MWC
		mwcWin2 := e.getMWCAfterWin(state, player, cubeValue*2)
		mwcLose2 := e.getMWCAfterLoss(state, player, cubeValue*2)
		mwcDoubleTake := p*mwcWin2 + (1-p)*mwcLose2
		analysis.DoubleTakeEq = e.Mwc2Eq(float32(mwcDoubleTake), pci)
		arDouble[OUTPUT_TAKE] = analysis.DoubleTakeEq
	}

	analysis.DoublePassEq = dpEq
	analysis.ArDouble = arDouble

	// Find best cube decision
	analysis.DecisionType = e.FindBestCubeDecision(arDouble[:], aarOutput, pci)

	// Calculate decision points for display
	w := 1.0 + (eval.WinG + eval.WinBG)
	l := 1.0 + (eval.LoseG + eval.LoseBG)
	analysis.TakePoint = (l - 0.5) / (w + l + 0.5)
	analysis.DoublePoint = analysis.TakePoint
	analysis.TooGoodPoint = (l + 1) / (w + l + 0.5)

	// Convert to simple CubeDecision for public API
	analysis.Decision = e.cubeDecisionTypeToAction(analysis.DecisionType, analysis)

	return analysis, nil
}

// cubeDecisionTypeToAction converts detailed decision type to simple action
func (e *Engine) cubeDecisionTypeToAction(cdt CubeDecisionType, analysis *CubeAnalysis) CubeDecision {
	switch cdt {
	case DOUBLE_TAKE, DOUBLE_BEAVER, REDOUBLE_TAKE,
		OPTIONAL_DOUBLE_TAKE, OPTIONAL_REDOUBLE_TAKE, OPTIONAL_DOUBLE_BEAVER:
		return CubeDecision{
			Action:         Double,
			DoubleEquity:   analysis.DoubleTakeEq,
			NoDoubleEquity: analysis.NoDoubleEquity,
			TakeEquity:     -analysis.DoubleTakeEq,
		}
	case DOUBLE_PASS, REDOUBLE_PASS, OPTIONAL_DOUBLE_PASS, OPTIONAL_REDOUBLE_PASS:
		return CubeDecision{
			Action:         Double,
			DoubleEquity:   analysis.DoublePassEq,
			NoDoubleEquity: analysis.NoDoubleEquity,
		}
	default:
		return CubeDecision{
			Action:         NoDouble,
			NoDoubleEquity: analysis.NoDoubleEquity,
		}
	}
}

// getMWCAfterWin returns match winning chance after winning the game
func (e *Engine) getMWCAfterWin(state *GameState, player, points int) float64 {
	if e.met == nil {
		return 0.5
	}
	newScore := [2]int{state.Score[0], state.Score[1]}
	newScore[player] += points
	if newScore[player] >= state.MatchLength {
		return 1.0
	}
	return float64(e.met.GetME(newScore[0], newScore[1], state.MatchLength, player, state.Crawford))
}

// getMWCAfterLoss returns match winning chance after losing the game
func (e *Engine) getMWCAfterLoss(state *GameState, player, points int) float64 {
	if e.met == nil {
		return 0.5
	}
	opponent := 1 - player
	newScore := [2]int{state.Score[0], state.Score[1]}
	newScore[opponent] += points
	if newScore[opponent] >= state.MatchLength {
		return 0.0
	}
	return float64(e.met.GetME(newScore[0], newScore[1], state.MatchLength, player, state.Crawford))
}

// Mwc2Eq converts match winning chance to equity
// This matches gnubg's mwc2eq function
func (e *Engine) Mwc2Eq(rMwc float32, pci *CubeInfo) float64 {
	if pci.NMatchTo == 0 {
		return float64(rMwc)
	}
	// Get current MWC
	currentMwc := float64(e.met.GetME(pci.AnScore[0], pci.AnScore[1],
		pci.NMatchTo, pci.FMove, pci.FCrawford))

	// Convert to normalized equity
	// eq = (mwc - 0.5) * 2 scaled by current position
	if currentMwc > 0.0001 && currentMwc < 0.9999 {
		return (float64(rMwc) - currentMwc) / math.Min(currentMwc, 1.0-currentMwc)
	}
	return 2.0*float64(rMwc) - 1.0
}

// FindBestCubeDecision finds the optimal cube decision
// This matches gnubg's FindBestCubeDecision function
func (e *Engine) FindBestCubeDecision(arDouble []float64, aarOutput [2][]float64, pci *CubeInfo) CubeDecisionType {
	// Check if cube is available
	fCube, _ := e.GetDPEq(pci)
	if !fCube {
		arDouble[OUTPUT_OPTIMAL] = arDouble[OUTPUT_NODOUBLE]

		// For match play distinguish between dead cube and not available cube
		if pci.NMatchTo > 0 && (pci.FCubeOwner < 0 || pci.FCubeOwner == pci.FMove) {
			if pci.FCubeOwner == -1 {
				return NODOUBLE_DEADCUBE
			}
			return NO_REDOUBLE_DEADCUBE
		}
		return NOT_AVAILABLE
	}

	// Cube is available: find optimal cube action
	if arDouble[OUTPUT_TAKE] >= arDouble[OUTPUT_NODOUBLE] &&
		arDouble[OUTPUT_DROP] >= arDouble[OUTPUT_NODOUBLE] {
		// DT >= ND and DP >= ND - we have a double

		if arDouble[OUTPUT_DROP] > arDouble[OUTPUT_TAKE] {
			// DP > DT >= ND: Double, take
			f := isOptional(arDouble[OUTPUT_TAKE], arDouble[OUTPUT_NODOUBLE])
			arDouble[OUTPUT_OPTIMAL] = arDouble[OUTPUT_TAKE]

			// Check for beaver (money game only)
			if pci.NMatchTo == 0 && arDouble[OUTPUT_TAKE] >= -2.0 &&
				arDouble[OUTPUT_TAKE] <= 0.0 && pci.FBeavers {
				if arDouble[OUTPUT_TAKE]*2.0 < arDouble[OUTPUT_NODOUBLE] {
					return NODOUBLE_BEAVER
				}
				if f {
					return OPTIONAL_DOUBLE_BEAVER
				}
				return DOUBLE_BEAVER
			} else if winAny(aarOutput[0]) {
				// Take
				if f {
					if pci.FCubeOwner == -1 {
						return OPTIONAL_DOUBLE_TAKE
					}
					return OPTIONAL_REDOUBLE_TAKE
				}
				if pci.FCubeOwner == -1 {
					return DOUBLE_TAKE
				}
				return REDOUBLE_TAKE
			} else {
				// No double if we're sure to lose
				if pci.FCubeOwner == -1 {
					return NODOUBLE_TAKE
				}
				return NO_REDOUBLE_TAKE
			}
		} else {
			// DT >= DP >= ND: Double, pass
			arDouble[OUTPUT_OPTIMAL] = arDouble[OUTPUT_DROP]

			// Double is optional if equities are close and can win gammon
			if isOptional(arDouble[OUTPUT_NODOUBLE], arDouble[OUTPUT_DROP]) &&
				winGammon(aarOutput[0]) &&
				(pci.NMatchTo > 0 || pci.FCubeOwner != -1 || !pci.FJacoby) {
				if pci.FCubeOwner == -1 {
					return OPTIONAL_DOUBLE_PASS
				}
				return OPTIONAL_REDOUBLE_PASS
			}
			if pci.FCubeOwner == -1 {
				return DOUBLE_PASS
			}
			return REDOUBLE_PASS
		}
	} else {
		// No double: ND > DT or ND > DP
		arDouble[OUTPUT_OPTIMAL] = arDouble[OUTPUT_NODOUBLE]

		if arDouble[OUTPUT_NODOUBLE] > arDouble[OUTPUT_TAKE] {
			// ND > DT
			if arDouble[OUTPUT_TAKE] > arDouble[OUTPUT_DROP] {
				// ND > DT > DP: Too good, pass
				if winGammon(aarOutput[0]) {
					if pci.FCubeOwner == -1 {
						return TOOGOOD_PASS
					}
					return TOOGOODRE_PASS
				}
				if pci.FCubeOwner == -1 {
					return DOUBLE_PASS
				}
				return REDOUBLE_PASS
			} else if arDouble[OUTPUT_NODOUBLE] > arDouble[OUTPUT_DROP] {
				// ND > DP > DT: Too good, take
				if winGammon(aarOutput[0]) {
					if pci.FCubeOwner == -1 {
						return TOOGOOD_TAKE
					}
					return TOOGOODRE_TAKE
				}
				if pci.FCubeOwner == -1 {
					return NODOUBLE_TAKE
				}
				return NO_REDOUBLE_TAKE
			} else {
				// DP > ND > DT: No double, take/beaver
				if arDouble[OUTPUT_TAKE] >= -2.0 && arDouble[OUTPUT_TAKE] <= 0.0 &&
					pci.NMatchTo == 0 && pci.FBeavers {
					if pci.FCubeOwner == -1 {
						return NODOUBLE_BEAVER
					}
					return NO_REDOUBLE_BEAVER
				}
				if pci.FCubeOwner == -1 {
					return NODOUBLE_TAKE
				}
				return NO_REDOUBLE_TAKE
			}
		} else {
			// DT >= ND > DP: Too good, pass
			if winGammon(aarOutput[0]) {
				if pci.FCubeOwner == -1 {
					return TOOGOOD_PASS
				}
				return TOOGOODRE_PASS
			}
			if pci.FCubeOwner == -1 {
				return DOUBLE_PASS
			}
			return REDOUBLE_PASS
		}
	}
}
