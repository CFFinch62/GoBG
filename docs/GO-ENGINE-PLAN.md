# Go Backgammon Engine

A high-performance backgammon analysis engine ported from GNU Backgammon's
evaluation core, rewritten in idiomatic Go.

## Project Status: Phase 5 Complete ✅

**Completed Features:**
- Position encoding/decoding (gnubg-compatible position IDs)
- Move generation for all dice combinations
- Neural network evaluation (contact, race, crashed positions)
- Bearoff database integration (1-sided and 2-sided)
- Match equity table support
- Cube decision analysis (double/take/pass with Janowski formula)
- Match play with full MET integration (Crawford/post-Crawford)
- Monte Carlo rollouts with parallel execution
- Multi-ply evaluation (0-ply, 1-ply, 2-ply)
- Move pruning and evaluation caching
- Command-line interface (CLI)
- Tutor mode (error detection, skill ratings, luck analysis)
- External player protocol (gnubg socket interface)
- SGF/MAT match file import/export

**Next Phase:** Web Service (REST API, WebSocket, Docker)

**See:** [docs/USAGE.md](USAGE.md) for complete usage instructions.

## Project Goals

1. **Standalone engine** - No GUI, no CLI parsing, just pure evaluation logic ✅
2. **Multiple interfaces** - Library, CLI tool, web service (in that order) ✅ (Library + CLI done)
3. **Maximum performance** - Leverage all available cores for rollouts ✅
4. **Validated accuracy** - Test against gnubg to ensure correctness ✅
5. **Clean API** - Easy to embed in any Go application ✅

## Target Hardware

- Memory: 32-96 GB available
- Cores: 4-6 cores (design for easy scaling)
- Rollouts: Up to 1,000 games per position

## What We're Porting from gnubg

Only the essential evaluation logic (~8,000 lines of C):

| Component | Source Files | Purpose |
|-----------|--------------|---------|
| Neural Network | lib/neuralnet.c, lib/inputs.c | Position evaluation |
| Position Encoding | positionid.c | Board ↔ ID conversion |
| Move Generation | eval.c (GenerateMoves) | Legal move enumeration |
| Bearoff Database | bearoff.c | Endgame exact equity |
| Match Equity Tables | matchequity.c | Match play adjustments |
| Cube Decisions | eval.c (cube functions) | Double/take/pass |
| Rollouts | rollout.c (core logic) | Monte Carlo simulation |

## What We're NOT Porting

- GUI (gtk*.c, board3d/*)
- Command parser (gnubg.c command handling)
- Database/relational features
- Sound, rendering, preferences
- Python bindings
- Most of backgammon.h globals

---

## Project Structure

```
bgengine/
├── go.mod
├── README.md
├── cmd/
│   └── bgengine/           # CLI tool
│       └── main.go
├── pkg/
│   ├── engine/             # Public API - Core evaluation
│   │   ├── position.go     # Board/GameState types
│   │   ├── move.go         # Move types and generation
│   │   ├── evaluate.go     # Neural net evaluation
│   │   ├── cube.go         # Cube decisions
│   │   ├── rollout.go      # Monte Carlo rollouts
│   │   ├── analysis.go     # Move ranking
│   │   ├── tutor.go        # Skill rating, error detection
│   │   ├── prune.go        # Move filtering
│   │   └── cache.go        # Evaluation cache
│   ├── match/              # Match file import/export
│   │   ├── types.go        # Match, Game, Action types
│   │   ├── mat.go          # MAT (Jellyfish) format
│   │   └── sgf.go          # SGF (Smart Game Format)
│   └── external/           # External player protocol
│       ├── protocol.go     # TCP server
│       └── fibs.go         # FIBS board format
├── internal/
│   ├── neuralnet/          # Neural network implementation
│   │   ├── network.go      # Network structure
│   │   ├── weights.go      # Weight file loading
│   │   ├── forward.go      # Forward propagation
│   │   ├── inputs.go       # Position → NN inputs
│   │   └── classify.go     # Position classification
│   ├── bearoff/            # Bearoff database
│   │   ├── database.go     # 1-sided and 2-sided
│   │   └── reader.go       # File format parsing
│   ├── met/                # Match equity tables
│   │   └── table.go        # XML loading, MET lookup
│   └── positionid/         # Position ID encoding
│       └── positionid.go   # gnubg-compatible IDs
├── data/                   # Embedded data files
│   ├── gnubg.weights       # Neural net weights
│   ├── gnubg_os.bd         # 1-sided bearoff (embedded)
│   ├── gnubg_ts.bd         # 2-sided bearoff (loaded)
│   └── g11.xml             # Match equity table
└── test/
    └── testdata/           # Test fixtures
```

---

## Core Types

```go
package engine

// Board represents checker positions for both players.
// Index 0 = bar, 1-24 = points, 25 = borne off
type Board [2][26]int8

// GameState represents the full state needed for evaluation
type GameState struct {
    Board       Board
    Turn        int    // 0 or 1
    Dice        [2]int // Current roll (0,0 if not rolled)
    CubeValue   int    // 1, 2, 4, 8, ...
    CubeOwner   int    // -1=centered, 0=player0, 1=player1
    MatchLength int    // 0 = money game
    Score       [2]int // Match score
    Crawford    bool   // Crawford game
}

// Evaluation contains equity estimates
type Evaluation struct {
    Equity      float64    // Expected value
    WinProb     float64    // P(win)
    WinG        float64    // P(win gammon)
    WinBG       float64    // P(win backgammon)
    LoseG       float64    // P(lose gammon)
    LoseBG      float64    // P(lose backgammon)
}

// Move represents a sequence of checker moves
type Move struct {
    From  [4]int8   // Starting points
    To    [4]int8   // Ending points
    Hits  int8      // Number of hits
    Eval  *Evaluation
}

// CubeDecision contains cube action recommendation
type CubeDecision struct {
    Action      CubeAction // NoDouble, Double, Take, Pass
    DoubleEquity float64
    NoDoubleEquity float64
    TakeEquity  float64
}

type CubeAction int
const (
    NoDouble CubeAction = iota
    Double
    Take
    Pass
)
```

---

## Public API


---

## Performance Design

### Concurrency Model

```go
// RolloutOptions controls parallel rollout execution
type RolloutOptions struct {
    Trials      int           // Number of games (default 1000)
    Truncate    int           // Truncate at ply N (0 = play to end)
    Seed        int64         // RNG seed (0 = random)
    Workers     int           // 0 = GOMAXPROCS
    JSD         bool          // Calculate Jensen-Shannon divergence
}

// Rollout distributes trials across workers
func (e *Engine) Rollout(state *GameState, opts RolloutOptions) (*RolloutResult, error) {
    workers := opts.Workers
    if workers == 0 {
        workers = runtime.GOMAXPROCS(0)
    }

    trialsPerWorker := opts.Trials / workers
    results := make(chan partialResult, workers)

    for i := 0; i < workers; i++ {
        go func(seed int64) {
            // Each worker has independent RNG
            results <- e.rolloutWorker(state, trialsPerWorker, seed)
        }(opts.Seed + int64(i))
    }

    // Aggregate results...
}
```

### Memory Considerations

With 32-96 GB available, we can be generous:

| Component | Memory | Notes |
|-----------|--------|-------|
| Neural net weights | ~50 MB | Loaded once, shared |
| Bearoff DB (2-sided) | ~130 MB | Loaded once, shared |
| Bearoff DB (1-sided) | ~35 MB | Loaded once, shared |
| Per-rollout state | ~1 KB | Trivial |
| Move lists | ~50 KB | Per evaluation |

**Total baseline: ~220 MB** - Plenty of headroom.

For maximum performance, we'll:
- Load all databases into memory at startup
- Use sync.Pool for frequently allocated objects
- Avoid allocations in hot paths

---

## Implementation Phases

### Phase 1: Foundation ✅ COMPLETE

**Goal:** Port position encoding and move generation, with tests.

Tasks:
- [x] Set up Go module and project structure
- [x] Port positionid.c → internal/positionid/
  - [x] PositionID encoding
  - [x] PositionFromID decoding
  - [x] Test against gnubg outputs
- [x] Define Board and GameState types
- [x] Port move generation from eval.c
  - [x] GenerateMoves for all 21 dice combinations
  - [x] Handle bar entry, bearing off, blocked points
  - [x] Test against gnubg move lists

### Phase 2: Evaluation ✅ COMPLETE

**Goal:** Port neural network and get accurate position evaluations.

Tasks:
- [x] Port lib/neuralnet.c → internal/neuralnet/
  - [x] Network structure (layers, weights)
  - [x] Forward propagation
  - [x] Weight file parser (.weights format)
- [x] Port lib/inputs.c → internal/inputs/
  - [x] CalculateInputs - base inputs (position → 200 floats for pruning nets)
  - [x] Race inputs (position → 214 floats)
  - [x] Contact/race/crashed classification (classify.go)
  - [x] Full contact inputs (250 floats with heuristics)
- [x] Port bearoff.c → internal/bearoff/
  - [x] Read gnubg_os.bd (1-sided bearoff)
  - [x] Compressed and uncompressed format support
  - [x] Normal distribution approximation format support
  - [x] Position encoding/decoding
  - [x] Win probability evaluation for bearoff positions
- [x] Port matchequity.c → internal/met/
  - [x] Load match equity table from XML (g11.xml format)
  - [x] GetME function for match equity lookup
  - [x] Pre-Crawford and post-Crawford table support
  - [x] Default table with Jacobs-Trice approximation
- [x] Implement Evaluate function (pkg/engine/evaluate.go)
  - [x] Select correct neural net (contact/race/crashed)
  - [x] Detect bearoff positions
  - [x] Apply bearoff for endgame positions
  - [x] Game over detection (gammon/backgammon)

### Phase 3: Analysis ✅ COMPLETE

**Goal:** Full analysis capabilities with rollouts.

Tasks:
- [x] Implement BestMove (1-ply lookahead)
- [x] Implement RankMoves (evaluate top N moves)
- [x] Port cube decision logic
  - [x] Double decision (opponent's take point)
  - [x] Take/pass decision
  - [x] Full CubeInfo with match equity integration
  - [x] Janowski formula for live cube equity
- [x] Implement Rollout
  - [x] Monte Carlo game simulation
  - [x] Parallel execution with goroutines (configurable workers)
  - [x] Proper RNG seeding per worker
  - [x] Result aggregation with variance/CI calculation
  - [x] Truncation support for faster evaluation
- [x] Create CLI tool (cmd/bgengine/)
  - [x] `eval` - Position evaluation
  - [x] `move` - Best move analysis
  - [x] `cube` - Cube decision analysis
  - [x] `rollout` - Monte Carlo rollout

**Performance Achieved:**
- 1,000 game rollout: ~360ms (full) or ~93ms (truncated at 10 plies)
- 10,000+ evaluations/second on single core

---

## Testing Strategy

### Unit Tests

Each internal package has comprehensive tests:

```go
// internal/positionid/positionid_test.go
func TestPositionIDRoundTrip(t *testing.T) {
    // Test that Encode(Decode(id)) == id for known positions
}

func TestPositionIDCompatibility(t *testing.T) {
    // Test against known gnubg position IDs
    cases := []struct{
        board Board
        id    string
    }{
        {StartingPosition, "4HPwATDgc/ABMA"},
        // ... more test cases from gnubg
    }
}
```

### Integration Tests (Compare to gnubg)

We'll create a test harness that:
1. Runs gnubg CLI with a position
2. Captures its evaluation
3. Runs our engine
4. Compares results

```bash
# scripts/compare_eval.sh
echo "set evaluation cubeful on" | gnubg -t
echo "set evaluation plies 2" | gnubg -t
echo "external posid:$1" | gnubg -t
# Parse output, compare to our engine
```

### Benchmark Tests

```go
func BenchmarkEvaluate(b *testing.B) {
    e, _ := NewEngine()
    state := StartingPosition()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        e.Evaluate(state)
    }
}

func BenchmarkRollout1000(b *testing.B) {
    e, _ := NewEngine()
    state := StartingPosition()
    opts := RolloutOptions{Trials: 1000}
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        e.Rollout(state, opts)
    }
}
```

---

## Data Files

The engine needs these data files from gnubg:

| File | Size | Purpose | Embedding |
|------|------|---------|-----------|
| gnubg.weights | ~50 MB | Neural net weights | Embed or load |
| gnubg_os.bd | ~35 MB | 1-sided bearoff | Embed or load |
| gnubg_ts.bd | ~130 MB | 2-sided bearoff | Load at runtime |
| g11.xml | ~10 KB | Match equity table | Embed |

For the library, we'll:
- Embed gnubg.weights and gnubg_os.bd using `//go:embed`
- Load gnubg_ts.bd from disk (optional, for more accuracy)
- Embed g11.xml (small)

---

## CLI Tool Design (Phase 3)

```bash
# Evaluate a position
$ bgengine eval --position "4HPwATDgc/ABMA:cIkqAAAAAAAA"
Equity: +0.234
  Win:    58.2% (G: 18.4%, BG: 0.8%)
  Lose:   41.8% (G: 12.1%, BG: 0.4%)

# Find best move
$ bgengine move --position "4HPwATDgc/ABMA:cIkqAAAAAAAA" --dice 3,1
Best: 8/5 6/5
  Equity: +0.312

# Cube decision
$ bgengine cube --position "4HPwATDgc/ABMA:cIkqAAAAAAAA"
Action: Double
  Double equity: +0.892
  No double:     +0.756

# Rollout
$ bgengine rollout --position "4HPwATDgc/ABMA:cIkqAAAAAAAA" --trials 1000
Rollout (1000 trials, 6 workers):
  Equity: +0.241 ± 0.012
  Time: 2.3s
```

---

### Phase 4: Optimizations ✅

**Goal:** Performance improvements for production use.

Tasks:
- [x] Implement Multi-Ply Evaluation (2-ply lookahead)
- [x] Implement Move Filtering/Pruning
- [x] Implement Evaluation Cache

**Performance Achieved:**
- 0-ply: 17,500 evals/sec
- 1-ply: 275 evals/sec
- 2-ply: 0.6 evals/sec (with 11.5% cache hit rate)
- Rollout 1000 games: 248ms

---

### Phase 5: Core gnubg Features ✅ COMPLETE

**Goal:** Complete parity with gnubg's core analysis functionality.

Tasks:
- [x] **Cube Decisions** - Full take/pass/double/beaver analysis with cubeful equity
  - Janowski formula for live cube equity in money games
  - All decision types: Double, Redouble, NoDouble, Take, Pass, Beaver
  - CubeInfo struct with match context
- [x] **Match Play** - Complete match equity integration, Crawford/post-Crawford handling
  - MET loading from XML (g11.xml format)
  - Crawford and post-Crawford table support
  - Match gammon prices calculation
  - Equity to MWC conversion (Mwc2Eq, Eq2Mwc)
- [x] **Two-Sided Bearoff Database** - Load and use gnubg_ts.bd for accurate endgame
  - Generated 6x6 database (6.5 MB) using makebearoff
  - Position classification detects two-sided bearoff
  - Engine uses two-sided when available with fallback
- [x] **Tutor Mode** - Error detection, skill ratings, luck analysis
  - SkillType: None, Doubtful (?!), Bad (?), Very Bad (??) with gnubg thresholds
  - LuckType: VeryBad, Bad, None, Good, VeryGood based on equity swing
  - Player ratings: Supernatural to Awful based on error-per-move
  - AnalyzeMoveSkill: Compare played move vs best, calculate equity loss
  - AnalyzeCubeSkill: Evaluate cube decision errors
- [x] **External Player Protocol** - gnubg's socket interface for engine vs engine play
  - TCP server in pkg/external/protocol.go
  - FIBS board format parsing (pkg/external/fibs.go)
  - Commands: version, help, set, evaluation, fibsboard, exit
  - Settings: plies, cubeful, jacoby, crawford
- [x] **SGF/MAT Import/Export** - Match file format support
  - MAT format (Jellyfish/gnubg) - pkg/match/mat.go
  - SGF format (Smart Game Format) - pkg/match/sgf.go
  - Match, Game, Action types - pkg/match/types.go

---

### Phase 6: Integration APIs 🔄

**Goal:** Make GoBG easily embeddable in any backgammon application, regardless of language or framework. Provide a drop-in replacement for gnubg that developers can actually use.

**Why This Matters:**
- gnubg is notoriously difficult to integrate with external applications
- Developers building backgammon GUIs need a simple, performant AI engine
- Language-agnostic APIs enable use from Python, JavaScript, C#, etc.

Tasks:

**HTTP/JSON REST API**
- [ ] REST API server with JSON request/response
- [ ] Endpoints: `/evaluate`, `/move`, `/cube`, `/rollout`
- [ ] Configurable port and host binding
- [ ] Daemon mode (run as background service)
- [ ] Graceful shutdown handling

**C Shared Library (FFI)**
- [ ] Build as C shared library (`libbgengine.so` / `bgengine.dll`)
- [ ] CGO wrapper functions with C-compatible signatures
- [ ] Header file generation for C/C++ integration
- [ ] Memory management documentation

**Python Integration**
- [ ] Python package using ctypes (for C library) or requests (for REST API)
- [ ] Pythonic wrapper classes (Engine, Position, Move, etc.)
- [ ] Example: integrate with PyGame or Tkinter backgammon UI
- [ ] pip-installable package

**Documentation**
- [ ] API reference with all endpoints/functions
- [ ] Integration guide for each language (Python, JavaScript, C#)
- [ ] Example projects showing integration patterns
- [ ] Performance tuning guide

Files to create:
```
cmd/
  bgserver/           # HTTP API server
    main.go
pkg/
  api/                # REST API handlers
    server.go
    handlers.go
    types.go          # JSON request/response types
  capi/               # C shared library exports
    exports.go        # //export functions for CGO
bindings/
  python/
    gobg/
      __init__.py
      engine.py       # Python wrapper
      types.py
    setup.py
    examples/
      simple_eval.py
      pygame_integration.py
```

---

### Phase 7: Advanced Features & Deployment 🔮

**Goal:** Production-ready deployment and advanced optimizations.

Tasks:

**Deployment**
- [ ] Docker image with pre-loaded data files
- [ ] Docker Compose for easy local deployment
- [ ] Health check endpoints
- [ ] Prometheus metrics (optional)

**Streaming & Real-time**
- [ ] WebSocket support for streaming rollout progress
- [ ] Server-sent events (SSE) alternative
- [ ] Progress callbacks for long operations

**Performance Optimizations**
- [ ] SIMD for neural net (using Go assembly or CGO)
- [ ] Connection pooling for high-throughput scenarios
- [ ] Evaluation caching with configurable size

**Extended Features**
- [ ] Opening book database
- [ ] Match analysis from position list
- [ ] Position database for common scenarios
- [ ] Tutor API for analyzing played games

---

## Success Criteria

1. **Accuracy:** Evaluations within 0.02 equity of gnubg 2-ply ✅
2. **Performance:** 10,000+ evaluations/second on single core ✅
3. **Rollouts:** 1,000 game rollout in <5 seconds (6 cores) ✅
4. **Embeddable:** Single HTTP call or function call to get best move
5. **Language-agnostic:** Working examples in Python, JavaScript
6. **Easy setup:** `docker run` or `pip install` to get started
7. **Testing:** >90% code coverage, validation against gnubg

---

## Integration Examples

### Python (REST API)
```python
import requests

class GoBGEngine:
    def __init__(self, host="localhost", port=8080):
        self.base_url = f"http://{host}:{port}/api"

    def best_move(self, position_id, dice):
        response = requests.post(f"{self.base_url}/move", json={
            "position": position_id,
            "dice": dice
        })
        return response.json()

# Usage
engine = GoBGEngine()
result = engine.best_move("4HPwATDgc/ABMA", [3, 1])
print(f"Best move: {result['move']}, Equity: {result['equity']}")
```

### Python (C Library)
```python
from gobg import Engine, Position

engine = Engine("./data")
pos = Position.starting()
move, equity = engine.best_move(pos, dice=(3, 1))
print(f"Best: {move} (Eq: {equity:+.3f})")
```

### JavaScript (REST API)
```javascript
const response = await fetch('http://localhost:8080/api/move', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
        position: '4HPwATDgc/ABMA',
        dice: [3, 1]
    })
});
const { move, equity } = await response.json();
```

### cURL
```bash
curl -X POST http://localhost:8080/api/move \
  -H "Content-Type: application/json" \
  -d '{"position": "4HPwATDgc/ABMA", "dice": [3, 1]}'
```

---

## Current Status

| Phase | Status | Description |
|-------|--------|-------------|
| Phase 1 | ✅ Complete | Core Infrastructure |
| Phase 2 | ✅ Complete | Neural Network Evaluation |
| Phase 3 | ✅ Complete | Move Generation & Search |
| Phase 4 | ✅ Complete | Cube Decisions & Match Play |
| Phase 5 | ✅ Complete | External Protocol & Match Files |
| Phase 6 | 🔄 Next | Integration APIs |
| Phase 7 | 🔮 Future | Advanced Features & Deployment |

