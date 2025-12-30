package engine

import (
	"fmt"
	"sync"

	"github.com/yourusername/bgengine/internal/bearoff"
	"github.com/yourusername/bgengine/internal/met"
	"github.com/yourusername/bgengine/internal/neuralnet"
	"github.com/yourusername/bgengine/internal/positionid"
)

// Engine is the main evaluation engine
type Engine struct {
	// Neural networks
	contact *neuralnet.NeuralNet
	race    *neuralnet.NeuralNet
	crashed *neuralnet.NeuralNet

	// Pruning networks
	pContact *neuralnet.NeuralNet
	pCrashed *neuralnet.NeuralNet
	pRace    *neuralnet.NeuralNet

	// Bearoff databases
	bearoff   *bearoff.Database // One-sided bearoff database
	bearoffTS *bearoff.Database // Two-sided bearoff database

	// Match equity table
	met *met.Table

	// Evaluation cache
	cache *EvalCache

	// Reusable buffers
	inputPool sync.Pool

	// SIMD optimization: pre-allocated evaluation buffers (per-network)
	contactBufPool sync.Pool
	raceBufPool    sync.Pool
	crashedBufPool sync.Pool
	outputPool     sync.Pool
}

// EngineOptions configures the engine
type EngineOptions struct {
	WeightsFile     string // Path to neural network weights (binary .wd format)
	WeightsFileText string // Path to text format weights (alternative)
	BearoffFile     string // Path to one-sided bearoff database
	BearoffTSFile   string // Path to two-sided bearoff database
	METFile         string // Path to match equity table
	CacheSize       uint32 // Evaluation cache size (0 = default, negative = disabled)
}

// NewEngine creates a new evaluation engine with the given options
func NewEngine(opts EngineOptions) (*Engine, error) {
	e := &Engine{
		inputPool: sync.Pool{
			New: func() interface{} {
				return make([]float32, neuralnet.NumContactInputs)
			},
		},
		outputPool: sync.Pool{
			New: func() interface{} {
				return make([]float32, 5)
			},
		},
	}

	// Load neural network weights (try binary first, then text)
	if opts.WeightsFile != "" {
		weights, err := neuralnet.LoadWeightsBinary(opts.WeightsFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load binary weights: %w", err)
		}
		e.contact = weights.Contact
		e.race = weights.Race
		e.crashed = weights.Crashed
		e.pContact = weights.PContact
		e.pCrashed = weights.PCrashed
		e.pRace = weights.PRace

		// Initialize SIMD buffer pools
		e.initBufferPools()
	} else if opts.WeightsFileText != "" {
		weights, err := neuralnet.LoadWeightsText(opts.WeightsFileText)
		if err != nil {
			return nil, fmt.Errorf("failed to load text weights: %w", err)
		}
		e.contact = weights.Contact
		e.race = weights.Race
		e.crashed = weights.Crashed
		e.pContact = weights.PContact
		e.pCrashed = weights.PCrashed
		e.pRace = weights.PRace

		// Initialize SIMD buffer pools
		e.initBufferPools()
	}

	// Load one-sided bearoff database
	if opts.BearoffFile != "" {
		db, err := bearoff.LoadOneSided(opts.BearoffFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load one-sided bearoff database: %w", err)
		}
		e.bearoff = db
	}

	// Load two-sided bearoff database
	if opts.BearoffTSFile != "" {
		db, err := bearoff.LoadOneSided(opts.BearoffTSFile) // Same loader works for both
		if err != nil {
			return nil, fmt.Errorf("failed to load two-sided bearoff database: %w", err)
		}
		if db.Type != bearoff.BearoffTwoSided {
			return nil, fmt.Errorf("expected two-sided bearoff database, got type %d", db.Type)
		}
		e.bearoffTS = db
	}

	// Load match equity table
	if opts.METFile != "" {
		table, err := met.LoadXML(opts.METFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load MET: %w", err)
		}
		e.met = table
	} else {
		e.met = met.Default()
	}

	// Create evaluation cache
	cacheSize := opts.CacheSize
	if cacheSize == 0 {
		cacheSize = DefaultCacheSize
	}
	if cacheSize > 0 {
		e.cache = NewEvalCache(cacheSize)
	}

	return e, nil
}

// initBufferPools initializes the SIMD evaluation buffer pools based on loaded networks
func (e *Engine) initBufferPools() {
	if e.contact != nil {
		e.contactBufPool = sync.Pool{
			New: func() interface{} {
				return neuralnet.NewEvaluateBuffer(e.contact.CInput, e.contact.CHidden)
			},
		}
	}
	if e.race != nil {
		e.raceBufPool = sync.Pool{
			New: func() interface{} {
				return neuralnet.NewEvaluateBuffer(e.race.CInput, e.race.CHidden)
			},
		}
	}
	if e.crashed != nil {
		e.crashedBufPool = sync.Pool{
			New: func() interface{} {
				return neuralnet.NewEvaluateBuffer(e.crashed.CInput, e.crashed.CHidden)
			},
		}
	}
}

// Cache returns the evaluation cache (may be nil if disabled)
func (e *Engine) Cache() *EvalCache {
	return e.cache
}

// SetCache sets the evaluation cache (use nil to disable caching)
func (e *Engine) SetCache(cache *EvalCache) {
	e.cache = cache
}

// Evaluate evaluates a position and returns the expected equities
func (e *Engine) Evaluate(state *GameState) (*Evaluation, error) {
	board := neuralnet.Board(state.Board)

	// Classify the position
	class := neuralnet.ClassifyPosition(board)

	var output [5]float32
	var err error

	switch class {
	case neuralnet.ClassOver:
		// Game is over
		return e.evaluateGameOver(board)

	case neuralnet.ClassBearoffTS:
		// Use two-sided bearoff database if available
		if e.bearoffTS != nil {
			boBoard := neuralnet.GetBearoffBoard(board)
			output, err = e.bearoffTS.Evaluate(boBoard)
			if err != nil {
				// Fall back to one-sided or race net
				if e.bearoff != nil {
					output, err = e.bearoff.Evaluate(boBoard)
				}
				if err != nil {
					output, err = e.evaluateRace(board)
				}
			}
		} else if e.bearoff != nil {
			// Fall back to one-sided database
			boBoard := neuralnet.GetBearoffBoard(board)
			output, err = e.bearoff.Evaluate(boBoard)
			if err != nil {
				output, err = e.evaluateRace(board)
			}
		} else {
			output, err = e.evaluateRace(board)
		}

	case neuralnet.ClassBearoff1, neuralnet.ClassBearoff2, neuralnet.ClassBearoffOS:
		// Use one-sided bearoff database
		if e.bearoff != nil {
			boBoard := neuralnet.GetBearoffBoard(board)
			output, err = e.bearoff.Evaluate(boBoard)
			if err != nil {
				// Fall back to race net
				output, err = e.evaluateRace(board)
			}
		} else {
			output, err = e.evaluateRace(board)
		}

	case neuralnet.ClassRace:
		output, err = e.evaluateRace(board)

	case neuralnet.ClassCrashed:
		output, err = e.evaluateCrashed(board)

	case neuralnet.ClassContact:
		output, err = e.evaluateContact(board)

	default:
		return nil, fmt.Errorf("unknown position class: %d", class)
	}

	if err != nil {
		return nil, err
	}

	eval := &Evaluation{
		WinProb: float64(output[0]),
		WinG:    float64(output[1]),
		WinBG:   float64(output[2]),
		LoseG:   float64(output[3]),
		LoseBG:  float64(output[4]),
	}

	// Calculate equity
	eval.Equity = eval.WinProb - (1 - eval.WinProb) +
		eval.WinG - eval.LoseG +
		eval.WinBG - eval.LoseBG

	return eval, nil
}

// EvaluateCached evaluates a position with caching support
// plies specifies the ply depth for cache context (0 for neural net only)
func (e *Engine) EvaluateCached(state *GameState, plies int) (*Evaluation, error) {
	// If no cache, just evaluate directly
	if e.cache == nil {
		return e.Evaluate(state)
	}

	// Create position key and eval context
	key := positionid.MakePositionKey(positionid.Board(state.Board))
	evalCtx := MakeEvalContext(plies, false, state.CubeOwner, state.CubeValue)

	// Check cache
	output := make([]float32, 5)
	slot := e.cache.Lookup(key, evalCtx, output)
	if slot == CacheHit {
		// Cache hit - reconstruct evaluation from cached output
		eval := &Evaluation{
			WinProb: float64(output[0]),
			WinG:    float64(output[1]),
			WinBG:   float64(output[2]),
			LoseG:   float64(output[3]),
			LoseBG:  float64(output[4]),
		}
		eval.Equity = eval.WinProb - (1 - eval.WinProb) +
			eval.WinG - eval.LoseG +
			eval.WinBG - eval.LoseBG
		return eval, nil
	}

	// Cache miss - evaluate and store
	eval, err := e.Evaluate(state)
	if err != nil {
		return nil, err
	}

	// Store in cache
	output[0] = float32(eval.WinProb)
	output[1] = float32(eval.WinG)
	output[2] = float32(eval.WinBG)
	output[3] = float32(eval.LoseG)
	output[4] = float32(eval.LoseBG)
	e.cache.Add(key, evalCtx, output, slot)

	return eval, nil
}

// evaluateGameOver handles positions where the game is over
func (e *Engine) evaluateGameOver(board neuralnet.Board) (*Evaluation, error) {
	// Count checkers for each side
	var count [2]int
	for side := 0; side < 2; side++ {
		for i := 0; i < 25; i++ {
			count[side] += int(board[side][i])
		}
	}

	eval := &Evaluation{}

	if count[1] == 0 {
		// Player 1 (on roll) has borne off all checkers - they win
		eval.WinProb = 1.0
		eval.Equity = 1.0
		// Check for gammon/backgammon
		if count[0] > 0 {
			// Opponent still has checkers
			hasInHomeBoard := false
			hasOnBar := board[0][24] > 0
			for i := 0; i < 6; i++ {
				if board[0][i] > 0 {
					hasInHomeBoard = true
					break
				}
			}
			if !hasInHomeBoard && !hasOnBar {
				eval.WinG = 1.0
				eval.Equity = 2.0
				// Check for backgammon
				hasInOpponentHome := false
				for i := 18; i < 24; i++ {
					if board[0][i] > 0 {
						hasInOpponentHome = true
						break
					}
				}
				if hasInOpponentHome || hasOnBar {
					eval.WinBG = 1.0
					eval.Equity = 3.0
				}
			}
		}
	} else if count[0] == 0 {
		// Player 0 (not on roll) has borne off - they win (player 1 loses)
		eval.WinProb = 0.0
		eval.Equity = -1.0
		// Check for gammon/backgammon
		if count[1] > 0 {
			hasInHomeBoard := false
			hasOnBar := board[1][24] > 0
			for i := 0; i < 6; i++ {
				if board[1][i] > 0 {
					hasInHomeBoard = true
					break
				}
			}
			if !hasInHomeBoard && !hasOnBar {
				eval.LoseG = 1.0
				eval.Equity = -2.0
				hasInOpponentHome := false
				for i := 18; i < 24; i++ {
					if board[1][i] > 0 {
						hasInOpponentHome = true
						break
					}
				}
				if hasInOpponentHome || hasOnBar {
					eval.LoseBG = 1.0
					eval.Equity = -3.0
				}
			}
		}
	}

	return eval, nil
}

// evaluateRace evaluates a race position using the race neural network (SIMD optimized)
func (e *Engine) evaluateRace(board neuralnet.Board) ([5]float32, error) {
	if e.race == nil {
		return [5]float32{0.5, 0, 0, 0, 0}, nil
	}

	inputs := neuralnet.RaceInputs(board)

	// Get buffer from pool and output slice
	buf := e.raceBufPool.Get().(*neuralnet.EvaluateBuffer)
	output := e.outputPool.Get().([]float32)
	defer e.raceBufPool.Put(buf)
	defer e.outputPool.Put(output)

	e.race.EvaluateFast(inputs, output, buf)

	var result [5]float32
	copy(result[:], output[:5])
	return result, nil
}

// evaluateCrashed evaluates a crashed position using the crashed neural network (SIMD optimized)
func (e *Engine) evaluateCrashed(board neuralnet.Board) ([5]float32, error) {
	if e.crashed == nil {
		return e.evaluateContact(board)
	}

	inputs := neuralnet.CrashedInputs(board)

	// Get buffer from pool
	buf := e.crashedBufPool.Get().(*neuralnet.EvaluateBuffer)
	output := e.outputPool.Get().([]float32)
	defer e.crashedBufPool.Put(buf)
	defer e.outputPool.Put(output)

	e.crashed.EvaluateFast(inputs, output, buf)

	var result [5]float32
	copy(result[:], output[:5])
	return result, nil
}

// evaluateContact evaluates a contact position using the contact neural network (SIMD optimized)
func (e *Engine) evaluateContact(board neuralnet.Board) ([5]float32, error) {
	if e.contact == nil {
		return [5]float32{0.5, 0.15, 0.01, 0.15, 0.01}, nil
	}

	inputs := neuralnet.ContactInputs(board)

	// Get buffer from pool
	buf := e.contactBufPool.Get().(*neuralnet.EvaluateBuffer)
	output := e.outputPool.Get().([]float32)
	defer e.contactBufPool.Put(buf)
	defer e.outputPool.Put(output)

	e.contact.EvaluateFast(inputs, output, buf)

	var result [5]float32
	copy(result[:], output[:5])
	return result, nil
}

// GetMatchEquity returns the match winning probability at the current score
func (e *Engine) GetMatchEquity(state *GameState, player int) float32 {
	if e.met == nil {
		return 0.5
	}
	return e.met.GetME(state.Score[0], state.Score[1], state.MatchLength, player, state.Crawford)
}
