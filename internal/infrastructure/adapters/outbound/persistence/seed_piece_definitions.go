package persistence

import (
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// predefinedPieceDefinition is a compact struct for declaring seed data.
type predefinedPieceDefinition struct {
	ID              uuid.UUID
	Name            string
	ImageURL        string
	DimensionSchema []string
}

// predefinedPieceDefinitions is the single source of truth for system-defined piece types.
// Edit this list to add, rename, or change dimensions of predefined definitions.
// On every application startup, these are upserted into the database.
var predefinedPieceDefinitions = []predefinedPieceDefinition{
	{
		ID:              uuid.MustParse("a0000000-0000-0000-0000-000000000001"),
		Name:            "Rectangular",
		DimensionSchema: []string{"Length", "Width"},
	},
	{
		ID:              uuid.MustParse("a0000000-0000-0000-0000-000000000002"),
		Name:            "Circular",
		DimensionSchema: []string{"Radius"},
	},
}

// SeedPredefinedPieceDefinitions upserts all predefined piece definitions into the database.
// If a definition already exists (by ID), its name, image_url, and dimension_schema are updated.
// If it doesn't exist, it is created. This is idempotent and safe to run on every startup.
func SeedPredefinedPieceDefinitions(db *gorm.DB) error {
	for _, def := range predefinedPieceDefinitions {
		schemaJSON, err := json.Marshal(def.DimensionSchema)
		if err != nil {
			return err
		}

		model := PieceDefinitionModel{
			ID:              def.ID,
			Name:            def.Name,
			ImageURL:        def.ImageURL,
			DimensionSchema: string(schemaJSON),
			Predefined:      true,
			UserID:          nil,
		}

		result := db.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"name", "image_url", "dimension_schema",
			}),
		}).Create(&model)

		if result.Error != nil {
			return result.Error
		}
	}

	slog.Info("predefined piece definitions seeded", "count", len(predefinedPieceDefinitions))
	return nil
}
