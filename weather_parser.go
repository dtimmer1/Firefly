package main

import (
	"fmt"
	"net/http"
	"time"
)

/*
 * readWeather()
 * Process weather data for a chunk of the search area.
 * Launches a single goroutine that scans the region defined by lat0, lat1, long0, and long1,
 * sending GET requests to each 2.5x2.5km grid square.
 * Sends the output to the main goroutine via a generated channel.
 */

func readWeather(lat0 float64, lat1 float64, long0 float64, long1 float64) <-chan int {
	c := make(chan int)
	go func() {
		total := 0
		for lat := lat0; lat < lat1; lat += 0.02 { //2.5km is roughly 0.02 degrees of latitude/longitude
			for long := long0; long < long1; long += 0.02 {
				url := fmt.Sprintf("https://api.weather.gov/points/%f,%f", lat, long)
				val, err := http.Get(url) //val contains the body of the HTTP request, err contains any error code
				if err == nil && val != nil {
					total += 1 //for debugging purposes we're just returning the number of processed regions
				} else {
					fmt.Println("error!")
				}
				time.Sleep(time.Millisecond * time.Duration(100))
			}
		}
		c <- total
	}()

	return c
}

func main() {

	var outputs []<-chan int //array containing receive-only int channels

	lat0, lat1, long0, long1 := 40.97, 44.98, -111.03, -104.06
	latInterval := (lat1 - lat0) / 50

	for i := 0; i < 50; i++ {
		channel := readWeather(lat0, lat0+latInterval, long0, long1)
		outputs = append(outputs, channel) //appends to the array
		lat0 += latInterval
	}

	total := 0

	for i := 0; i < 50; i++ {
		total += <-outputs[i]
	}

	fmt.Printf("Main goroutine ended, processed %d regions\n", total)
}
