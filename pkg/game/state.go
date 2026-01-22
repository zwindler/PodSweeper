// Package game contains the core game logic and state management for PodSweeper.
package game

import (
	"encoding/json"
	"fmt"
	"time"
)

// GameStatus represents the current status of the game.
type GameStatus string

const (
	// StatusPlaying indicates the game is in progress.
	StatusPlaying GameStatus = "playing"
	// StatusWon indicates the player has won.
	StatusWon GameStatus = "won"
	// StatusLost indicates the player has lost (hit a mine).
	StatusLost GameStatus = "lost"
)

// Coordinate represents a position on the game grid.
type Coordinate struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// String returns a string representation of the coordinate.
func (c Coordinate) String() string {
	return fmt.Sprintf("(%d,%d)", c.X, c.Y)
}

// PodName returns the Kubernetes pod name for this coordinate.
func (c Coordinate) PodName() string {
	return fmt.Sprintf("pod-%d-%d", c.X, c.Y)
}

// HintPodName returns the hint pod name for this coordinate.
func (c Coordinate) HintPodName() string {
	return fmt.Sprintf("hint-%d-%d", c.X, c.Y)
}

// GameState holds the complete state of a PodSweeper game.
// This is serialized to JSON and stored in a Kubernetes Secret.
type GameState struct {
	// Size is the dimension of the grid (Size x Size).
	Size int `json:"size"`

	// Seed is the random seed used to generate the mine placement.
	// Using the same seed produces the same mine layout.
	Seed int64 `json:"seed"`

	// Level is the current difficulty/hardening level (0-9).
	Level int `json:"level"`

	// Status is the current game status (playing, won, lost).
	Status GameStatus `json:"status"`

	// MineMap is a 2D boolean array where true indicates a mine.
	// MineMap[x][y] corresponds to pod-x-y.
	MineMap [][]bool `json:"mineMap"`

	// Revealed is a 2D boolean array tracking which cells have been revealed.
	// Revealed[x][y] is true if the cell has been clicked/deleted.
	Revealed [][]bool `json:"revealed"`

	// HintCells tracks cells that have been converted to hint pods.
	// These are cells adjacent to mines that show a number.
	HintCells []Coordinate `json:"hintCells,omitempty"`

	// MineCount is the total number of mines on the grid.
	MineCount int `json:"mineCount"`

	// StartedAt is when the game was started.
	StartedAt time.Time `json:"startedAt"`

	// EndedAt is when the game ended (won or lost). Zero if still playing.
	EndedAt time.Time `json:"endedAt,omitempty"`

	// Clicks is the number of cells the player has clicked/deleted.
	Clicks int `json:"clicks"`
}

// NewGameState creates a new empty GameState with the given size.
// The MineMap and Revealed grids are initialized but empty (no mines placed).
// Use a grid generator to populate the MineMap.
func NewGameState(size int, seed int64) *GameState {
	mineMap := make([][]bool, size)
	revealed := make([][]bool, size)
	for i := 0; i < size; i++ {
		mineMap[i] = make([]bool, size)
		revealed[i] = make([]bool, size)
	}

	return &GameState{
		Size:      size,
		Seed:      seed,
		Level:     0,
		Status:    StatusPlaying,
		MineMap:   mineMap,
		Revealed:  revealed,
		HintCells: []Coordinate{},
		StartedAt: time.Now(),
	}
}

// IsValidCoordinate checks if the given coordinate is within the grid bounds.
func (g *GameState) IsValidCoordinate(x, y int) bool {
	return x >= 0 && x < g.Size && y >= 0 && y < g.Size
}

// IsMine checks if the cell at (x, y) contains a mine.
// Returns false if the coordinate is out of bounds.
func (g *GameState) IsMine(x, y int) bool {
	if !g.IsValidCoordinate(x, y) {
		return false
	}
	return g.MineMap[x][y]
}

// IsRevealed checks if the cell at (x, y) has been revealed.
// Returns false if the coordinate is out of bounds.
func (g *GameState) IsRevealed(x, y int) bool {
	if !g.IsValidCoordinate(x, y) {
		return false
	}
	return g.Revealed[x][y]
}

// Reveal marks the cell at (x, y) as revealed.
// Returns false if the coordinate is out of bounds or already revealed.
func (g *GameState) Reveal(x, y int) bool {
	if !g.IsValidCoordinate(x, y) || g.Revealed[x][y] {
		return false
	}
	g.Revealed[x][y] = true
	g.Clicks++
	return true
}

// SetMine places a mine at the given coordinate.
// Returns false if the coordinate is out of bounds.
func (g *GameState) SetMine(x, y int) bool {
	if !g.IsValidCoordinate(x, y) {
		return false
	}
	if !g.MineMap[x][y] {
		g.MineMap[x][y] = true
		g.MineCount++
	}
	return true
}

// AdjacentMines returns the count of mines adjacent to the cell at (x, y).
// This includes all 8 neighboring cells (diagonals included).
func (g *GameState) AdjacentMines(x, y int) int {
	count := 0
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			if dx == 0 && dy == 0 {
				continue // Skip the cell itself
			}
			if g.IsMine(x+dx, y+dy) {
				count++
			}
		}
	}
	return count
}

// GetNeighbors returns all valid neighboring coordinates for the cell at (x, y).
func (g *GameState) GetNeighbors(x, y int) []Coordinate {
	neighbors := make([]Coordinate, 0, 8)
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			if dx == 0 && dy == 0 {
				continue
			}
			nx, ny := x+dx, y+dy
			if g.IsValidCoordinate(nx, ny) {
				neighbors = append(neighbors, Coordinate{X: nx, Y: ny})
			}
		}
	}
	return neighbors
}

// UnrevealedSafeCells returns the count of cells that are not mines and not revealed.
func (g *GameState) UnrevealedSafeCells() int {
	count := 0
	for x := 0; x < g.Size; x++ {
		for y := 0; y < g.Size; y++ {
			if !g.MineMap[x][y] && !g.Revealed[x][y] {
				count++
			}
		}
	}
	return count
}

// CheckVictory checks if the player has won.
// Victory occurs when all non-mine cells have been revealed.
func (g *GameState) CheckVictory() bool {
	return g.UnrevealedSafeCells() == 0
}

// SetWon marks the game as won and records the end time.
func (g *GameState) SetWon() {
	g.Status = StatusWon
	g.EndedAt = time.Now()
}

// SetLost marks the game as lost and records the end time.
func (g *GameState) SetLost() {
	g.Status = StatusLost
	g.EndedAt = time.Now()
}

// AddHintCell records that a hint pod was created at the given coordinate.
func (g *GameState) AddHintCell(x, y int) {
	g.HintCells = append(g.HintCells, Coordinate{X: x, Y: y})
}

// ToJSON serializes the GameState to JSON bytes.
func (g *GameState) ToJSON() ([]byte, error) {
	return json.Marshal(g)
}

// ToJSONPretty serializes the GameState to indented JSON bytes.
func (g *GameState) ToJSONPretty() ([]byte, error) {
	return json.MarshalIndent(g, "", "  ")
}

// FromJSON deserializes a GameState from JSON bytes.
func FromJSON(data []byte) (*GameState, error) {
	var state GameState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal game state: %w", err)
	}
	return &state, nil
}

// Clone creates a deep copy of the GameState.
func (g *GameState) Clone() *GameState {
	clone := &GameState{
		Size:      g.Size,
		Seed:      g.Seed,
		Level:     g.Level,
		Status:    g.Status,
		MineCount: g.MineCount,
		StartedAt: g.StartedAt,
		EndedAt:   g.EndedAt,
		Clicks:    g.Clicks,
	}

	// Deep copy MineMap
	clone.MineMap = make([][]bool, g.Size)
	for i := 0; i < g.Size; i++ {
		clone.MineMap[i] = make([]bool, g.Size)
		copy(clone.MineMap[i], g.MineMap[i])
	}

	// Deep copy Revealed
	clone.Revealed = make([][]bool, g.Size)
	for i := 0; i < g.Size; i++ {
		clone.Revealed[i] = make([]bool, g.Size)
		copy(clone.Revealed[i], g.Revealed[i])
	}

	// Deep copy HintCells
	clone.HintCells = make([]Coordinate, len(g.HintCells))
	copy(clone.HintCells, g.HintCells)

	return clone
}

// Stats returns a summary of the current game state.
func (g *GameState) Stats() map[string]interface{} {
	totalCells := g.Size * g.Size
	revealedCount := 0
	for x := 0; x < g.Size; x++ {
		for y := 0; y < g.Size; y++ {
			if g.Revealed[x][y] {
				revealedCount++
			}
		}
	}

	return map[string]interface{}{
		"size":           g.Size,
		"level":          g.Level,
		"status":         g.Status,
		"mines":          g.MineCount,
		"totalCells":     totalCells,
		"revealedCells":  revealedCount,
		"remainingSafe":  g.UnrevealedSafeCells(),
		"clicks":         g.Clicks,
		"hintPodsPlaced": len(g.HintCells),
	}
}
