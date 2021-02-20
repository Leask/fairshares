package poolapi

import (
	"encoding/json"
	"errors"
	"fmt"
)

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

type FlexAPI struct {
	Endpoint string
	Apikey   string
}

func NewFlexAPI(e, a string) *FlexAPI {
	return &FlexAPI{e, a}
}

func (api *FlexAPI) GetWorkers(address string) (error, []*FlexpoolWorker) {
	resultworkers := &FlexpoolWorkers{}

	url := fmt.Sprintf("%s/miner/%s/workers/", api.Endpoint, address)
	err, jsondata := HttpGet(url, api.Apikey)
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
