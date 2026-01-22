// Package grid handles grid generation and mine placement for PodSweeper.
package grid

import (
	"fmt"
	"math/rand"

	"github.com/zwindler/podsweeper/pkg/game"
)

// DefaultSize is the default grid dimension.
const DefaultSize = 10

// DefaultMineDensity is the default percentage of cells that are mines.
const DefaultMineDensity = 0.15 // 15%

// MinMineDensity is the minimum allowed mine density.
const MinMineDensity = 0.05 // 5%

// MaxMineDensity is the maximum allowed mine density.
const MaxMineDensity = 0.50 // 50%

// Config holds the configuration for grid generation.
type Config struct {
	// Size is the grid dimension (Size x Size).
	// Default: 10
	Size int

	// Seed is the random seed for reproducible mine placement.
	// If 0, a random seed will be used.
	Seed int64

	// MineDensity is the percentage of cells that should be mines (0.0 to 1.0).
	// Default: 0.15 (15%)
	MineDensity float64

	// MinMineCount is the minimum number of mines regardless of density.
	// Default: 1
	MinMineCount int

	// MaxMineCount is the maximum number of mines regardless of density.
	// If 0, no maximum is enforced.
	MaxMineCount int
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() Config {
	return Config{
		Size:         DefaultSize,
		Seed:         0,
		MineDensity:  DefaultMineDensity,
		MinMineCount: 1,
		MaxMineCount: 0,
	}
}

// Validate checks if the config values are valid and returns an error if not.
func (c *Config) Validate() error {
	if c.Size < 1 {
		return fmt.Errorf("size must be at least 1, got %d", c.Size)
	}
	if c.Size > 100 {
		return fmt.Errorf("size must be at most 100, got %d", c.Size)
	}
	if c.MineDensity < MinMineDensity {
		return fmt.Errorf("mine density must be at least %.2f, got %.2f", MinMineDensity, c.MineDensity)
	}
	if c.MineDensity > MaxMineDensity {
		return fmt.Errorf("mine density must be at most %.2f, got %.2f", MaxMineDensity, c.MineDensity)
	}
	if c.MinMineCount < 0 {
		return fmt.Errorf("min mine count cannot be negative, got %d", c.MinMineCount)
	}
	if c.MaxMineCount < 0 {
		return fmt.Errorf("max mine count cannot be negative, got %d", c.MaxMineCount)
	}
	if c.MaxMineCount > 0 && c.MinMineCount > c.MaxMineCount {
		return fmt.Errorf("min mine count (%d) cannot exceed max mine count (%d)", c.MinMineCount, c.MaxMineCount)
	}
	return nil
}

// CalculateMineCount returns the number of mines based on config.
func (c *Config) CalculateMineCount() int {
	totalCells := c.Size * c.Size
	mineCount := int(float64(totalCells) * c.MineDensity)

	// Enforce minimum
	if mineCount < c.MinMineCount {
		mineCount = c.MinMineCount
	}

	// Enforce maximum
	if c.MaxMineCount > 0 && mineCount > c.MaxMineCount {
		mineCount = c.MaxMineCount
	}

	// Cannot exceed total cells - 1 (need at least one safe cell)
	maxPossible := totalCells - 1
	if mineCount > maxPossible {
		mineCount = maxPossible
	}

	return mineCount
}

// Generator creates game grids with randomly placed mines.
type Generator struct {
	config Config
	rng    *rand.Rand
}

// NewGenerator creates a new grid generator with the given config.
func NewGenerator(config Config) (*Generator, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Use provided seed or generate one
	seed := config.Seed
	if seed == 0 {
		seed = rand.Int63()
	}

	return &Generator{
		config: config,
		rng:    rand.New(rand.NewSource(seed)),
	}, nil
}

// NewDefaultGenerator creates a generator with default configuration.
func NewDefaultGenerator() *Generator {
	gen, _ := NewGenerator(DefaultConfig())
	return gen
}

// Generate creates a new GameState with mines randomly placed.
func (g *Generator) Generate() *game.GameState {
	state := game.NewGameState(g.config.Size, g.config.Seed)
	g.placeMines(state)
	return state
}

// GenerateWithSeed creates a new GameState using a specific seed.
// This is useful for reproducible game generation.
func (g *Generator) GenerateWithSeed(seed int64) *game.GameState {
	// Create a new RNG with the specific seed
	rng := rand.New(rand.NewSource(seed))
	state := game.NewGameState(g.config.Size, seed)
	g.placeMinesWithRNG(state, rng)
	return state
}

// placeMines randomly places mines on the grid using the generator's RNG.
func (g *Generator) placeMines(state *game.GameState) {
	g.placeMinesWithRNG(state, g.rng)
}

// placeMinesWithRNG places mines using a specific RNG instance.
func (g *Generator) placeMinesWithRNG(state *game.GameState, rng *rand.Rand) {
	mineCount := g.config.CalculateMineCount()
	totalCells := g.config.Size * g.config.Size

	// Create a slice of all possible positions
	positions := make([]int, totalCells)
	for i := 0; i < totalCells; i++ {
		positions[i] = i
	}

	// Fisher-Yates shuffle
	for i := len(positions) - 1; i > 0; i-- {
		j := rng.Intn(i + 1)
		positions[i], positions[j] = positions[j], positions[i]
	}

	// Place mines at the first mineCount positions
	for i := 0; i < mineCount; i++ {
		pos := positions[i]
		x := pos / g.config.Size
		y := pos % g.config.Size
		state.SetMine(x, y)
	}
}

// Config returns the generator's configuration.
func (g *Generator) Config() Config {
	return g.config
}

// GenerateGrid is a convenience function that creates a game with default settings.
func GenerateGrid(size int, seed int64, density float64) (*game.GameState, error) {
	config := Config{
		Size:         size,
		Seed:         seed,
		MineDensity:  density,
		MinMineCount: 1,
		MaxMineCount: 0,
	}

	gen, err := NewGenerator(config)
	if err != nil {
		return nil, err
	}

	return gen.GenerateWithSeed(seed), nil
}

// GenerateDefaultGrid creates a 10x10 grid with 15% mine density.
func GenerateDefaultGrid(seed int64) *game.GameState {
	state, _ := GenerateGrid(DefaultSize, seed, DefaultMineDensity)
	return state
}

// DifficultyPreset represents predefined difficulty levels.
type DifficultyPreset string

const (
	// DifficultyEasy is 8x8 with 10% mines (6-7 mines).
	DifficultyEasy DifficultyPreset = "easy"
	// DifficultyMedium is 10x10 with 15% mines (15 mines).
	DifficultyMedium DifficultyPreset = "medium"
	// DifficultyHard is 16x16 with 20% mines (51 mines).
	DifficultyHard DifficultyPreset = "hard"
	// DifficultyExpert is 20x20 with 25% mines (100 mines).
	DifficultyExpert DifficultyPreset = "expert"
)

// GetDifficultyConfig returns a Config for the given difficulty preset.
func GetDifficultyConfig(preset DifficultyPreset) Config {
	switch preset {
	case DifficultyEasy:
		return Config{
			Size:         8,
			MineDensity:  0.10,
			MinMineCount: 5,
			MaxMineCount: 10,
		}
	case DifficultyMedium:
		return Config{
			Size:         10,
			MineDensity:  0.15,
			MinMineCount: 10,
			MaxMineCount: 20,
		}
	case DifficultyHard:
		return Config{
			Size:         16,
			MineDensity:  0.20,
			MinMineCount: 40,
			MaxMineCount: 60,
		}
	case DifficultyExpert:
		return Config{
			Size:         20,
			MineDensity:  0.25,
			MinMineCount: 80,
			MaxMineCount: 120,
		}
	default:
		return DefaultConfig()
	}
}

// GenerateWithDifficulty creates a game grid with the specified difficulty.
func GenerateWithDifficulty(preset DifficultyPreset, seed int64) (*game.GameState, error) {
	config := GetDifficultyConfig(preset)
	config.Seed = seed

	gen, err := NewGenerator(config)
	if err != nil {
		return nil, err
	}

	return gen.GenerateWithSeed(seed), nil
}
