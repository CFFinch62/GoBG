# Performance Tuning Guide

This guide covers performance optimization for the GoBG backgammon engine.

## Performance Baselines

The engine achieves the following performance on modern hardware (AMD Ryzen 5600X):

| Operation | Speed | Notes |
|-----------|-------|-------|
| 0-ply evaluation | ~17,500 evals/sec | Neural net only |
| 1-ply evaluation | ~275 evals/sec | 21 opponent rolls |
| 2-ply evaluation | ~0.6 evals/sec | Full lookahead |
| Rollout 1000 games (full) | ~360ms | Play to completion |
| Rollout 1000 games (truncated) | ~93ms | Truncate at 10 plies |

## Configuration Options

### Engine Options

```go
opts := engine.EngineOptions{
    WeightsFile:   "data/gnubg.wd",       // Binary weights (faster load)
    BearoffFile:   "data/gnubg_os.bd",    // One-sided bearoff DB
    BearoffTSFile: "data/gnubg_ts.bd",    // Two-sided bearoff DB
    CacheSize:     1 << 20,               // 1M entries (~48MB)
}
```

### Cache Sizing

The evaluation cache significantly improves multi-ply performance:

```go
// Default: 1M entries (~48MB memory)
opts.CacheSize = engine.DefaultCacheSize

// Larger cache for 2-ply heavy workloads
opts.CacheSize = 1 << 22  // 4M entries (~192MB)

// Disable cache for memory-constrained environments
opts.CacheSize = 0
```

**Cache performance at 2-ply:**
- Without cache: ~0.3 evals/sec
- With cache: ~0.6 evals/sec (11.5% hit rate)

### Rollout Options

```go
opts := engine.RolloutOptions{
    Trials:   1296,    // Default: 36^2 for variance coverage
    Truncate: 0,       // 0 = play to end, 10-15 for speed
    Workers:  0,       // 0 = use all CPU cores
    Seed:     0,       // 0 = random seed
    Cubeful:  false,   // Cube decisions in rollout
}
```

**Tuning tips:**
- `Truncate: 10` reduces rollout time by ~75% with minimal accuracy loss
- `Workers` scales linearly up to core count
- `Trials: 324` (18^2) is sufficient for casual analysis

## Memory Usage

| Component | Memory | Notes |
|-----------|--------|-------|
| Neural networks | ~50 MB | 6 networks (contact, race, crashed + pruning) |
| One-sided bearoff | ~80 MB | Exact probabilities for endgame |
| Two-sided bearoff | ~80 MB | Cubeful equity for bearoff |
| Evaluation cache | ~48 MB | Default 1M entries |
| **Total baseline** | **~258 MB** | With all components loaded |

### Reducing Memory

For memory-constrained environments:

```go
opts := engine.EngineOptions{
    WeightsFile: "data/gnubg.wd",
    CacheSize:   1 << 18,  // 256K entries (~12MB)
    // Omit bearoff DBs for smaller footprint
}
```

## CPU Optimization

### Parallel Rollouts

Rollouts automatically use all available cores:

```go
// Let engine choose based on GOMAXPROCS
opts := engine.RolloutOptions{Workers: 0}

// Limit workers to leave headroom for other tasks
opts := engine.RolloutOptions{Workers: runtime.GOMAXPROCS(0) / 2}
```

### Object Pooling

The engine uses `sync.Pool` internally to reduce allocations:

```go
// Neural net input buffers are pooled automatically
// No user configuration needed
```

### Move Pruning

Multi-ply evaluation uses pruning networks to reduce move count:

```go
// Pruning keeps 5-16 moves from potentially thousands
// Controlled internally - no user configuration
// Default threshold: 0.16 equity from best move
```

## API Server Tuning

### HTTP Timeouts

```go
config := api.ServerConfig{
    ReadTimeout:  30 * time.Second,
    WriteTimeout: 30 * time.Second,
    IdleTimeout:  60 * time.Second,
}
```

For rollout endpoints with many trials, increase timeouts:

```go
config.WriteTimeout = 5 * time.Minute
```

### Concurrent Requests

The engine is thread-safe. Multiple API requests can run concurrently:
- Evaluation cache uses `sync.RWMutex` for concurrent reads
- Each rollout uses its own RNG state
- No global locks on neural net evaluation

## Benchmarking

Run benchmarks to measure performance on your hardware:

```bash
# Basic benchmarks
go test -bench=. ./pkg/engine/... -benchtime=5s

# Memory allocation benchmarks
go test -bench=. ./pkg/engine/... -benchmem

# Specific benchmark
go test -bench=BenchmarkRollout1000 ./pkg/engine/...
```

### Example Benchmark Output

```
BenchmarkEvaluate0Ply-12         17482         68542 ns/op
BenchmarkEvaluatePlied1-12         275       4360892 ns/op
BenchmarkRollout1000-12              3     362847291 ns/op
BenchmarkRolloutTruncated-12        13      93127843 ns/op
```

## Profiling

### CPU Profiling

```bash
# Generate CPU profile
go test -bench=BenchmarkEvaluate0Ply -cpuprofile=cpu.prof ./pkg/engine/...

# Analyze with pprof
go tool pprof cpu.prof
```

Key hotspots:
1. Neural net forward pass (~70% of 0-ply time)
2. Input feature computation (~15%)
3. Move generation (~10%)

### Memory Profiling

```bash
# Generate memory profile
go test -bench=. -memprofile=mem.prof ./pkg/engine/...

# Analyze allocations
go tool pprof -alloc_space mem.prof
```

## Common Performance Pitfalls

### 1. Not Loading Bearoff Databases

Without bearoff databases, endgame evaluation uses the neural net which is slower and less accurate:

```go
// Slow - neural net for all positions
opts := engine.EngineOptions{WeightsFile: "gnubg.wd"}

// Fast - exact bearoff calculations
opts := engine.EngineOptions{
    WeightsFile:   "gnubg.wd",
    BearoffFile:   "gnubg_os.bd",
    BearoffTSFile: "gnubg_ts.bd",
}
```

### 2. Unnecessary 2-Ply Evaluation

2-ply is ~50x slower than 1-ply. Use it only when needed:

```go
// For casual play, 0-ply is sufficient
eval, _ := engine.Evaluate(state)

// For analysis, 1-ply is usually enough
eval, _ := engine.EvaluatePlied(state, 1)

// Reserve 2-ply for critical cube decisions
eval, _ := engine.EvaluatePlied(state, 2)
```

### 3. Small Rollout Trial Counts

Too few trials increase variance without improving speed much:

```go
// Bad - high variance, not much faster
opts := engine.RolloutOptions{Trials: 100}

// Good - balanced speed/accuracy
opts := engine.RolloutOptions{Trials: 1296, Truncate: 10}
```

### 4. Not Using Binary Weights

Text weights take ~5x longer to load:

```bash
# Convert text weights to binary (one-time)
./bgcli convert gnubg.weights gnubg.wd
```

```go
// Slow startup
opts := engine.EngineOptions{WeightsFileText: "gnubg.weights"}

// Fast startup
opts := engine.EngineOptions{WeightsFile: "gnubg.wd"}
```

## Production Recommendations

For production deployments:

1. **Use binary weights** - 5x faster startup
2. **Load all bearoff databases** - Accurate endgame
3. **Size cache appropriately** - 1M entries for typical workloads
4. **Truncate rollouts** - 10-15 plies for speed
5. **Set appropriate timeouts** - Match expected evaluation time
6. **Monitor cache hit rate** - Low rates indicate cache sizing issues

```go
// Production configuration
opts := engine.EngineOptions{
    WeightsFile:   "/data/gnubg.wd",
    BearoffFile:   "/data/gnubg_os.bd",
    BearoffTSFile: "/data/gnubg_ts.bd",
    CacheSize:     1 << 20,
}

engine, err := engine.NewEngine(opts)
if err != nil {
    log.Fatal(err)
}

// Periodic cache monitoring
go func() {
    ticker := time.NewTicker(1 * time.Minute)
    for range ticker.C {
        if cache := engine.Cache(); cache != nil {
            log.Printf("Cache hit rate: %.1f%%", cache.HitRate())
        }
    }
}()
```

