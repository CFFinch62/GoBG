# GoBG - Go Backgammon Engine

A high-performance backgammon analysis engine written in Go. Originally ported from GNU Backgammon's evaluation core, GoBG has evolved into a feature-rich engine with modern APIs and optimizations not found in the original.

## Features

### Core Analysis (GNU BG Parity)
- **Neural Network Evaluation** - Contact, race, and crashed position networks
- **Bearoff Databases** - 1-sided and 2-sided for accurate endgame play
- **Cube Decisions** - Full double/take/pass analysis with Janowski formula
- **Match Play** - Complete MET integration with Crawford rule handling
- **Monte Carlo Rollouts** - Parallel execution with configurable workers
- **Multi-Ply Analysis** - 0-ply, 1-ply, and 2-ply evaluation
- **Tutor Mode** - Error detection, skill ratings, and luck analysis
- **Match File Support** - Import/export MAT and SGF formats
- **External Protocol** - gnubg-compatible socket interface

### Beyond GNU Backgammon
- **REST API** - Modern JSON API for easy integration (gnubg has no HTTP interface)
- **WebSocket Streaming** - Real-time rollout progress updates
- **Server-Sent Events** - Alternative streaming for browser clients
- **SIMD Optimization** - AVX2/SSE4.1 accelerated neural network inference
- **Connection Pooling** - High-throughput concurrent request handling
- **Tutor API** - Programmatic game analysis with `/api/tutor/*` endpoints
- **Opening Book** - 21 optimal opening moves with instant lookup
- **C Shared Library** - FFI integration for any language
- **Python Package** - pip-installable bindings

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

### CLI Usage

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

### API Server

```bash
# Start the REST API server
go run ./cmd/bgserver -port 8080

# Or build and run
go build -o bgserver ./cmd/bgserver/
./bgserver -port 8080
```

**API Endpoints:**
| Endpoint | Description |
|----------|-------------|
| `GET /api/health` | Health check |
| `POST /api/evaluate` | Evaluate a position |
| `POST /api/move` | Find best moves for a roll |
| `POST /api/cube` | Cube decision analysis |
| `POST /api/rollout` | Monte Carlo rollout |
| `GET /api/rollout/stream` | SSE streaming rollout |
| `WS /api/ws` | WebSocket for real-time analysis |
| `POST /api/tutor/move` | Analyze a played move |
| `POST /api/tutor/cube` | Analyze a cube decision |
| `POST /api/tutor/game` | Analyze a complete game |

**Example:**
```bash
# Get best move via API
curl -X POST http://localhost:8080/api/move \
  -H "Content-Type: application/json" \
  -d '{"position": "4HPwATDgc/ABMA", "dice": [3, 1]}'

# Analyze a played move
curl -X POST http://localhost:8080/api/tutor/move \
  -H "Content-Type: application/json" \
  -d '{"position": "4HPwATDgc/ABMA", "dice": [3, 1], "move": "24/21 24/23"}'
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
GoBG/
├── cmd/
│   ├── bgengine/     # CLI tool
│   └── bgserver/     # REST API server
├── pkg/
│   ├── engine/       # Core evaluation engine
│   ├── api/          # REST API handlers
│   ├── capi/         # C shared library exports
│   ├── match/        # MAT/SGF import/export
│   └── external/     # External player protocol
├── internal/
│   ├── neuralnet/    # Neural network (with SIMD)
│   ├── bearoff/      # Bearoff databases
│   ├── met/          # Match equity tables
│   └── positionid/   # Position ID encoding
├── bindings/
│   └── python/       # Python package
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
| Neural net (SIMD) | ~2x faster than scalar |

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

## GoBG vs GNU Backgammon

| Feature | GoBG | GNU Backgammon |
|---------|------|----------------|
| Core evaluation | ✅ Same neural networks | ✅ |
| Position IDs | ✅ Compatible | ✅ |
| REST API | ✅ Full JSON API | ❌ |
| WebSocket streaming | ✅ Real-time updates | ❌ |
| SIMD acceleration | ✅ AVX2/SSE4.1 | ❌ |
| Tutor API | ✅ Programmatic | CLI only |
| C library | ✅ Easy FFI | Complex |
| Python package | ✅ pip install | Embedded only |
| Concurrency | ✅ Goroutines | Threads |
| GUI | ❌ Headless | ✅ GTK |

GoBG is designed as an **embeddable analysis engine** for developers building backgammon applications, while GNU Backgammon is a complete desktop application with GUI.

## License

This project is a reimplementation of gnubg's evaluation logic in Go, extended with modern APIs.
The neural network weights and bearoff databases are from GNU Backgammon (GPL).

## Acknowledgments

- [GNU Backgammon](https://www.gnu.org/software/gnubg/) - The original reference implementation
- The gnubg development team for decades of excellent work on backgammon AI
