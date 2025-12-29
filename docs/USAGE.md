# GoBG Engine Usage Guide

A comprehensive guide to using the GoBG backgammon analysis engine.

## Table of Contents

1. [Installation](#installation)
2. [Quick Start](#quick-start)
3. [Command-Line Interface](#command-line-interface)
4. [Library API](#library-api)
5. [Match Play](#match-play)
6. [Tutor Mode](#tutor-mode)
7. [Position ID Format](#position-id-format)
8. [Understanding Output](#understanding-output)
9. [External Player Protocol](#external-player-protocol)
10. [Match Import/Export](#match-importexport)

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

