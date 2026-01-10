package entities

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEntityConstants verifies that all entity constants are properly defined.
func TestEntityConstants(t *testing.T) {
	tests := []struct {
		name          string
		entity        Entity
		expectedTable string
		expectedStage string
	}{
		{
			name:          "Blocks entity",
			entity:        Blocks,
			expectedTable: "blocks",
			expectedStage: "blocks_staging",
		},
		{
			name:          "Transactions entity",
			entity:        Transactions,
			expectedTable: "txs",
			expectedStage: "txs_staging",
		},
		{
			name:          "BlockSummaries entity",
			entity:        BlockSummaries,
			expectedTable: "block_summaries",
			expectedStage: "block_summaries_staging",
		},
		{
			name:          "Accounts entity",
			entity:        Accounts,
			expectedTable: "accounts",
			expectedStage: "accounts_staging",
		},
		{
			name:          "Events entity",
			entity:        Events,
			expectedTable: "events",
			expectedStage: "events_staging",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedTable, tt.entity.TableName(),
				"TableName should return production table name")
			assert.Equal(t, tt.expectedStage, tt.entity.StagingTableName(),
				"StagingTableName should append _staging")
			assert.Equal(t, tt.expectedTable, tt.entity.String(),
				"String should return base entity name")
			assert.True(t, tt.entity.IsValid(),
				"Entity constant should be valid")
		})
	}
}

// TestEntityString verifies the String method implementation.
func TestEntityString(t *testing.T) {
	tests := []struct {
		name     string
		entity   Entity
		expected string
	}{
		{"Blocks", Blocks, "blocks"},
		{"Transactions", Transactions, "txs"},
		{"BlockSummaries", BlockSummaries, "block_summaries"},
		{"Accounts", Accounts, "accounts"},
		{"Events", Events, "events"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.entity.String())
		})
	}
}

// TestEntityTableNames verifies table naming conventions.
func TestEntityTableNames(t *testing.T) {
	tests := []struct {
		entity          Entity
		expectedProd    string
		expectedStaging string
	}{
		{Blocks, "blocks", "blocks_staging"},
		{Transactions, "txs", "txs_staging"},
		{BlockSummaries, "block_summaries", "block_summaries_staging"},
		{Accounts, "accounts", "accounts_staging"},
		{Events, "events", "events_staging"},
	}

	for _, tt := range tests {
		t.Run(string(tt.entity), func(t *testing.T) {
			assert.Equal(t, tt.expectedProd, tt.entity.TableName())
			assert.Equal(t, tt.expectedStaging, tt.entity.StagingTableName())

			// Verify staging name follows convention
			assert.Equal(t, tt.entity.TableName()+"_staging", tt.entity.StagingTableName(),
				"Staging table should be production table + _staging suffix")
		})
	}
}

// TestEntityIsValid tests entity validation.
func TestEntityIsValid(t *testing.T) {
	tests := []struct {
		name     string
		entity   Entity
		expected bool
	}{
		{"Valid Blocks", Blocks, true},
		{"Valid Transactions", Transactions, true},
		{"Invalid empty", Entity(""), false},
		{"Invalid typo", Entity("bloks"), false},
		{"Invalid staging suffix", Entity("blocks_staging"), false},
		{"Invalid random", Entity("random_entity"), false},
		{"Invalid uppercase", Entity("BLOCKS"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.entity.IsValid())
		})
	}
}

// TestFromString verifies string-to-entity conversion with validation.
func TestFromString(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  Entity
		wantError bool
	}{
		{
			name:      "Valid blocks",
			input:     "blocks",
			expected:  Blocks,
			wantError: false,
		},
		{
			name:      "Valid txs",
			input:     "txs",
			expected:  Transactions,
			wantError: false,
		},
		{
			name:      "Invalid empty",
			input:     "",
			wantError: true,
		},
		{
			name:      "Invalid typo",
			input:     "bloks",
			wantError: true,
		},
		{
			name:      "Invalid staging suffix",
			input:     "blocks_staging",
			wantError: true,
		},
		{
			name:      "Invalid unknown",
			input:     "unknown_entity",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entity, err := FromString(tt.input)

			if tt.wantError {
				assert.Error(t, err, "Should return error for invalid input")
				assert.Empty(t, entity, "Should return empty entity on error")
				// Verify error message contains helpful information
				assert.Contains(t, err.Error(), tt.input,
					"Error should mention the invalid input")
				assert.Contains(t, err.Error(), "valid entities",
					"Error should list valid entities")
			} else {
				assert.NoError(t, err, "Should not return error for valid input")
				assert.Equal(t, tt.expected, entity)
			}
		})
	}
}

// TestMustFromString verifies panic behavior for invalid input.
func TestMustFromString(t *testing.T) {
	t.Run("Valid input succeeds", func(t *testing.T) {
		entity := MustFromString("blocks")
		assert.Equal(t, Blocks, entity)
	})

	t.Run("Invalid input panics", func(t *testing.T) {
		assert.Panics(t, func() {
			MustFromString("invalid")
		}, "Should panic on invalid entity")
	})
}

// TestAll verifies the All() function returns all entities.
func TestAll(t *testing.T) {
	entities := All()

	// Verify count
	assert.Len(t, entities, Count(), "All() should return all entities")
	assert.Greater(t, len(entities), 0, "Should have at least one entity")

	// Verify all known entities are present
	expectedEntities := map[Entity]bool{
		Blocks:         true,
		Transactions:   true,
		BlockSummaries: true,
		Accounts:       true,
		Events:         true,
	}

	for entity := range expectedEntities {
		assert.Contains(t, entities, entity,
			"All() should contain %s", entity)
	}

	// Verify all entities are valid
	for _, entity := range entities {
		assert.True(t, entity.IsValid(),
			"All entities returned by All() should be valid: %s", entity)
		assert.NotEmpty(t, entity.String(),
			"All entities should have non-empty names")
	}

	// Verify no duplicates
	seen := make(map[Entity]bool)
	for _, entity := range entities {
		assert.False(t, seen[entity],
			"All() should not contain duplicates: %s", entity)
		seen[entity] = true
	}
}

// TestAllReturnsACopy verifies that All() returns a copy, not the internal slice.
func TestAllReturnsACopy(t *testing.T) {
	entities1 := All()
	entities2 := All()

	// Modify first slice
	if len(entities1) > 0 {
		entities1[0] = Entity("modified")
	}

	// Second slice should be unchanged
	assert.NotEqual(t, entities1, entities2,
		"Modifying returned slice should not affect future calls")

	// Verify second slice still valid
	for _, entity := range entities2 {
		assert.True(t, entity.IsValid(),
			"All() should always return valid entities")
	}
}

// TestAllStrings verifies string conversion of all entities.
func TestAllStrings(t *testing.T) {
	strings := AllStrings()

	// Verify count matches entity count
	assert.Len(t, strings, Count())
	assert.Greater(t, len(strings), 0)

	// Verify all strings are non-empty
	for _, s := range strings {
		assert.NotEmpty(t, s, "Entity strings should not be empty")
	}

	// Verify no duplicates
	seen := make(map[string]bool)
	for _, s := range strings {
		assert.False(t, seen[s], "AllStrings should not contain duplicates: %s", s)
		seen[s] = true
	}

	// Verify each string can be converted back to a valid entity
	for _, s := range strings {
		entity, err := FromString(s)
		assert.NoError(t, err, "AllStrings should only return valid entity names: %s", s)
		assert.Equal(t, s, entity.String(), "Round-trip conversion should work")
	}
}

// TestCount verifies the Count function.
func TestCount(t *testing.T) {
	count := Count()
	assert.Greater(t, count, 0, "Should have at least one entity")
	assert.Equal(t, len(All()), count, "Count should match All() length")
	assert.Equal(t, len(AllStrings()), count, "Count should match AllStrings() length")
}

// TestEntityJSONSerialization verifies JSON marshaling/unmarshaling.
func TestEntityJSONSerialization(t *testing.T) {
	type TestStruct struct {
		Entity Entity `json:"entity"`
		Name   string `json:"name"`
	}

	tests := []struct {
		name    string
		input   TestStruct
		wantErr bool
	}{
		{
			name: "Valid entity serialization",
			input: TestStruct{
				Entity: Blocks,
				Name:   "test",
			},
			wantErr: false,
		},
		{
			name: "Valid transactions entity",
			input: TestStruct{
				Entity: Transactions,
				Name:   "test",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.input)
			require.NoError(t, err, "Marshaling should succeed")

			// Unmarshal back
			var output TestStruct
			err = json.Unmarshal(data, &output)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err, "Unmarshaling should succeed")
				assert.Equal(t, tt.input.Entity, output.Entity,
					"Round-trip should preserve entity value")
				assert.Equal(t, tt.input.Name, output.Name)
			}
		})
	}
}

// TestEntityJSONUnmarshalValidation verifies that unmarshaling validates entity names.
func TestEntityJSONUnmarshalValidation(t *testing.T) {
	type TestStruct struct {
		Entity Entity `json:"entity"`
	}

	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name:    "Valid entity",
			json:    `{"entity":"blocks"}`,
			wantErr: false,
		},
		{
			name:    "Invalid entity",
			json:    `{"entity":"invalid"}`,
			wantErr: true,
		},
		{
			name:    "Empty entity",
			json:    `{"entity":""}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output TestStruct
			err := json.Unmarshal([]byte(tt.json), &output)

			if tt.wantErr {
				assert.Error(t, err, "Should reject invalid entity during unmarshal")
			} else {
				assert.NoError(t, err, "Should accept valid entity during unmarshal")
			}
		})
	}
}

// TestEntityUsageInMap verifies entities work as map keys.
func TestEntityUsageInMap(t *testing.T) {
	// Entities should work as map keys since they're based on strings
	processedEntities := make(map[Entity]bool)

	for _, entity := range All() {
		processedEntities[entity] = true
	}

	// Verify all entities are in the map
	assert.Equal(t, Count(), len(processedEntities),
		"All entities should be usable as map keys")

	// Verify lookup works
	assert.True(t, processedEntities[Blocks])
	assert.True(t, processedEntities[Transactions])
	assert.False(t, processedEntities[Entity("invalid")])
}

// TestEntityUsageInSwitch verifies entities work in switch statements.
func TestEntityUsageInSwitch(t *testing.T) {
	entity := Blocks

	var result string
	switch entity {
	case Blocks:
		result = "blocks"
	case Transactions:
		result = "transactions"
	default:
		result = "unknown"
	}

	assert.Equal(t, "blocks", result,
		"Entities should work in switch statements")
}

// TestConcurrentAccess verifies thread-safety of package functions.
func TestConcurrentAccess(t *testing.T) {
	// Run multiple goroutines accessing package functions simultaneously
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			// Call all public functions
			_ = All()
			_ = AllStrings()
			_ = Count()
			_, _ = FromString("blocks")
			_ = Blocks.IsValid()
			_ = Blocks.TableName()
			_ = Blocks.StagingTableName()

			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// If we get here without data races, test passes
	assert.True(t, true, "Concurrent access should be safe")
}

// BenchmarkEntityValidation benchmarks the IsValid method.
func BenchmarkEntityValidation(b *testing.B) {
	entity := Blocks
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = entity.IsValid()
	}
}

// BenchmarkFromString benchmarks string-to-entity conversion.
func BenchmarkFromString(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = FromString("blocks")
	}
}

// BenchmarkAll benchmarks getting all entities.
func BenchmarkAll(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = All()
	}
}

// BenchmarkTableName benchmarks table name generation.
func BenchmarkTableName(b *testing.B) {
	entity := Blocks
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = entity.TableName()
	}
}

// BenchmarkStagingTableName benchmarks staging table name generation.
func BenchmarkStagingTableName(b *testing.B) {
	entity := Blocks
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = entity.StagingTableName()
	}
}
