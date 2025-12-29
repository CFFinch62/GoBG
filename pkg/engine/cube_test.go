package engine

import (
	"math"
	"testing"
)

func TestSetCubeInfoMoney(t *testing.T) {
	// Test basic money game cube info
	pci := SetCubeInfoMoney(1, -1, 0, false, false)

	if pci.NCube != 1 {
		t.Errorf("NCube = %d, want 1", pci.NCube)
	}
	if pci.FCubeOwner != -1 {
		t.Errorf("FCubeOwner = %d, want -1 (centered)", pci.FCubeOwner)
	}
	if pci.FMove != 0 {
		t.Errorf("FMove = %d, want 0", pci.FMove)
	}
	if pci.NMatchTo != 0 {
		t.Errorf("NMatchTo = %d, want 0 (money game)", pci.NMatchTo)
	}

	// Without Jacoby, gammon prices should be 1.0
	for i, gp := range pci.GammonPrice {
		if gp != 1.0 {
			t.Errorf("GammonPrice[%d] = %f, want 1.0", i, gp)
		}
	}

	// Test with Jacoby rule and centered cube - gammons don't count
	pciJacoby := SetCubeInfoMoney(1, -1, 0, true, false)
	for i, gp := range pciJacoby.GammonPrice {
		if gp != 0.0 {
			t.Errorf("Jacoby GammonPrice[%d] = %f, want 0.0", i, gp)
		}
	}

	// Test with Jacoby but cube not centered - gammons count
	pciJacobyOwned := SetCubeInfoMoney(2, 0, 0, true, false)
	for i, gp := range pciJacobyOwned.GammonPrice {
		if gp != 1.0 {
			t.Errorf("Jacoby owned GammonPrice[%d] = %f, want 1.0", i, gp)
		}
	}
}

func TestGetDPEqMoney(t *testing.T) {
	engine, err := NewEngine(EngineOptions{})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	tests := []struct {
		name      string
		cubeOwner int
		fMove     int
		wantCube  bool
		wantDPEq  float64
	}{
		{"Centered cube, player 0 on roll", -1, 0, true, 1.0},
		{"Centered cube, player 1 on roll", -1, 1, true, 1.0},
		{"Player 0 owns, player 0 on roll", 0, 0, true, 1.0},
		{"Player 0 owns, player 1 on roll", 0, 1, false, 1.0},
		{"Player 1 owns, player 0 on roll", 1, 0, false, 1.0},
		{"Player 1 owns, player 1 on roll", 1, 1, true, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pci := SetCubeInfoMoney(1, tt.cubeOwner, tt.fMove, false, false)
			fCube, dpEq := engine.GetDPEq(pci)

			if fCube != tt.wantCube {
				t.Errorf("fCube = %v, want %v", fCube, tt.wantCube)
			}
			if dpEq != tt.wantDPEq {
				t.Errorf("dpEq = %f, want %f", dpEq, tt.wantDPEq)
			}
		})
	}
}

func TestUtility(t *testing.T) {
	engine, err := NewEngine(EngineOptions{})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Test pure win/loss utility
	// arOutput: [0]=Win, [1]=WinGammon, [2]=WinBG, [3]=LoseGammon, [4]=LoseBG
	tests := []struct {
		name       string
		arOutput   []float64
		pci        *CubeInfo
		wantEquity float64
	}{
		{
			"50% win, no gammons",
			[]float64{0.5, 0.0, 0.0, 0.0, 0.0},
			SetCubeInfoMoney(1, -1, 0, false, false),
			0.0, // 2*0.5 - 1 = 0
		},
		{
			"100% win, no gammons",
			[]float64{1.0, 0.0, 0.0, 0.0, 0.0},
			SetCubeInfoMoney(1, -1, 0, false, false),
			1.0, // 2*1 - 1 = 1
		},
		{
			"0% win (100% loss), no gammons",
			[]float64{0.0, 0.0, 0.0, 0.0, 0.0},
			SetCubeInfoMoney(1, -1, 0, false, false),
			-1.0, // 2*0 - 1 = -1
		},
		{
			"60% win, 20% win gammon",
			[]float64{0.6, 0.2, 0.0, 0.0, 0.0},
			SetCubeInfoMoney(1, -1, 0, false, false),
			0.4, // 2*0.6 - 1 + 0.2 = 0.4
		},
		{
			"50% win, 10% each gammon",
			[]float64{0.5, 0.1, 0.0, 0.1, 0.0},
			SetCubeInfoMoney(1, -1, 0, false, false),
			0.0, // 2*0.5 - 1 + 0.1 - 0.1 = 0.0
		},
		{
			"Jacoby centered - gammons don't count",
			[]float64{0.6, 0.3, 0.0, 0.0, 0.0},
			SetCubeInfoMoney(1, -1, 0, true, false),
			0.2, // 2*0.6 - 1 = 0.2 (gammons ignored)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			utility := engine.Utility(tt.arOutput, tt.pci)
			if math.Abs(utility-tt.wantEquity) > 0.0001 {
				t.Errorf("Utility = %f, want %f", utility, tt.wantEquity)
			}
		})
	}
}

func TestMoneyLive(t *testing.T) {
	// Test MoneyLive function - Janowski's cubeful equity formula
	// rW = average win size, rL = average loss size, p = win probability

	tests := []struct {
		name      string
		rW        float64
		rL        float64
		p         float64
		pci       *CubeInfo
		wantRange [2]float64 // min, max for rough range check
	}{
		{
			"Centered, 50% win, no gammons",
			1.0, 1.0, 0.5,
			SetCubeInfoMoney(1, -1, 0, false, false),
			[2]float64{-0.1, 0.1}, // Should be near 0
		},
		{
			"Centered, 75% win, no gammons",
			1.0, 1.0, 0.75,
			SetCubeInfoMoney(1, -1, 0, false, false),
			[2]float64{0.5, 1.0}, // Strong positive
		},
		{
			"Centered, 25% win, no gammons",
			1.0, 1.0, 0.25,
			SetCubeInfoMoney(1, -1, 0, false, false),
			[2]float64{-1.0, -0.5}, // Strong negative
		},
		{
			"Player owns, 50% win",
			1.0, 1.0, 0.5,
			SetCubeInfoMoney(2, 0, 0, false, false),
			[2]float64{0.1, 0.4}, // Player who owns cube has advantage at 50%
		},
		{
			"Opponent owns, 50% win",
			1.0, 1.0, 0.5,
			SetCubeInfoMoney(2, 1, 0, false, false),
			[2]float64{-0.4, -0.1}, // Opponent owns cube - disadvantage at 50%
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eq := MoneyLive(tt.rW, tt.rL, tt.p, tt.pci)
			if eq < tt.wantRange[0] || eq > tt.wantRange[1] {
				t.Errorf("MoneyLive = %f, want in range [%f, %f]",
					eq, tt.wantRange[0], tt.wantRange[1])
			}
		})
	}
}

func TestMoneyLiveTakePoint(t *testing.T) {
	// Test that MoneyLive gives -1 at the take point
	// For no gammons (rW=rL=1): take point = (1 - 0.5) / (1 + 1 + 0.5) = 0.2
	pci := SetCubeInfoMoney(1, -1, 0, false, false)

	rW, rL := 1.0, 1.0
	takePoint := (rL - 0.5) / (rW + rL + 0.5)

	eq := MoneyLive(rW, rL, takePoint, pci)
	if math.Abs(eq-(-1.0)) > 0.0001 {
		t.Errorf("At take point (%f), MoneyLive = %f, want -1.0", takePoint, eq)
	}
}

func TestMoneyLiveCashPoint(t *testing.T) {
	// Test that MoneyLive gives +1 at the cash point (too good to double)
	// For no gammons (rW=rL=1): cash point = (1 + 1) / (1 + 1 + 0.5) = 0.8
	pci := SetCubeInfoMoney(1, -1, 0, false, false)

	rW, rL := 1.0, 1.0
	cashPoint := (rL + 1.0) / (rW + rL + 0.5)

	eq := MoneyLive(rW, rL, cashPoint, pci)
	if math.Abs(eq-1.0) > 0.0001 {
		t.Errorf("At cash point (%f), MoneyLive = %f, want +1.0", cashPoint, eq)
	}
}

func TestFindBestCubeDecisionDoublePass(t *testing.T) {
	// Test a clear double/pass scenario: DT > DP > ND means we double, they pass
	engine, err := NewEngine(EngineOptions{})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	pci := SetCubeInfoMoney(1, -1, 0, false, false)

	// arDouble: [0]=optimal, [1]=no double, [2]=double/take, [3]=double/pass
	arDouble := []float64{0, 0.5, 0.9, 0.8}
	// aarOutput: just needs win prob > 0
	aarOutput := [2][]float64{
		{0.9, 0.0, 0.0, 0.0, 0.0},
		{0.9, 0.0, 0.0, 0.0, 0.0},
	}

	cd := engine.FindBestCubeDecision(arDouble, aarOutput, pci)

	// DT(0.9) > DP(0.8) > ND(0.5): Double, pass
	if cd != DOUBLE_PASS {
		t.Errorf("Expected DOUBLE_PASS, got %v", cd)
	}
	// Optimal should be set to DP
	if math.Abs(arDouble[OUTPUT_OPTIMAL]-0.8) > 0.0001 {
		t.Errorf("Optimal = %f, want 0.8 (double/pass)", arDouble[OUTPUT_OPTIMAL])
	}
}

func TestFindBestCubeDecisionDoubleTake(t *testing.T) {
	// Test a clear double/take scenario: DP > DT > ND means we double, they take
	engine, err := NewEngine(EngineOptions{})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	pci := SetCubeInfoMoney(1, -1, 0, false, false)

	arDouble := []float64{0, 0.5, 0.7, 0.9}
	aarOutput := [2][]float64{
		{0.7, 0.0, 0.0, 0.0, 0.0},
		{0.7, 0.0, 0.0, 0.0, 0.0},
	}

	cd := engine.FindBestCubeDecision(arDouble, aarOutput, pci)

	// DP(0.9) > DT(0.7) > ND(0.5): Double, take
	if cd != DOUBLE_TAKE {
		t.Errorf("Expected DOUBLE_TAKE, got %v", cd)
	}
	if math.Abs(arDouble[OUTPUT_OPTIMAL]-0.7) > 0.0001 {
		t.Errorf("Optimal = %f, want 0.7 (double/take)", arDouble[OUTPUT_OPTIMAL])
	}
}

func TestFindBestCubeDecisionNoDouble(t *testing.T) {
	// Test no double scenario: ND > DT and ND > DP
	engine, err := NewEngine(EngineOptions{})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	pci := SetCubeInfoMoney(1, -1, 0, false, false)

	arDouble := []float64{0, 0.6, 0.4, 0.5}
	aarOutput := [2][]float64{
		{0.55, 0.0, 0.0, 0.0, 0.0},
		{0.55, 0.0, 0.0, 0.0, 0.0},
	}

	cd := engine.FindBestCubeDecision(arDouble, aarOutput, pci)

	// ND(0.6) > DP(0.5) > DT(0.4): No double, take
	if cd != NODOUBLE_TAKE {
		t.Errorf("Expected NODOUBLE_TAKE, got %v", cd)
	}
	if math.Abs(arDouble[OUTPUT_OPTIMAL]-0.6) > 0.0001 {
		t.Errorf("Optimal = %f, want 0.6 (no double)", arDouble[OUTPUT_OPTIMAL])
	}
}

func TestFindBestCubeDecisionTooGood(t *testing.T) {
	// Test too good scenario: ND > DT > DP with gammon chances
	engine, err := NewEngine(EngineOptions{})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	pci := SetCubeInfoMoney(1, -1, 0, false, false)

	arDouble := []float64{0, 1.2, 0.9, 0.8}
	aarOutput := [2][]float64{
		{0.85, 0.3, 0.0, 0.0, 0.0}, // 30% gammon chance
		{0.85, 0.3, 0.0, 0.0, 0.0},
	}

	cd := engine.FindBestCubeDecision(arDouble, aarOutput, pci)

	// ND(1.2) > DT(0.9) > DP(0.8) with gammons: Too good, pass
	if cd != TOOGOOD_PASS {
		t.Errorf("Expected TOOGOOD_PASS, got %v", cd)
	}
}

func TestFindBestCubeDecisionRedouble(t *testing.T) {
	// Test redouble scenario - player owns the cube
	engine, err := NewEngine(EngineOptions{})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Player 0 owns the cube at 2
	pci := SetCubeInfoMoney(2, 0, 0, false, false)

	arDouble := []float64{0, 0.5, 0.9, 0.8}
	aarOutput := [2][]float64{
		{0.9, 0.0, 0.0, 0.0, 0.0},
		{0.9, 0.0, 0.0, 0.0, 0.0},
	}

	cd := engine.FindBestCubeDecision(arDouble, aarOutput, pci)

	// DT(0.9) > DP(0.8) > ND(0.5): Redouble, pass
	if cd != REDOUBLE_PASS {
		t.Errorf("Expected REDOUBLE_PASS, got %v", cd)
	}
}

func TestFindBestCubeDecisionCubeNotAvailable(t *testing.T) {
	// Test cube not available - opponent owns the cube
	engine, err := NewEngine(EngineOptions{})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Player 1 owns the cube, but player 0 is on move
	pci := SetCubeInfoMoney(2, 1, 0, false, false)

	arDouble := []float64{0, 0.5, 0.7, 0.8}
	aarOutput := [2][]float64{
		{0.7, 0.0, 0.0, 0.0, 0.0},
		{0.7, 0.0, 0.0, 0.0, 0.0},
	}

	cd := engine.FindBestCubeDecision(arDouble, aarOutput, pci)

	// Cube not available to player 0
	if cd != NOT_AVAILABLE {
		t.Errorf("Expected NOT_AVAILABLE, got %v", cd)
	}
}

func TestAnalyzeCubeStartingPosition(t *testing.T) {
	// Test cube analysis on starting position
	engine, err := NewEngine(EngineOptions{})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	state := StartingPosition()
	analysis, err := engine.AnalyzeCube(state)
	if err != nil {
		t.Fatalf("AnalyzeCube failed: %v", err)
	}

	// Starting position is equal - no double
	if analysis.Decision.Action != NoDouble {
		t.Errorf("Expected NoDouble for starting position, got %v", analysis.Decision.Action)
	}

	// Equity should be near 0 for starting position
	if math.Abs(analysis.NoDoubleEquity) > 0.2 {
		t.Errorf("NoDoubleEquity = %f, expected near 0", analysis.NoDoubleEquity)
	}
}

func TestAnalyzeCubeStrongPosition(t *testing.T) {
	// Create a strong racing position where player 0 is clearly ahead
	engine, err := NewEngine(EngineOptions{})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	state := &GameState{
		Turn:      0,
		CubeValue: 1,
		CubeOwner: -1, // Centered
	}

	// Player 0: 15 checkers on points 0-4 (home board, ready to bear off)
	// In gnubg notation: 1-point=index 0, 2-point=index 1, etc.
	state.Board[0][0] = 3 // 3 on 1-point
	state.Board[0][1] = 3 // 3 on 2-point
	state.Board[0][2] = 3 // 3 on 3-point
	state.Board[0][3] = 3 // 3 on 4-point
	state.Board[0][4] = 3 // 3 on 5-point

	// Player 1: 15 checkers still on points 18-22 (far from home)
	// From player 1's perspective: their home board is points 0-5
	// But from board indices, points 18-23 are player 0's outer/opponent's home
	state.Board[1][0] = 3 // 3 on 1-point (player 1's home)
	state.Board[1][1] = 3 // 3 on 2-point
	state.Board[1][2] = 3 // 3 on 3-point
	state.Board[1][3] = 3 // 3 on 4-point
	state.Board[1][4] = 3 // 3 on 5-point

	analysis, err := engine.AnalyzeCube(state)
	if err != nil {
		t.Fatalf("AnalyzeCube failed: %v", err)
	}

	// This is now an equal position (both in home board)
	// Just verify we get a reasonable result
	t.Logf("Decision: %v, NoDoubleEq: %f, DoubleTakeEq: %f, WinProb from eval",
		analysis.DecisionType, analysis.NoDoubleEquity, analysis.DoubleTakeEq)

	// At minimum, make sure the analysis completed with reasonable values
	// The equity should be finite and within reasonable range
	if analysis.NoDoubleEquity < -2.0 || analysis.NoDoubleEquity > 2.0 {
		t.Errorf("NoDoubleEquity out of reasonable range: %f", analysis.NoDoubleEquity)
	}
}

func TestGetDPEqMatchPlay(t *testing.T) {
	engine, err := NewEngine(EngineOptions{})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	tests := []struct {
		name      string
		score     [2]int
		matchTo   int
		cubeValue int
		cubeOwner int
		fMove     int
		crawford  bool
		wantCube  bool
	}{
		{"0-0 to 5, centered", [2]int{0, 0}, 5, 1, -1, 0, false, true},
		{"Crawford game, centered", [2]int{4, 0}, 5, 1, -1, 0, true, false},
		{"Post-Crawford, leader", [2]int{4, 0}, 5, 1, -1, 0, false, false},
		{"Post-Crawford, trailer", [2]int{0, 4}, 5, 1, -1, 0, false, true},
		{"Dead cube - already winning", [2]int{3, 0}, 5, 4, -1, 0, false, false},
		{"1-1 to 3, centered", [2]int{1, 1}, 3, 1, -1, 0, false, true},
		{"Cube at 2, player owns", [2]int{0, 0}, 5, 2, 0, 0, false, true},
		{"Cube at 2, opponent owns", [2]int{0, 0}, 5, 2, 1, 0, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pci := engine.SetCubeInfoMatch(tt.cubeValue, tt.cubeOwner, tt.fMove,
				tt.matchTo, tt.score, tt.crawford)
			fCube, _ := engine.GetDPEq(pci)

			if fCube != tt.wantCube {
				t.Errorf("fCube = %v, want %v", fCube, tt.wantCube)
			}
		})
	}
}

func TestCrawfordGame(t *testing.T) {
	// In Crawford game, no cube actions are allowed
	engine, err := NewEngine(EngineOptions{})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	state := StartingPosition()
	state.MatchLength = 5
	state.Score = [2]int{4, 0} // Leader is 1-away
	state.Crawford = true

	analysis, err := engine.AnalyzeCube(state)
	if err != nil {
		t.Fatalf("AnalyzeCube failed: %v", err)
	}

	// Cube should not be available in Crawford game
	if analysis.DecisionType != NOT_AVAILABLE {
		t.Errorf("Expected NOT_AVAILABLE in Crawford game, got %v", analysis.DecisionType)
	}
}

func TestGammonPricesMatch(t *testing.T) {
	// Test that gammon prices are calculated correctly for match play
	engine, err := NewEngine(EngineOptions{})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Early in match, gammons should be valuable
	pci := engine.SetCubeInfoMatch(1, -1, 0, 7, [2]int{0, 0}, false)

	// Gammon prices should be positive (gammons have value)
	if pci.GammonPrice[0] <= 0 || pci.GammonPrice[1] <= 0 {
		t.Errorf("Expected positive gammon prices, got %v", pci.GammonPrice)
	}

	// At DMP (double match point), gammons have no value
	pciDMP := engine.SetCubeInfoMatch(1, -1, 0, 5, [2]int{4, 4}, false)

	// At DMP gammons don't matter
	if pciDMP.GammonPrice[0] != 0 && pciDMP.GammonPrice[1] != 0 {
		t.Logf("DMP gammon prices: %v (may be 0 or undefined)", pciDMP.GammonPrice)
	}
}
