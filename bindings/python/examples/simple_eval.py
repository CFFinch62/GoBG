#!/usr/bin/env python3
"""
Simple example of using the GoBG Python bindings.

Before running, start the server:
    cd /path/to/gobg
    go run ./cmd/bgserver -port 8080

Then run this script:
    python simple_eval.py
"""

import sys
sys.path.insert(0, '..')  # Add parent directory to path

from gobg import Engine

def main():
    # Connect to the server
    engine = Engine(host="localhost", port=8080)

    # Check if server is ready
    if not engine.is_ready():
        print("Error: Server is not ready. Make sure bgserver is running.")
        print("Start with: go run ./cmd/bgserver -port 8080")
        return

    print("=== GoBG Python Example ===\n")

    # Evaluate starting position
    position = "4HPwATDgc/ABMA"
    print(f"Position: {position} (starting position)")
    print()

    # Get evaluation
    result = engine.evaluate(position)
    print("Evaluation:")
    print(f"  Equity: {result.equity:+.3f}")
    print(f"  Win:    {result.win:.1f}%")
    print(f"  Win G:  {result.win_g:.1f}%")
    print(f"  Win BG: {result.win_bg:.1f}%")
    print(f"  Lose G: {result.lose_g:.1f}%")
    print(f"  Lose BG:{result.lose_bg:.1f}%")
    print()

    # Find best moves for 3-1
    dice = (3, 1)
    print(f"Best moves for {dice[0]}-{dice[1]}:")
    moves = engine.best_move(position, dice, num_moves=5)
    for i, move in enumerate(moves, 1):
        print(f"  {i}. {move.move:<20} Eq: {move.equity:+.3f} (Win: {move.win:.1f}%)")
    print()

    # Cube decision
    print("Cube decision:")
    cube = engine.cube_decision(position)
    print(f"  Action: {cube.action}")
    print(f"  Double equity:    {cube.double_equity:+.3f}")
    print(f"  No double equity: {cube.no_double_equity:+.3f}")
    print(f"  Difference:       {cube.double_diff:+.3f}")


if __name__ == "__main__":
    main()

