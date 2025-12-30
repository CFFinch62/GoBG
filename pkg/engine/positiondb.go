// Package engine provides a position database for common backgammon scenarios.
package engine

import (
	"sort"
	"sync"

	"github.com/yourusername/bgengine/internal/positionid"
)

// PositionCategory represents the type of position.
type PositionCategory int

const (
	CategoryUnknown    PositionCategory = iota
	CategoryOpening                     // Opening positions (first few moves)
	CategoryBearoff                     // Pure bearoff positions
	CategoryContact                     // Contact positions with blots
	CategoryBackgame                    // Defensive backgame positions
	CategoryBlitz                       // Aggressive attack positions
	CategoryHolding                     // Holding game positions
	CategoryRace                        // Pure racing positions
	CategoryPriming                     // Priming game positions
	CategorySafetyPlay                  // Safety-focused positions
)

// String returns the human-readable name of the category.
func (c PositionCategory) String() string {
	return [...]string{
		"Unknown", "Opening", "Bearoff", "Contact", "Backgame",
		"Blitz", "Holding", "Race", "Priming", "Safety Play",
	}[c]
}

// PositionEntry represents a position in the database.
type PositionEntry struct {
	ID          string           `json:"id"`          // Position ID
	Name        string           `json:"name"`        // Human-readable name
	Category    PositionCategory `json:"category"`    // Position category
	Description string           `json:"description"` // Detailed description
	Board       Board            `json:"board"`       // Board position
	Tags        []string         `json:"tags"`        // Searchable tags

	// Pre-computed evaluation (optional)
	Evaluation *Evaluation `json:"evaluation,omitempty"`

	// Key concepts this position demonstrates
	Concepts []string `json:"concepts,omitempty"`

	// Difficulty level (1-5)
	Difficulty int `json:"difficulty"`
}

// PositionDB is an in-memory position database.
type PositionDB struct {
	positions  map[string]*PositionEntry
	byCategory map[PositionCategory][]*PositionEntry
	byTag      map[string][]*PositionEntry
	mu         sync.RWMutex
}

// NewPositionDB creates a new empty position database.
func NewPositionDB() *PositionDB {
	return &PositionDB{
		positions:  make(map[string]*PositionEntry),
		byCategory: make(map[PositionCategory][]*PositionEntry),
		byTag:      make(map[string][]*PositionEntry),
	}
}

// Add adds a position to the database.
func (db *PositionDB) Add(entry *PositionEntry) {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Generate ID if not set
	if entry.ID == "" {
		entry.ID = EncodePositionID(entry.Board)
	}

	db.positions[entry.ID] = entry

	// Index by category
	db.byCategory[entry.Category] = append(db.byCategory[entry.Category], entry)

	// Index by tags
	for _, tag := range entry.Tags {
		db.byTag[tag] = append(db.byTag[tag], entry)
	}
}

// Get retrieves a position by ID.
func (db *PositionDB) Get(id string) *PositionEntry {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.positions[id]
}

// GetByCategory returns all positions in a category.
func (db *PositionDB) GetByCategory(cat PositionCategory) []*PositionEntry {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.byCategory[cat]
}

// GetByTag returns all positions with a given tag.
func (db *PositionDB) GetByTag(tag string) []*PositionEntry {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.byTag[tag]
}

// Search finds positions matching a query.
func (db *PositionDB) Search(query string) []*PositionEntry {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var results []*PositionEntry
	for _, p := range db.positions {
		if matchesQuery(p, query) {
			results = append(results, p)
		}
	}
	return results
}

// Count returns the total number of positions.
func (db *PositionDB) Count() int {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return len(db.positions)
}

// All returns all positions.
func (db *PositionDB) All() []*PositionEntry {
	db.mu.RLock()
	defer db.mu.RUnlock()

	results := make([]*PositionEntry, 0, len(db.positions))
	for _, p := range db.positions {
		results = append(results, p)
	}
	return results
}

// matchesQuery checks if a position matches a search query.
func matchesQuery(p *PositionEntry, query string) bool {
	// Simple substring match on name, description, tags
	if containsIgnoreCase(p.Name, query) || containsIgnoreCase(p.Description, query) {
		return true
	}
	for _, tag := range p.Tags {
		if containsIgnoreCase(tag, query) {
			return true
		}
	}
	return false
}

// containsIgnoreCase checks if s contains substr (case-insensitive).
func containsIgnoreCase(s, substr string) bool {
	sLower := toLower(s)
	substrLower := toLower(substr)
	for i := 0; i <= len(sLower)-len(substrLower); i++ {
		if sLower[i:i+len(substrLower)] == substrLower {
			return true
		}
	}
	return false
}

// toLower converts a string to lowercase (ASCII only).
func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

// PositionSimilarity contains similarity information between two positions.
type PositionSimilarity struct {
	Entry      *PositionEntry
	Similarity float64 // 0.0 to 1.0
}

// FindSimilar finds positions similar to the given board.
func (db *PositionDB) FindSimilar(board Board, maxResults int) []PositionSimilarity {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var results []PositionSimilarity
	for _, p := range db.positions {
		sim := calculateBoardSimilarity(board, p.Board)
		if sim > 0.5 { // Threshold for "similar"
			results = append(results, PositionSimilarity{
				Entry:      p,
				Similarity: sim,
			})
		}
	}

	// Sort by similarity descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	if len(results) > maxResults {
		results = results[:maxResults]
	}
	return results
}

// calculateBoardSimilarity returns a similarity score (0-1) between two boards.
func calculateBoardSimilarity(a, b Board) float64 {
	// Count matching checkers per point
	matches := 0
	total := 0

	for player := 0; player < 2; player++ {
		for point := 0; point < 25; point++ {
			aCount := int(a[player][point])
			bCount := int(b[player][point])
			if aCount == bCount {
				matches += aCount
			} else {
				// Partial match for close counts
				minCount := aCount
				if bCount < minCount {
					minCount = bCount
				}
				matches += minCount
			}
			if aCount > bCount {
				total += aCount
			} else {
				total += bCount
			}
		}
	}

	if total == 0 {
		return 1.0
	}
	return float64(matches) / float64(total)
}

// ClassifyPosition determines the category of a position.
func ClassifyPosition(board Board) PositionCategory {
	// Count checkers and their positions
	var p0home, p0mid, p0outer, p0bar int
	var p1home, p1mid, p1outer, p1bar int

	for point := 0; point < 6; point++ {
		p0home += int(board[0][point])
		p1home += int(board[1][point])
	}
	for point := 6; point < 12; point++ {
		p0mid += int(board[0][point])
		p1mid += int(board[1][point])
	}
	for point := 12; point < 24; point++ {
		p0outer += int(board[0][point])
		p1outer += int(board[1][point])
	}
	p0bar = int(board[0][24])
	p1bar = int(board[1][24])

	// Pure bearoff: all checkers in home board
	if p0mid == 0 && p0outer == 0 && p0bar == 0 {
		return CategoryBearoff
	}

	// Race: no contact means all checkers have passed each other
	// In TanBoard, player 0's point N is player 1's point 23-N
	// Contact exists if any player has checkers that haven't passed opponent's checkers

	// Find the furthest point (toward opponent) that has checkers for each player
	p0Furthest := -1
	p1Furthest := -1
	for point := 23; point >= 0; point-- {
		if board[0][point] > 0 && p0Furthest == -1 {
			p0Furthest = point
		}
		if board[1][point] > 0 && p1Furthest == -1 {
			p1Furthest = point
		}
	}

	// Check for contact: if player 0's furthest point + player 1's furthest point >= 23,
	// then there's contact (they haven't fully passed each other)
	hasContact := false
	if p0Furthest >= 0 && p1Furthest >= 0 {
		// In TanBoard, p0's point X corresponds to p1's point 23-X
		// Contact if p0Furthest and p1Furthest haven't passed
		hasContact = (p0Furthest + p1Furthest) >= 23
	}
	if !hasContact && p0bar == 0 && p1bar == 0 {
		return CategoryRace
	}

	// Backgame: player has anchors in opponent's home board
	p0anchors := 0
	for point := 18; point < 24; point++ {
		if board[0][point] >= 2 {
			p0anchors++
		}
	}
	if p0anchors >= 2 && p0outer > 6 {
		return CategoryBackgame
	}

	// Holding: single strong anchor
	for point := 18; point < 24; point++ {
		if board[0][point] >= 2 && p0outer <= 4 {
			return CategoryHolding
		}
	}

	// Priming: 4+ consecutive points made
	maxPrime := 0
	currentPrime := 0
	for point := 0; point < 24; point++ {
		if board[0][point] >= 2 {
			currentPrime++
			if currentPrime > maxPrime {
				maxPrime = currentPrime
			}
		} else {
			currentPrime = 0
		}
	}
	if maxPrime >= 4 {
		return CategoryPriming
	}

	// Default to contact
	return CategoryContact
}

// CreatePositionEntry creates a position entry from a position ID.
func CreatePositionEntry(posID, name string, cat PositionCategory, desc string, tags []string) (*PositionEntry, error) {
	board, err := positionid.BoardFromPositionID(posID)
	if err != nil {
		return nil, err
	}

	return &PositionEntry{
		ID:          posID,
		Name:        name,
		Category:    cat,
		Description: desc,
		Board:       Board(board),
		Tags:        tags,
	}, nil
}

// DefaultPositionDB creates a database with common reference positions.
func DefaultPositionDB() *PositionDB {
	db := NewPositionDB()

	// Add starting position
	db.Add(&PositionEntry{
		ID:          "4HPwATDgc/ABMA",
		Name:        "Starting Position",
		Category:    CategoryOpening,
		Description: "Standard backgammon starting position",
		Board:       StartingPosition().Board,
		Tags:        []string{"opening", "standard", "initial"},
		Concepts:    []string{"game start", "opening theory"},
		Difficulty:  1,
	})

	// Common reference positions - these are well-known training positions
	referencePositions := []struct {
		id       string
		name     string
		cat      PositionCategory
		desc     string
		tags     []string
		concepts []string
		diff     int
	}{
		// Bearoff positions
		{"AQAABAAAAAAA", "Simple Bearoff 1", CategoryBearoff,
			"One checker to bear off", []string{"bearoff", "endgame"},
			[]string{"bearing off", "pip count"}, 1},

		// The following are placeholder IDs - they'll be set from actual positions
		// In a real implementation, we'd have actual position IDs from gnubg

		// Racing positions are common training scenarios
		// Holding game examples
		// Backgame examples
		// Blitz attack examples
	}

	for _, rp := range referencePositions {
		entry, err := CreatePositionEntry(rp.id, rp.name, rp.cat, rp.desc, rp.tags)
		if err != nil {
			continue // Skip invalid positions
		}
		entry.Concepts = rp.concepts
		entry.Difficulty = rp.diff
		db.Add(entry)
	}

	return db
}

// PrecomputeEvaluations adds evaluations to all positions in the database.
func (db *PositionDB) PrecomputeEvaluations(e *Engine) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, p := range db.positions {
		if p.Evaluation == nil {
			state := &GameState{
				Board:     p.Board,
				Turn:      0,
				CubeValue: 1,
				CubeOwner: -1,
			}
			eval, err := e.Evaluate(state)
			if err != nil {
				continue
			}
			p.Evaluation = eval
		}
	}
	return nil
}
