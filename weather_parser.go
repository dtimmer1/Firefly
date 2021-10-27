package main

import (
	"fmt"
	"net/http"
)

func readWeather(lat0 float64, lat1 float64, long0 float64, long1 float64) <-chan int {
	c := make(chan int)
	go func() {
		total := 0
		for lat := lat0; lat < lat1; lat += 0.05 {
			for long := long0; long < long1; long += 0.05 {
				url := fmt.Sprintf("https://api.weather.gov/points/%f,%f", lat, long)
				val, err := http.Get(url)
				if err == nil && val != nil {
					fmt.Println(url)
					total += 1
				} else {
					fmt.Println("error!")
				}
			}
		}
		c <- total
	}()

	return c
}

func main() {

	var outputs []<-chan int

	lat0, lat1, long0, long1 := 30., 31., -110., -109.
	latInterval := (lat1 - lat0) / 10
	longInterval := (long1 - long0) / 10

	for i := 0; i < 10; i++ {
		outputs = append(outputs, readWeather(lat0, lat0+latInterval, long0, long0+longInterval))
		lat0 += latInterval
		long0 += longInterval
	}

	total := 0

	for i := 0; i < 10; i++ {
		total += <-outputs[i]
	}

	fmt.Printf("Main goroutine ended, processed %d regions\n", total)
}
