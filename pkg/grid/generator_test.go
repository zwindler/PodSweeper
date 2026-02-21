package grid

import (
	"testing"
)

func TestGetLevelConfig(t *testing.T) {
	tests := []struct {
		level         int
		expectedSize  int
		expectedMines int
		expectedTier  int
	}{
		// Tier 1: Levels 0-3
		{0, Tier1Size, Tier1Mines, 1},
		{1, Tier1Size, Tier1Mines, 1},
		{2, Tier1Size, Tier1Mines, 1},
		{3, Tier1Size, Tier1Mines, 1},
		// Tier 2: Levels 4-6
		{4, Tier2Size, Tier2Mines, 2},
		{5, Tier2Size, Tier2Mines, 2},
		{6, Tier2Size, Tier2Mines, 2},
		// Tier 3: Levels 7-9
		{7, Tier3Size, Tier3Mines, 3},
		{8, Tier3Size, Tier3Mines, 3},
		{9, Tier3Size, Tier3Mines, 3},
	}

	for _, tt := range tests {
		t.Run("level_"+string(rune('0'+tt.level)), func(t *testing.T) {
			config := GetLevelConfig(tt.level)

			if config.Size != tt.expectedSize {
				t.Errorf("Level %d: expected size %d, got %d", tt.level, tt.expectedSize, config.Size)
			}
			if config.MineCount != tt.expectedMines {
				t.Errorf("Level %d: expected mines %d, got %d", tt.level, tt.expectedMines, config.MineCount)
			}
			if config.Tier != tt.expectedTier {
				t.Errorf("Level %d: expected tier %d, got %d", tt.level, tt.expectedTier, config.Tier)
			}
			if config.Level != tt.level {
				t.Errorf("Level %d: expected level %d in config, got %d", tt.level, tt.level, config.Level)
			}
		})
	}
}

func TestGetLevelConfigClamping(t *testing.T) {
	// Level below minimum should clamp to 0
	config := GetLevelConfig(-5)
	if config.Level != 0 {
		t.Errorf("expected level 0 for negative input, got %d", config.Level)
	}
	if config.Tier != 1 {
		t.Errorf("expected tier 1 for negative input, got %d", config.Tier)
	}

	// Level above maximum should clamp to 9
	config = GetLevelConfig(99)
	if config.Level != 9 {
		t.Errorf("expected level 9 for overflow, got %d", config.Level)
	}
	if config.Tier != 3 {
		t.Errorf("expected tier 3 for overflow, got %d", config.Tier)
	}
}

func TestGetTierDescription(t *testing.T) {
	tests := []struct {
		tier     int
		contains string
	}{
		{1, "Tutorial"},
		{2, "Intermediate"},
		{3, "Expert"},
		{99, "Unknown"},
	}

	for _, tt := range tests {
		desc := GetTierDescription(tt.tier)
		if desc == "" {
			t.Errorf("Tier %d: description should not be empty", tt.tier)
		}
		// Just check it contains expected keyword
		found := false
		for i := 0; i < len(desc)-len(tt.contains)+1; i++ {
			if desc[i:i+len(tt.contains)] == tt.contains {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Tier %d: expected description to contain %q, got %q", tt.tier, tt.contains, desc)
		}
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name:    "valid tier 1",
			config:  ConfigForLevel(0, 12345),
			wantErr: false,
		},
		{
			name:    "valid tier 2",
			config:  ConfigForLevel(5, 12345),
			wantErr: false,
		},
		{
			name:    "valid tier 3",
			config:  ConfigForLevel(9, 12345),
			wantErr: false,
		},
		{
			name: "size too small",
			config: Config{
				Size:      0,
				MineCount: 5,
				Level:     0,
			},
			wantErr: true,
		},
		{
			name: "size too large",
			config: Config{
				Size:      101,
				MineCount: 5,
				Level:     0,
			},
			wantErr: true,
		},
		{
			name: "mine count too low",
			config: Config{
				Size:      10,
				MineCount: 0,
				Level:     0,
			},
			wantErr: true,
		},
		{
			name: "mine count exceeds grid",
			config: Config{
				Size:      5,
				MineCount: 25, // 5x5 = 25, need at least 1 safe cell
				Level:     0,
			},
			wantErr: true,
		},
		{
			name: "level too low",
			config: Config{
				Size:      10,
				MineCount: 5,
				Level:     -1,
			},
			wantErr: true,
		},
		{
			name: "level too high",
			config: Config{
				Size:      10,
				MineCount: 5,
				Level:     10,
			},
			wantErr: true,
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

func TestConfigForLevel(t *testing.T) {
	seed := int64(12345)

	for level := 0; level <= 9; level++ {
		config := ConfigForLevel(level, seed)

		if config.Seed != seed {
			t.Errorf("Level %d: expected seed %d, got %d", level, seed, config.Seed)
		}
		if config.Level != level {
			t.Errorf("Level %d: expected level in config, got %d", level, config.Level)
		}

		// Verify config matches expected tier
		lc := GetLevelConfig(level)
		if config.Size != lc.Size {
			t.Errorf("Level %d: size mismatch", level)
		}
		if config.MineCount != lc.MineCount {
			t.Errorf("Level %d: mine count mismatch", level)
		}
	}
}

func TestNewGenerator(t *testing.T) {
	config := ConfigForLevel(5, 12345)
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
		Size:      0, // invalid
		MineCount: 5,
		Level:     0,
	}

	_, err := NewGenerator(config)
	if err == nil {
		t.Error("expected error for invalid config")
	}
}

func TestNewGeneratorForLevel(t *testing.T) {
	for level := 0; level <= 9; level++ {
		gen, err := NewGeneratorForLevel(level, 12345)
		if err != nil {
			t.Fatalf("Level %d: NewGeneratorForLevel failed: %v", level, err)
		}

		config := gen.Config()
		expected := GetLevelConfig(level)

		if config.Size != expected.Size {
			t.Errorf("Level %d: expected size %d, got %d", level, expected.Size, config.Size)
		}
		if config.MineCount != expected.MineCount {
			t.Errorf("Level %d: expected mines %d, got %d", level, expected.MineCount, config.MineCount)
		}
		if config.Level != level {
			t.Errorf("Level %d: expected level in config, got %d", level, config.Level)
		}
	}
}

func TestGenerate(t *testing.T) {
	gen, err := NewGeneratorForLevel(5, 12345)
	if err != nil {
		t.Fatalf("NewGeneratorForLevel failed: %v", err)
	}

	state := gen.Generate()

	expectedConfig := GetLevelConfig(5)
	if state.Size != expectedConfig.Size {
		t.Errorf("expected size %d, got %d", expectedConfig.Size, state.Size)
	}
	if state.MineCount != expectedConfig.MineCount {
		t.Errorf("expected %d mines, got %d", expectedConfig.MineCount, state.MineCount)
	}
	if state.Level != 5 {
		t.Errorf("expected level 5, got %d", state.Level)
	}
}

func TestGenerateWithSeedReproducibility(t *testing.T) {
	gen, _ := NewGeneratorForLevel(5, 0)
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
	gen, _ := NewGeneratorForLevel(5, 0)

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

func TestGenerateForLevel(t *testing.T) {
	for level := 0; level <= 9; level++ {
		state, err := GenerateForLevel(level, 12345)
		if err != nil {
			t.Fatalf("Level %d: GenerateForLevel failed: %v", level, err)
		}

		expected := GetLevelConfig(level)
		if state.Size != expected.Size {
			t.Errorf("Level %d: expected size %d, got %d", level, expected.Size, state.Size)
		}
		if state.MineCount != expected.MineCount {
			t.Errorf("Level %d: expected mines %d, got %d", level, expected.MineCount, state.MineCount)
		}
		if state.Level != level {
			t.Errorf("Level %d: expected level in state, got %d", level, state.Level)
		}
		if state.Seed != 12345 {
			t.Errorf("Level %d: expected seed 12345, got %d", level, state.Seed)
		}
	}
}

func TestValidateLevel(t *testing.T) {
	// Valid levels
	for level := 0; level <= 9; level++ {
		if err := ValidateLevel(level); err != nil {
			t.Errorf("Level %d should be valid, got error: %v", level, err)
		}
	}

	// Invalid levels
	invalidLevels := []int{-1, -10, 10, 99}
	for _, level := range invalidLevels {
		if err := ValidateLevel(level); err == nil {
			t.Errorf("Level %d should be invalid", level)
		}
	}
}

func TestAllLevelConfigs(t *testing.T) {
	configs := AllLevelConfigs()

	if len(configs) != 10 {
		t.Errorf("expected 10 level configs, got %d", len(configs))
	}

	for i, config := range configs {
		if config.Level != i {
			t.Errorf("config[%d].Level = %d, expected %d", i, config.Level, i)
		}
	}
}

func TestTierSizes(t *testing.T) {
	// Verify tier constants are what we expect
	if Tier1Size != 5 {
		t.Errorf("Tier1Size = %d, expected 5", Tier1Size)
	}
	if Tier2Size != 10 {
		t.Errorf("Tier2Size = %d, expected 10", Tier2Size)
	}
	if Tier3Size != 20 {
		t.Errorf("Tier3Size = %d, expected 20", Tier3Size)
	}

	if Tier1Mines != 4 {
		t.Errorf("Tier1Mines = %d, expected 4", Tier1Mines)
	}
	if Tier2Mines != 15 {
		t.Errorf("Tier2Mines = %d, expected 15", Tier2Mines)
	}
	if Tier3Mines != 60 {
		t.Errorf("Tier3Mines = %d, expected 60", Tier3Mines)
	}
}

func TestMinesAreDistributed(t *testing.T) {
	// Test with Tier 3 (20x20, 60 mines) for good distribution
	gen, _ := NewGeneratorForLevel(9, 12345)
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
	for level := 0; level <= 9; level++ {
		gen, _ := NewGeneratorForLevel(level, 99999)
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
			t.Errorf("Level %d: counted %d mines but MineCount is %d", level, minesFound, state.MineCount)
		}
	}
}

func TestAtLeastOneSafeCell(t *testing.T) {
	// All levels should have at least one safe cell
	for level := 0; level <= 9; level++ {
		state, _ := GenerateForLevel(level, 12345)

		totalCells := state.Size * state.Size
		if state.MineCount >= totalCells {
			t.Errorf("Level %d: should have at least one safe cell", level)
		}
	}
}

func TestGeneratorConfig(t *testing.T) {
	gen, _ := NewGeneratorForLevel(7, 123)
	config := gen.Config()

	expected := GetLevelConfig(7)
	if config.Size != expected.Size {
		t.Error("Config() should return the original config size")
	}
	if config.MineCount != expected.MineCount {
		t.Error("Config() should return the original config mine count")
	}
	if config.Level != 7 {
		t.Error("Config() should return the original config level")
	}
}

func TestLevelConstants(t *testing.T) {
	if MinLevel != 0 {
		t.Errorf("MinLevel = %d, expected 0", MinLevel)
	}
	if MaxLevel != 9 {
		t.Errorf("MaxLevel = %d, expected 9", MaxLevel)
	}
}
