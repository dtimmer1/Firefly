package firefly

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Response struct {
	Properties struct {
		ForecastHourly string
	}
}

type ForecastData struct {
	Temperature   int
	WindSpeed     string
	WindDirection string
}

type Contents struct {
	Geometry struct {
		Coordinates [][][]float64
	}
	Properties struct {
		Periods []ForecastData
	}
}

type Output struct {
	Latitude  float64
	Longitude float64
	Score     int
	Total     int
}

func SendGetRequest(url string) (*http.Response, int) {
	val, err := http.Get(url)
	if err != nil || val.StatusCode != 200 {
		log.Printf("access denied! status code is %d\n", val.StatusCode)
		return nil, 1
	}
	time.Sleep(time.Millisecond * time.Duration(50))
	return val, 0
}

func GetScore(region Contents) Output {
	count := 0
	total := 0
	for _, i := range region.Properties.Periods {
		parts := strings.Split(i.WindSpeed, " ")
		speed, err := strconv.Atoi(parts[0])

		if err == nil {
			if speed >= 3 && speed <= 7 {
				if i.Temperature >= 40 && i.Temperature <= 60 {
					count++
				}
			}
		}
		total++
	}
	return Output{
		region.Geometry.Coordinates[0][1][1],
		region.Geometry.Coordinates[0][1][0],
		count,
		total,
	}
}
