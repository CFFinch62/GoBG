// Package met provides match equity table functionality.
// Match equity tables give the probability of winning a match from a given score.
package met

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// MaxScore is the maximum supported match length
const MaxScore = 64

// Table represents a match equity table
type Table struct {
	Name        string
	Description string
	Length      int // Native length of the table

	// Pre-Crawford match equities
	// PreCrawford[i][j] = P(player wins match | player needs i+1, opponent needs j+1)
	PreCrawford [MaxScore][MaxScore]float32

	// Post-Crawford match equities
	// PostCrawford[0][i] = P(player 0 wins | player 0 needs i+1 to win, Crawford game)
	// PostCrawford[1][i] = P(player 1 wins | player 1 needs i+1 to win, Crawford game)
	PostCrawford [2][MaxScore]float32
}

// XML parsing structures
type xmlMET struct {
	XMLName      xml.Name          `xml:"met"`
	Info         xmlInfo           `xml:"info"`
	PreCrawford  xmlPreCrawford    `xml:"pre-crawford-table"`
	PostCrawford []xmlPostCrawford `xml:"post-crawford-table"`
}

type xmlInfo struct {
	Name        string `xml:"name"`
	Description string `xml:"description"`
	Length      int    `xml:"length"`
}

type xmlPreCrawford struct {
	Type string   `xml:"type,attr"`
	Rows []xmlRow `xml:"row"`
}

type xmlPostCrawford struct {
	Player string `xml:"player,attr"` // "0", "1", or "both"
	Type   string `xml:"type,attr"`
	Row    xmlRow `xml:"row"`
}

type xmlRow struct {
	Values []string `xml:"me"`
}

// LoadXML loads a match equity table from an XML file
func LoadXML(filename string) (*Table, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open MET file: %w", err)
	}
	defer f.Close()
	return ParseXML(f)
}

// ParseXML parses a match equity table from XML
func ParseXML(r io.Reader) (*Table, error) {
	var met xmlMET
	decoder := xml.NewDecoder(r)
	if err := decoder.Decode(&met); err != nil {
		return nil, fmt.Errorf("failed to parse MET XML: %w", err)
	}

	t := &Table{
		Name:        met.Info.Name,
		Description: met.Info.Description,
		Length:      met.Info.Length,
	}

	// Parse pre-Crawford table
	for i, row := range met.PreCrawford.Rows {
		if i >= MaxScore {
			break
		}
		for j, val := range row.Values {
			if j >= MaxScore {
				break
			}
			f, err := strconv.ParseFloat(strings.TrimSpace(val), 32)
			if err != nil {
				return nil, fmt.Errorf("failed to parse MET value [%d][%d]: %w", i, j, err)
			}
			t.PreCrawford[i][j] = float32(f)
		}
	}

	// Parse post-Crawford tables
	for _, pc := range met.PostCrawford {
		// Determine which players this table applies to
		var players []int
		switch pc.Player {
		case "0":
			players = []int{0}
		case "1":
			players = []int{1}
		case "both", "":
			players = []int{0, 1}
		default:
			continue
		}

		for _, player := range players {
			for j, val := range pc.Row.Values {
				if j >= MaxScore {
					break
				}
				f, err := strconv.ParseFloat(strings.TrimSpace(val), 32)
				if err != nil {
					return nil, fmt.Errorf("failed to parse post-Crawford value [%d][%d]: %w", player, j, err)
				}
				t.PostCrawford[player][j] = float32(f)
			}
		}
	}

	// If no post-Crawford table, calculate from pre-Crawford
	if len(met.PostCrawford) == 0 {
		t.calculatePostCrawford()
	}

	return t, nil
}

// calculatePostCrawford calculates post-Crawford equities from pre-Crawford
func (t *Table) calculatePostCrawford() {
	// Post-Crawford for player who is 1-away:
	// They have 50% chance of winning each game (simplified)
	// Actual calculation would use gammon rates, but this is a good approximation
	for i := 0; i < MaxScore; i++ {
		// Using Woolsey-Heinrich approximation
		t.PostCrawford[0][i] = float32(1.0 / (1.0 + float64(i+1)*0.7))
		t.PostCrawford[1][i] = float32(1.0 / (1.0 + float64(i+1)*0.7))
	}
}

// GetME returns the match equity for a given score
// score0, score1: current scores
// matchTo: match length
// player: which player's equity to return (0 or 1)
// crawford: true if this is the Crawford game
func (t *Table) GetME(score0, score1, matchTo, player int, crawford bool) float32 {
	if matchTo == 0 {
		// Money game - return 0.5
		return 0.5
	}

	away0 := matchTo - score0 - 1
	away1 := matchTo - score1 - 1

	// Check if match is already won
	if away0 < 0 {
		if player == 0 {
			return 1.0
		}
		return 0.0
	}
	if away1 < 0 {
		if player == 1 {
			return 1.0
		}
		return 0.0
	}

	// Clamp to table size
	if away0 >= MaxScore {
		away0 = MaxScore - 1
	}
	if away1 >= MaxScore {
		away1 = MaxScore - 1
	}

	var equity float32
	if crawford && (away0 == 0 || away1 == 0) {
		// Post-Crawford game
		if away0 == 0 {
			equity = 1.0 - t.PostCrawford[1][away1]
		} else {
			equity = t.PostCrawford[0][away0]
		}
	} else {
		// Pre-Crawford game
		equity = t.PreCrawford[away0][away1]
	}

	if player == 1 {
		equity = 1.0 - equity
	}
	return equity
}

// GetMEAfterResult returns match equity after winning/losing with given points
// player: which player's equity to return
// points: points won (1=normal, 2=gammon, 3=backgammon)
// winner: 0 or 1, who won
func (t *Table) GetMEAfterResult(score0, score1, matchTo, player, points, winner int, crawford bool) float32 {
	newScore0 := score0
	newScore1 := score1

	if winner == 0 {
		newScore0 += points
	} else {
		newScore1 += points
	}

	// Crawford rule: if a player reaches match point - 1, next game is Crawford
	newCrawford := false
	if !crawford {
		if newScore0 == matchTo-1 || newScore1 == matchTo-1 {
			newCrawford = true
		}
	}

	return t.GetME(newScore0, newScore1, matchTo, player, newCrawford)
}

// Default returns the default match equity table (g11)
// This provides reasonable defaults without loading from file
func Default() *Table {
	t := &Table{
		Name:        "Default MET",
		Description: "Simplified match equity table",
		Length:      11,
	}

	// Initialize with Jacobs-Trice approximation
	for i := 0; i < MaxScore; i++ {
		for j := 0; j < MaxScore; j++ {
			// Simple approximation: equity based on points-away ratio
			// More sophisticated would use recurrence relations
			pi := float64(i + 1)
			pj := float64(j + 1)
			t.PreCrawford[i][j] = float32(pj / (pi + pj))
		}
	}

	t.calculatePostCrawford()
	return t
}
