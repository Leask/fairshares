package storage

import (
	"crypto/sha1"
	"database/sql"
	"fmt"
	"github.com/huo-ju/fairshares/internal/pkg/poolapi"
	"github.com/mattn/go-sqlite3"
	"time"
)

type Storage struct {
	db *sql.DB
}

func NewStorage(db *sql.DB) *Storage {
	return &Storage{db}
}

func (s *Storage) NewDatabase() error {
	sqlStmt := `
	create table setting (version INTEGER); 
	create table workershares (checksum TEXT primary key, poolname TEXT not null, address TEXT not null, workername TEXT not null, valid_shares INTEGER, stale_shares INTEGER, invalid_shares INTEGER, lastseen TIMESTAMP); 
	create table addresses (checksum TEXT primary key, address TEXT, poolname TEXT, created_at TIMESTAMP);
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

func (s *Storage) RegAddress(address string, poolname string) error {

	str := fmt.Sprintf("%s %s", address, poolname)
	checksumstr := fmt.Sprintf("%x", sha1.Sum([]byte(str)))
	stmt, err := s.db.Prepare("insert into addresses (checksum, address, poolname, created_at) values(?,?,?,?)")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(checksumstr, address, poolname, time.Now())
	return err
}

func (s *Storage) GetAddresses(poolname string) (error, []string) {
	results := []string{}
	rows, err := s.db.Query(`select address from addresses where poolname=$1`, poolname)
	for rows.Next() {
		var address string
		err = rows.Scan(&address)
		if err == nil {
			results = append(results, address)

		}
	}
	rows.Close()
	return nil, results
}

func (s *Storage) SaveWorkerShares(poolname string, address string, workers []*poolapi.FlexpoolWorker) int {
	tx, err := s.db.Begin()
	stmt, err := tx.Prepare("insert into workershares(checksum, poolname, address, workername, valid_shares, stale_shares, invalid_shares, lastseen) values(?, ?,?,?,?,?,?,?)")
	defer stmt.Close()
	insertcount := 0
	for _, worker := range workers {
		str := fmt.Sprintf("%s %s %s %d %d %d %d", poolname, address, worker.Name, worker.ValidShares, worker.StaleShares, worker.InvalidShares, worker.LastSeen)
		checksumstr := fmt.Sprintf("%x", sha1.Sum([]byte(str)))
		_, err = stmt.Exec(checksumstr, poolname, address, worker.Name, worker.ValidShares, worker.StaleShares, worker.InvalidShares, worker.LastSeen)
		if err != nil {
			sqliteErr := err.(sqlite3.Error)
			if sqliteErr.ExtendedCode == sqlite3.ErrConstraintPrimaryKey {
				insertcount++
				//data exist, it's ok
			} else {
				fmt.Printf("save worker err:%s\n", err)
			}
		} else {
			insertcount++
		}
	}
	tx.Commit()
	return insertcount
}
