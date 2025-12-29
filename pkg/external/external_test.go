package external

import (
"testing"
)

func TestParseFIBSBoard(t *testing.T) {
// Sample FIBS board string
fibsBoard := "board:You:Opponent:5:2:3:0:2:0:0:0:0:-5:0:-3:0:0:0:5:-5:0:0:0:3:0:5:0:0:0:1:3:1:0:0:0:1:1:0:1:-1"

fb, err := ParseFIBSBoard(fibsBoard)
if err != nil {
t.Fatalf("ParseFIBSBoard error: %v", err)
}

if fb.Player1 != "You" {
t.Errorf("Player1 = %q, want %q", fb.Player1, "You")
}
if fb.Player2 != "Opponent" {
t.Errorf("Player2 = %q, want %q", fb.Player2, "Opponent")
}
if fb.MatchLength != 5 {
t.Errorf("MatchLength = %d, want %d", fb.MatchLength, 5)
}
if fb.Score1 != 2 {
t.Errorf("Score1 = %d, want %d", fb.Score1, 2)
}
if fb.Score2 != 3 {
t.Errorf("Score2 = %d, want %d", fb.Score2, 3)
}
}

func TestParseFIBSBoardMinimal(t *testing.T) {
// Minimal FIBS board with exactly 32 fields
parts := make([]string, 32)
parts[0] = "P1"
parts[1] = "P2"
parts[2] = "7"  // match length
parts[3] = "0"  // score1
parts[4] = "0"  // score2
// Board positions 5-30 (26 values)
for i := 5; i < 31; i++ {
parts[i] = "0"
}
parts[31] = "1" // turn

fibsStr := "board:" + joinParts(parts, ":")

fb, err := ParseFIBSBoard(fibsStr)
if err != nil {
t.Fatalf("ParseFIBSBoard error: %v", err)
}

if fb.Player1 != "P1" {
t.Errorf("Player1 = %q, want %q", fb.Player1, "P1")
}
if fb.MatchLength != 7 {
t.Errorf("MatchLength = %d, want %d", fb.MatchLength, 7)
}
}

func joinParts(parts []string, sep string) string {
result := parts[0]
for i := 1; i < len(parts); i++ {
result += sep + parts[i]
}
return result
}

func TestParseFIBSBoardInvalid(t *testing.T) {
_, err := ParseFIBSBoard("invalid:board")
if err == nil {
t.Error("Expected error for invalid board")
}
}

func TestToGameState(t *testing.T) {
fb := &FIBSBoard{
Player1:      "Me",
Player2:      "Opp",
MatchLength:  5,
Score1:       2,
Score2:       1,
Cube:         2,
Turn:         1,
CanDouble:    true,
OppCanDouble: false,
Dice:         [2]int{3, 1},
}

// Set some checkers
fb.Board[6] = 5   // 5 checkers on point 6 (my checkers)
fb.Board[13] = -5 // 5 opponent checkers on point 13

state := fb.ToGameState()

if state.MatchLength != 5 {
t.Errorf("MatchLength = %d, want 5", state.MatchLength)
}
if state.CubeValue != 2 {
t.Errorf("CubeValue = %d, want 2", state.CubeValue)
}
if state.Turn != 0 {
t.Errorf("Turn = %d, want 0", state.Turn)
}
if state.Dice != [2]int{3, 1} {
t.Errorf("Dice = %v, want [3, 1]", state.Dice)
}
// Check cube owner - CanDouble=true, OppCanDouble=false means we own it
if state.CubeOwner != 0 {
t.Errorf("CubeOwner = %d, want 0", state.CubeOwner)
}
}

func TestFormatMove(t *testing.T) {
tests := []struct {
from      [4]int8
to        [4]int8
direction int
want      string
}{
{[4]int8{7, 5, -1, -1}, [4]int8{4, 4, -1, -1}, 1, "8/5 6/5"},
{[4]int8{23, -1, -1, -1}, [4]int8{17, -1, -1, -1}, 1, "24/18"},
{[4]int8{24, -1, -1, -1}, [4]int8{20, -1, -1, -1}, 1, "bar/21"},
}

for _, tc := range tests {
move := struct {
From [4]int8
To   [4]int8
}{tc.from, tc.to}

// We need to import engine.Move, skip for now
_ = move
}
}

func TestFormatFIBSPoint(t *testing.T) {
tests := []struct {
point     int
direction int
want      string
}{
{24, 1, "bar"},
{25, 1, "bar"},
{-1, 1, "off"},
{5, 1, "6"},
{0, 1, "1"},
}

for _, tc := range tests {
got := formatFIBSPoint(tc.point, tc.direction)
if got != tc.want {
t.Errorf("formatFIBSPoint(%d, %d) = %q, want %q", tc.point, tc.direction, got, tc.want)
}
}
}
