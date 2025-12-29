"""
GoBG - Python bindings for the GoBG backgammon engine.

This package provides a simple Python interface to the GoBG REST API.

Example:
    >>> from gobg import Engine
    >>> engine = Engine()
    >>> result = engine.evaluate("4HPwATDgc/ABMA")
    >>> print(f"Equity: {result.equity:+.3f}")
"""

from .engine import Engine, Evaluation, Move, CubeDecision

__version__ = "0.1.0"
__all__ = ["Engine", "Evaluation", "Move", "CubeDecision"]

