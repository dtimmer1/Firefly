package main

import (
	"container/heap"
	"encoding/json"
	"firefly/firefly"
	"flag"
	"fmt"
	"os"
	"time"
)

// Slice of Intervals
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

/*
 * scanWeather()
 * Process weather data for a chunk of the search area.
 * Launches a single goroutine that scans the region defined by lat0, lat1, long0, and long1,
 * sending GET requests to each 2.5x2.5km grid square.
 * Sends the output to the main goroutine via a generated channel.
 */

func scanWeather(lat0 float64, lat1 float64, long0 float64, long1 float64, agg chan firefly.Output) {
	for lat := lat0; lat < lat1; lat += 0.02 { //2.5km is roughly 0.02 degrees of latitude/longitude
		for long := long0; long < long1; long += 0.02 {
			url := fmt.Sprintf("https://api.weather.gov/points/%f,%f", lat, long)
			val, err := firefly.SendGetRequest(url)
			if err != 0 {
				continue
			}

			dec := json.NewDecoder(val.Body)
			var forecast firefly.Response

			dec.Decode(&forecast)

			val, err = firefly.SendGetRequest(forecast.Properties.ForecastHourly)
			if err != 0 {
				continue
			}

			var data firefly.Contents
			dec = json.NewDecoder(val.Body)

			dec.Decode(&data)

			total := firefly.GetScore(data)
			agg <- total
		}
	}
}

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
	latInterval := (lat1 - lat0) / 8

	agg := make(chan firefly.Output, 8)
	results := &OutputHeap{}
	heap.Init(results)

	for i := 0; i < 8; i++ {
		go scanWeather(lat0, lat0+latInterval, long0, long1, agg)
		time.Sleep(time.Millisecond * time.Duration(100))
		lat0 += latInterval
	}

	count := 0

Loop:
	for {
		select {
		case output := <-agg:
			if results.Len() < 10 || output.Score > results.Peek().(*firefly.Output).Score {
				heap.Push(results, &output)
				if results.Len() > 10 {
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
	checker := flag.NewFlagSet("check", flag.ContinueOnError)
	check_lat := checker.Float64("lat", 0.0, "latitude to check")
	check_long := checker.Float64("long", 0.0, "longitude to check")

	scanner := flag.NewFlagSet("scan", flag.ContinueOnError)
	scan_lat0 := scanner.Float64("start_lat", 0.0, "beginning of latitude region")
	scan_lat1 := scanner.Float64("end_lat", 0.0, "end of latitude region")
	scan_long0 := scanner.Float64("start_long", 0.0, "beginning of longitude region")
	scan_long1 := scanner.Float64("end_long", 0.0, "end of longitude region")

	switch os.Args[1] {
	case "check":
		if err := checker.Parse(os.Args[2:]); err == nil {
			check(*check_lat, *check_long)
		}
	case "scan":
		if err := scanner.Parse(os.Args[2:]); err == nil {
			scan(*scan_lat0, *scan_lat1, *scan_long0, *scan_long1)
		}
	default:
		fmt.Println("command must be one of check or scan")
	}
}
