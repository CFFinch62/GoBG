// Package main provides C-compatible functions for building a shared library.
// Build with: go build -buildmode=c-shared -o libbgengine.so ./pkg/capi
package main

/*
#include <stdlib.h>
#include <stdint.h>
*/
import "C"
import (
	"encoding/json"
	"strings"
	"sync"
	"unsafe"

	"github.com/yourusername/bgengine/internal/positionid"
	"github.com/yourusername/bgengine/pkg/engine"
)

var (
	globalEngine *engine.Engine
	engineMutex  sync.RWMutex
	lastError    string
	errorMutex   sync.Mutex
)

// setError stores an error message for later retrieval.
func setError(err error) {
	errorMutex.Lock()
	defer errorMutex.Unlock()
	if err != nil {
		lastError = err.Error()
	} else {
		lastError = ""
	}
}

// parsePosition converts a position ID string to a GameState.
func parsePosition(posStr string) (*engine.GameState, error) {
	// Handle gnubg format "positionID:matchID" - we only need the position part
	if idx := strings.Index(posStr, ":"); idx >= 0 {
		posStr = posStr[:idx]
	}

	board, err := positionid.BoardFromPositionID(posStr)
	if err != nil {
		return nil, err
	}

	return &engine.GameState{
		Board:     engine.Board(board),
		Turn:      0,
		CubeValue: 1,
		CubeOwner: -1,
	}, nil
}

//export bgengine_version
func bgengine_version() *C.char {
	return C.CString("0.1.0")
}

//export bgengine_last_error
func bgengine_last_error() *C.char {
	errorMutex.Lock()
	defer errorMutex.Unlock()
	if lastError == "" {
		return nil
	}
	return C.CString(lastError)
}

//export bgengine_init
func bgengine_init(weightsFile, bearoffFile, bearoffTSFile, metFile *C.char) C.int {
	engineMutex.Lock()
	defer engineMutex.Unlock()

	opts := engine.EngineOptions{}
	if weightsFile != nil {
		opts.WeightsFileText = C.GoString(weightsFile)
	}
	if bearoffFile != nil {
		opts.BearoffFile = C.GoString(bearoffFile)
	}
	if bearoffTSFile != nil {
		opts.BearoffTSFile = C.GoString(bearoffTSFile)
	}
	if metFile != nil {
		opts.METFile = C.GoString(metFile)
	}

	eng, err := engine.NewEngine(opts)
	if err != nil {
		setError(err)
		return -1
	}

	globalEngine = eng
	setError(nil)
	return 0
}

//export bgengine_shutdown
func bgengine_shutdown() {
	engineMutex.Lock()
	defer engineMutex.Unlock()
	globalEngine = nil
}

//export bgengine_evaluate
func bgengine_evaluate(positionID *C.char, resultJSON **C.char) C.int {
	engineMutex.RLock()
	eng := globalEngine
	engineMutex.RUnlock()

	if eng == nil {
		setError(nil)
		*resultJSON = C.CString(`{"error": "engine not initialized"}`)
		return -1
	}

	posID := C.GoString(positionID)
	gs, err := parsePosition(posID)
	if err != nil {
		setError(err)
		*resultJSON = C.CString(`{"error": "invalid position"}`)
		return -1
	}

	eval, err := eng.Evaluate(gs)
	if err != nil {
		setError(err)
		*resultJSON = C.CString(`{"error": "evaluation failed"}`)
		return -1
	}

	result := map[string]interface{}{
		"equity":  eval.Equity,
		"win":     eval.WinProb * 100,
		"win_g":   eval.WinG * 100,
		"win_bg":  eval.WinBG * 100,
		"lose_g":  eval.LoseG * 100,
		"lose_bg": eval.LoseBG * 100,
	}

	jsonBytes, _ := json.Marshal(result)
	*resultJSON = C.CString(string(jsonBytes))
	setError(nil)
	return 0
}

//export bgengine_best_move
func bgengine_best_move(positionID *C.char, die1, die2 C.int, resultJSON **C.char) C.int {
	engineMutex.RLock()
	eng := globalEngine
	engineMutex.RUnlock()

	if eng == nil {
		*resultJSON = C.CString(`{"error": "engine not initialized"}`)
		return -1
	}

	posID := C.GoString(positionID)
	gs, err := parsePosition(posID)
	if err != nil {
		setError(err)
		*resultJSON = C.CString(`{"error": "invalid position"}`)
		return -1
	}

	dice := [2]int{int(die1), int(die2)}
	analysis, err := eng.AnalyzePosition(gs, dice)
	if err != nil {
		setError(err)
		*resultJSON = C.CString(`{"error": "analysis failed"}`)
		return -1
	}

	if len(analysis.Moves) == 0 {
		*resultJSON = C.CString(`{"move": "", "equity": 0, "num_legal": 0}`)
		setError(nil)
		return 0
	}

	best := analysis.Moves[0]
	result := map[string]interface{}{
		"move":      formatMove(best.Move),
		"equity":    best.Equity,
		"win":       best.Eval.WinProb * 100,
		"win_g":     best.Eval.WinG * 100,
		"num_legal": analysis.NumMoves,
	}

	jsonBytes, _ := json.Marshal(result)
	*resultJSON = C.CString(string(jsonBytes))
	setError(nil)
	return 0
}

//export bgengine_cube_decision
func bgengine_cube_decision(positionID *C.char, resultJSON **C.char) C.int {
	engineMutex.RLock()
	eng := globalEngine
	engineMutex.RUnlock()

	if eng == nil {
		*resultJSON = C.CString(`{"error": "engine not initialized"}`)
		return -1
	}

	posID := C.GoString(positionID)
	gs, err := parsePosition(posID)
	if err != nil {
		setError(err)
		*resultJSON = C.CString(`{"error": "invalid position"}`)
		return -1
	}

	decision, err := eng.AnalyzeCube(gs)
	if err != nil {
		setError(err)
		*resultJSON = C.CString(`{"error": "cube analysis failed"}`)
		return -1
	}

	action := "no_double"
	diff := decision.DoubleTakeEq - decision.NoDoubleEquity
	if diff > 0 {
		if decision.DoublePassEq > decision.DoubleTakeEq {
			action = "double_pass"
		} else {
			action = "double_take"
		}
	}

	result := map[string]interface{}{
		"action":           action,
		"double_equity":    decision.DoubleTakeEq,
		"no_double_equity": decision.NoDoubleEquity,
		"double_diff":      diff,
	}

	jsonBytes, _ := json.Marshal(result)
	*resultJSON = C.CString(string(jsonBytes))
	setError(nil)
	return 0
}

// formatMove converts a Move to human-readable notation.
func formatMove(m engine.Move) string {
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
			result += itoa(from)
		}
		result += "/"
		if m.To[i] < 0 {
			result += "off"
		} else {
			result += itoa(int(m.To[i]) + 1)
		}
	}
	return result
}

// itoa converts an int to a string (simple implementation for small numbers).
func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}

//export bgengine_free_string
func bgengine_free_string(s *C.char) {
	if s != nil {
		C.free(unsafe.Pointer(s))
	}
}

func main() {}
