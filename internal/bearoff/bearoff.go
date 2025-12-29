// Package bearoff provides bearoff database functionality for endgame evaluation.
// It reads gnubg's .bd bearoff database files.
package bearoff

import (
	"fmt"
	"io"
	"math"
	"os"
)

// BearoffType identifies the type of bearoff database
type BearoffType int

const (
	BearoffInvalid BearoffType = iota
	BearoffOneSided
	BearoffTwoSided
	BearoffHypergammon
)

// Database represents a bearoff database
type Database struct {
	Type       BearoffType
	NPoints    int  // Number of points covered
	NChequers  int  // Number of checkers (for one-sided)
	Compressed bool // Is the database compressed?
	HasGammon  bool // Includes gammon probabilities?
	ND         bool // Uses normal distribution approximation?
	Cubeful    bool // Includes cubeful equities? (two-sided only)

	data     []byte // Memory-mapped or loaded database content
	filename string
}

// LoadOneSided loads a one-sided bearoff database from disk
func LoadOneSided(filename string) (*Database, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open bearoff database: %w", err)
	}
	defer f.Close()

	// Read entire file into memory
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read bearoff database: %w", err)
	}

	if len(data) < 40 {
		return nil, fmt.Errorf("bearoff database too small: %d bytes", len(data))
	}

	// Parse header
	header := string(data[:40])
	if header[:5] != "gnubg" {
		return nil, fmt.Errorf("not a gnubg bearoff database")
	}

	db := &Database{
		data:     data,
		filename: filename,
	}

	// Parse database type
	dbType := header[6:8]
	switch dbType {
	case "OS":
		db.Type = BearoffOneSided
	case "TS":
		db.Type = BearoffTwoSided
	default:
		if header[6] == 'H' {
			db.Type = BearoffHypergammon
		} else {
			return nil, fmt.Errorf("unknown bearoff database type: %s", dbType)
		}
	}

	// Parse points and checkers (format: XX-YY where XX=points, YY=checkers)
	if _, err := fmt.Sscanf(header[9:14], "%02d-%02d", &db.NPoints, &db.NChequers); err != nil {
		return nil, fmt.Errorf("failed to parse points/checkers: %w", err)
	}

	// Parse options for one-sided database
	if db.Type == BearoffOneSided {
		db.HasGammon = header[15] == '1'
		db.Compressed = header[17] == '1'
		db.ND = header[19] == '1'
	} else if db.Type == BearoffTwoSided {
		db.Cubeful = header[15] == '1'
	}

	return db, nil
}

// NumPositions returns the number of positions in the database
func (db *Database) NumPositions() int {
	return Combination(db.NPoints+db.NChequers, db.NPoints)
}

// Evaluate evaluates a bearoff position and returns win/gammon probabilities
// anBoard should be [2][6]uint8 representing checkers on points 1-6 for each player
func (db *Database) Evaluate(board [2][6]uint8) (output [5]float32, err error) {
	if db.Type == BearoffOneSided {
		return db.evaluateOneSided(board)
	} else if db.Type == BearoffTwoSided {
		return db.evaluateTwoSided(board)
	}
	return output, fmt.Errorf("unsupported bearoff database type")
}

// evaluateOneSided evaluates using one-sided database
func (db *Database) evaluateOneSided(board [2][6]uint8) (output [5]float32, err error) {
	// Convert board arrays to slices for PositionBearoff
	us := boardToSlice(board[1])
	them := boardToSlice(board[0])

	posUs := PositionBearoff(us, db.NPoints, db.NChequers)
	posThem := PositionBearoff(them, db.NPoints, db.NChequers)

	// Get probability distributions
	probUs, _, err := db.GetDistribution(posUs)
	if err != nil {
		return output, err
	}
	probThem, _, err := db.GetDistribution(posThem)
	if err != nil {
		return output, err
	}

	// Calculate winning probability
	// P(win) = sum over i of P(us finishes in i) * P(them finishes in >= i)
	var winProb float32
	for i := 0; i < 32; i++ {
		for j := i; j < 32; j++ {
			winProb += probUs[i] * probThem[j]
		}
	}
	output[0] = winProb

	// Gammon probabilities are 0 for pure bearoff (no backgammons possible)
	return output, nil
}

// evaluateTwoSided evaluates using two-sided database
func (db *Database) evaluateTwoSided(board [2][6]uint8) (output [5]float32, err error) {
	us := boardToSlice(board[1])
	them := boardToSlice(board[0])

	posUs := PositionBearoff(us, db.NPoints, db.NChequers)
	posThem := PositionBearoff(them, db.NPoints, db.NChequers)

	n := db.NumPositions()
	iPos := posUs*n + posThem

	equity, err := db.readTwoSidedEquity(iPos)
	if err != nil {
		return output, err
	}

	output[0] = equity/2.0 + 0.5
	return output, nil
}

// GetDistribution returns the probability distribution for a position
// Returns probabilities of bearing off in 0-31 rolls
func (db *Database) GetDistribution(posID int) (prob [32]float32, gammonProb [32]float32, err error) {
	if db.ND {
		return db.getDistributionND(posID)
	}
	if db.Compressed {
		return db.getDistributionCompressed(posID)
	}
	return db.getDistributionUncompressed(posID)
}

// getDistributionND reads distribution using normal distribution approximation
func (db *Database) getDistributionND(posID int) (prob [32]float32, gammonProb [32]float32, err error) {
	offset := 40 + posID*16
	if offset+16 > len(db.data) {
		return prob, gammonProb, fmt.Errorf("position %d out of range", posID)
	}

	// Read 4 floats: mean, stddev, gammon_mean, gammon_stddev
	mean := math.Float32frombits(uint32(db.data[offset]) | uint32(db.data[offset+1])<<8 |
		uint32(db.data[offset+2])<<16 | uint32(db.data[offset+3])<<24)
	stddev := math.Float32frombits(uint32(db.data[offset+4]) | uint32(db.data[offset+5])<<8 |
		uint32(db.data[offset+6])<<16 | uint32(db.data[offset+7])<<24)
	gammonMean := math.Float32frombits(uint32(db.data[offset+8]) | uint32(db.data[offset+9])<<8 |
		uint32(db.data[offset+10])<<16 | uint32(db.data[offset+11])<<24)
	gammonStddev := math.Float32frombits(uint32(db.data[offset+12]) | uint32(db.data[offset+13])<<8 |
		uint32(db.data[offset+14])<<16 | uint32(db.data[offset+15])<<24)

	for i := 0; i < 32; i++ {
		prob[i] = normalDist(float32(i), mean, stddev)
		gammonProb[i] = normalDist(float32(i), gammonMean, gammonStddev)
	}

	return prob, gammonProb, nil
}

// getDistributionCompressed reads compressed distribution
func (db *Database) getDistributionCompressed(posID int) (prob [32]float32, gammonProb [32]float32, err error) {
	nPos := db.NumPositions()
	indexEntrySize := 6
	if db.HasGammon {
		indexEntrySize = 8
	}

	// Read index entry
	indexOffset := 40 + posID*indexEntrySize
	if indexOffset+indexEntrySize > len(db.data) {
		return prob, gammonProb, fmt.Errorf("position %d out of range", posID)
	}

	// Parse index entry
	dataOffset := int(db.data[indexOffset]) | int(db.data[indexOffset+1])<<8 |
		int(db.data[indexOffset+2])<<16 | int(db.data[indexOffset+3])<<24
	nz := int(db.data[indexOffset+4])
	ioff := int(db.data[indexOffset+5])

	var nzg, ioffg int
	if db.HasGammon {
		nzg = int(db.data[indexOffset+6])
		ioffg = int(db.data[indexOffset+7])
	}

	// Calculate actual data offset
	actualOffset := 40 + nPos*indexEntrySize + 2*dataOffset
	nBytes := 2 * (nz + nzg)

	if actualOffset+nBytes > len(db.data) {
		return prob, gammonProb, fmt.Errorf("data offset out of range for position %d", posID)
	}

	// Read probability values
	for i := 0; i < nz; i++ {
		idx := actualOffset + 2*i
		val := uint16(db.data[idx]) | uint16(db.data[idx+1])<<8
		prob[ioff+i] = float32(val) / 65535.0
	}

	// Read gammon probability values
	for i := 0; i < nzg; i++ {
		idx := actualOffset + 2*nz + 2*i
		val := uint16(db.data[idx]) | uint16(db.data[idx+1])<<8
		gammonProb[ioffg+i] = float32(val) / 65535.0
	}

	return prob, gammonProb, nil
}

// getDistributionUncompressed reads uncompressed distribution
func (db *Database) getDistributionUncompressed(posID int) (prob [32]float32, gammonProb [32]float32, err error) {
	recordSize := 64
	if db.HasGammon {
		recordSize = 128
	}

	offset := 40 + posID*recordSize
	if offset+recordSize > len(db.data) {
		return prob, gammonProb, fmt.Errorf("position %d out of range", posID)
	}

	// Read probability values
	for i := 0; i < 32; i++ {
		idx := offset + 2*i
		val := uint16(db.data[idx]) | uint16(db.data[idx+1])<<8
		prob[i] = float32(val) / 65535.0
	}

	// Read gammon probability values
	if db.HasGammon {
		for i := 0; i < 32; i++ {
			idx := offset + 64 + 2*i
			val := uint16(db.data[idx]) | uint16(db.data[idx+1])<<8
			gammonProb[i] = float32(val) / 65535.0
		}
	}

	return prob, gammonProb, nil
}

// readTwoSidedEquity reads equity from two-sided database
// Returns equity in range [-1, 1] where 1 = certain win, -1 = certain loss
func (db *Database) readTwoSidedEquity(posID int) (float32, error) {
	recordSize := 2
	if db.Cubeful {
		recordSize = 8
	}

	offset := 40 + posID*recordSize
	if offset+recordSize > len(db.data) {
		return 0, fmt.Errorf("position %d out of range", posID)
	}

	// gnubg stores as unsigned short, converts with: us / 32767.5f - 1.0f
	val := uint16(db.data[offset]) | uint16(db.data[offset+1])<<8
	return float32(val)/32767.5 - 1.0, nil
}

// boardToSlice converts a [6]uint8 to []uint8
func boardToSlice(board [6]uint8) []uint8 {
	return board[:]
}

// normalDist calculates normal distribution probability
func normalDist(x, mu, sigma float32) float32 {
	const epsilon = 1e-7
	if sigma <= epsilon {
		if float32(math.Abs(float64(mu-x))) < epsilon {
			return 1.0
		}
		return 0.0
	}
	xm := (x - mu) / sigma
	return float32(1.0 / (float64(sigma) * math.Sqrt(2*math.Pi)) * math.Exp(float64(-xm*xm/2)))
}

// AverageRolls calculates the mean and standard deviation of rolls to bear off
// from a probability distribution
func AverageRolls(prob [32]float32) (mean, stddev float32) {
	var sx, sx2 float32
	for i := 1; i < 32; i++ {
		p := float32(i) * prob[i]
		sx += p
		sx2 += float32(i) * p
	}
	mean = sx
	variance := sx2 - sx*sx
	if variance > 0 {
		stddev = float32(math.Sqrt(float64(variance)))
	}
	return mean, stddev
}

// GetAverageRolls returns the average rolls to bear off for a position
func (db *Database) GetAverageRolls(posID int) (mean, stddev float32, err error) {
	prob, _, err := db.GetDistribution(posID)
	if err != nil {
		return 0, 0, err
	}
	mean, stddev = AverageRolls(prob)
	return mean, stddev, nil
}
