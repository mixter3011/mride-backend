package db

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

func (db *DB) Migrate() error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT NOW()
		)
	`)
	if err != nil {
		return err
	}

	migrations := []string{
		"001_create_users_table.sql",
		"002_create_rides_table.sql",
		"003_create_ride_passengers_table.sql",
		"004_create_notifications_table.sql",
		"005_create_fcm_table.sql",
	}

	for _, migration := range migrations {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = $1", migration).Scan(&count)
		if err != nil {
			return err
		}

		if count > 0 {
			log.Printf("Migration %s already applied, skipping", migration)
			continue
		}

		migrationFile := filepath.Join("migrations", migration)
		if _, err := os.Stat(migrationFile); os.IsNotExist(err) {
			continue
		}

		content, err := ioutil.ReadFile(migrationFile)
		if err != nil {
			return err
		}

		tx, err := db.Begin()
		if err != nil {
			return err
		}

		if _, err := tx.Exec(string(content)); err != nil {
			tx.Rollback()
			return err
		}

		_, err = tx.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", migration)
		if err != nil {
			tx.Rollback()
			return err
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		log.Printf("Migration %s executed successfully", migration)
	}

	log.Println("All migrations completed successfully")
	return nil
}
