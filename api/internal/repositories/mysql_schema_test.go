package repositories

import (
	"strings"
	"sync"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm/schema"
)

// TestMigrationModelsUseMySQLCompatibleIndexedStrings prevents indexed string
// fields from silently becoming TEXT/LONGTEXT under the MySQL dialector. MySQL
// cannot create a full-value index over those types without a prefix length.
func TestMigrationModelsUseMySQLCompatibleIndexedStrings(t *testing.T) {
	dialector := mysql.Dialector{Config: &mysql.Config{}}

	for _, model := range migrationModels() {
		modelSchema, err := schema.Parse(model, &sync.Map{}, schema.NamingStrategy{})
		if err != nil {
			t.Fatalf("parse schema for %T: %v", model, err)
		}

		indexedFields := make(map[*schema.Field]bool)
		for _, index := range modelSchema.ParseIndexes() {
			for _, option := range index.Fields {
				indexedFields[option.Field] = true
			}
		}

		for _, field := range modelSchema.Fields {
			if field.DataType != schema.String {
				continue
			}
			if !field.PrimaryKey && !field.Unique && !indexedFields[field] {
				continue
			}

			dataType := strings.ToLower(dialector.DataTypeOf(field))
			if strings.Contains(dataType, "text") || strings.Contains(dataType, "blob") {
				t.Errorf(
					"%s.%s is an indexed string with MySQL type %s; declare a bounded size",
					modelSchema.Name,
					field.Name,
					dataType,
				)
			}
		}
	}
}
