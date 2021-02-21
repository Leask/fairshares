package poolapi

import (
	"context"
	"io/ioutil"
	"net/http"
)

func HttpGet(ctx context.Context, url, apiKey string) (error, []byte) {
	request, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err, nil
	}

	if len(apiKey) > 0 {
		request.Header.Set("X-Auth-Token", apiKey)
	}

	var client = http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return err, nil
	}
	defer response.Body.Close()

	jsonByte, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err, nil
	}

	return nil, jsonByte
}
