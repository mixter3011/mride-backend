package db

import (
	"io/ioutil"
	"log"
	"path/filepath"
)

func (db *DB) Migrate() error {
	migrationFile := filepath.Join("migrations", "001_create_users_table.sql")

	content, err := ioutil.ReadFile(migrationFile)
	if err != nil {
		return err
	}

	if _, err := db.Exec(string(content)); err != nil {
		return err
	}

	log.Println("Database migrated successfully")
	return nil
}
