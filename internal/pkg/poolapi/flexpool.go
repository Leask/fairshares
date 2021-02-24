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
	EffectiveHashrate        float32 `json:"effective_hashrate"`
	AverageEffectiveHashrate float32 `json:"average_effective_hashrate"`
	ValidShares              int     `json:"valid_shares"`
	StaleShares              int     `json:"stale_shares""`
	InvalidShares            int     `json:"invalid_shares"`
}

type FlexpoolWorkersChart struct {
	Error  *string                 `json:"error"`
	Result []*FlexpoolWorkerTicker `json:"result"`
}

type FlexpoolWorker struct {
	Name          string `json:"name"`
	Online        bool   `json:"online"`
	ValidShares   int    `json:"valid_shares"`
	StaleShares   int    `json:"stale_shares""`
	InvalidShares int    `json:"invalid_shares"`
	LastSeen      int64  `json:"last_seen"`
}

type FlexpoolWorkers struct {
	Error   *string           `json:"error"`
	Workers []*FlexpoolWorker `json:"result"`
}
type FlexpoolBalance struct {
	Error  *string `json:"error"`
	Result int64   `json:"result"`
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
	url := fmt.Sprintf("%s/miner/%s/balance/", api.Endpoint, address)
	err, jsondata := HttpGet(ctx, url, api.Apikey)

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
	return nil, resultbalance.Result
}

func (api *FlexAPI) GetWorkers(ctx context.Context, address string) (error, []*FlexpoolWorker) {
	resultworkers := &FlexpoolWorkers{}

	url := fmt.Sprintf("%s/miner/%s/workers/", api.Endpoint, address)
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

	url := fmt.Sprintf("%s/worker/%s/%s/chart/", api.Endpoint, address, name)
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
