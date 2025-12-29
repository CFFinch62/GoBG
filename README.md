# GoBG - Go Backgammon Engine

A high-performance backgammon analysis engine written in Go, ported from GNU Backgammon's evaluation core.

## Features

- **Neural Network Evaluation** - Contact, race, and crashed position networks
- **Bearoff Databases** - 1-sided and 2-sided for accurate endgame play
- **Cube Decisions** - Full double/take/pass analysis with Janowski formula
- **Match Play** - Complete MET integration with Crawford rule handling
- **Monte Carlo Rollouts** - Parallel execution with configurable workers
- **Multi-Ply Analysis** - 0-ply, 1-ply, and 2-ply evaluation
- **Tutor Mode** - Error detection, skill ratings, and luck analysis
- **Match File Support** - Import/export MAT and SGF formats
- **External Protocol** - gnubg-compatible socket interface

## Quick Start

### Prerequisites

- Go 1.21 or later
- GNU Backgammon data files

### Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/bgengine.git
cd bgengine

# Copy required data files from gnubg installation
cp /path/to/gnubg/gnubg.weights data/
cp /path/to/gnubg/gnubg_os0.bd data/
cp /path/to/gnubg/gnubg_ts.bd data/    # Optional

# Build
go build -o bgengine ./cmd/bgengine/

# Test
go test ./...
```

### Usage

```bash
# Evaluate a position
./bgengine eval -position "4HPwATDgc/ABMA"

# Find best move for a roll
./bgengine move -position "4HPwATDgc/ABMA" -dice 3,1

# Analyze cube decision
./bgengine cube -position "4HPwATDgc/ABMA"

# Run a rollout
./bgengine rollout -position "4HPwATDgc/ABMA" -trials 2000
```

## Library Usage

```go
package main

import (
    "fmt"
    "github.com/yourusername/bgengine/pkg/engine"
)

func main() {
    // Create engine
    e, _ := engine.NewEngine(engine.EngineOptions{})
    
    // Evaluate starting position
    state := engine.StartingPosition()
    eval, _ := e.Evaluate(state)
    
    fmt.Printf("Equity: %+.3f\n", eval.Equity)
    fmt.Printf("Win: %.1f%%\n", eval.WinProb*100)
}
```

## Project Structure

```
bgengine/
├── cmd/bgengine/     # CLI tool
├── pkg/
│   ├── engine/       # Core evaluation engine
│   ├── match/        # MAT/SGF import/export
│   └── external/     # External player protocol
├── internal/
│   ├── neuralnet/    # Neural network implementation
│   ├── bearoff/      # Bearoff databases
│   ├── met/          # Match equity tables
│   └── positionid/   # Position ID encoding
├── data/             # Neural net weights, databases
└── docs/             # Documentation
```

## Performance

| Operation | Speed |
|-----------|-------|
| 0-ply evaluation | ~17,500 evals/sec |
| 1-ply evaluation | ~275 evals/sec |
| 2-ply evaluation | ~0.6 evals/sec |
| 1000-game rollout | ~250ms |

## Documentation

- [Usage Guide](docs/USAGE.md) - Complete usage instructions
- [Development Plan](docs/GO-ENGINE-PLAN.md) - Architecture and roadmap

## Data Files

Required files from GNU Backgammon:

| File | Size | Purpose |
|------|------|---------|
| `gnubg.weights` | ~50 MB | Neural network weights |
| `gnubg_os0.bd` | ~35 MB | 1-sided bearoff database |
| `gnubg_ts.bd` | ~6.5 MB | 2-sided bearoff (optional) |
| `g11.xml` | ~10 KB | Match equity table (optional) |

## Accuracy

Evaluations are validated against GNU Backgammon:
- Within 0.02 equity of gnubg 2-ply
- Position ID encoding is fully compatible
- Match equity calculations match gnubg

## License

This project is a clean-room reimplementation of gnubg's evaluation logic in Go.
The neural network weights and bearoff databases are from GNU Backgammon.

## Acknowledgments

- [GNU Backgammon](https://www.gnu.org/software/gnubg/) - The reference implementation
- The gnubg development team for their excellent work

