package game

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewGameState(t *testing.T) {
	size := 10
	seed := int64(12345)

	state := NewGameState(size, seed)

	if state.Size != size {
		t.Errorf("expected size %d, got %d", size, state.Size)
	}
	if state.Seed != seed {
		t.Errorf("expected seed %d, got %d", seed, state.Seed)
	}
	if state.Level != 0 {
		t.Errorf("expected level 0, got %d", state.Level)
	}
	if state.Status != StatusPlaying {
		t.Errorf("expected status %s, got %s", StatusPlaying, state.Status)
	}
	if state.MineCount != 0 {
		t.Errorf("expected mine count 0, got %d", state.MineCount)
	}
	if len(state.MineMap) != size {
		t.Errorf("expected MineMap length %d, got %d", size, len(state.MineMap))
	}
	if len(state.Revealed) != size {
		t.Errorf("expected Revealed length %d, got %d", size, len(state.Revealed))
	}
	if state.StartedAt.IsZero() {
		t.Error("expected StartedAt to be set")
	}
}

func TestCoordinate(t *testing.T) {
	c := Coordinate{X: 3, Y: 5}

	if c.String() != "(3,5)" {
		t.Errorf("expected (3,5), got %s", c.String())
	}
	if c.PodName() != "pod-3-5" {
		t.Errorf("expected pod-3-5, got %s", c.PodName())
	}
	if c.HintPodName() != "hint-3-5" {
		t.Errorf("expected hint-3-5, got %s", c.HintPodName())
	}
}

func TestIsValidCoordinate(t *testing.T) {
	state := NewGameState(10, 0)

	tests := []struct {
		x, y     int
		expected bool
	}{
		{0, 0, true},
		{9, 9, true},
		{5, 5, true},
		{-1, 0, false},
		{0, -1, false},
		{10, 0, false},
		{0, 10, false},
		{-1, -1, false},
		{10, 10, false},
	}

	for _, tt := range tests {
		result := state.IsValidCoordinate(tt.x, tt.y)
		if result != tt.expected {
			t.Errorf("IsValidCoordinate(%d, %d) = %v, expected %v", tt.x, tt.y, result, tt.expected)
		}
	}
}

func TestSetMineAndIsMine(t *testing.T) {
	state := NewGameState(10, 0)

	// Set a mine
	if !state.SetMine(3, 5) {
		t.Error("SetMine should return true for valid coordinate")
	}
	if state.MineCount != 1 {
		t.Errorf("expected mine count 1, got %d", state.MineCount)
	}
	if !state.IsMine(3, 5) {
		t.Error("IsMine should return true for cell with mine")
	}

	// Set same mine again (should not increment count)
	state.SetMine(3, 5)
	if state.MineCount != 1 {
		t.Errorf("expected mine count 1 after duplicate set, got %d", state.MineCount)
	}

	// Check cell without mine
	if state.IsMine(0, 0) {
		t.Error("IsMine should return false for cell without mine")
	}

	// Check out of bounds
	if state.SetMine(-1, 0) {
		t.Error("SetMine should return false for invalid coordinate")
	}
	if state.IsMine(-1, 0) {
		t.Error("IsMine should return false for invalid coordinate")
	}
}

func TestRevealAndIsRevealed(t *testing.T) {
	state := NewGameState(10, 0)

	// Initially not revealed
	if state.IsRevealed(3, 5) {
		t.Error("cell should not be revealed initially")
	}

	// Reveal a cell
	if !state.Reveal(3, 5) {
		t.Error("Reveal should return true for valid unrevealed cell")
	}
	if state.Clicks != 1 {
		t.Errorf("expected clicks 1, got %d", state.Clicks)
	}
	if !state.IsRevealed(3, 5) {
		t.Error("cell should be revealed after Reveal")
	}

	// Reveal same cell again (should return false)
	if state.Reveal(3, 5) {
		t.Error("Reveal should return false for already revealed cell")
	}
	if state.Clicks != 1 {
		t.Errorf("clicks should not increment on duplicate reveal, got %d", state.Clicks)
	}

	// Check out of bounds
	if state.Reveal(-1, 0) {
		t.Error("Reveal should return false for invalid coordinate")
	}
	if state.IsRevealed(-1, 0) {
		t.Error("IsRevealed should return false for invalid coordinate")
	}
}

func TestAdjacentMines(t *testing.T) {
	state := NewGameState(5, 0)

	// Place mines around (2,2)
	// . M .
	// M . M
	// . M .
	state.SetMine(2, 1) // top
	state.SetMine(1, 2) // left
	state.SetMine(3, 2) // right
	state.SetMine(2, 3) // bottom

	if count := state.AdjacentMines(2, 2); count != 4 {
		t.Errorf("expected 4 adjacent mines, got %d", count)
	}

	// Corner case
	state.SetMine(0, 1)
	if count := state.AdjacentMines(0, 0); count != 1 {
		t.Errorf("expected 1 adjacent mine for corner, got %d", count)
	}

	// Cell with no adjacent mines
	if count := state.AdjacentMines(4, 4); count != 0 {
		t.Errorf("expected 0 adjacent mines, got %d", count)
	}
}

func TestGetNeighbors(t *testing.T) {
	state := NewGameState(5, 0)

	// Center cell should have 8 neighbors
	neighbors := state.GetNeighbors(2, 2)
	if len(neighbors) != 8 {
		t.Errorf("center cell should have 8 neighbors, got %d", len(neighbors))
	}

	// Corner cell should have 3 neighbors
	neighbors = state.GetNeighbors(0, 0)
	if len(neighbors) != 3 {
		t.Errorf("corner cell should have 3 neighbors, got %d", len(neighbors))
	}

	// Edge cell should have 5 neighbors
	neighbors = state.GetNeighbors(0, 2)
	if len(neighbors) != 5 {
		t.Errorf("edge cell should have 5 neighbors, got %d", len(neighbors))
	}
}

func TestUnrevealedSafeCells(t *testing.T) {
	state := NewGameState(3, 0)
	// 3x3 = 9 cells

	// No mines, no reveals
	if count := state.UnrevealedSafeCells(); count != 9 {
		t.Errorf("expected 9 unrevealed safe cells, got %d", count)
	}

	// Add a mine
	state.SetMine(0, 0)
	if count := state.UnrevealedSafeCells(); count != 8 {
		t.Errorf("expected 8 unrevealed safe cells, got %d", count)
	}

	// Reveal a safe cell
	state.Reveal(1, 1)
	if count := state.UnrevealedSafeCells(); count != 7 {
		t.Errorf("expected 7 unrevealed safe cells, got %d", count)
	}
}

func TestCheckVictory(t *testing.T) {
	state := NewGameState(2, 0)
	// 2x2 grid with 1 mine

	state.SetMine(0, 0)

	// Not yet won
	if state.CheckVictory() {
		t.Error("should not be victory yet")
	}

	// Reveal all safe cells
	state.Reveal(0, 1)
	state.Reveal(1, 0)
	state.Reveal(1, 1)

	if !state.CheckVictory() {
		t.Error("should be victory after revealing all safe cells")
	}
}

func TestSetWonAndSetLost(t *testing.T) {
	state := NewGameState(10, 0)

	state.SetWon()
	if state.Status != StatusWon {
		t.Errorf("expected status %s, got %s", StatusWon, state.Status)
	}
	if state.EndedAt.IsZero() {
		t.Error("EndedAt should be set after SetWon")
	}

	state2 := NewGameState(10, 0)
	state2.SetLost()
	if state2.Status != StatusLost {
		t.Errorf("expected status %s, got %s", StatusLost, state2.Status)
	}
	if state2.EndedAt.IsZero() {
		t.Error("EndedAt should be set after SetLost")
	}
}

func TestAddHintCell(t *testing.T) {
	state := NewGameState(10, 0)

	state.AddHintCell(3, 5)
	state.AddHintCell(4, 6)

	if len(state.HintCells) != 2 {
		t.Errorf("expected 2 hint cells, got %d", len(state.HintCells))
	}
	if state.HintCells[0].X != 3 || state.HintCells[0].Y != 5 {
		t.Errorf("first hint cell should be (3,5), got %v", state.HintCells[0])
	}
}

func TestJSONSerialization(t *testing.T) {
	state := NewGameState(5, 12345)
	state.Level = 3
	state.SetMine(1, 2)
	state.SetMine(3, 4)
	state.Reveal(0, 0)
	state.AddHintCell(0, 1)

	// Serialize to JSON
	data, err := state.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// Deserialize
	restored, err := FromJSON(data)
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	// Verify fields
	if restored.Size != state.Size {
		t.Errorf("Size mismatch: expected %d, got %d", state.Size, restored.Size)
	}
	if restored.Seed != state.Seed {
		t.Errorf("Seed mismatch: expected %d, got %d", state.Seed, restored.Seed)
	}
	if restored.Level != state.Level {
		t.Errorf("Level mismatch: expected %d, got %d", state.Level, restored.Level)
	}
	if restored.MineCount != state.MineCount {
		t.Errorf("MineCount mismatch: expected %d, got %d", state.MineCount, restored.MineCount)
	}
	if !restored.IsMine(1, 2) || !restored.IsMine(3, 4) {
		t.Error("mines not preserved after serialization")
	}
	if !restored.IsRevealed(0, 0) {
		t.Error("revealed cells not preserved after serialization")
	}
	if len(restored.HintCells) != 1 {
		t.Errorf("HintCells not preserved: expected 1, got %d", len(restored.HintCells))
	}
}

func TestJSONPretty(t *testing.T) {
	state := NewGameState(3, 0)

	data, err := state.ToJSONPretty()
	if err != nil {
		t.Fatalf("ToJSONPretty failed: %v", err)
	}

	// Should contain newlines (formatted)
	if len(data) < 50 {
		t.Error("pretty JSON should be longer than compact")
	}

	// Should still be valid JSON
	var check map[string]interface{}
	if err := json.Unmarshal(data, &check); err != nil {
		t.Errorf("pretty JSON is not valid: %v", err)
	}
}

func TestFromJSONInvalid(t *testing.T) {
	_, err := FromJSON([]byte("invalid json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}

	_, err = FromJSON([]byte("{}"))
	if err != nil {
		t.Error("empty object should parse without error")
	}
}

func TestClone(t *testing.T) {
	state := NewGameState(5, 12345)
	state.Level = 3
	state.SetMine(1, 2)
	state.Reveal(0, 0)
	state.AddHintCell(0, 1)

	clone := state.Clone()

	// Verify clone has same values
	if clone.Size != state.Size || clone.Seed != state.Seed || clone.Level != state.Level {
		t.Error("clone basic fields don't match")
	}
	if !clone.IsMine(1, 2) {
		t.Error("clone should have same mines")
	}
	if !clone.IsRevealed(0, 0) {
		t.Error("clone should have same revealed cells")
	}

	// Modify original, verify clone is independent
	state.SetMine(4, 4)
	state.Reveal(1, 1)
	state.Level = 9

	if clone.IsMine(4, 4) {
		t.Error("clone should be independent - mine change affected clone")
	}
	if clone.IsRevealed(1, 1) {
		t.Error("clone should be independent - reveal change affected clone")
	}
	if clone.Level == 9 {
		t.Error("clone should be independent - level change affected clone")
	}
}

func TestStats(t *testing.T) {
	state := NewGameState(5, 0)
	state.SetMine(0, 0)
	state.SetMine(1, 1)
	state.Reveal(2, 2)
	state.Reveal(3, 3)
	state.AddHintCell(2, 2)

	stats := state.Stats()

	if stats["size"] != 5 {
		t.Errorf("stats size mismatch")
	}
	if stats["mines"] != 2 {
		t.Errorf("stats mines mismatch")
	}
	if stats["totalCells"] != 25 {
		t.Errorf("stats totalCells mismatch")
	}
	if stats["revealedCells"] != 2 {
		t.Errorf("stats revealedCells mismatch")
	}
	if stats["clicks"] != 2 {
		t.Errorf("stats clicks mismatch")
	}
	if stats["hintPodsPlaced"] != 1 {
		t.Errorf("stats hintPodsPlaced mismatch")
	}
}

func TestGameStateZeroValues(t *testing.T) {
	// Test with size 0 (edge case)
	state := NewGameState(0, 0)

	if state.Size != 0 {
		t.Errorf("expected size 0, got %d", state.Size)
	}
	if len(state.MineMap) != 0 {
		t.Errorf("expected empty MineMap, got length %d", len(state.MineMap))
	}

	// These should not panic
	if state.IsValidCoordinate(0, 0) {
		t.Error("(0,0) should be invalid for size 0 grid")
	}
	if state.IsMine(0, 0) {
		t.Error("IsMine should return false for invalid coordinate")
	}
	if state.Reveal(0, 0) {
		t.Error("Reveal should return false for invalid coordinate")
	}
}

func TestGameStateLargeGrid(t *testing.T) {
	// Test with large grid
	state := NewGameState(100, 0)

	if state.Size != 100 {
		t.Errorf("expected size 100, got %d", state.Size)
	}

	// Set mines in corners
	state.SetMine(0, 0)
	state.SetMine(99, 99)
	state.SetMine(0, 99)
	state.SetMine(99, 0)

	if state.MineCount != 4 {
		t.Errorf("expected 4 mines, got %d", state.MineCount)
	}

	// Verify serialization works for large grids
	data, err := state.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed for large grid: %v", err)
	}

	restored, err := FromJSON(data)
	if err != nil {
		t.Fatalf("FromJSON failed for large grid: %v", err)
	}

	if restored.MineCount != 4 {
		t.Errorf("mine count not preserved for large grid")
	}
}

func TestTimeFields(t *testing.T) {
	before := time.Now()
	state := NewGameState(5, 0)
	after := time.Now()

	if state.StartedAt.Before(before) || state.StartedAt.After(after) {
		t.Error("StartedAt should be set to current time")
	}

	if !state.EndedAt.IsZero() {
		t.Error("EndedAt should be zero for new game")
	}

	beforeEnd := time.Now()
	state.SetWon()
	afterEnd := time.Now()

	if state.EndedAt.Before(beforeEnd) || state.EndedAt.After(afterEnd) {
		t.Error("EndedAt should be set when game ends")
	}
}
