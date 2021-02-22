package storage

import (
	"crypto/sha1"
	"database/sql"
	"fmt"
	"github.com/huo-ju/fairshares/internal/pkg/poolapi"
	"github.com/mattn/go-sqlite3"
	"log"
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
	create table workershares (logid INTEGER PRIMARY KEY AUTOINCREMENT, poolname TEXT not null, address TEXT not null, workername TEXT not null, valid_shares INTEGER, stale_shares INTEGER, invalid_shares INTEGER, lastseen INTEGER); 
	CREATE INDEX workerindex ON workershares(poolname,address, workername);
	create table addresses (checksum TEXT primary key, address TEXT, poolname TEXT, created_at TIMESTAMP);
	create table balance (checksum TEXT primary key, address TEXT, poolname TEXT, balance INTEGER, created_at TIMESTAMP);
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

	filtedworkers := []poolapi.FlexpoolWorker{}
	for _, worker := range workers {
		err, lastestlogworker := s.GetLastestWorkerShare(poolname, address, worker.Name)
		if err != nil {
			log.Println(err)
		} else {
			if worker.ValidShares > lastestlogworker.ValidShares && worker.LastSeen > lastestlogworker.LastSeen {
				filtedworkers = append(filtedworkers, *worker)
			} else {
				log.Printf("lastestlogworker:%s share %d lastseen %d\n", lastestlogworker.Name, lastestlogworker.ValidShares, lastestlogworker.LastSeen)
				log.Printf("skip worker %s share %d lastseen %d\n", worker.Name, worker.ValidShares, worker.LastSeen)
			}
		}
	}

	tx, err := s.db.Begin()
	if err != nil {
		log.Printf("SaveWorkerShares db.Begin err: %s\n", err)
		return 0
	}

	//poolname TEXT not null, address TEXT not null, workername TEXT not null, valid_shares INTEGER, stale_shares INTEGER, invalid_shares INTEGER, lastseen TIMESTAMP

	stmt, err := tx.Prepare("insert into workershares(poolname, address, workername, valid_shares, stale_shares, invalid_shares, lastseen) values( ?,?,?,?,?,?,?)")
	if err != nil {
		log.Printf("SaveWorkerShares tx.Prepare err: %s\n", err)
		return 0
	}
	defer stmt.Close()
	insertcount := 0
	for _, worker := range filtedworkers {
		_, err = stmt.Exec(poolname, address, worker.Name, worker.ValidShares, worker.StaleShares, worker.InvalidShares, worker.LastSeen)
		if err != nil {
			sqliteErr := err.(sqlite3.Error)
			if sqliteErr.ExtendedCode == sqlite3.ErrConstraintPrimaryKey {
				insertcount++
				//data exist, it's ok
			} else {
				log.Printf("save worker err:%s\n", err)
			}
		} else {
			insertcount++
		}
	}
	tx.Commit()
	return insertcount
}

func (s *Storage) GetLastestWorkerShare(poolname string, address string, name string) (error, *poolapi.FlexpoolWorker) {
	worker := poolapi.FlexpoolWorker{Name: name}
	rows, err := s.db.Query(`select valid_shares, lastseen from workershares where poolname=$1 and address=$2 and workername=$3 order by logid desc limit 1;`, poolname, address, name)
	for rows.Next() {
		err = rows.Scan(&worker.ValidShares, &worker.LastSeen)
		if err != nil {
			return err, nil

		}
	}
	defer rows.Close()
	return nil, &worker
}

func (s *Storage) SaveBalance(poolname string, address string, balance int64) error {
	str := fmt.Sprintf("%s %s %d", address, poolname, balance)
	checksumstr := fmt.Sprintf("%x", sha1.Sum([]byte(str)))
	stmt, err := s.db.Prepare("insert into balance (checksum, address, poolname, balance, created_at) values(?,?,?,?,?)")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(checksumstr, address, poolname, balance, time.Now())
	return err
}
