package db

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

type DB struct {
	*sql.DB
}

func New(dbUrl string) (*DB, error) {
	db, err := sql.Open("postgres", dbUrl)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	log.Println("Connected to database")
	return &DB{db}, nil
}

func (db *DB) Close() error {
	return db.DB.Close()
}
