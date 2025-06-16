package db

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

func (db *DB) Migrate() error {
	migrations := []string{
		"001_create_users_table.sql",
		"002_create_rides_table.sql",
		"003_create_ride_passengers_table.sql",
	}

	for _, migration := range migrations {
		migrationFile := filepath.Join("migrations", migration)

		if _, err := os.Stat(migrationFile); os.IsNotExist(err) {
			continue
		}

		content, err := ioutil.ReadFile(migrationFile)
		if err != nil {
			return err
		}

		if _, err := db.Exec("SELECT 1 FROM rides LIMIT 1"); err == nil {
			log.Printf("Migration %s already applied, skipping", migration)
			continue
		}

		if _, err := db.Exec(string(content)); err != nil {
			return err
		}

		log.Printf("Migration %s executed successfully", migration)
	}

	log.Println("All migrations completed successfully")
	return nil
}
