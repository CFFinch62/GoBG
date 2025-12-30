// Package engine provides match analysis for complete backgammon matches.
package engine

import (
	"fmt"

	"github.com/yourusername/bgengine/internal/positionid"
)

// MatchAnalysis contains the complete analysis of a backgammon match.
type MatchAnalysis struct {
	// Overall match statistics
	TotalGames    int               `json:"total_games"`
	TotalMoves    int               `json:"total_moves"`  // Total checker moves (both players)
	TotalCubeActs int               `json:"total_cube"`   // Total cube actions
	PlayerStats   [2]PlayerAnalysis `json:"player_stats"` // Stats per player
	GameStats     []GameAnalysis    `json:"game_stats"`   // Stats per game

	// Error lists
	MoveErrors []MoveErrorDetail `json:"move_errors"` // All move errors
	CubeErrors []CubeErrorDetail `json:"cube_errors"` // All cube errors

	// Luck analysis
	PlayerLuck [2]LuckAnalysis `json:"player_luck"` // Luck per player
}

// PlayerAnalysis contains analysis stats for one player across the match.
type PlayerAnalysis struct {
	Name         string     `json:"name"`
	TotalMoves   int        `json:"total_moves"`    // Unforced moves
	TotalCube    int        `json:"total_cube"`     // Cube decisions
	TotalError   float64    `json:"total_error"`    // Sum of equity lost
	ErrorPerMove float64    `json:"error_per_move"` // EPM
	Rating       RatingType `json:"rating"`         // Overall rating
	RatingStr    string     `json:"rating_str"`     // Human-readable rating
	Blunders     int        `json:"blunders"`       // Very bad (>= 0.12)
	Errors       int        `json:"errors"`         // Bad (0.06-0.12)
	Doubtful     int        `json:"doubtful"`       // Doubtful (0.03-0.06)

	// Cube-specific stats
	CubeError     float64 `json:"cube_error"`     // Total cube error
	MissedDoubles int     `json:"missed_doubles"` // Should have doubled
	WrongDoubles  int     `json:"wrong_doubles"`  // Shouldn't have doubled
	WrongTakes    int     `json:"wrong_takes"`    // Should have passed
	WrongPasses   int     `json:"wrong_passes"`   // Should have taken
}

// GameAnalysis contains analysis of a single game.
type GameAnalysis struct {
	GameNumber   int               `json:"game_number"`
	Winner       int               `json:"winner"`      // 0 or 1, -1 if unfinished
	Points       int               `json:"points"`      // Points won
	MoveCount    [2]int            `json:"move_count"`  // Moves per player
	TotalError   [2]float64        `json:"total_error"` // Error per player
	ErrorPerMove [2]float64        `json:"error_per_move"`
	CubeActions  int               `json:"cube_actions"`
	Errors       []MoveErrorDetail `json:"errors"` // Errors in this game
}

// MoveErrorDetail contains details about a single move error.
type MoveErrorDetail struct {
	GameNumber int       `json:"game_number"`
	MoveNumber int       `json:"move_number"`
	Player     int       `json:"player"`
	Position   string    `json:"position"` // Position ID
	Dice       [2]int    `json:"dice"`
	Played     string    `json:"played_move"` // What was played
	Best       string    `json:"best_move"`   // What should have been played
	EquityLoss float64   `json:"equity_loss"`
	Skill      SkillType `json:"skill"`
	SkillStr   string    `json:"skill_str"`
}

// CubeErrorDetail contains details about a cube decision error.
type CubeErrorDetail struct {
	GameNumber int        `json:"game_number"`
	MoveNumber int        `json:"move_number"`
	Player     int        `json:"player"`
	Position   string     `json:"position"`
	Played     CubeAction `json:"played"`
	Optimal    CubeAction `json:"optimal"`
	PlayedStr  string     `json:"played_str"`
	OptimalStr string     `json:"optimal_str"`
	EquityLoss float64    `json:"equity_loss"`
	Skill      SkillType  `json:"skill"`
	SkillStr   string     `json:"skill_str"`
}

// LuckAnalysis contains luck statistics for a player.
type LuckAnalysis struct {
	TotalLuck   float64 `json:"total_luck"`   // Sum of luck values
	AvgLuck     float64 `json:"avg_luck"`     // Average luck per roll
	VeryLucky   int     `json:"very_lucky"`   // Count of very lucky rolls
	Lucky       int     `json:"lucky"`        // Count of lucky rolls
	Unlucky     int     `json:"unlucky"`      // Count of unlucky rolls
	VeryUnlucky int     `json:"very_unlucky"` // Count of very unlucky rolls
}

// AnalyzedPosition represents a position with analysis for match reconstruction.
type AnalyzedPosition struct {
	Board      Board      `json:"board"`
	Turn       int        `json:"turn"`
	Dice       [2]int     `json:"dice"`
	CubeValue  int        `json:"cube_value"`
	CubeOwner  int        `json:"cube_owner"`
	Score      [2]int     `json:"score"`
	Move       *Move      `json:"move,omitempty"`
	CubeAction CubeAction `json:"cube_action,omitempty"`
	GameNumber int        `json:"game_number"`
	MoveNumber int        `json:"move_number"`
	Player     int        `json:"player"`
}

// MatchAnalysisOptions configures match analysis behavior.
type MatchAnalysisOptions struct {
	IncludeLuck    bool    `json:"include_luck"`    // Calculate luck (slower)
	ErrorThreshold float64 `json:"error_threshold"` // Min error to report (default 0)
	Ply            int     `json:"ply"`             // Analysis ply (0, 1, 2)
	Player1Name    string  `json:"player1_name"`
	Player2Name    string  `json:"player2_name"`
}

// DefaultMatchAnalysisOptions returns sensible defaults.
func DefaultMatchAnalysisOptions() MatchAnalysisOptions {
	return MatchAnalysisOptions{
		IncludeLuck:    false,
		ErrorThreshold: 0.0,
		Ply:            0,
		Player1Name:    "Player 1",
		Player2Name:    "Player 2",
	}
}

// AnalyzePositionList analyzes a list of positions with moves/cube actions.
// This is the core function for match analysis from recorded games.
func (e *Engine) AnalyzePositionList(positions []AnalyzedPosition, opts MatchAnalysisOptions) (*MatchAnalysis, error) {
	if len(positions) == 0 {
		return &MatchAnalysis{}, nil
	}

	result := &MatchAnalysis{
		PlayerStats: [2]PlayerAnalysis{
			{Name: opts.Player1Name},
			{Name: opts.Player2Name},
		},
		GameStats:  make([]GameAnalysis, 0),
		MoveErrors: make([]MoveErrorDetail, 0),
		CubeErrors: make([]CubeErrorDetail, 0),
	}

	// Track current game
	currentGame := -1
	var gameAnalysis *GameAnalysis

	for _, pos := range positions {
		// Start new game if needed
		if pos.GameNumber != currentGame {
			if gameAnalysis != nil {
				// Finalize previous game
				for p := 0; p < 2; p++ {
					if gameAnalysis.MoveCount[p] > 0 {
						gameAnalysis.ErrorPerMove[p] = gameAnalysis.TotalError[p] / float64(gameAnalysis.MoveCount[p])
					}
				}
				result.GameStats = append(result.GameStats, *gameAnalysis)
			}
			currentGame = pos.GameNumber
			gameAnalysis = &GameAnalysis{
				GameNumber: currentGame,
				Winner:     -1,
				Errors:     make([]MoveErrorDetail, 0),
			}
			result.TotalGames++
		}

		player := pos.Player

		// Analyze move if present
		if pos.Move != nil {
			result.TotalMoves++
			gameAnalysis.MoveCount[player]++

			gs := &GameState{
				Board:     pos.Board,
				Turn:      pos.Turn,
				CubeValue: pos.CubeValue,
				CubeOwner: pos.CubeOwner,
				Score:     pos.Score,
			}

			analysis, err := e.AnalyzeMoveSkill(gs, *pos.Move, pos.Dice)
			if err != nil {
				continue
			}

			if !analysis.IsForced {
				result.PlayerStats[player].TotalMoves++

				if analysis.EquityLoss >= opts.ErrorThreshold {
					result.PlayerStats[player].TotalError += analysis.EquityLoss
					gameAnalysis.TotalError[player] += analysis.EquityLoss

					// Classify and count
					switch analysis.Skill {
					case SkillVeryBad:
						result.PlayerStats[player].Blunders++
					case SkillBad:
						result.PlayerStats[player].Errors++
					case SkillDoubtful:
						result.PlayerStats[player].Doubtful++
					}

					// Record error if significant
					if analysis.Skill != SkillNone {
						posID := EncodePositionID(pos.Board)
						errDetail := MoveErrorDetail{
							GameNumber: pos.GameNumber,
							MoveNumber: pos.MoveNumber,
							Player:     player,
							Position:   posID,
							Dice:       pos.Dice,
							Played:     FormatMove(*pos.Move),
							Best:       FormatMove(analysis.BestMove),
							EquityLoss: analysis.EquityLoss,
							Skill:      analysis.Skill,
							SkillStr:   analysis.Skill.String(),
						}
						result.MoveErrors = append(result.MoveErrors, errDetail)
						gameAnalysis.Errors = append(gameAnalysis.Errors, errDetail)
					}
				}
			}
		}

		// Analyze cube action if present
		if pos.CubeAction != NoDouble && pos.CubeAction != 0 {
			result.TotalCubeActs++
			result.PlayerStats[player].TotalCube++
			gameAnalysis.CubeActions++

			gs := &GameState{
				Board:     pos.Board,
				Turn:      pos.Turn,
				CubeValue: pos.CubeValue,
				CubeOwner: pos.CubeOwner,
				Score:     pos.Score,
			}

			analysis, err := e.AnalyzeCubeSkill(gs, pos.CubeAction)
			if err != nil {
				continue
			}

			if analysis.EquityLoss > 0 {
				result.PlayerStats[player].CubeError += analysis.EquityLoss
				result.PlayerStats[player].TotalError += analysis.EquityLoss

				// Categorize cube errors
				switch {
				case pos.CubeAction == NoDouble && analysis.OptimalPlay == Double:
					result.PlayerStats[player].MissedDoubles++
				case pos.CubeAction == Double && analysis.OptimalPlay == NoDouble:
					result.PlayerStats[player].WrongDoubles++
				case pos.CubeAction == Take && (analysis.OptimalPlay == Pass):
					result.PlayerStats[player].WrongTakes++
				case pos.CubeAction == Pass && (analysis.OptimalPlay == Take):
					result.PlayerStats[player].WrongPasses++
				}

				if analysis.Skill != SkillNone {
					posID := EncodePositionID(pos.Board)
					result.CubeErrors = append(result.CubeErrors, CubeErrorDetail{
						GameNumber: pos.GameNumber,
						MoveNumber: pos.MoveNumber,
						Player:     player,
						Position:   posID,
						Played:     pos.CubeAction,
						Optimal:    analysis.OptimalPlay,
						PlayedStr:  pos.CubeAction.String(),
						OptimalStr: analysis.OptimalPlay.String(),
						EquityLoss: analysis.EquityLoss,
						Skill:      analysis.Skill,
						SkillStr:   analysis.Skill.String(),
					})
				}
			}
		}
	}

	// Finalize last game
	if gameAnalysis != nil {
		for p := 0; p < 2; p++ {
			if gameAnalysis.MoveCount[p] > 0 {
				gameAnalysis.ErrorPerMove[p] = gameAnalysis.TotalError[p] / float64(gameAnalysis.MoveCount[p])
			}
		}
		result.GameStats = append(result.GameStats, *gameAnalysis)
	}

	// Calculate overall stats
	for p := 0; p < 2; p++ {
		if result.PlayerStats[p].TotalMoves > 0 {
			result.PlayerStats[p].ErrorPerMove = result.PlayerStats[p].TotalError / float64(result.PlayerStats[p].TotalMoves)
		}
		result.PlayerStats[p].Rating = GetRating(result.PlayerStats[p].ErrorPerMove)
		result.PlayerStats[p].RatingStr = result.PlayerStats[p].Rating.String()
	}

	return result, nil
}

// EncodePositionID returns the base64 position ID for a board.
func EncodePositionID(b Board) string {
	return positionid.PositionID(positionid.Board(b))
}

// FormatMove converts a Move to human-readable notation.
func FormatMove(m Move) string {
	result := ""
	for i := 0; i < 4; i++ {
		if m.From[i] < 0 {
			break
		}
		if i > 0 {
			result += " "
		}
		from := int(m.From[i]) + 1
		if m.From[i] == 24 {
			result += "bar"
		} else {
			result += fmt.Sprintf("%d", from)
		}
		result += "/"
		to := int(m.To[i]) + 1
		if m.To[i] == -1 || m.To[i] == 25 {
			result += "off"
		} else {
			result += fmt.Sprintf("%d", to)
		}
	}
	return result
}

// CubeActionString returns the string representation of a CubeAction.
func CubeActionString(a CubeAction) string {
	switch a {
	case NoDouble:
		return "no_double"
	case Double:
		return "double"
	case Redouble:
		return "redouble"
	case Take:
		return "take"
	case Pass:
		return "pass"
	case Beaver:
		return "beaver"
	default:
		return "unknown"
	}
}

// String returns the string representation of a CubeAction.
func (a CubeAction) String() string {
	return CubeActionString(a)
}

// MatchToPositions converts a Match into a list of AnalyzedPositions for analysis.
// This reconstructs positions from game actions.
type MatchActions struct {
	Actions     []MatchAction
	Player1Name string
	Player2Name string
}

// MatchAction represents an action in a match for analysis.
type MatchAction struct {
	GameNumber int
	MoveNumber int
	Player     int
	Dice       [2]int
	Move       *Move
	CubeAction CubeAction
}

// ConvertMatchActionsToPositions reconstructs positions from match actions.
// This walks through the actions and reconstructs the board state at each decision.
func ConvertMatchActionsToPositions(actions MatchActions, startBoard Board, score [2]int, matchLen int) []AnalyzedPosition {
	positions := make([]AnalyzedPosition, 0, len(actions.Actions))

	currentBoard := startBoard
	cubeValue := 1
	cubeOwner := -1
	gameNum := 1
	moveNum := 0

	for _, action := range actions.Actions {
		if action.GameNumber != gameNum {
			// New game - reset
			gameNum = action.GameNumber
			currentBoard = StartingPosition().Board
			cubeValue = 1
			cubeOwner = -1
			moveNum = 0
		}

		pos := AnalyzedPosition{
			Board:      currentBoard,
			Turn:       action.Player,
			Dice:       action.Dice,
			CubeValue:  cubeValue,
			CubeOwner:  cubeOwner,
			Score:      score,
			GameNumber: action.GameNumber,
			MoveNumber: action.MoveNumber,
			Player:     action.Player,
		}

		if action.Move != nil {
			pos.Move = action.Move
			moveNum++
			positions = append(positions, pos)
			// Apply move to update board
			currentBoard = ApplyMove(currentBoard, *action.Move)
			// Swap sides
			currentBoard = swapBoard(currentBoard)
		}

		if action.CubeAction != NoDouble && action.CubeAction != 0 {
			pos.CubeAction = action.CubeAction
			positions = append(positions, pos)
			// Update cube state
			if action.CubeAction == Double || action.CubeAction == Redouble {
				cubeValue *= 2
				cubeOwner = 1 - action.Player // Opponent now owns cube
			}
		}
	}

	return positions
}
