// Package positionid implements position encoding/decoding for backgammon boards.
// This is a port of gnubg's positionid.c
//
// The encoding creates compact position IDs compatible with GNU Backgammon.
// Position IDs are 14-character base64 strings that uniquely identify a board position.
package positionid

import (
	"errors"
)

const (
	// PositionIDLength is the length of a position ID string
	PositionIDLength = 14
	// MaxN is the maximum n for combination calculations
	MaxN = 40
	// MaxR is the maximum r for combination calculations
	MaxR = 25
)

// Base64 alphabet used for position ID encoding
const base64Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

// Board represents a backgammon board position
// [2][25] where [player][point], point 24 is the bar
type Board [2][25]uint8

// PositionKey is a compact binary representation of a board position
// Uses 7 uint32s to encode the position (4 bits per point)
type PositionKey struct {
	Data [7]uint32
}

// OldPositionKey is the legacy position key format (80 bits)
// Used for generating the base64 position ID string
type OldPositionKey struct {
	Data [10]uint8
}

// Combination table for bearoff calculations
var combinationTable [MaxN][MaxR]uint32
var combinationInitialized = false

// initCombination initializes the combination table
func initCombination() {
	for i := 0; i < MaxN; i++ {
		combinationTable[i][0] = uint32(i + 1)
	}

	for j := 1; j < MaxR; j++ {
		combinationTable[0][j] = 0
	}

	for i := 1; i < MaxN; i++ {
		for j := 1; j < MaxR; j++ {
			combinationTable[i][j] = combinationTable[i-1][j-1] + combinationTable[i-1][j]
		}
	}

	combinationInitialized = true
}

// Combination returns C(n, r) - n choose r
func Combination(n, r uint32) uint32 {
	if n > MaxN || r > MaxR || n == 0 || r == 0 {
		return 0
	}

	if !combinationInitialized {
		initCombination()
	}

	return combinationTable[n-1][r-1]
}

// PositionKey creates a compact key from a board position
// This is the fast internal representation (4 bits per point)
func MakePositionKey(board Board) PositionKey {
	var key PositionKey

	for i, j := 0, 0; i < 3; i, j = i+1, j+8 {
		key.Data[i] = uint32(board[1][j]) + (uint32(board[1][j+1]) << 4) +
			(uint32(board[1][j+2]) << 8) + (uint32(board[1][j+3]) << 12) +
			(uint32(board[1][j+4]) << 16) + (uint32(board[1][j+5]) << 20) +
			(uint32(board[1][j+6]) << 24) + (uint32(board[1][j+7]) << 28)

		key.Data[i+3] = uint32(board[0][j]) + (uint32(board[0][j+1]) << 4) +
			(uint32(board[0][j+2]) << 8) + (uint32(board[0][j+3]) << 12) +
			(uint32(board[0][j+4]) << 16) + (uint32(board[0][j+5]) << 20) +
			(uint32(board[0][j+6]) << 24) + (uint32(board[0][j+7]) << 28)
	}
	key.Data[6] = uint32(board[0][24]) + (uint32(board[1][24]) << 4)

	return key
}

// BoardFromKey reconstructs a board from a position key
func BoardFromKey(key PositionKey) Board {
	var board Board

	for i, j := 0, 0; i < 3; i, j = i+1, j+8 {
		board[1][j] = uint8(key.Data[i] & 0x0f)
		board[1][j+1] = uint8((key.Data[i] >> 4) & 0x0f)
		board[1][j+2] = uint8((key.Data[i] >> 8) & 0x0f)
		board[1][j+3] = uint8((key.Data[i] >> 12) & 0x0f)
		board[1][j+4] = uint8((key.Data[i] >> 16) & 0x0f)
		board[1][j+5] = uint8((key.Data[i] >> 20) & 0x0f)
		board[1][j+6] = uint8((key.Data[i] >> 24) & 0x0f)
		board[1][j+7] = uint8((key.Data[i] >> 28) & 0x0f)

		board[0][j] = uint8(key.Data[i+3] & 0x0f)
		board[0][j+1] = uint8((key.Data[i+3] >> 4) & 0x0f)
		board[0][j+2] = uint8((key.Data[i+3] >> 8) & 0x0f)
		board[0][j+3] = uint8((key.Data[i+3] >> 12) & 0x0f)
		board[0][j+4] = uint8((key.Data[i+3] >> 16) & 0x0f)
		board[0][j+5] = uint8((key.Data[i+3] >> 20) & 0x0f)
		board[0][j+6] = uint8((key.Data[i+3] >> 24) & 0x0f)
		board[0][j+7] = uint8((key.Data[i+3] >> 28) & 0x0f)
	}
	board[0][24] = uint8(key.Data[6] & 0x0f)
	board[1][24] = uint8((key.Data[6] >> 4) & 0x0f)

	return board
}

// BoardFromKeySwapped reconstructs a board from a position key with players swapped
func BoardFromKeySwapped(key PositionKey) Board {
	var board Board

	for i, j := 0, 0; i < 3; i, j = i+1, j+8 {
		board[0][j] = uint8(key.Data[i] & 0x0f)
		board[0][j+1] = uint8((key.Data[i] >> 4) & 0x0f)
		board[0][j+2] = uint8((key.Data[i] >> 8) & 0x0f)
		board[0][j+3] = uint8((key.Data[i] >> 12) & 0x0f)
		board[0][j+4] = uint8((key.Data[i] >> 16) & 0x0f)
		board[0][j+5] = uint8((key.Data[i] >> 20) & 0x0f)
		board[0][j+6] = uint8((key.Data[i] >> 24) & 0x0f)
		board[0][j+7] = uint8((key.Data[i] >> 28) & 0x0f)

		board[1][j] = uint8(key.Data[i+3] & 0x0f)
		board[1][j+1] = uint8((key.Data[i+3] >> 4) & 0x0f)
		board[1][j+2] = uint8((key.Data[i+3] >> 8) & 0x0f)
		board[1][j+3] = uint8((key.Data[i+3] >> 12) & 0x0f)
		board[1][j+4] = uint8((key.Data[i+3] >> 16) & 0x0f)
		board[1][j+5] = uint8((key.Data[i+3] >> 20) & 0x0f)
		board[1][j+6] = uint8((key.Data[i+3] >> 24) & 0x0f)
		board[1][j+7] = uint8((key.Data[i+3] >> 28) & 0x0f)
	}
	board[1][24] = uint8(key.Data[6] & 0x0f)
	board[0][24] = uint8((key.Data[6] >> 4) & 0x0f)

	return board
}

// addBits adds nBits 1-bits to the old position key starting at bitPos
func addBits(key *OldPositionKey, bitPos, nBits uint32) {
	k := bitPos / 8
	r := bitPos & 0x7
	b := ((uint32(1) << nBits) - 1) << r

	key.Data[k] |= uint8(b)

	if k < 8 {
		key.Data[k+1] |= uint8(b >> 8)
		key.Data[k+2] |= uint8(b >> 16)
	} else if k == 8 {
		key.Data[k+1] |= uint8(b >> 8)
	}
}

// MakeOldPositionKey creates the legacy position key from a board
// This is used for generating the base64 position ID string
func MakeOldPositionKey(board Board) OldPositionKey {
	var key OldPositionKey
	var bitPos uint32 = 0

	for i := 0; i < 2; i++ {
		for j := 0; j < 25; j++ {
			nc := uint32(board[i][j])
			if nc > 0 {
				addBits(&key, bitPos, nc)
				bitPos += nc + 1
			} else {
				bitPos++
			}
		}
	}

	return key
}

// BoardFromOldKey reconstructs a board from a legacy position key
func BoardFromOldKey(key OldPositionKey) Board {
	var board Board
	i, j := 0, 0

	for a := 0; a < 10; a++ {
		cur := key.Data[a]

		for k := 0; k < 8; k++ {
			if cur&0x1 != 0 {
				if i >= 2 || j >= 25 {
					// Error - return what we have
					return board
				}
				board[i][j]++
			} else {
				j++
				if j == 25 {
					i++
					j = 0
				}
			}
			cur >>= 1
		}
	}

	return board
}

// PositionIDFromOldKey generates a base64 position ID string from an old position key
func PositionIDFromOldKey(key OldPositionKey) string {
	result := make([]byte, PositionIDLength)
	puch := key.Data[:]

	for i := 0; i < 3; i++ {
		result[i*4] = base64Chars[puch[0]>>2]
		result[i*4+1] = base64Chars[((puch[0]&0x03)<<4)|(puch[1]>>4)]
		result[i*4+2] = base64Chars[((puch[1]&0x0F)<<2)|(puch[2]>>6)]
		result[i*4+3] = base64Chars[puch[2]&0x3F]
		puch = puch[3:]
	}

	result[12] = base64Chars[puch[0]>>2]
	result[13] = base64Chars[(puch[0]&0x03)<<4]

	return string(result)
}

// PositionID generates a base64 position ID string from a board
func PositionID(board Board) string {
	key := MakeOldPositionKey(board)
	return PositionIDFromOldKey(key)
}

// PositionIDFromKey generates a base64 position ID string from a position key
func PositionIDFromKey(key PositionKey) string {
	board := BoardFromKey(key)
	oldKey := MakeOldPositionKey(board)
	return PositionIDFromOldKey(oldKey)
}

// base64Decode decodes a base64 character to its value
func base64Decode(ch byte) uint8 {
	if ch >= 'A' && ch <= 'Z' {
		return ch - 'A'
	}
	if ch >= 'a' && ch <= 'z' {
		return ch - 'a' + 26
	}
	if ch >= '0' && ch <= '9' {
		return ch - '0' + 52
	}
	if ch == '+' {
		return 62
	}
	if ch == '/' {
		return 63
	}
	return 255
}

// ErrInvalidPositionID is returned when a position ID is invalid
var ErrInvalidPositionID = errors.New("invalid position ID")

// BoardFromPositionID decodes a base64 position ID string to a board
func BoardFromPositionID(posID string) (Board, error) {
	var key OldPositionKey
	var board Board

	if len(posID) < PositionIDLength {
		return board, ErrInvalidPositionID
	}

	// Decode base64 characters
	ach := make([]uint8, PositionIDLength)
	for i := 0; i < PositionIDLength; i++ {
		ach[i] = base64Decode(posID[i])
		if ach[i] == 255 {
			return board, ErrInvalidPositionID
		}
	}

	// Convert to old position key bytes
	pch := ach
	puchIdx := 0
	for i := 0; i < 3; i++ {
		key.Data[puchIdx] = (pch[0] << 2) | (pch[1] >> 4)
		key.Data[puchIdx+1] = (pch[1] << 4) | (pch[2] >> 2)
		key.Data[puchIdx+2] = (pch[2] << 6) | pch[3]
		puchIdx += 3
		pch = pch[4:]
	}
	key.Data[9] = (pch[0] << 2) | (pch[1] >> 4)

	board = BoardFromOldKey(key)

	if !CheckPosition(board) {
		return board, ErrInvalidPositionID
	}

	return board, nil
}

// CheckPosition validates that a board position is legal
func CheckPosition(board Board) bool {
	var ac [2]uint32

	// Check for a player with over 15 checkers
	for i := 0; i < 25; i++ {
		ac[0] += uint32(board[0][i])
		ac[1] += uint32(board[1][i])
		if ac[0] > 15 || ac[1] > 15 {
			return false
		}
	}

	// Check for both players having checkers on the same point
	for i := 0; i < 24; i++ {
		if board[0][i] > 0 && board[1][23-i] > 0 {
			return false
		}
	}

	// Check for both players on the bar against closed boards
	for i := 0; i < 6; i++ {
		if board[0][i] < 2 || board[1][i] < 2 {
			return true // Open board
		}
	}

	if board[0][24] == 0 || board[1][24] == 0 {
		return true
	}

	return false
}

// EqualBoards returns true if two boards are identical
func EqualBoards(b1, b2 Board) bool {
	for i := 0; i < 2; i++ {
		for j := 0; j < 25; j++ {
			if b1[i][j] != b2[i][j] {
				return false
			}
		}
	}
	return true
}

// EqualKeys returns true if two position keys are identical
func EqualKeys(k1, k2 PositionKey) bool {
	for i := 0; i < 7; i++ {
		if k1.Data[i] != k2.Data[i] {
			return false
		}
	}
	return true
}

// positionF is a helper for bearoff position calculation
func positionF(fBits, n, r uint32) uint32 {
	if n == r {
		return 0
	}

	if fBits&(1<<(n-1)) != 0 {
		return Combination(n-1, r) + positionF(fBits, n-1, r-1)
	}
	return positionF(fBits, n-1, r)
}

// PositionBearoff calculates the bearoff database index for a position
func PositionBearoff(board []uint8, nPoints, nChequers uint32) uint32 {
	if nPoints == 0 {
		return 0
	}

	var j uint32 = nPoints - 1
	for i := uint32(0); i < nPoints; i++ {
		j += uint32(board[i])
	}

	fBits := uint32(1) << j

	for i := uint32(0); i < nPoints-1; i++ {
		j -= uint32(board[i]) + 1
		fBits |= uint32(1) << j
	}

	return positionF(fBits, nChequers+nPoints, nPoints)
}

// positionInv is a helper for inverse bearoff position calculation
func positionInv(nID, n, r uint32) uint32 {
	if r == 0 {
		return 0
	}
	if n == r {
		return (1 << n) - 1
	}

	nC := Combination(n-1, r)

	if nID >= nC {
		return (1 << (n - 1)) | positionInv(nID-nC, n-1, r-1)
	}
	return positionInv(nID, n-1, r)
}

// PositionFromBearoff reconstructs a board position from a bearoff database index
func PositionFromBearoff(usID, nPoints, nChequers uint32) []uint8 {
	board := make([]uint8, nPoints)
	fBits := positionInv(usID, nChequers+nPoints, nPoints)

	j := nPoints - 1
	for i := uint32(0); i < nChequers+nPoints; i++ {
		if fBits&(1<<i) != 0 {
			if j == 0 {
				break
			}
			j--
		} else {
			board[j]++
		}
	}

	return board
}

// PositionIndex calculates the position index for bearoff gammon calculations
func PositionIndex(g uint32, board []uint8) uint16 {
	if g == 0 {
		return 0
	}

	j := g - 1
	for i := uint32(0); i < g; i++ {
		j += uint32(board[i])
	}

	fBits := uint32(1) << j

	for i := uint32(0); i < g-1; i++ {
		j -= uint32(board[i]) + 1
		fBits |= uint32(1) << j
	}

	// 15 is the number of checkers
	return uint16(positionF(fBits, 15, g))
}

// SwapSides swaps the two sides of the board
// This is used to flip perspective after a move
func SwapSides(board Board) Board {
	var result Board
	for i := 0; i < 25; i++ {
		result[0][i] = board[1][i]
		result[1][i] = board[0][i]
	}
	return result
}
