package ticker

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gocraft/health"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

type fetchFn func() (exchangeRates, error)

// Fetch gets data from all sources, formats it, and sends it to the Writers.
func Fetch(stream *health.Stream, btcAvgPubkey string, btcAvgPrivkey string, writers ...Writer) error {
	job := stream.NewJob("fetch")

	// Fetch data from each provider
	allRates := []exchangeRates{{"BTC": {Ask: "1", Bid: "1", Last: "1"}}}
	for _, f := range []fetchFn{
		NewBTCAVGFetcher(btcAvgPubkey, btcAvgPrivkey),
		FetchCMC,
	} {
		rates, err := f()
		if err != nil {
			job.EventErr("fetch_data", err)
			job.Complete(health.Error)
			return err
		}
		allRates = append(allRates, rates)
	}

	// Format responses
	responseBytes, err := json.Marshal(mergeRates(allRates))
	if err != nil {
		job.EventErr("marshal", err)
		job.Complete(health.Error)
		return err
	}

	// Write
	for _, writer := range writers {
		err := writer(job, responseBytes)
		if err != nil {
			job.EventErr("write", err)
			job.Complete(health.Error)
			return err
		}
	}

	job.Complete(health.Success)
	return nil
}