package poolapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

//{
//  "effective_hashrate": 0,
//  "average_effective_hashrate": 31333333,
//  "reported_hashrate": 0,
//  "valid_shares": 0,
//  "stale_shares": 0,
//  "invalid_shares": 0
//},

type FlexpoolWorkerTicker struct {
	Name                     string
	Timestamp                int64   `json:"timestamp"`
	EffectiveHashrate        float32 `json:"effectiveHashrate"`
	AverageEffectiveHashrate float32 `json:"averageEffectiveHashrate"`
	ValidShares              int     `json:"validShares"`
	StaleShares              int     `json:"staleShares""`
	InvalidShares            int     `json:"invalidShares"`
}

type FlexpoolWorkersChart struct {
	Error  *string                 `json:"error"`
	Result []*FlexpoolWorkerTicker `json:"result"`
}

type FlexpoolWorker struct {
	Name          string `json:"name"`
	Online        bool   `json:"isOnline"`
	ValidShares   int    `json:"validShares"`
	StaleShares   int    `json:"staleShares""`
	InvalidShares int    `json:"invalidShares"`
	LastSeen      int64  `json:"lastSeen"`
}

type FlexpoolWorkers struct {
	Error   *string           `json:"error"`
	Workers []*FlexpoolWorker `json:"result"`
}

type BalanceResult struct {
	Balance int64 `json:"balance"`
}
type FlexpoolBalance struct {
	Error  *string       `json:"error"`
	Result BalanceResult `json:"result"`
	//{"error":null,"result":{"balance":484337889491274530,"balanceCountervalue":1589.28,"price":3281.34}}
}

type FlexAPI struct {
	Endpoint string
	Apikey   string
}

func NewFlexAPI(e, a string) *FlexAPI {
	return &FlexAPI{e, a}
}

func (api *FlexAPI) GetBalance(ctx context.Context, address string) (error, int64) {
	resultbalance := &FlexpoolBalance{}
	url := fmt.Sprintf("%s/miner/balance?address=%s&coin=eth&countervalue=usd", api.Endpoint, address)
	fmt.Println(url)

	err, jsondata := HttpGet(ctx, url, api.Apikey)

	fmt.Println(string(jsondata))
	if err != nil {
		return err, 0
	}

	err = json.Unmarshal(jsondata, &resultbalance)

	if err != nil {
		return err, 0
	}

	if resultbalance.Error != nil {
		return errors.New(*resultbalance.Error), 0
	}
	return nil, resultbalance.Result.Balance
}

func (api *FlexAPI) GetWorkers(ctx context.Context, address string) (error, []*FlexpoolWorker) {
	resultworkers := &FlexpoolWorkers{}
	url := fmt.Sprintf("%s/miner/workers?address=%s&coin=eth", api.Endpoint, address)
	fmt.Println(url)
	err, jsondata := HttpGet(ctx, url, api.Apikey)
	if err != nil {
		return err, nil
	}

	err = json.Unmarshal(jsondata, &resultworkers)

	if err != nil {
		return err, nil
	}

	if resultworkers.Error != nil {
		return errors.New(*resultworkers.Error), nil
	}
	return nil, resultworkers.Workers
}

func (api *FlexAPI) GetWorkersChart(ctx context.Context, address string, name string) (error, []*FlexpoolWorkerTicker) {
	resultworkerchart := &FlexpoolWorkersChart{}
	url := fmt.Sprintf("%s/miner/chart?address=%s&coin=eth&worker=%s", api.Endpoint, address, name)
	err, jsondata := HttpGet(ctx, url, api.Apikey)
	if err != nil {
		return err, nil
	}

	err = json.Unmarshal(jsondata, &resultworkerchart)

	if err != nil {
		return err, nil
	}

	if resultworkerchart.Error != nil {
		return errors.New(*resultworkerchart.Error), nil
	}
	return nil, resultworkerchart.Result
}
