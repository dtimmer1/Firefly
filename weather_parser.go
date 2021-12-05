package main

import (
    "container/heap"
	"encoding/json"
	"firefly/firefly"
	"fmt"
	"os"
	"time"
    "log"
)

type OutputHeap []*firefly.Output

func (h OutputHeap) Len() int {
	return len(h)
}
func (h OutputHeap) Less(i, j int) bool {
	return h[i].Score < h[j].Score
}
func (h OutputHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *OutputHeap) Push(x interface{}) {
	// Push and Pop use pointer receivers because they modify the slice's length,
	// not just its contents.
	*h = append(*h, x.(*firefly.Output))
}

func (h *OutputHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

func (h *OutputHeap) Peek() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	return x
}

var cfg firefly.Config

func check(lat float64, long float64) {
	url := fmt.Sprintf("https://api.weather.gov/points/%f,%f", lat, long)
	val, err := firefly.SendGetRequest(url)
	for err != 0 {
		val, err = firefly.SendGetRequest(url)
	}

	dec := json.NewDecoder(val.Body)
	var forecast firefly.Response

	dec.Decode(&forecast)

	val, err = firefly.SendGetRequest(forecast.Properties.ForecastHourly)
	for err != 0 {
		val, err = firefly.SendGetRequest(forecast.Properties.ForecastHourly)
	}

	var data firefly.Contents
	dec = json.NewDecoder(val.Body)

	dec.Decode(&data)

	result := firefly.GetScore(data)
	fmt.Printf("region %f,%f has %d/%d valid times\n", result.Latitude, result.Longitude, result.Score, result.Total)
}

func scan(lat0 float64, lat1 float64, long0 float64, long1 float64) {
	latInterval := (lat1 - lat0) / float64(cfg.Num_goroutines)

	agg := make(chan firefly.Output, cfg.Num_goroutines)
	results := &OutputHeap{}
	heap.Init(results)

	for i := 0; i < cfg.Num_goroutines; i++ {
		go firefly.ScanWeather(lat0, lat0+latInterval, long0, long1, agg)
		time.Sleep(time.Millisecond * time.Duration(100))
		lat0 += latInterval
	}

	count := 0

Loop:
	for {
		select {
		case output := <-agg:
			if results.Len() < cfg.Num_ranked_regions || output.Score > results.Peek().(*firefly.Output).Score {
				heap.Push(results, &output)
				if results.Len() > cfg.Num_ranked_regions {
					heap.Pop(results)
				}
			}
			count++
			if count%1 == 0 {
				fmt.Printf("processed %d regions, current one is %f,%f\n", count, output.Latitude, output.Longitude)
			}

		case <-time.After(time.Second * time.Duration(2)):
			break Loop
		}
	}

	for results.Len() > 0 {
		result := heap.Pop(results).(*firefly.Output)
		fmt.Printf("region %f,%f has %d/%d valid times\n", result.Latitude, result.Longitude, result.Score, result.Total)
	}
}

func main() {
    config, err := firefly.ConfigInit("config.yml")
    cfg = *config
    if err != nil {
        log.Fatal(err)
    }

	switch os.Args[1] {
	case "check":
		check(cfg.Check_lat, cfg.Check_long)
	case "scan":
	    scan(cfg.Start_lat, cfg.End_lat, cfg.Start_long, cfg.End_long)
	default:
		fmt.Println("command must be one of check or scan")
	}
}
