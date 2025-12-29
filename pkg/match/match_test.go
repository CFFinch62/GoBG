package match

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yourusername/bgengine/pkg/engine"
)

func TestNewMatch(t *testing.T) {
	m := NewMatch("Alice", "Bob", 7)
	if m.Player1 != "Alice" {
		t.Errorf("Player1 = %q, want %q", m.Player1, "Alice")
	}
	if m.Player2 != "Bob" {
		t.Errorf("Player2 = %q, want %q", m.Player2, "Bob")
	}
	if m.MatchLength != 7 {
		t.Errorf("MatchLength = %d, want %d", m.MatchLength, 7)
	}
}

func TestNewGame(t *testing.T) {
	g := NewGame(1, 3, 2, true)
	if g.Number != 1 {
		t.Errorf("Number = %d, want %d", g.Number, 1)
	}
	if g.Score1 != 3 {
		t.Errorf("Score1 = %d, want %d", g.Score1, 3)
	}
	if !g.Crawford {
		t.Error("Crawford = false, want true")
	}
	if g.CubeValue != 1 {
		t.Errorf("CubeValue = %d, want %d", g.CubeValue, 1)
	}
}

func TestGameActions(t *testing.T) {
	g := NewGame(1, 0, 0, false)

	g.AddRoll(0, 3, 1)
	if len(g.Actions) != 1 || g.Actions[0].Type != ActionRoll {
		t.Error("AddRoll failed")
	}
	if g.Actions[0].Dice != [2]int{3, 1} {
		t.Errorf("Dice = %v, want [3, 1]", g.Actions[0].Dice)
	}

	move := engine.Move{From: [4]int8{8, 6, -1, -1}, To: [4]int8{5, 5, -1, -1}}
	g.AddMove(0, move)
	if len(g.Actions) != 2 || g.Actions[1].Type != ActionMove {
		t.Error("AddMove failed")
	}

	g.AddDouble(0, 2)
	if len(g.Actions) != 3 || g.Actions[2].Type != ActionDouble {
		t.Error("AddDouble failed")
	}

	g.AddTake(1)
	if g.Actions[3].Type != ActionTake {
		t.Error("AddTake failed")
	}

	g.AddPass(1)
	if g.Actions[4].Type != ActionPass {
		t.Error("AddPass failed")
	}
}

func TestImportMAT(t *testing.T) {
	matContent := " ; [Site \"TestSite\"]\n ; [Player 1 \"Alice\"]\n ; [Player 2 \"Bob\"]\n 7 point match\n\n Game 1\n Alice : 0                          Bob : 0\n  1) 31: 8/5 6/5                    52: 24/22 13/8\n"

	match, err := ImportMAT(strings.NewReader(matContent))
	if err != nil {
		t.Fatalf("ImportMAT error: %v", err)
	}

	if match.Player1 != "Alice" {
		t.Errorf("Player1 = %q, want %q", match.Player1, "Alice")
	}
	if match.MatchLength != 7 {
		t.Errorf("MatchLength = %d, want %d", match.MatchLength, 7)
	}
	if len(match.Games) != 1 {
		t.Fatalf("Games = %d, want 1", len(match.Games))
	}
	if len(match.Games[0].Actions) < 4 {
		t.Errorf("Actions = %d, want at least 4", len(match.Games[0].Actions))
	}
}

func TestExportMAT(t *testing.T) {
	match := NewMatch("Alice", "Bob", 5)
	game := NewGame(1, 0, 0, false)
	game.AddRoll(0, 3, 1)
	game.AddMove(0, engine.Move{From: [4]int8{8, 6, -1, -1}, To: [4]int8{5, 5, -1, -1}})
	match.Games = append(match.Games, game)

	var buf bytes.Buffer
	err := ExportMAT(&buf, match)
	if err != nil {
		t.Fatalf("ExportMAT error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "Alice") || !strings.Contains(output, "5 point match") {
		t.Error("Output missing expected content")
	}
}

func TestImportSGF(t *testing.T) {
	sgfContent := "(;FF[4]GM[6]AP[gnubg]PW[Alice]PB[Bob]MI[length:7]DT[2024-01-15];W[31];B[52])"

	match, err := ImportSGF(strings.NewReader(sgfContent))
	if err != nil {
		t.Fatalf("ImportSGF error: %v", err)
	}

	if match.Player1 != "Alice" {
		t.Errorf("Player1 = %q, want %q", match.Player1, "Alice")
	}
	if match.MatchLength != 7 {
		t.Errorf("MatchLength = %d, want %d", match.MatchLength, 7)
	}
	if len(match.Games) != 1 {
		t.Fatalf("Games = %d, want 1", len(match.Games))
	}
}

func TestExportSGF(t *testing.T) {
	match := NewMatch("Alice", "Bob", 5)
	game := NewGame(1, 0, 0, false)
	game.AddRoll(0, 3, 1)
	game.AddMove(0, engine.Move{From: [4]int8{8, 6, -1, -1}, To: [4]int8{5, 5, -1, -1}})
	match.Games = append(match.Games, game)

	var buf bytes.Buffer
	err := ExportSGF(&buf, match)
	if err != nil {
		t.Fatalf("ExportSGF error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "FF[4]") || !strings.Contains(output, "GM[6]") {
		t.Error("Output missing SGF markers")
	}
}

func TestParsePoint(t *testing.T) {
	if parsePoint("bar", 0) != 25 {
		t.Error("parsePoint(bar, 0) should be 25")
	}
	if parsePoint("bar", 1) != 0 {
		t.Error("parsePoint(bar, 1) should be 0")
	}
	if parsePoint("off", 0) != 0 {
		t.Error("parsePoint(off, 0) should be 0")
	}
	if parsePoint("off", 1) != 25 {
		t.Error("parsePoint(off, 1) should be 25")
	}
}

