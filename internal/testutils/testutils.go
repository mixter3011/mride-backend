package testutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func SetupTestDB(t *testing.T, modelsToMigrate ...interface{}) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	err = db.AutoMigrate(modelsToMigrate...)
	assert.NoError(t, err)
	return db
}

func Float64Ptr(f float64) *float64 {
	return &f
}
