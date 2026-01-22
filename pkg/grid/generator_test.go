package grid

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Size != DefaultSize {
		t.Errorf("expected size %d, got %d", DefaultSize, config.Size)
	}
	if config.MineDensity != DefaultMineDensity {
		t.Errorf("expected density %f, got %f", DefaultMineDensity, config.MineDensity)
	}
	if config.MinMineCount != 1 {
		t.Errorf("expected min mine count 1, got %d", config.MinMineCount)
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name:    "valid default",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "size too small",
			config: Config{
				Size:        0,
				MineDensity: 0.15,
			},
			wantErr: true,
		},
		{
			name: "size too large",
			config: Config{
				Size:        101,
				MineDensity: 0.15,
			},
			wantErr: true,
		},
		{
			name: "density too low",
			config: Config{
				Size:        10,
				MineDensity: 0.01,
			},
			wantErr: true,
		},
		{
			name: "density too high",
			config: Config{
				Size:        10,
				MineDensity: 0.60,
			},
			wantErr: true,
		},
		{
			name: "negative min mine count",
			config: Config{
				Size:         10,
				MineDensity:  0.15,
				MinMineCount: -1,
			},
			wantErr: true,
		},
		{
			name: "min exceeds max",
			config: Config{
				Size:         10,
				MineDensity:  0.15,
				MinMineCount: 20,
				MaxMineCount: 10,
			},
			wantErr: true,
		},
		{
			name: "valid custom config",
			config: Config{
				Size:         15,
				MineDensity:  0.20,
				MinMineCount: 10,
				MaxMineCount: 50,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigCalculateMineCount(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected int
	}{
		{
			name: "default 10x10 15%",
			config: Config{
				Size:         10,
				MineDensity:  0.15,
				MinMineCount: 1,
			},
			expected: 15, // 100 * 0.15 = 15
		},
		{
			name: "enforces minimum",
			config: Config{
				Size:         5,
				MineDensity:  0.05,
				MinMineCount: 5,
			},
			expected: 5, // 25 * 0.05 = 1, but min is 5
		},
		{
			name: "enforces maximum",
			config: Config{
				Size:         10,
				MineDensity:  0.50,
				MinMineCount: 1,
				MaxMineCount: 20,
			},
			expected: 20, // 100 * 0.50 = 50, but max is 20
		},
		{
			name: "cannot exceed total cells minus 1",
			config: Config{
				Size:         3,
				MineDensity:  0.50,
				MinMineCount: 1,
				MaxMineCount: 0, // no max
			},
			expected: 4, // 9 * 0.50 = 4, and 4 < 8 (9-1), so 4
		},
		{
			name: "tiny grid with high density",
			config: Config{
				Size:         2,
				MineDensity:  0.50,
				MinMineCount: 1,
				MaxMineCount: 0,
			},
			expected: 2, // 4 * 0.50 = 2, max possible is 3
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.CalculateMineCount()
			if result != tt.expected {
				t.Errorf("CalculateMineCount() = %d, expected %d", result, tt.expected)
			}
		})
	}
}

func TestNewGenerator(t *testing.T) {
	config := DefaultConfig()
	gen, err := NewGenerator(config)

	if err != nil {
		t.Fatalf("NewGenerator failed: %v", err)
	}
	if gen == nil {
		t.Fatal("generator should not be nil")
	}
}

func TestNewGeneratorInvalidConfig(t *testing.T) {
	config := Config{
		Size:        0, // invalid
		MineDensity: 0.15,
	}

	_, err := NewGenerator(config)
	if err == nil {
		t.Error("expected error for invalid config")
	}
}

func TestNewDefaultGenerator(t *testing.T) {
	gen := NewDefaultGenerator()
	if gen == nil {
		t.Fatal("generator should not be nil")
	}
	if gen.Config().Size != DefaultSize {
		t.Errorf("expected size %d, got %d", DefaultSize, gen.Config().Size)
	}
}

func TestGenerate(t *testing.T) {
	config := Config{
		Size:         10,
		Seed:         12345,
		MineDensity:  0.15,
		MinMineCount: 1,
	}

	gen, err := NewGenerator(config)
	if err != nil {
		t.Fatalf("NewGenerator failed: %v", err)
	}

	state := gen.Generate()

	if state.Size != 10 {
		t.Errorf("expected size 10, got %d", state.Size)
	}
	if state.MineCount < 1 {
		t.Error("expected at least 1 mine")
	}
	if state.MineCount > 50 {
		t.Error("too many mines for 15% density")
	}
}

func TestGenerateWithSeedReproducibility(t *testing.T) {
	config := Config{
		Size:         10,
		Seed:         0, // will be overridden
		MineDensity:  0.15,
		MinMineCount: 1,
	}

	gen, _ := NewGenerator(config)
	seed := int64(42)

	// Generate two grids with the same seed
	state1 := gen.GenerateWithSeed(seed)
	state2 := gen.GenerateWithSeed(seed)

	// They should be identical
	if state1.MineCount != state2.MineCount {
		t.Errorf("mine counts differ: %d vs %d", state1.MineCount, state2.MineCount)
	}

	// Check all mine positions match
	for x := 0; x < state1.Size; x++ {
		for y := 0; y < state1.Size; y++ {
			if state1.IsMine(x, y) != state2.IsMine(x, y) {
				t.Errorf("mine mismatch at (%d,%d)", x, y)
			}
		}
	}
}

func TestDifferentSeedsDifferentGrids(t *testing.T) {
	config := Config{
		Size:         10,
		MineDensity:  0.15,
		MinMineCount: 1,
	}

	gen, _ := NewGenerator(config)

	state1 := gen.GenerateWithSeed(1)
	state2 := gen.GenerateWithSeed(2)

	// Count differences
	differences := 0
	for x := 0; x < state1.Size; x++ {
		for y := 0; y < state1.Size; y++ {
			if state1.IsMine(x, y) != state2.IsMine(x, y) {
				differences++
			}
		}
	}

	// With different seeds, grids should differ
	if differences == 0 {
		t.Error("different seeds should produce different grids")
	}
}

func TestGenerateGrid(t *testing.T) {
	state, err := GenerateGrid(10, 12345, 0.15)
	if err != nil {
		t.Fatalf("GenerateGrid failed: %v", err)
	}

	if state.Size != 10 {
		t.Errorf("expected size 10, got %d", state.Size)
	}
	if state.Seed != 12345 {
		t.Errorf("expected seed 12345, got %d", state.Seed)
	}
	if state.MineCount < 1 {
		t.Error("expected at least 1 mine")
	}
}

func TestGenerateGridInvalid(t *testing.T) {
	_, err := GenerateGrid(0, 12345, 0.15) // invalid size
	if err == nil {
		t.Error("expected error for invalid size")
	}

	_, err = GenerateGrid(10, 12345, 0.01) // invalid density
	if err == nil {
		t.Error("expected error for invalid density")
	}
}

func TestGenerateDefaultGrid(t *testing.T) {
	state := GenerateDefaultGrid(12345)

	if state.Size != DefaultSize {
		t.Errorf("expected size %d, got %d", DefaultSize, state.Size)
	}
	if state.MineCount == 0 {
		t.Error("expected at least 1 mine")
	}
}

func TestGetDifficultyConfig(t *testing.T) {
	tests := []struct {
		preset       DifficultyPreset
		expectedSize int
	}{
		{DifficultyEasy, 8},
		{DifficultyMedium, 10},
		{DifficultyHard, 16},
		{DifficultyExpert, 20},
		{"unknown", DefaultSize}, // unknown defaults
	}

	for _, tt := range tests {
		t.Run(string(tt.preset), func(t *testing.T) {
			config := GetDifficultyConfig(tt.preset)
			if config.Size != tt.expectedSize {
				t.Errorf("expected size %d, got %d", tt.expectedSize, config.Size)
			}
		})
	}
}

func TestGenerateWithDifficulty(t *testing.T) {
	presets := []DifficultyPreset{
		DifficultyEasy,
		DifficultyMedium,
		DifficultyHard,
		DifficultyExpert,
	}

	for _, preset := range presets {
		t.Run(string(preset), func(t *testing.T) {
			state, err := GenerateWithDifficulty(preset, 12345)
			if err != nil {
				t.Fatalf("GenerateWithDifficulty failed: %v", err)
			}

			config := GetDifficultyConfig(preset)
			if state.Size != config.Size {
				t.Errorf("expected size %d, got %d", config.Size, state.Size)
			}
			if state.MineCount < config.MinMineCount {
				t.Errorf("expected at least %d mines, got %d", config.MinMineCount, state.MineCount)
			}
			if config.MaxMineCount > 0 && state.MineCount > config.MaxMineCount {
				t.Errorf("expected at most %d mines, got %d", config.MaxMineCount, state.MineCount)
			}
		})
	}
}

func TestMinesAreDistributed(t *testing.T) {
	// Ensure mines aren't all clustered in one area
	config := Config{
		Size:         20,
		Seed:         12345,
		MineDensity:  0.20,
		MinMineCount: 1,
	}

	gen, _ := NewGenerator(config)
	state := gen.GenerateWithSeed(12345)

	// Divide grid into 4 quadrants and count mines in each
	quadrants := make([]int, 4)
	half := state.Size / 2

	for x := 0; x < state.Size; x++ {
		for y := 0; y < state.Size; y++ {
			if state.IsMine(x, y) {
				quadrant := 0
				if x >= half {
					quadrant += 1
				}
				if y >= half {
					quadrant += 2
				}
				quadrants[quadrant]++
			}
		}
	}

	// Each quadrant should have at least some mines
	for i, count := range quadrants {
		if count == 0 {
			t.Errorf("quadrant %d has no mines - poor distribution", i)
		}
	}
}

func TestEveryGeneratedCellIsValid(t *testing.T) {
	config := Config{
		Size:         10,
		Seed:         99999,
		MineDensity:  0.20,
		MinMineCount: 1,
	}

	gen, _ := NewGenerator(config)
	state := gen.Generate()

	minesFound := 0
	for x := 0; x < state.Size; x++ {
		for y := 0; y < state.Size; y++ {
			if state.IsMine(x, y) {
				minesFound++
			}
		}
	}

	if minesFound != state.MineCount {
		t.Errorf("counted %d mines but MineCount is %d", minesFound, state.MineCount)
	}
}

func TestAtLeastOneSafeCell(t *testing.T) {
	// Even with max density, there should be at least one safe cell
	config := Config{
		Size:         5,
		Seed:         12345,
		MineDensity:  MaxMineDensity,
		MinMineCount: 1,
	}

	gen, _ := NewGenerator(config)
	state := gen.Generate()

	totalCells := state.Size * state.Size
	if state.MineCount >= totalCells {
		t.Error("should have at least one safe cell")
	}
}

func TestGeneratorConfig(t *testing.T) {
	config := Config{
		Size:         15,
		Seed:         123,
		MineDensity:  0.20,
		MinMineCount: 10,
		MaxMineCount: 50,
	}

	gen, _ := NewGenerator(config)
	retrieved := gen.Config()

	if retrieved.Size != config.Size {
		t.Error("Config() should return the original config")
	}
	if retrieved.MineDensity != config.MineDensity {
		t.Error("Config() should return the original config")
	}
}
