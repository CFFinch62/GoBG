"""
GoBG Engine - Python wrapper for the GoBG REST API.
"""

from dataclasses import dataclass
from typing import List, Optional, Tuple
import requests


@dataclass
class Evaluation:
    """Result of position evaluation."""
    equity: float
    win: float
    win_g: float
    win_bg: float
    lose_g: float
    lose_bg: float
    ply: int = 0
    cubeful: bool = False


@dataclass
class Move:
    """A ranked move with equity."""
    move: str
    equity: float
    win: float
    win_g: float


@dataclass
class CubeDecision:
    """Result of cube analysis."""
    action: str  # "no_double", "double_take", "double_pass"
    double_equity: float
    no_double_equity: float
    take_equity: float
    double_diff: float


class Engine:
    """
    Python client for the GoBG REST API.

    Args:
        host: Server hostname (default: "localhost")
        port: Server port (default: 8080)
        timeout: Request timeout in seconds (default: 30)

    Example:
        >>> engine = Engine()
        >>> eval_result = engine.evaluate("4HPwATDgc/ABMA")
        >>> print(f"Win: {eval_result.win:.1f}%")
    """

    def __init__(self, host: str = "localhost", port: int = 8080, timeout: int = 30):
        self.base_url = f"http://{host}:{port}/api"
        self.timeout = timeout
        self._session = requests.Session()

    def health(self) -> dict:
        """Check server health."""
        resp = self._session.get(f"{self.base_url}/health", timeout=self.timeout)
        resp.raise_for_status()
        return resp.json()

    def is_ready(self) -> bool:
        """Check if server is ready."""
        try:
            return self.health().get("ready", False)
        except Exception:
            return False

    def evaluate(self, position: str, ply: int = 0, cubeful: bool = False) -> Evaluation:
        """
        Evaluate a position.

        Args:
            position: Position ID string (gnubg format)
            ply: Evaluation depth (0-2)
            cubeful: Include cube in equity calculation

        Returns:
            Evaluation result with equity and win probabilities
        """
        resp = self._session.post(
            f"{self.base_url}/evaluate",
            json={"position": position, "ply": ply, "cubeful": cubeful},
            timeout=self.timeout,
        )
        resp.raise_for_status()
        data = resp.json()
        return Evaluation(
            equity=data["equity"],
            win=data["win"],
            win_g=data["win_g"],
            win_bg=data["win_bg"],
            lose_g=data["lose_g"],
            lose_bg=data["lose_bg"],
            ply=data.get("ply", 0),
            cubeful=data.get("cubeful", False),
        )

    def best_move(self, position: str, dice: Tuple[int, int], num_moves: int = 5) -> List[Move]:
        """
        Find the best moves for a position and dice roll.

        Args:
            position: Position ID string
            dice: Tuple of (die1, die2)
            num_moves: Number of top moves to return

        Returns:
            List of ranked moves
        """
        resp = self._session.post(
            f"{self.base_url}/move",
            json={"position": position, "dice": list(dice), "num_moves": num_moves},
            timeout=self.timeout,
        )
        resp.raise_for_status()
        data = resp.json()
        return [
            Move(
                move=m["move"],
                equity=m["equity"],
                win=m["win"],
                win_g=m["win_g"],
            )
            for m in data["moves"]
        ]

    def cube_decision(self, position: str) -> CubeDecision:
        """
        Analyze cube decision.

        Args:
            position: Position ID string

        Returns:
            Cube decision analysis
        """
        resp = self._session.post(
            f"{self.base_url}/cube",
            json={"position": position},
            timeout=self.timeout,
        )
        resp.raise_for_status()
        data = resp.json()
        return CubeDecision(
            action=data["action"],
            double_equity=data["double_equity"],
            no_double_equity=data["no_double_equity"],
            take_equity=data["take_equity"],
            double_diff=data["double_diff"],
        )

