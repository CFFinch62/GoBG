package engine

import (
	"testing"
)

func TestPositionDB(t *testing.T) {
	db := NewPositionDB()
	
	// Add a position
	entry := &PositionEntry{
		ID:          "test123",
		Name:        "Test Position",
		Category:    CategoryContact,
		Description: "A test position",
		Board:       StartingPosition().Board,
		Tags:        []string{"test", "contact"},
	}
	db.Add(entry)
	
	if db.Count() != 1 {
		t.Errorf("Count() = %d, want 1", db.Count())
	}
	
	// Get by ID
	got := db.Get("test123")
	if got == nil {
		t.Fatal("Get() returned nil")
	}
	if got.Name != "Test Position" {
		t.Errorf("Name = %q, want %q", got.Name, "Test Position")
	}
	
	// Get by category
	catPositions := db.GetByCategory(CategoryContact)
	if len(catPositions) != 1 {
		t.Errorf("GetByCategory() returned %d positions, want 1", len(catPositions))
	}
	
	// Get by tag
	tagPositions := db.GetByTag("test")
	if len(tagPositions) != 1 {
		t.Errorf("GetByTag() returned %d positions, want 1", len(tagPositions))
	}
}

func TestPositionDBSearch(t *testing.T) {
	db := NewPositionDB()
	
	db.Add(&PositionEntry{
		ID:          "pos1",
		Name:        "Holding Game Example",
		Category:    CategoryHolding,
		Description: "Classic holding position",
		Board:       StartingPosition().Board,
		Tags:        []string{"holding", "anchor"},
	})
	
	db.Add(&PositionEntry{
		ID:          "pos2",
		Name:        "Backgame Strategy",
		Category:    CategoryBackgame,
		Description: "Two-point backgame",
		Board:       StartingPosition().Board,
		Tags:        []string{"backgame", "defense"},
	})
	
	// Search by name
	results := db.Search("holding")
	if len(results) != 1 {
		t.Errorf("Search('holding') returned %d, want 1", len(results))
	}
	
	// Search by description
	results = db.Search("backgame")
	if len(results) != 1 {
		t.Errorf("Search('backgame') returned %d, want 1", len(results))
	}
	
	// Search by tag
	results = db.Search("anchor")
	if len(results) != 1 {
		t.Errorf("Search('anchor') returned %d, want 1", len(results))
	}
}

func TestDefaultPositionDB(t *testing.T) {
	db := DefaultPositionDB()
	
	if db.Count() < 1 {
		t.Error("DefaultPositionDB() should have at least 1 position")
	}
	
	// Should have starting position
	startPos := db.Get("4HPwATDgc/ABMA")
	if startPos == nil {
		t.Error("Should have starting position")
	} else {
		if startPos.Category != CategoryOpening {
			t.Errorf("Starting position category = %v, want Opening", startPos.Category)
		}
	}
}

func TestClassifyPosition(t *testing.T) {
	// Starting position should be contact (has contact)
	startBoard := StartingPosition().Board
	cat := ClassifyPosition(startBoard)
	
	// Starting position has contact
	if cat != CategoryContact {
		t.Errorf("Starting position classified as %v, want Contact", cat)
	}
	
	// Create a pure bearoff position (all checkers in home board)
	bearoffBoard := Board{}
	bearoffBoard[0][0] = 3
	bearoffBoard[0][1] = 3
	bearoffBoard[0][2] = 3
	bearoffBoard[0][3] = 3
	bearoffBoard[0][4] = 2
	bearoffBoard[0][5] = 1
	
	cat = ClassifyPosition(bearoffBoard)
	if cat != CategoryBearoff {
		t.Errorf("Bearoff position classified as %v, want Bearoff", cat)
	}
}

func TestFindSimilar(t *testing.T) {
	db := DefaultPositionDB()
	
	// Find positions similar to starting
	startBoard := StartingPosition().Board
	similar := db.FindSimilar(startBoard, 5)
	
	// Should find the starting position itself with high similarity
	found := false
	for _, s := range similar {
		if s.Entry.ID == "4HPwATDgc/ABMA" && s.Similarity > 0.9 {
			found = true
			break
		}
	}
	
	if !found {
		t.Error("FindSimilar should find starting position with high similarity")
	}
}

func TestPositionCategoryString(t *testing.T) {
	tests := []struct {
		cat  PositionCategory
		want string
	}{
		{CategoryOpening, "Opening"},
		{CategoryBearoff, "Bearoff"},
		{CategoryBackgame, "Backgame"},
		{CategoryRace, "Race"},
	}
	
	for _, tt := range tests {
		if got := tt.cat.String(); got != tt.want {
			t.Errorf("%v.String() = %q, want %q", tt.cat, got, tt.want)
		}
	}
}

