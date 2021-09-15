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

const FlexApiEndpoint = "https://api.flexpool.io/v2"
const DBVersion = 1

//internal/pkg/poolapi/flexpool.go

const (
	JobFetchWorkers = iota
	JobFetchChart   = iota
	JobFetchBalance = iota
)

type Job struct {
	Id         int
	Poolname   string
	Address    string
	Workername string
	Type       int
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
	//address := "0xc3722311F6f43476174C1ea46E09fF07f007C386"
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
	go FetchBalanceTicker(ctx, store, jobch, []string{poolname})
	defer db.Close()

	exitSignal := make(chan os.Signal)
	signal.Notify(exitSignal, syscall.SIGINT, syscall.SIGTERM)
	<-exitSignal
}

func fetchWorker(ctx context.Context, jobch chan Job, store *storage.Storage, poolname string, address string) {
	log.Println("run fetchWorker: ", poolname, address)
	flexapi := poolapi.NewFlexAPI(FlexApiEndpoint, "")
	err, workers := flexapi.GetWorkers(ctx, address)
	if err != nil {
		log.Println("flexapi.GetWorkers error", err)
	} else {
		for _, worker := range workers {
			log.Printf("add fetch chart job poolname %s address %s workers %s\n", poolname, address, worker.Name)
			jobch <- Job{Type: JobFetchChart, Address: address, Poolname: poolname, Workername: worker.Name}
		}
		//log.Printf("save address %s workers\n", address)
		//savecount := store.SaveWorkerShares(poolname, address, workers)
		//log.Println("save count:", savecount)
	}
}

func fetchWorkerChart(ctx context.Context, store *storage.Storage, poolname string, address string, workername string) {
	log.Println("run fetchWorkerChart: ", poolname, address, workername)
	flexapi := poolapi.NewFlexAPI(FlexApiEndpoint, "")
	err, result := flexapi.GetWorkersChart(ctx, address, workername)
	if err != nil {
		log.Println("flexapi.GetWorkersChart error", err)
	} else {
		store.SaveWorkerChart(poolname, address, workername, result)
		//log.Printf("save address %s workers\n", address)
		//savecount := store.SaveWorkerShares(poolname, address, workers)
		//log.Println("save count:", savecount)
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
			log.Printf("save balance error: %s %s %d\n", err, address, balance)
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
			if j.Type == JobFetchWorkers {
				ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*10))
				defer cancel()
				go fetchWorker(ctx, jobch, store, j.Poolname, j.Address)
				//resultch <- Result{Type: j.Type}
			} else if j.Type == JobFetchChart {
				ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*10))
				defer cancel()
				go fetchWorkerChart(ctx, store, j.Poolname, j.Address, j.Workername)
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
	dataTicker := time.NewTicker(time.Second * 60 * 30)
	for {
		select {
		case <-dataTicker.C:
			for _, poolname := range poolnames {
				err, addresses := store.GetAddresses(poolname)
				if err != nil {
					log.Println(err)
				} else {
					for _, address := range addresses {
						jobch <- Job{Type: JobFetchWorkers, Address: address, Poolname: poolname}
					}
				}

			}
		}
	}
}

func FetchBalanceTicker(ctx context.Context, store *storage.Storage, jobch chan Job, poolnames []string) {
	log.Println("run fetchBalanceTicker")
	dataTicker := time.NewTicker(time.Second * 60 * 10)
	for {
		select {
		case <-dataTicker.C:
			for _, poolname := range poolnames {
				err, addresses := store.GetAddresses(poolname)
				if err != nil {
					log.Println(err)
				} else {
					for _, address := range addresses {
						jobch <- Job{Type: JobFetchBalance, Address: address, Poolname: poolname}
					}
				}

			}
		}
	}
}
