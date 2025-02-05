package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func NewSqliteConnection(path string) (*sql.DB, error) {
	config := map[string]string{
		"_busy_timeout": "5000",
		"_foreign_keys": "ON",
		"cache_size": "-2000",
		"_synchronous": "NORMAL",
		 "_journal_mode": "WAL",
		"_incremental_vacuum": "1",
		"_mmap_size": "2147483648",
		"_temp_store": "MEMORY",
		"_page_size": "32768",
		"_auto_vacuum": "incremental",
	}
	params := ""
	for k, v := range config {
		params += fmt.Sprintf("%s=%s&", k, v)
	}
	dsn := fmt.Sprintf("%s?%s", path, params)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error connecting to the database: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)
	return db, nil
}
