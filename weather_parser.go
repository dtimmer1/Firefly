package main

import (
	"container/heap"
	"encoding/json"
	"firefly/firefly"
	"fmt"
	"log"
	"os"
	"time"
)

// All of this is declaration of a type that implements the heap interface, designed for use in region ranking.

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

var cfg firefly.Config // Stores a copy of the config in the scope of this file. This is primarily just for convenience

/*
 * Check()
 * Get the value for a given region and print out its score.
 */

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

/*
 * Scan()
 * Launch multiple goroutines to evaluate every region within a chunk and find the best regions.
 * We then log these regions in an output file in ascending order and the user can pick a location to burn.
 */

func scan(lat0 float64, lat1 float64, long0 float64, long1 float64) {
	latInterval := (lat1 - lat0) / float64(cfg.Num_goroutines)

	agg := make(chan firefly.Output, cfg.Num_goroutines)
	results := &OutputHeap{}
	heap.Init(results)

	for i := 0; i < cfg.Num_goroutines; i++ {
		go firefly.ScanWeather(lat0, lat0+latInterval, long0, long1, agg)
		time.Sleep(time.Millisecond * time.Duration(cfg.Scanner_delay))
		lat0 += latInterval
	}

	fmt.Println("Starting scanners...")

	count := 0

Loop: //we use a label to break out of both the infinite loop and the select statement
	for {
		select {
		case output := <-agg:

			/* The algorithm here is as follows:
			 * If our current heap size is smaller than the max size, append the output to the heap.
			 * If our heap is "full", then we add the output to the heap if it is higher-scoring than the smallest in the heap.
			 * Then, if adding this item overflowed the heap, we pop the smallest output region off the heap.
			 */

			if results.Len() < cfg.Num_ranked_regions || output.Score > results.Peek().(*firefly.Output).Score {
				heap.Push(results, &output)
				if results.Len() > cfg.Num_ranked_regions {
					heap.Pop(results)
				}
			}
			count++
			if count%cfg.Processing_interval == 0 {
				fmt.Printf("Processed %d regions...\n", count)
			}

		case <-time.After(time.Second * time.Duration(2)): //timeout period for the aggregation process
			break Loop
		}
	}

	// After all regions have been processed, we empty the heap, outputting each item to our output file and printing the best region.

	f, err := os.OpenFile(cfg.Output_file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	var output string

	for results.Len() > 0 {
		result := heap.Pop(results).(*firefly.Output)
		output = fmt.Sprintf("Region %f,%f Score: %d/%d\n", result.Latitude, result.Longitude, result.Score, result.Total)
		if _, err := f.Write([]byte(output)); err != nil {
			log.Fatal(err)
		}
	}

	fmt.Printf("\n##### Scan Complete: %d regions scanned. #####\n\n", count)
	fmt.Printf("Found the best region for a controlled burn. See %s for more locations.\n\n", cfg.Output_file)
	fmt.Println(output)
}

func main() {
	config, err := firefly.ConfigInit("config.yml") // Parses the configuration file
	cfg = *config
	if err != nil {
		log.Fatal(err)
	}

	logfile, err := os.OpenFile(cfg.Log_file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644) // Sets up the log file
	if err != nil {
		log.Fatal(err)
	}
	log.SetOutput(logfile)
	defer logfile.Close()

	switch os.Args[1] {
	case "check":
		check(cfg.Check_lat, cfg.Check_long)
	case "scan":
		scan(cfg.Start_lat, cfg.End_lat, cfg.Start_long, cfg.End_long)
	default:
		fmt.Println("command must be one of check or scan")
	}
}
