//go:build integration

package repositories

import (
	"os"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestMySQLMigrationFromEmptyDatabase(t *testing.T) {
	dsn := os.Getenv("MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("MYSQL_TEST_DSN is not set")
	}

	mysqlDB, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect to MySQL: %v", err)
	}

	sqlDB, err := mysqlDB.DB()
	if err != nil {
		t.Fatalf("get MySQL connection: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := sqlDB.Close(); closeErr != nil {
			t.Errorf("close MySQL connection: %v", closeErr)
		}
	})

	tables, err := mysqlDB.Migrator().GetTables()
	if err != nil {
		t.Fatalf("list MySQL tables: %v", err)
	}
	if len(tables) != 0 {
		t.Fatalf("integration database must be empty, found tables: %v", tables)
	}

	previousDB := db
	db = mysqlDB
	t.Cleanup(func() { db = previousDB })

	if err := MakeMigrations(); err != nil {
		t.Fatalf("migrate empty MySQL database: %v", err)
	}

	expectedColumns := []struct {
		table  string
		column string
		length int64
	}{
		{table: "users", column: "username", length: 255},
		{table: "categories", column: "name", length: 255},
		{table: "tags", column: "name", length: 255},
		{table: "prompts", column: "name", length: 255},
		{table: "receipt_processing_settings", column: "name", length: 255},
		{table: "data_migrations", column: "name", length: 128},
	}

	for _, expected := range expectedColumns {
		assertMySQLVarcharUniqueColumn(t, mysqlDB, expected.table, expected.column, expected.length)
	}
}

func assertMySQLVarcharUniqueColumn(t *testing.T, mysqlDB *gorm.DB, table, column string, expectedLength int64) {
	t.Helper()

	var definition struct {
		DataType        string `gorm:"column:DATA_TYPE"`
		CharacterLength int64  `gorm:"column:CHARACTER_MAXIMUM_LENGTH"`
	}
	if err := mysqlDB.Raw(`
		SELECT DATA_TYPE, CHARACTER_MAXIMUM_LENGTH
		FROM information_schema.columns
		WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?
	`, table, column).Scan(&definition).Error; err != nil {
		t.Fatalf("inspect %s.%s: %v", table, column, err)
	}
	if definition.DataType != "varchar" || definition.CharacterLength != expectedLength {
		t.Errorf(
			"%s.%s type = %s(%d), want varchar(%d)",
			table,
			column,
			definition.DataType,
			definition.CharacterLength,
			expectedLength,
		)
	}

	var uniqueIndexCount int64
	if err := mysqlDB.Raw(`
		SELECT COUNT(*)
		FROM information_schema.statistics
		WHERE table_schema = DATABASE()
			AND table_name = ?
			AND column_name = ?
			AND non_unique = 0
			AND sub_part IS NULL
	`, table, column).Scan(&uniqueIndexCount).Error; err != nil {
		t.Fatalf("inspect unique index for %s.%s: %v", table, column, err)
	}
	if uniqueIndexCount == 0 {
		t.Errorf("%s.%s has no full-value unique index", table, column)
	}
}
