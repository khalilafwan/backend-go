package config

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

var (
	DB         *sql.DB
	initDBOnce sync.Once
)

// InitDB initializes the PostgreSQL connection as a singleton
func InitDB() error {
	var initError error
	initDBOnce.Do(func() {
		// Build connection string for PostgreSQL
		connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			PostgresUser, PostgresPassword, PostgresHost, PostgresPort, PostgresDB)

		// Open a connection to PostgreSQL
		db, err := sql.Open("postgres", connStr)
		if err != nil {
			initError = fmt.Errorf("gagal membuka koneksi PostgreSQL: %v", err)
			return
		}

		// Set connection pool limits
		db.SetMaxOpenConns(25)                  // Maximum number of open connections
		db.SetMaxIdleConns(25)                  // Maximum number of idle connections
		db.SetConnMaxLifetime(10 * time.Minute) // Limit lifetime to 10 minutes for periodic refresh

		// Ping the database to ensure the connection is successful
		if err := db.Ping(); err != nil {
			initError = fmt.Errorf("gagal melakukan ping ke basis data PostgreSQL: %v", err)
			return
		}

		DB = db
		log.Println("✅ Terhubung ke basis data PostgreSQL!")
	})

	return initError
}

// CloseDB closes the database connection gracefully
func CloseDB() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}
