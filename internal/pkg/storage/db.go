package storage

import (
	"database/sql"
)

type Storage struct {
	db *sql.DB
}

func NewStorage(db *sql.DB) *Storage {
	return &Storage{db}
}

func (s *Storage) NewDatabase() error {
	sqlStmt := `create table workershares (checksum TEXT primary key, address TEXT not null, workername TEXT not null, valid_shares INTEGER, stale_shares INTEGER, invalid_shares INTEGER, lastseen TIMESTAMP); 
	create table setting (version INTEGER); 
	`
	_, err := s.db.Exec(sqlStmt)
	if err == nil {
		_, err = s.db.Exec("insert into setting(version) values('1');")
	}
	return err
}

func (s *Storage) DatabaseVersion() int {
	var dbVersion int
	err := s.db.QueryRow(`SELECT version from setting;`).Scan(&dbVersion)
	if err != nil {
		return 0
	}
	return dbVersion
}
