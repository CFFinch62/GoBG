package engine

import (
	"math"
	"math/rand"
	"runtime"
	"sync"
)

// RolloutOptions controls rollout execution
type RolloutOptions struct {
	Trials   int   // Number of games to simulate (default 1296)
	Truncate int   // Truncate at ply N and use evaluation (0 = play to end)
	Seed     int64 // RNG seed (0 = use current time)
	Workers  int   // Number of parallel workers (0 = GOMAXPROCS)
	Cubeful  bool  // Include cube decisions in rollout
}

// RolloutProgress contains progress information during a rollout
type RolloutProgress struct {
	TrialsCompleted int     // Number of trials completed so far
	TrialsTotal     int     // Total number of trials
	Percent         float64 // Percentage complete (0-100)
	CurrentEquity   float64 // Current equity estimate
	CurrentCI       float64 // Current 95% confidence interval
}

// ProgressCallback is called periodically during rollout with progress updates
type ProgressCallback func(progress RolloutProgress)

// RolloutResult contains the results of a rollout
type RolloutResult struct {
	// Average probabilities
	WinProb float64
	WinG    float64
	WinBG   float64
	LoseG   float64
	LoseBG  float64
	Equity  float64

	// Standard deviations
	WinProbStdDev float64
	WinGStdDev    float64
	WinBGStdDev   float64
	LoseGStdDev   float64
	LoseBGStdDev  float64
	EquityStdDev  float64

	// Confidence interval (95%)
	EquityCI float64

	// Statistics
	TrialsCompleted int
	GamesWon        int
	GammonsWon      int
	BackgammonsWon  int
	GamesLost       int
	GammonsLost     int
	BackgammonsLost int
}

// partialResult holds results from a single worker
type partialResult struct {
	sumProbs    [5]float64 // WinProb, WinG, WinBG, LoseG, LoseBG
	sumSqProbs  [5]float64 // Sum of squares for variance
	sumEquity   float64
	sumSqEquity float64
	trials      int
	wins        int
	gammonsWon  int
	bgsWon      int
	losses      int
	gammonsLost int
	bgsLost     int
}

// DefaultRolloutOptions returns sensible defaults
func DefaultRolloutOptions() RolloutOptions {
	return RolloutOptions{
		Trials:   1296, // 36^2 for full dice coverage
		Truncate: 0,    // Play to completion
		Seed:     0,    // Random seed
		Workers:  0,    // Use all cores
		Cubeful:  false,
	}
}

// Rollout performs a Monte Carlo rollout of the position
func (e *Engine) Rollout(state *GameState, opts RolloutOptions) (*RolloutResult, error) {
	// Set defaults
	if opts.Trials <= 0 {
		opts.Trials = 1296
	}
	if opts.Workers <= 0 {
		opts.Workers = runtime.GOMAXPROCS(0)
	}
	if opts.Seed == 0 {
		opts.Seed = rand.Int63()
	}

	// Distribute trials across workers
	trialsPerWorker := opts.Trials / opts.Workers
	extraTrials := opts.Trials % opts.Workers

	// Channel for collecting results
	results := make(chan partialResult, opts.Workers)
	var wg sync.WaitGroup

	// Launch workers
	for i := 0; i < opts.Workers; i++ {
		wg.Add(1)
		workerTrials := trialsPerWorker
		if i < extraTrials {
			workerTrials++
		}
		workerSeed := opts.Seed + int64(i)*1000000

		go func(trials int, seed int64) {
			defer wg.Done()
			results <- e.rolloutWorker(state, trials, seed, opts.Truncate, opts.Cubeful)
		}(workerTrials, workerSeed)
	}

	// Close channel when all workers done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Aggregate results
	return e.aggregateResults(results, opts.Trials)
}

// RolloutWithProgress performs a rollout with periodic progress callbacks
// The callback is called after each batch of trials completes
func (e *Engine) RolloutWithProgress(state *GameState, opts RolloutOptions, callback ProgressCallback) (*RolloutResult, error) {
	// Set defaults
	if opts.Trials <= 0 {
		opts.Trials = 1296
	}
	if opts.Workers <= 0 {
		opts.Workers = runtime.GOMAXPROCS(0)
	}
	if opts.Seed == 0 {
		opts.Seed = rand.Int63()
	}

	// For progress reporting, we break trials into batches
	// Report progress approximately 20 times during the rollout
	batchSize := opts.Trials / 20
	if batchSize < 1 {
		batchSize = 1
	}
	if batchSize > opts.Trials/opts.Workers {
		batchSize = opts.Trials / opts.Workers
	}
	if batchSize < 1 {
		batchSize = 1
	}

	// Channel for collecting incremental results
	incrementalResults := make(chan partialResult, opts.Workers*20)
	var wg sync.WaitGroup

	// Launch workers with batch reporting
	trialsPerWorker := opts.Trials / opts.Workers
	extraTrials := opts.Trials % opts.Workers

	for i := 0; i < opts.Workers; i++ {
		wg.Add(1)
		workerTrials := trialsPerWorker
		if i < extraTrials {
			workerTrials++
		}
		workerSeed := opts.Seed + int64(i)*1000000

		go func(trials int, seed int64, batch int) {
			defer wg.Done()
			e.rolloutWorkerWithProgress(state, trials, seed, opts.Truncate, opts.Cubeful, batch, incrementalResults)
		}(workerTrials, workerSeed, batchSize)
	}

	// Close channel when all workers done
	go func() {
		wg.Wait()
		close(incrementalResults)
	}()

	// Aggregate results with progress reporting
	return e.aggregateResultsWithProgress(incrementalResults, opts.Trials, callback)
}

// rolloutWorkerWithProgress performs rollouts and reports progress in batches
func (e *Engine) rolloutWorkerWithProgress(state *GameState, trials int, seed int64, truncate int, cubeful bool, batchSize int, results chan<- partialResult) {
	rng := rand.New(rand.NewSource(seed))

	for trialsRemaining := trials; trialsRemaining > 0; {
		// Process a batch
		currentBatch := batchSize
		if currentBatch > trialsRemaining {
			currentBatch = trialsRemaining
		}

		pr := partialResult{}
		for i := 0; i < currentBatch; i++ {
			result := e.playOutGame(state, rng, truncate, cubeful)

			pr.sumProbs[0] += result.WinProb
			pr.sumProbs[1] += result.WinG
			pr.sumProbs[2] += result.WinBG
			pr.sumProbs[3] += result.LoseG
			pr.sumProbs[4] += result.LoseBG

			pr.sumSqProbs[0] += result.WinProb * result.WinProb
			pr.sumSqProbs[1] += result.WinG * result.WinG
			pr.sumSqProbs[2] += result.WinBG * result.WinBG
			pr.sumSqProbs[3] += result.LoseG * result.LoseG
			pr.sumSqProbs[4] += result.LoseBG * result.LoseBG

			pr.sumEquity += result.Equity
			pr.sumSqEquity += result.Equity * result.Equity
			pr.trials++

			if result.WinProb > 0.5 {
				pr.wins++
				if result.WinBG > 0 {
					pr.bgsWon++
				} else if result.WinG > 0 {
					pr.gammonsWon++
				}
			} else {
				pr.losses++
				if result.LoseBG > 0 {
					pr.bgsLost++
				} else if result.LoseG > 0 {
					pr.gammonsLost++
				}
			}
		}

		results <- pr
		trialsRemaining -= currentBatch
	}
}

// aggregateResultsWithProgress combines results and calls progress callback
func (e *Engine) aggregateResultsWithProgress(results chan partialResult, totalTrials int, callback ProgressCallback) (*RolloutResult, error) {
	var (
		sumProbs                     [5]float64
		sumSqProbs                   [5]float64
		sumEquity                    float64
		sumSqEquity                  float64
		trials                       int
		wins, gammonsWon, bgsWon     int
		losses, gammonsLost, bgsLost int
	)

	for pr := range results {
		for i := 0; i < 5; i++ {
			sumProbs[i] += pr.sumProbs[i]
			sumSqProbs[i] += pr.sumSqProbs[i]
		}
		sumEquity += pr.sumEquity
		sumSqEquity += pr.sumSqEquity
		trials += pr.trials
		wins += pr.wins
		gammonsWon += pr.gammonsWon
		bgsWon += pr.bgsWon
		losses += pr.losses
		gammonsLost += pr.gammonsLost
		bgsLost += pr.bgsLost

		// Call progress callback
		if callback != nil && trials > 0 {
			n := float64(trials)
			currentEquity := sumEquity / n
			currentStdDev := calcStdDev(sumEquity, sumSqEquity, n)
			currentCI := 1.96 * currentStdDev / math.Sqrt(n)

			callback(RolloutProgress{
				TrialsCompleted: trials,
				TrialsTotal:     totalTrials,
				Percent:         100.0 * float64(trials) / float64(totalTrials),
				CurrentEquity:   currentEquity,
				CurrentCI:       currentCI,
			})
		}
	}

	n := float64(trials)
	if n == 0 {
		return &RolloutResult{}, nil
	}

	// Calculate means
	result := &RolloutResult{
		WinProb:         sumProbs[0] / n,
		WinG:            sumProbs[1] / n,
		WinBG:           sumProbs[2] / n,
		LoseG:           sumProbs[3] / n,
		LoseBG:          sumProbs[4] / n,
		Equity:          sumEquity / n,
		TrialsCompleted: trials,
		GamesWon:        wins,
		GammonsWon:      gammonsWon,
		BackgammonsWon:  bgsWon,
		GamesLost:       losses,
		GammonsLost:     gammonsLost,
		BackgammonsLost: bgsLost,
	}

	// Calculate standard deviations
	if n > 1 {
		result.WinProbStdDev = calcStdDev(sumProbs[0], sumSqProbs[0], n)
		result.WinGStdDev = calcStdDev(sumProbs[1], sumSqProbs[1], n)
		result.WinBGStdDev = calcStdDev(sumProbs[2], sumSqProbs[2], n)
		result.LoseGStdDev = calcStdDev(sumProbs[3], sumSqProbs[3], n)
		result.LoseBGStdDev = calcStdDev(sumProbs[4], sumSqProbs[4], n)
		result.EquityStdDev = calcStdDev(sumEquity, sumSqEquity, n)
		result.EquityCI = 1.96 * result.EquityStdDev / math.Sqrt(n)
	}

	return result, nil
}

// aggregateResults combines partial results from workers
func (e *Engine) aggregateResults(results chan partialResult, _ int) (*RolloutResult, error) {
	var (
		sumProbs                     [5]float64
		sumSqProbs                   [5]float64
		sumEquity                    float64
		sumSqEquity                  float64
		trials                       int
		wins, gammonsWon, bgsWon     int
		losses, gammonsLost, bgsLost int
	)

	for pr := range results {
		for i := 0; i < 5; i++ {
			sumProbs[i] += pr.sumProbs[i]
			sumSqProbs[i] += pr.sumSqProbs[i]
		}
		sumEquity += pr.sumEquity
		sumSqEquity += pr.sumSqEquity
		trials += pr.trials
		wins += pr.wins
		gammonsWon += pr.gammonsWon
		bgsWon += pr.bgsWon
		losses += pr.losses
		gammonsLost += pr.gammonsLost
		bgsLost += pr.bgsLost
	}

	n := float64(trials)
	if n == 0 {
		return &RolloutResult{}, nil
	}

	// Calculate means
	result := &RolloutResult{
		WinProb:         sumProbs[0] / n,
		WinG:            sumProbs[1] / n,
		WinBG:           sumProbs[2] / n,
		LoseG:           sumProbs[3] / n,
		LoseBG:          sumProbs[4] / n,
		Equity:          sumEquity / n,
		TrialsCompleted: trials,
		GamesWon:        wins,
		GammonsWon:      gammonsWon,
		BackgammonsWon:  bgsWon,
		GamesLost:       losses,
		GammonsLost:     gammonsLost,
		BackgammonsLost: bgsLost,
	}

	// Calculate standard deviations using Welford's formula
	// Var(X) = E(X^2) - E(X)^2
	if n > 1 {
		result.WinProbStdDev = calcStdDev(sumProbs[0], sumSqProbs[0], n)
		result.WinGStdDev = calcStdDev(sumProbs[1], sumSqProbs[1], n)
		result.WinBGStdDev = calcStdDev(sumProbs[2], sumSqProbs[2], n)
		result.LoseGStdDev = calcStdDev(sumProbs[3], sumSqProbs[3], n)
		result.LoseBGStdDev = calcStdDev(sumProbs[4], sumSqProbs[4], n)
		result.EquityStdDev = calcStdDev(sumEquity, sumSqEquity, n)

		// 95% confidence interval = 1.96 * stdErr = 1.96 * stdDev / sqrt(n)
		result.EquityCI = 1.96 * result.EquityStdDev / math.Sqrt(n)
	}

	return result, nil
}

// calcStdDev calculates standard deviation from sum and sum of squares
func calcStdDev(sum, sumSq, n float64) float64 {
	if n <= 1 {
		return 0
	}
	mean := sum / n
	variance := (sumSq/n - mean*mean) * n / (n - 1) // Bessel's correction
	if variance < 0 {
		variance = 0 // Handle numerical errors
	}
	return math.Sqrt(variance)
}

// rolloutWorker performs rollouts for a single worker
func (e *Engine) rolloutWorker(state *GameState, trials int, seed int64, truncate int, cubeful bool) partialResult {
	rng := rand.New(rand.NewSource(seed))
	pr := partialResult{}

	for trial := 0; trial < trials; trial++ {
		result := e.playOutGame(state, rng, truncate, cubeful)

		// Accumulate probabilities
		pr.sumProbs[0] += result.WinProb
		pr.sumProbs[1] += result.WinG
		pr.sumProbs[2] += result.WinBG
		pr.sumProbs[3] += result.LoseG
		pr.sumProbs[4] += result.LoseBG

		pr.sumSqProbs[0] += result.WinProb * result.WinProb
		pr.sumSqProbs[1] += result.WinG * result.WinG
		pr.sumSqProbs[2] += result.WinBG * result.WinBG
		pr.sumSqProbs[3] += result.LoseG * result.LoseG
		pr.sumSqProbs[4] += result.LoseBG * result.LoseBG

		pr.sumEquity += result.Equity
		pr.sumSqEquity += result.Equity * result.Equity
		pr.trials++

		// Count outcomes
		if result.WinProb > 0.5 {
			pr.wins++
			if result.WinBG > 0 {
				pr.bgsWon++
			} else if result.WinG > 0 {
				pr.gammonsWon++
			}
		} else {
			pr.losses++
			if result.LoseBG > 0 {
				pr.bgsLost++
			} else if result.LoseG > 0 {
				pr.gammonsLost++
			}
		}
	}

	return pr
}

// playOutGame plays a single game to completion or truncation
// Returns evaluation from the perspective of the original player (state.Turn)
// cubeful parameter reserved for future cubeful rollouts
func (e *Engine) playOutGame(state *GameState, rng *rand.Rand, truncate int, _ bool) Evaluation {
	// Copy the board so we don't modify the original
	board := state.Board
	originalPlayer := state.Turn // Remember who we're evaluating for
	turn := state.Turn
	ply := 0

	const maxPlies = 1000 // Safety limit

	for ply < maxPlies {
		// Check for truncation
		if truncate > 0 && ply >= truncate {
			return e.evaluateForRollout(&board, originalPlayer)
		}

		// Check if game is over
		status := e.gameStatus(&board)
		if status != 0 {
			return e.gameOverEvaluation(status, originalPlayer)
		}

		// Roll dice
		die1 := rng.Intn(6) + 1
		die2 := rng.Intn(6) + 1

		// Generate moves for current player
		moves := e.generateMovesForBoard(&board, turn, die1, die2)

		if len(moves) > 0 {
			// Find best move using neural net
			bestMove := e.findBestMoveFromList(&board, turn, moves)
			// Apply the move
			e.applyMoveToBoard(&board, turn, bestMove)
		}

		// Switch turns
		turn = 1 - turn
		ply++
	}

	// If we hit max plies, evaluate current position
	return e.evaluateForRollout(&board, originalPlayer)
}

// gameStatus returns the game status
// 0 = game in progress, 1 = player 0 wins, -1 = player 1 wins
// 2/-2 = gammon, 3/-3 = backgammon
func (e *Engine) gameStatus(board *Board) int {
	// Check if player 0 has borne off all checkers
	p0Total := 0
	for i := 0; i < 25; i++ {
		p0Total += int(board[0][i])
	}
	if p0Total == 0 {
		// Player 0 wins - check for gammon/backgammon
		return e.winType(board, 1) // Check opponent's position
	}

	// Check if player 1 has borne off all checkers
	p1Total := 0
	for i := 0; i < 25; i++ {
		p1Total += int(board[1][i])
	}
	if p1Total == 0 {
		// Player 1 wins
		return -e.winType(board, 0)
	}

	return 0 // Game in progress
}

// winType determines if it's a gammon (2) or backgammon (3) or regular win (1)
func (e *Engine) winType(board *Board, loser int) int {
	// Check if loser has borne off any checkers
	total := 0
	for i := 0; i < 25; i++ {
		total += int(board[loser][i])
	}
	if total == 15 {
		// Loser has all 15 checkers - it's a gammon or backgammon
		// Check if any checkers in winner's home board or on the bar
		hasInHome := false
		if board[loser][24] > 0 { // On bar
			hasInHome = true
		} else {
			// Check winner's home board (opponent's points 18-23)
			// From loser's perspective, winner's home is points 0-5
			for i := 0; i < 6; i++ {
				if board[loser][i] > 0 {
					hasInHome = true
					break
				}
			}
		}
		if hasInHome {
			return 3 // Backgammon
		}
		return 2 // Gammon
	}
	return 1 // Regular win
}

// gameOverEvaluation returns an evaluation for a completed game
func (e *Engine) gameOverEvaluation(status int, perspective int) Evaluation {
	// Status > 0 means player 0 won
	// Perspective determines who we're evaluating for
	eval := Evaluation{}

	if status > 0 {
		// Player 0 wins
		if perspective == 0 {
			eval.WinProb = 1.0
			if status >= 3 {
				eval.WinBG = 1.0
				eval.WinG = 1.0
				eval.Equity = 3.0
			} else if status >= 2 {
				eval.WinG = 1.0
				eval.Equity = 2.0
			} else {
				eval.Equity = 1.0
			}
		} else {
			eval.WinProb = 0.0
			if status >= 3 {
				eval.LoseBG = 1.0
				eval.LoseG = 1.0
				eval.Equity = -3.0
			} else if status >= 2 {
				eval.LoseG = 1.0
				eval.Equity = -2.0
			} else {
				eval.Equity = -1.0
			}
		}
	} else {
		// Player 1 wins (status < 0)
		absStatus := -status
		if perspective == 1 {
			eval.WinProb = 1.0
			if absStatus >= 3 {
				eval.WinBG = 1.0
				eval.WinG = 1.0
				eval.Equity = 3.0
			} else if absStatus >= 2 {
				eval.WinG = 1.0
				eval.Equity = 2.0
			} else {
				eval.Equity = 1.0
			}
		} else {
			eval.WinProb = 0.0
			if absStatus >= 3 {
				eval.LoseBG = 1.0
				eval.LoseG = 1.0
				eval.Equity = -3.0
			} else if absStatus >= 2 {
				eval.LoseG = 1.0
				eval.Equity = -2.0
			} else {
				eval.Equity = -1.0
			}
		}
	}

	return eval
}

// evaluateForRollout evaluates the current board position from the specified player's perspective
func (e *Engine) evaluateForRollout(board *Board, perspective int) Evaluation {
	state := &GameState{
		Board: *board,
		Turn:  perspective,
	}
	eval, err := e.Evaluate(state)
	if err != nil || eval == nil {
		return Evaluation{WinProb: 0.5}
	}

	// If perspective is player 1, we need to invert
	if perspective == 1 {
		return Evaluation{
			WinProb: 1 - eval.WinProb,
			WinG:    eval.LoseG,
			WinBG:   eval.LoseBG,
			LoseG:   eval.WinG,
			LoseBG:  eval.WinBG,
			Equity:  -eval.Equity,
		}
	}
	return *eval
}

// generateMovesForBoard generates moves for the specified player
func (e *Engine) generateMovesForBoard(board *Board, turn int, die1, die2 int) []Move {
	// GenerateMoves assumes player 1 is on roll, so we need to swap if turn == 0
	var workBoard Board
	if turn == 0 {
		// Swap sides
		workBoard = swapBoardSides(*board)
	} else {
		workBoard = *board
	}

	ml := GenerateMoves(workBoard, die1, die2)
	return ml.Moves
}

// findBestMoveFromList finds the best move from a list using 1-ply evaluation
func (e *Engine) findBestMoveFromList(board *Board, turn int, moves []Move) Move {
	if len(moves) == 0 {
		return Move{}
	}
	if len(moves) == 1 {
		return moves[0]
	}

	// Work board from the perspective of the moving player
	var workBoard Board
	if turn == 0 {
		workBoard = swapBoardSides(*board)
	} else {
		workBoard = *board
	}

	bestMove := moves[0]
	bestEquity := float64(-999)

	for _, m := range moves {
		// Apply move
		resultBoard := ApplyMove(workBoard, m)
		// Swap back for evaluation (evaluate from moving player's perspective)
		swapped := swapBoardSides(resultBoard)

		state := &GameState{Board: swapped}
		eval, err := e.Evaluate(state)
		if err != nil || eval == nil {
			continue
		}

		// Higher equity is better for the moving player
		if eval.Equity > bestEquity {
			bestEquity = eval.Equity
			bestMove = m
		}
	}

	return bestMove
}

// applyMoveToBoard applies a move to the board in place
func (e *Engine) applyMoveToBoard(board *Board, turn int, m Move) {
	// Apply move from the perspective of the moving player
	var workBoard Board
	if turn == 0 {
		workBoard = swapBoardSides(*board)
	} else {
		workBoard = *board
	}

	resultBoard := ApplyMove(workBoard, m)

	if turn == 0 {
		*board = swapBoardSides(resultBoard)
	} else {
		*board = resultBoard
	}
}

// swapBoardSides swaps player 0 and player 1 on the board
// This should match positionid.SwapSides and analysis.swapBoard
func swapBoardSides(board Board) Board {
	var result Board
	for i := 0; i < 25; i++ {
		result[0][i] = board[1][i]
		result[1][i] = board[0][i]
	}
	return result
}
