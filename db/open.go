package db

import (
    "database/sql"

    _ "github.com/jackc/pgx/v5/stdlib"
)

// OpenDatabase opens a database/sql DB using the pgx driver.
func OpenDatabase(databaseURL string) (*sql.DB, error) {
    db, err := sql.Open("pgx", databaseURL)
    if err != nil {
        return nil, err
    }
    if err := db.Ping(); err != nil {
        db.Close()
        return nil, err
    }
    return db, nil
}
