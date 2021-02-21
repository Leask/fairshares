package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/huo-ju/fairshares/internal/pkg/poolapi"
	"github.com/huo-ju/fairshares/internal/pkg/storage"
	"github.com/mattn/go-sqlite3"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const FlexApiEndpoint = "https://flexpool.io/api/v1"
const DBVersion = 1

//internal/pkg/poolapi/flexpool.go

const (
	JobFetchData  = iota
	JobFetchBlock = iota
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

	jobch := make(chan Job)
	resultch := make(chan Result)

	address := "0x65146D70901C70188Eb02AeF452eEcCC3dA39208"
	poolname := "flexpool"

	db, err := sql.Open("sqlite3", "./faireshare.db")
	if err != nil {
		fmt.Println(err)
		return
	}
	store := storage.NewStorage(db)
	if store.DatabaseVersion() < DBVersion {
		err = store.NewDatabase()
	}

	fmt.Println("==store")
	fmt.Println(store)
	fmt.Println(err)
	ver := store.DatabaseVersion()
	fmt.Println(ver)
	err = store.RegAddress(address, poolname)
	if err != nil {
		sqliteErr := err.(sqlite3.Error)
		if sqliteErr.ExtendedCode == sqlite3.ErrConstraintPrimaryKey {
			fmt.Printf("address exist %s\n", address)
		}
	} else {
		fmt.Printf("reg address:%s result:", address)
		fmt.Println(err)
	}

	maxjob := 1
	for j := 0; j < maxjob; j++ {
		go runworker(jobch, resultch, store, j)
		go readresult(resultch, j)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*10))
	defer cancel()
	go FetchDataTicker(ctx, store, jobch, []string{poolname})

	//savecount := store.SaveWorkerShares(address, workers)
	//fmt.Printf("insert %d succ %d\n", len(workers), savecount)
	defer db.Close()

	exitSignal := make(chan os.Signal)
	signal.Notify(exitSignal, syscall.SIGINT, syscall.SIGTERM)
	<-exitSignal
}

func fetchData(ctx context.Context, store *storage.Storage, poolname string, address string) {
	fmt.Println("==run fetchData")
	flexapi := poolapi.NewFlexAPI(FlexApiEndpoint, "")
	err, workers := flexapi.GetWorkers(ctx, address)
	if err != nil {
		fmt.Println("flexapi.GetWorkers error")
		fmt.Println(err)
	} else {
		savecount := store.SaveWorkerShares(poolname, address, workers)
		fmt.Printf("save %d", savecount)
	}
}

func runworker(jobch chan Job, resultch chan Result, store *storage.Storage, jobid int) {
	for {
		select {
		case j := <-jobch:
			fmt.Printf("job %d data %s input: %d \n", j.Type, j.Address, jobid)
			if j.Type == JobFetchData {
				ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*10))
				defer cancel()
				go fetchData(ctx, store, j.Poolname, j.Address)
				//resultch <- Result{Type: j.Type}
			}
			time.Sleep(2 * time.Second)
		}
	}

}

func readresult(resultch chan Result, jobid int) {
	for {
		select {
		case r := <-resultch:
			fmt.Printf("result %d output: %d \n", r.Type, jobid)
		}
	}

}

func FetchDataTicker(ctx context.Context, store *storage.Storage, jobch chan Job, poolnames []string) {
	fmt.Println("run fetchDataTicker")
	dataTicker := time.NewTicker(time.Second * 10)
	for {
		select {
		case <-dataTicker.C:
			fmt.Println("ticker!")
			for _, poolname := range poolnames {
				fmt.Println(poolname)
				//flexapi := poolapi.NewFlexAPI(FlexApiEndpoint, "")
				err, addresses := store.GetAddresses(poolname)
				if err != nil {
					fmt.Println(err)
				} else {
					for _, address := range addresses {
						jobch <- Job{Type: JobFetchData, Address: address, Poolname: poolname}
					}
				}
			}
		}
	}
}
