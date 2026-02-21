// Package grid handles grid generation and mine placement for PodSweeper.
package grid

import (
	"fmt"
	"math/rand"

	"github.com/zwindler/podsweeper/pkg/game"
)

// Level constants define the valid range of game levels.
const (
	MinLevel = 0
	MaxLevel = 9
)

// Tier definitions - each tier has a specific grid size and mine count.
// Levels are grouped into 3 tiers for progressive difficulty.
const (
	// Tier 1 (Levels 0-3): Tutorial phase
	// Small grid, learn kubectl basics manually
	Tier1Size  = 5
	Tier1Mines = 4

	// Tier 2 (Levels 4-6): Intermediate phase
	// Medium grid, manual play becomes tedious, scripting helps
	Tier2Size  = 10
	Tier2Mines = 15

	// Tier 3 (Levels 7-9): Expert phase
	// Large grid, automation required to succeed
	Tier3Size  = 20
	Tier3Mines = 60
)

// LevelConfig holds the grid configuration for a specific level.
type LevelConfig struct {
	Level     int
	Size      int
	MineCount int
	Tier      int
}

// GetLevelConfig returns the grid configuration for a given level.
// Levels 0-3 use Tier 1, Levels 4-6 use Tier 2, Levels 7-9 use Tier 3.
func GetLevelConfig(level int) LevelConfig {
	// Clamp level to valid range
	if level < MinLevel {
		level = MinLevel
	}
	if level > MaxLevel {
		level = MaxLevel
	}

	switch {
	case level <= 3:
		return LevelConfig{
			Level:     level,
			Size:      Tier1Size,
			MineCount: Tier1Mines,
			Tier:      1,
		}
	case level <= 6:
		return LevelConfig{
			Level:     level,
			Size:      Tier2Size,
			MineCount: Tier2Mines,
			Tier:      2,
		}
	default:
		return LevelConfig{
			Level:     level,
			Size:      Tier3Size,
			MineCount: Tier3Mines,
			Tier:      3,
		}
	}
}

// GetTierDescription returns a human-readable description of the tier.
func GetTierDescription(tier int) string {
	switch tier {
	case 1:
		return "Tutorial (5×5, 4 mines) - Learn kubectl basics"
	case 2:
		return "Intermediate (10×10, 15 mines) - Scripting helps"
	case 3:
		return "Expert (20×20, 60 mines) - Automation required"
	default:
		return "Unknown tier"
	}
}

// Config holds the configuration for grid generation.
type Config struct {
	// Size is the grid dimension (Size x Size).
	Size int

	// Seed is the random seed for reproducible mine placement.
	// If 0, a random seed will be used.
	Seed int64

	// MineCount is the exact number of mines to place.
	MineCount int

	// Level is the game hardening level (0-9).
	Level int
}

// Validate checks if the config values are valid and returns an error if not.
func (c *Config) Validate() error {
	if c.Size < 1 {
		return fmt.Errorf("size must be at least 1, got %d", c.Size)
	}
	if c.Size > 100 {
		return fmt.Errorf("size must be at most 100, got %d", c.Size)
	}
	if c.MineCount < 1 {
		return fmt.Errorf("mine count must be at least 1, got %d", c.MineCount)
	}
	totalCells := c.Size * c.Size
	maxMines := totalCells - 1 // Need at least one safe cell
	if c.MineCount > maxMines {
		return fmt.Errorf("mine count (%d) exceeds maximum for grid size (%d)", c.MineCount, maxMines)
	}
	if c.Level < MinLevel || c.Level > MaxLevel {
		return fmt.Errorf("level must be between %d and %d, got %d", MinLevel, MaxLevel, c.Level)
	}
	return nil
}

// ConfigForLevel creates a Config based on the level's tier settings.
func ConfigForLevel(level int, seed int64) Config {
	lc := GetLevelConfig(level)
	return Config{
		Size:      lc.Size,
		Seed:      seed,
		MineCount: lc.MineCount,
		Level:     level,
	}
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

// NewGeneratorForLevel creates a generator configured for a specific level.
func NewGeneratorForLevel(level int, seed int64) (*Generator, error) {
	config := ConfigForLevel(level, seed)
	return NewGenerator(config)
}

// Generate creates a new GameState with mines randomly placed.
func (g *Generator) Generate() *game.GameState {
	state := game.NewGameState(g.config.Size, g.config.Seed)
	state.Level = g.config.Level
	g.placeMines(state)
	return state
}

// GenerateWithSeed creates a new GameState using a specific seed.
// This is useful for reproducible game generation.
func (g *Generator) GenerateWithSeed(seed int64) *game.GameState {
	// Create a new RNG with the specific seed
	rng := rand.New(rand.NewSource(seed))
	state := game.NewGameState(g.config.Size, seed)
	state.Level = g.config.Level
	g.placeMinesWithRNG(state, rng)
	return state
}

// placeMines randomly places mines on the grid using the generator's RNG.
func (g *Generator) placeMines(state *game.GameState) {
	g.placeMinesWithRNG(state, g.rng)
}

// placeMinesWithRNG places mines using a specific RNG instance.
func (g *Generator) placeMinesWithRNG(state *game.GameState, rng *rand.Rand) {
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

	// Place mines at the first MineCount positions
	for i := 0; i < g.config.MineCount; i++ {
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

// GenerateForLevel is a convenience function that creates a game for a specific level.
func GenerateForLevel(level int, seed int64) (*game.GameState, error) {
	gen, err := NewGeneratorForLevel(level, seed)
	if err != nil {
		return nil, err
	}
	return gen.GenerateWithSeed(seed), nil
}

// ValidateLevel checks if a level is within the valid range.
func ValidateLevel(level int) error {
	if level < MinLevel || level > MaxLevel {
		return fmt.Errorf("level must be between %d and %d, got %d", MinLevel, MaxLevel, level)
	}
	return nil
}

// AllLevelConfigs returns the configuration for all levels.
// Useful for displaying level selection or documentation.
func AllLevelConfigs() []LevelConfig {
	configs := make([]LevelConfig, MaxLevel+1)
	for i := MinLevel; i <= MaxLevel; i++ {
		configs[i] = GetLevelConfig(i)
	}
	return configs
}
