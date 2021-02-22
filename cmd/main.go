package main

import (
	"context"
	"database/sql"
	"flag"
	"github.com/huo-ju/fairshares/internal/pkg/poolapi"
	"github.com/huo-ju/fairshares/internal/pkg/storage"
	"github.com/mattn/go-sqlite3"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const FlexApiEndpoint = "https://flexpool.io/api/v1"
const DBVersion = 1

//internal/pkg/poolapi/flexpool.go

const (
	JobFetchData    = iota
	JobFetchBalance = iota
)

type Job struct {
	Id       int
	Poolname string
	Address  string
	Type     int
}
type Result struct {
	Id   int
	Type int
	err  error
}

func main() {

	dbname := flag.String("dbname", "faireshare.db", "database name")

	flag.Parse()

	jobch := make(chan Job)
	resultch := make(chan Result)

	address := "0x65146D70901C70188Eb02AeF452eEcCC3dA39208"
	poolname := "flexpool"

	db, err := sql.Open("sqlite3", *dbname)
	log.Println("open database:", *dbname)
	if err != nil {
		log.Fatal(err)
	}
	store := storage.NewStorage(db)
	if store.DatabaseVersion() < DBVersion {
		err = store.NewDatabase()
	}
	if err != nil {
		log.Fatal(err)
	}

	ver := store.DatabaseVersion()
	log.Println("database version :", ver)
	err = store.RegAddress(address, poolname)
	if err != nil {
		sqliteErr := err.(sqlite3.Error)
		if sqliteErr.ExtendedCode == sqlite3.ErrConstraintPrimaryKey {
			log.Printf("address exist %s\n", address)
		}
	} else {
		log.Printf("reg address:%s result:", address)
		log.Println(err)
	}

	maxjob := 1
	for j := 0; j < maxjob; j++ {
		go runworker(jobch, resultch, store, j)
		go readresult(resultch, j)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*10))
	defer cancel()
	go FetchDataTicker(ctx, store, jobch, []string{poolname})
	defer db.Close()

	exitSignal := make(chan os.Signal)
	signal.Notify(exitSignal, syscall.SIGINT, syscall.SIGTERM)
	<-exitSignal
}

func fetchData(ctx context.Context, store *storage.Storage, poolname string, address string) {
	log.Println("run fetchData: ", poolname, address)
	flexapi := poolapi.NewFlexAPI(FlexApiEndpoint, "")
	err, workers := flexapi.GetWorkers(ctx, address)
	if err != nil {
		log.Println("flexapi.GetWorkers error", err)
	} else {
		log.Printf("save address %s workers\n", address)
		savecount := store.SaveWorkerShares(poolname, address, workers)
		log.Println("save count:", savecount)
	}
}

func fetchBalance(ctx context.Context, store *storage.Storage, poolname string, address string) {
	log.Println("run fetchBalance: ", poolname, address)
	flexapi := poolapi.NewFlexAPI(FlexApiEndpoint, "")
	err, balance := flexapi.GetBalance(ctx, address)
	if err != nil {
		log.Println("flexapi.GetBalance error", err)
	} else {
		err = store.SaveBalance(poolname, address, balance)
		if err != nil {
			log.Println("fsave balance error", err)
		} else {
			log.Printf("balance saved")
		}
	}
}

func runworker(jobch chan Job, resultch chan Result, store *storage.Storage, jobid int) {
	for {
		select {
		case j := <-jobch:
			log.Printf("job %d data %s input: %d \n", j.Type, j.Address, jobid)
			if j.Type == JobFetchData {
				ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*10))
				defer cancel()
				go fetchData(ctx, store, j.Poolname, j.Address)
				//resultch <- Result{Type: j.Type}
			} else if j.Type == JobFetchBalance {
				ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*10))
				defer cancel()
				go fetchBalance(ctx, store, j.Poolname, j.Address)
			}
			time.Sleep(2 * time.Second)
		}
	}

}

func readresult(resultch chan Result, jobid int) {
	for {
		select {
		case r := <-resultch:
			log.Printf("result %d output: %d \n", r.Type, jobid)
		}
	}

}

func FetchDataTicker(ctx context.Context, store *storage.Storage, jobch chan Job, poolnames []string) {
	log.Println("run fetchDataTicker")
	dataTicker := time.NewTicker(time.Second * 60 * 5)
	for {
		select {
		case <-dataTicker.C:
			for _, poolname := range poolnames {
				err, addresses := store.GetAddresses(poolname)
				if err != nil {
					log.Println(err)
				} else {
					for _, address := range addresses {
						jobch <- Job{Type: JobFetchData, Address: address, Poolname: poolname}
						jobch <- Job{Type: JobFetchBalance, Address: address, Poolname: poolname}
					}
				}

			}
		}
	}
}
