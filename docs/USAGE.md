# GoBG Engine Usage Guide

A comprehensive guide to using the GoBG backgammon analysis engine.

## Table of Contents

1. [Installation](#installation)
2. [Quick Start](#quick-start)
3. [Command-Line Interface](#command-line-interface)
4. [REST API Server](#rest-api-server)
   - [WebSocket API](#websocket-apiws)
   - [Server-Sent Events (SSE)](#get-apirolloutstream-sse)
5. [Python Integration](#python-integration)
6. [C Shared Library](#c-shared-library)
7. [Library API](#library-api)
   - [Opening Book](#opening-book)
   - [Rollout with Progress](#rollout-with-progress-callbacks)
8. [Match Play](#match-play)
9. [Tutor Mode](#tutor-mode)
10. [Position ID Format](#position-id-format)
11. [Understanding Output](#understanding-output)
12. [External Player Protocol](#external-player-protocol)
13. [Match Import/Export](#match-importexport)

---

## Installation

### Prerequisites

- Go 1.21 or later
- gnubg data files (see below)

### Required Data Files

The engine requires these data files from GNU Backgammon:

| File | Size | Purpose | Required |
|------|------|---------|----------|
| `gnubg.weights` | ~50 MB | Neural network weights | Yes |
| `gnubg_os0.bd` | ~35 MB | 1-sided bearoff database | Yes |
| `gnubg_ts.bd` | ~6.5 MB | 2-sided bearoff database | Optional (more accurate endgame) |
| `g11.xml` | ~10 KB | Match equity table | Optional (has default) |

Place these files in the `data/` directory at the project root.

### Building

```bash
# Clone the repository
git clone https://github.com/yourusername/bgengine.git
cd bgengine

# Build the CLI tool
go build -o bgengine ./cmd/bgengine/

# Run tests to verify installation
go test ./...
```

---

## Quick Start

### Evaluate the Starting Position

```bash
./bgengine eval -position "4HPwATDgc/ABMA"
```

Output:
```
Equity: +0.000
  Win:    50.0% (G: 15.0%, BG: 1.0%)
  Lose:   50.0% (G: 15.0%, BG: 1.0%)
```

### Find the Best Move

```bash
./bgengine move -position "4HPwATDgc/ABMA" -dice 3,1
```

Output:
```
Best moves for roll 3-1:
  1. 24/21 24/23           Eq: +0.000
  2. 24/21 21/20           Eq: +0.000
  3. 8/5 6/5               Eq: +0.000
  ...
```

Note: For the symmetric starting position, many moves have similar equity.

### Analyze Cube Decision

```bash
./bgengine cube -position "4HPwATDgc/ABMA"
```

### Run a Rollout

```bash
./bgengine rollout -position "4HPwATDgc/ABMA" -trials 2000
```

---

## Command-Line Interface

### Global Options

```bash
bgengine <command> [options]

Commands:
  eval      Evaluate a position
  move      Find the best move for a dice roll
  cube      Analyze cube decisions
  rollout   Monte Carlo rollout
  help      Show help
```

### `eval` Command

Evaluates a position and shows win/loss probabilities.

```bash
bgengine eval -position <positionID>
bgengine eval -p <positionID>  # Short form
```

**Options:**
- `-position`, `-p`: Position ID in gnubg format (required)

**Example:**
```bash
./bgengine eval -p "4HPwATDgc/ABMA"
```

### `move` Command

Finds and ranks the best moves for a given dice roll.

```bash
bgengine move -position <positionID> -dice <roll> [-n <count>]
```

**Options:**
- `-position`, `-p`: Position ID (required)
- `-dice`, `-d`: Dice roll in format "3,1" or "3-1" (required)
- `-n`: Number of moves to show (default: 5)

**Examples:**
```bash
./bgengine move -p "4HPwATDgc/ABMA" -d 6,5
./bgengine move -p "4HPwATDgc/ABMA" -d 3-1 -n 10
```

### `cube` Command

Analyzes the cube decision (double/no-double/take/pass).

```bash
bgengine cube -position <positionID>
```

**Options:**
- `-position`, `-p`: Position ID (required)

**Example:**
```bash
./bgengine cube -p "sGfwATDgc/ABMA"
```

### `rollout` Command

Performs a Monte Carlo rollout to get more accurate equity estimates.

```bash
bgengine rollout -position <positionID> [options]
```

**Options:**
- `-position`, `-p`: Position ID (required)
- `-trials`: Number of games to simulate (default: 1296)
- `-workers`: Number of parallel workers (default: auto)
- `-truncate`: Truncate games at N plies, 0 = play to end (default: 0)
- `-seed`: Random seed for reproducibility (default: random)

**Examples:**
```bash
# Basic rollout
./bgengine rollout -p "4HPwATDgc/ABMA" -trials 2000

# Fast truncated rollout
./bgengine rollout -p "4HPwATDgc/ABMA" -trials 5000 -truncate 10

# Reproducible rollout
./bgengine rollout -p "4HPwATDgc/ABMA" -trials 1000 -seed 12345
```

---

## REST API Server

The engine can be run as an HTTP server for integration with any language.

### Starting the Server

```bash
# Build the server
go build -o bgserver ./cmd/bgserver/

# Run with defaults (localhost:8080)
./bgserver

# Run on a different port, accessible from network
./bgserver -host 0.0.0.0 -port 8888
```

### Command-Line Options

| Option | Default | Description |
|--------|---------|-------------|
| `-host` | localhost | Host to bind to |
| `-port` | 8080 | Port to listen on |
| `-weights` | data/gnubg.weights | Neural network weights file |
| `-bearoff` | data/gnubg_os0.bd | One-sided bearoff database |
| `-bearoff-ts` | data/gnubg_ts.bd | Two-sided bearoff database |
| `-met` | data/g11.xml | Match equity table |
| `-max-fast-workers` | 100 | Max concurrent fast operations (evaluate, move, cube) |
| `-max-slow-workers` | 4 | Max concurrent slow operations (rollout) |

### Worker Pool Configuration

The server uses a worker pool to manage concurrent requests:

- **Fast operations** (evaluate, move, cube): Quick evaluations that take milliseconds. Default: 100 concurrent.
- **Slow operations** (rollout): CPU-intensive rollouts that can take seconds. Default: 4 concurrent.

When the pool is full, new requests wait until a slot becomes available. If the request context is cancelled (e.g., client disconnects), the request returns `503 Service Unavailable`.

For high-throughput scenarios, tune these values based on your hardware:
- Fast workers: Can be high since evaluations are quick
- Slow workers: Keep low (typically number of CPU cores) since rollouts are CPU-bound

### API Endpoints

#### GET /api/health

Check server health and worker pool status.

```bash
curl http://localhost:8080/api/health
```

Response:
```json
{
  "status": "ok",
  "version": "0.1.0",
  "ready": true,
  "pool": {
    "active_fast": 5,
    "active_slow": 2,
    "queued_fast": 0,
    "queued_slow": 3,
    "total_fast": 1000,
    "total_slow": 50,
    "max_fast": 100,
    "max_slow": 4
  }
}
```

The `pool` field shows worker pool statistics for monitoring high-throughput scenarios:
- `active_fast/slow`: Currently processing requests
- `queued_fast/slow`: Requests waiting for a worker slot
- `total_fast/slow`: Total requests processed since server start
- `max_fast/slow`: Configured maximum concurrent workers

#### POST /api/evaluate

Evaluate a position.

```bash
curl -X POST http://localhost:8080/api/evaluate \
  -H "Content-Type: application/json" \
  -d '{"position": "4HPwATDgc/ABMA", "ply": 0}'
```

Request body:
```json
{
  "position": "4HPwATDgc/ABMA",
  "ply": 0,
  "cubeful": false
}
```

Response:
```json
{
  "equity": 0.079,
  "win": 52.4,
  "win_g": 14.9,
  "win_bg": 0.76,
  "lose_g": 11.9,
  "lose_bg": 0.75,
  "ply": 0,
  "cubeful": false
}
```

#### POST /api/move

Find best moves for a position and dice roll.

```bash
curl -X POST http://localhost:8080/api/move \
  -H "Content-Type: application/json" \
  -d '{"position": "4HPwATDgc/ABMA", "dice": [3, 1], "num_moves": 3}'
```

Response:
```json
{
  "moves": [
    {"move": "8/5 6/5", "equity": 0.145, "win": 54.9, "win_g": 16.0},
    {"move": "13/10 24/23", "equity": -0.018, "win": 49.5, "win_g": 12.3}
  ],
  "num_legal": 16,
  "dice": [3, 1],
  "position": "4HPwATDgc/ABMA"
}
```

#### POST /api/cube

Analyze cube decision.

```bash
curl -X POST http://localhost:8080/api/cube \
  -H "Content-Type: application/json" \
  -d '{"position": "4HPwATDgc/ABMA"}'
```

Response:
```json
{
  "action": "no_double",
  "double_equity": -0.166,
  "no_double_equity": 0.257,
  "take_equity": -0.166,
  "double_diff": -0.423
}
```

#### POST /api/rollout

Run Monte Carlo rollout.

```bash
curl -X POST http://localhost:8080/api/rollout \
  -H "Content-Type: application/json" \
  -d '{"position": "4HPwATDgc/ABMA", "trials": 1000}'
```

#### GET /api/rollout/stream (SSE)

Stream rollout progress via Server-Sent Events (SSE).

```bash
curl -N "http://localhost:8080/api/rollout/stream?position=4HPwATDgc/ABMA&trials=1000"
```

Response (event stream):
```
event: progress
data: {"trials_completed":50,"trials_total":1000,"percent":5,"current_equity":0.12,"current_ci":0.35}

event: progress
data: {"trials_completed":100,"trials_total":1000,"percent":10,"current_equity":0.08,"current_ci":0.22}

...

event: result
data: {"equity":0.02,"equity_ci":0.09,"win_prob":51.2,"trials_completed":1000}

event: done
```

JavaScript example:
```javascript
const eventSource = new EventSource(
  'http://localhost:8080/api/rollout/stream?position=4HPwATDgc/ABMA&trials=1000'
);

eventSource.addEventListener('progress', (e) => {
  const progress = JSON.parse(e.data);
  console.log(`${progress.percent}% complete, equity: ${progress.current_equity}`);
});

eventSource.addEventListener('result', (e) => {
  const result = JSON.parse(e.data);
  console.log(`Final equity: ${result.equity} ± ${result.equity_ci}`);
  eventSource.close();
});
```

#### WebSocket /api/ws

Real-time bidirectional communication with streaming support.

Connect:
```javascript
const ws = new WebSocket('ws://localhost:8080/api/ws');
```

Message types: `evaluate`, `move`, `cube`, `rollout`, `ping`

Request format:
```json
{
  "type": "rollout",
  "id": "req-123",
  "payload": {"position": "4HPwATDgc/ABMA", "trials": 1000}
}
```

Response types: `result`, `progress`, `error`, `pong`

Rollout with streaming progress:
```javascript
ws.send(JSON.stringify({
  type: 'rollout',
  id: 'roll-1',
  payload: {position: '4HPwATDgc/ABMA', trials: 1000}
}));

ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  if (msg.type === 'progress') {
    console.log(`${msg.payload.percent}% - equity: ${msg.payload.current_equity}`);
  } else if (msg.type === 'result') {
    console.log('Final:', msg.payload);
  }
};
```

---

## Python Integration

A Python package is provided for easy integration.

### Installation

```bash
cd bindings/python
pip install -e .
```

### Basic Usage

```python
from gobg import Engine

# Connect to the server
engine = Engine(host="localhost", port=8080)

# Evaluate a position
result = engine.evaluate("4HPwATDgc/ABMA")
print(f"Equity: {result.equity:+.3f}")
print(f"Win: {result.win:.1f}%")

# Find best moves
moves = engine.best_move("4HPwATDgc/ABMA", dice=(3, 1))
for move in moves:
    print(f"{move.move}: {move.equity:+.3f}")

# Cube decision
cube = engine.cube_decision("4HPwATDgc/ABMA")
print(f"Action: {cube.action}")
```

---

## C Shared Library

The engine can be built as a C shared library for integration with C/C++, C#, or any language supporting FFI.

### Building

```bash
go build -buildmode=c-shared -o libbgengine.so ./pkg/capi
```

This produces:
- `libbgengine.so` - Shared library
- `libbgengine.h` - C header file

### API Functions

```c
// Get library version
char* bgengine_version();

// Initialize engine with data file paths
int bgengine_init(char* weightsFile, char* bearoffFile,
                  char* bearoffTSFile, char* metFile);

// Shutdown engine
void bgengine_shutdown();

// Evaluate a position (returns JSON)
int bgengine_evaluate(char* positionID, char** resultJSON);

// Find best move (returns JSON)
int bgengine_best_move(char* positionID, int die1, int die2, char** resultJSON);

// Cube decision (returns JSON)
int bgengine_cube_decision(char* positionID, char** resultJSON);

// Free a string returned by other functions
void bgengine_free_string(char* s);

// Get last error message
char* bgengine_last_error();
```

### Example Usage

```c
#include "libbgengine.h"
#include <stdio.h>

int main() {
    // Initialize engine
    if (bgengine_init("data/gnubg.weights", "data/gnubg_os0.bd",
                      "data/gnubg_ts.bd", "data/g11.xml") != 0) {
        printf("Error: %s\n", bgengine_last_error());
        return 1;
    }

    // Evaluate position
    char* json = NULL;
    bgengine_evaluate("4HPwATDgc/ABMA", &json);
    printf("Result: %s\n", json);
    bgengine_free_string(json);

    // Cleanup
    bgengine_shutdown();
    return 0;
}
```

---

## Library API

The engine can be embedded in any Go application.

### Basic Usage

```go
package main

import (
    "fmt"
    "github.com/yourusername/bgengine/pkg/engine"
)

func main() {
    // Create a new engine
    e, err := engine.NewEngine(engine.EngineOptions{})
    if err != nil {
        panic(err)
    }

    // Get the starting position
    state := engine.StartingPosition()

    // Evaluate the position
    eval, err := e.Evaluate(state)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Equity: %+.3f\n", eval.Equity)
    fmt.Printf("Win probability: %.1f%%\n", eval.WinProb*100)
}
```

### Finding the Best Move

```go
// Find the best move for a 3-1 roll
bestMove, eval, err := e.BestMove(state, [2]int{3, 1})
if err != nil {
    panic(err)
}

fmt.Printf("Best move: %v\n", bestMove)
fmt.Printf("Expected equity: %+.3f\n", eval.Equity)
```

### Ranking Multiple Moves

```go
// Get top 5 moves ranked by equity
moves, err := e.RankMoves(state, [2]int{6, 5}, 5)
if err != nil {
    panic(err)
}

for i, m := range moves {
    fmt.Printf("%d. Equity: %+.3f\n", i+1, m.Equity)
}
```

### Cube Decision Analysis

```go
// Analyze cube decision
analysis, err := e.AnalyzeCube(state)
if err != nil {
    panic(err)
}

fmt.Printf("No double equity: %+.3f\n", analysis.NoDoubleEquity)
fmt.Printf("Double/Take equity: %+.3f\n", analysis.DoubleTakeEq)
```

### Monte Carlo Rollout

```go
opts := engine.RolloutOptions{
    Trials:   2000,   // Number of games
    Workers:  0,      // 0 = auto (GOMAXPROCS)
    Truncate: 0,      // 0 = play to end
    Seed:     12345,  // For reproducibility
}

result, err := e.Rollout(state, opts)
if err != nil {
    panic(err)
}

fmt.Printf("Equity: %+.3f ± %.3f\n", result.Equity, result.EquityCI)
```

### Rollout with Progress Callbacks

For long rollouts, use progress callbacks to report status:

```go
callback := func(p engine.RolloutProgress) {
    fmt.Printf("\r%.0f%% complete (%d/%d) - equity: %+.3f ± %.3f",
        p.Percent, p.TrialsCompleted, p.TrialsTotal,
        p.CurrentEquity, p.CurrentCI)
}

result, err := e.RolloutWithProgress(state, opts, callback)
```

The callback receives updates at 5% intervals with:
- `TrialsCompleted`, `TrialsTotal`, `Percent` - Progress info
- `CurrentEquity`, `CurrentCI` - Running equity and confidence interval

### Opening Book

The engine includes an opening book for the 21 standard opening rolls:

```go
// Look up opening move
if move, ok := e.LookupOpening(state, dice); ok {
    fmt.Printf("Book move: %v\n", move)
}
```

Supported rolls: 21, 31, 32, 41, 42, 43, 51, 52, 53, 54, 61, 62, 63, 64, 65,
and doubles 11, 22, 33, 44, 55, 66.

### Core Types

```go
// Board represents checker positions for both players
// Index 0-23 = points 1-24, index 24 = bar
type Board [2][25]uint8

// GameState contains all information needed for evaluation
type GameState struct {
    Board       Board
    Turn        int       // 0 or 1 (whose turn)
    CubeValue   int       // 1, 2, 4, 8, ...
    CubeOwner   int       // -1=centered, 0=player0, 1=player1
    MatchLength int       // 0 = money game
    Score       [2]int    // Match score [player0, player1]
    Crawford    bool      // Crawford game
}

// Evaluation contains position assessment
type Evaluation struct {
    WinProb float64  // Probability of winning
    WinG    float64  // Probability of winning gammon
    WinBG   float64  // Probability of winning backgammon
    LoseG   float64  // Probability of losing gammon
    LoseBG  float64  // Probability of losing backgammon
    Equity  float64  // Expected value (-1 to +1 for money, higher for gammons)
}

// Move represents a sequence of checker movements
type Move struct {
    From  [4]int8  // Starting points (-1 = unused)
    To    [4]int8  // Ending points (-1 = bear off, -2 = unused)
    Hits  int8     // Number of opponent checkers hit
}
```

---

## Match Play

The engine fully supports match play with Crawford rule handling.

### Setting Up Match Play

```go
state := engine.StartingPosition()
state.MatchLength = 7           // 7-point match
state.Score = [2]int{4, 5}      // Player 0 leads 4-5
state.Crawford = false          // Not Crawford game

// For Crawford game (leader is 1-away)
state.Crawford = true
```

### Match Equity

Match equity calculations adjust cube decisions based on the score:

```go
// Analyze cube in a match context
analysis, err := e.AnalyzeCube(state)

// Get match winning chances
fmt.Printf("Match equity if we take: %.1f%%\n", analysis.TakeEquity*100)
fmt.Printf("Match equity if we pass: %.1f%%\n", analysis.PassEquity*100)
```

### Crawford Rule

During Crawford games:
- The cube cannot be used
- Gammon values are different (no free drop)

Post-Crawford games automatically use the appropriate MET values.

---

## Tutor Mode

The engine provides comprehensive move and decision analysis for learning.

### Skill Classification

Moves are classified by equity loss:

| Skill Level | Symbol | Equity Loss |
|-------------|--------|-------------|
| None | - | < 0.025 |
| Doubtful | ?! | 0.025 - 0.05 |
| Bad | ? | 0.05 - 0.10 |
| Very Bad | ?? | > 0.10 |

### Analyzing Move Quality

```go
// Make a move and analyze it
playedMove := Move{
    From: [4]int8{23, 23, -1, -1},
    To:   [4]int8{20, 22, -1, -1},
}

analysis, err := e.AnalyzeMoveSkill(state, playedMove, [2]int{3, 1})
if err != nil {
    panic(err)
}

fmt.Printf("Played move equity: %+.4f\n", analysis.Equity)
fmt.Printf("Best move equity:   %+.4f\n", analysis.BestEquity)
fmt.Printf("Equity loss:        %.4f\n", analysis.EquityLoss)
fmt.Printf("Skill rating:       %s (%s)\n", analysis.Skill.String(), analysis.Skill.Abbr())
```

### Analyzing Cube Decisions

```go
// Analyze a cube decision
cubeAnalysis, err := e.AnalyzeCubeSkill(state, engine.CubeActionDouble)
if err != nil {
    panic(err)
}

fmt.Printf("Correct action: %v\n", cubeAnalysis.CorrectAction)
fmt.Printf("Equity loss:    %.4f\n", cubeAnalysis.EquityLoss)
fmt.Printf("Skill rating:   %s\n", cubeAnalysis.Skill.String())
```

### Luck Analysis

Dice rolls are classified by equity swing:

| Luck Level | Equity Swing |
|------------|--------------|
| Very Bad | < -0.6 |
| Bad | -0.6 to -0.3 |
| None | -0.3 to +0.3 |
| Good | +0.3 to +0.6 |
| Very Good | > +0.6 |

```go
luck := engine.ClassifyLuck(equitySwing)
fmt.Printf("Roll luck: %s\n", luck.String())
```

### Player Ratings

Overall player performance is rated by error per move (EPM):

| Rating | EPM Range |
|--------|-----------|
| Supernatural | < 0.002 |
| World Class | 0.002 - 0.005 |
| Expert | 0.005 - 0.008 |
| Advanced | 0.008 - 0.012 |
| Intermediate | 0.012 - 0.018 |
| Casual Player | 0.018 - 0.026 |
| Beginner | 0.026 - 0.035 |
| Awful | > 0.035 |

```go
rating := engine.GetRating(0.015)  // Returns RatingIntermediate
fmt.Printf("Player rating: %s\n", rating.String())
```

---

## Position ID Format

The engine uses GNU Backgammon's position ID format, a compact base64-encoded
representation of the board position.

### Format Structure

A full gnubg position string has two parts:
```
<position_id>:<match_id>
```

- **Position ID**: 14-character base64 string encoding checker positions
- **Match ID**: 12-character base64 string encoding game state (optional)

The engine currently only uses the position ID part.

### Common Position IDs

| Position | ID | Description |
|----------|-----|-------------|
| Starting | `4HPwATDgc/ABMA` | Initial setup |
| Empty | `AAAAAAAAAAAAAA` | No checkers |

### Getting Position IDs from gnubg

In GNU Backgammon:
1. Set up or load a position
2. Use `show positionid` command
3. Copy the position ID string

### Programmatic Conversion

```go
import "github.com/yourusername/bgengine/internal/positionid"

// Decode a position ID to a board
board, err := positionid.BoardFromPositionID("4HPwATDgc/ABMA")

// Encode a board to a position ID
id := positionid.PositionID(board)
```

---

## Understanding Output

### Equity

**Equity** represents the expected outcome of the game:

| Equity | Meaning |
|--------|---------|
| +1.000 | Expected to win a single game |
| +2.000 | Expected to win a gammon |
| +3.000 | Expected to win a backgammon |
| 0.000 | Even position |
| -1.000 | Expected to lose a single game |

For money games, equity directly translates to expected point gain/loss
per cube unit.

### Win Probabilities

The output shows five probabilities:

- **Win%**: Probability of winning the game
- **WinG%**: Probability of winning a gammon (opponent has not borne off)
- **WinBG%**: Probability of winning a backgammon (opponent still has checkers in your home or on bar)
- **LoseG%**: Probability of losing a gammon
- **LoseBG%**: Probability of losing a backgammon

### Cube Decision Types

| Decision | Meaning |
|----------|---------|
| No Double | Don't double, position not strong enough |
| Double, Take | Double is correct, opponent should take |
| Double, Pass | Double is correct, opponent should pass |
| Too Good | Position is too strong to double (more value in playing for gammon) |

### Rollout Statistics

- **Trials**: Number of games simulated
- **Equity ± StdDev**: Mean equity with standard deviation
- **95% CI**: 95% confidence interval for the true equity

The confidence interval narrows as more trials are run:
- 324 trials: CI ≈ ±0.12
- 1296 trials: CI ≈ ±0.06
- 5184 trials: CI ≈ ±0.03

---

## Performance Tips

### For Faster Analysis

1. **Use truncated rollouts** for quick estimates:
   ```bash
   ./bgengine rollout -p "..." -trials 5000 -truncate 10
   ```

2. **Increase workers** for parallel processing:
   ```bash
   ./bgengine rollout -p "..." -workers 12
   ```

### For More Accurate Results

1. **Increase trial count** (1296 or 5184 recommended):
   ```bash
   ./bgengine rollout -p "..." -trials 5184
   ```

2. **Use full rollouts** (no truncation) for critical decisions:
   ```bash
   ./bgengine rollout -p "..." -trials 2592 -truncate 0
   ```

### Memory Usage

The engine uses approximately:
- ~50 MB for neural network weights
- ~35 MB for 1-sided bearoff database
- ~6.5 MB for 2-sided bearoff database (optional)
- ~1 KB per concurrent game during rollouts

Total baseline: ~100-110 MB

---

## External Player Protocol

The engine implements gnubg's external player protocol for engine-vs-engine play.

### Starting the Server

```go
import "github.com/yourusername/bgengine/pkg/external"

server := external.NewServer(engine, external.ServerOptions{
    Port: 1234,
})
err := server.ListenAndServe()
```

### Protocol Commands

| Command | Description |
|---------|-------------|
| `version` | Returns engine version |
| `set plies N` | Set evaluation depth |
| `set cubeful on/off` | Enable/disable cubeful evaluation |
| `evaluation` | Evaluate current position |
| `fibsboard <board>` | Evaluate FIBS board position |
| `exit` | Close connection |

### FIBS Board Format

The server accepts FIBS board strings for position input:
```
board:player1:player2:matchlen:score1:score2:board[26]:turn:dice[4]:cube:...
```

---

## Match Import/Export

The engine supports reading and writing match files in MAT and SGF formats.

### MAT Format (Jellyfish)

```go
import "github.com/yourusername/bgengine/pkg/match"

// Import a match from MAT file
file, _ := os.Open("game.mat")
m, err := match.ImportMAT(file)

// Export a match to MAT format
file, _ := os.Create("output.mat")
err := match.ExportMAT(file, m)
```

### SGF Format (Smart Game Format)

```go
// Import from SGF
file, _ := os.Open("game.sgf")
m, err := match.ImportSGF(file)

// Export to SGF
file, _ := os.Create("output.sgf")
err := match.ExportSGF(file, m)
```

### Match Data Structure

```go
type Match struct {
    Player1     string    // Player 1 name
    Player2     string    // Player 2 name
    MatchLength int       // 0 = money game
    Date        string    // YYYY-MM-DD
    Event       string    // Event name
    Games       []*Game   // List of games
}

type Game struct {
    Number   int        // Game number
    Score1   int        // Player 1 score at start
    Score2   int        // Player 2 score at start
    Crawford bool       // Crawford game
    Actions  []Action   // Roll, move, cube actions
}
```

---

## Troubleshooting

### "Failed to load weights"

Ensure `gnubg.weights` is in the `data/` directory.

### "Failed to load bearoff database"

Ensure `gnubg_os.bd` is in the `data/` directory.

### Position ID Errors

- Ensure the position ID is exactly 14 characters
- Use only valid base64 characters: A-Z, a-z, 0-9, +, /
- Don't include the match ID part (after the colon)

### Unexpected Equity Values

- Check that the position is set up correctly
- Verify you're evaluating from the correct player's perspective
- For edge cases, try a rollout for more accurate results

