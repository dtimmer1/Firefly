package firefly

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Temp_max           int     `yaml:"temp_max"`
	Temp_min           int     `yaml:"temp_min"`
	Wspeed_max         int     `yaml:"wspeed_max"`
	Wspeed_min         int     `yaml:"wspeed_min"`
	Check_lat          float64 `yaml:"check_lat"`
	Check_long         float64 `yaml:"check_long"`
	Start_lat          float64 `yaml:"start_lat"`
	End_lat            float64 `yaml:"end_lat"`
	Start_long         float64 `yaml:"start_long"`
	End_long           float64 `yaml:"end_long"`
	Num_goroutines     int     `yaml:"num_goroutines"`
	Num_ranked_regions int     `yaml:"num_ranked_regions"`
}

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

var fconfig Config

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
		total++
		parts := strings.Split(i.WindSpeed, " ")
		speed, err := strconv.Atoi(parts[0])

		if err != nil {
			continue
		}
		if speed < fconfig.Wspeed_min || speed > fconfig.Wspeed_max {
			continue
		}
		if i.Temperature < fconfig.Temp_min || i.Temperature > fconfig.Temp_max {
			continue
		}

		count++
	}
	return Output{
		region.Geometry.Coordinates[0][1][1],
		region.Geometry.Coordinates[0][1][0],
		count,
		total,
	}
}

func ConfigInit(filePath string) (*Config, error) {
	config := &Config{}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	d := yaml.NewDecoder(file)
	if err := d.Decode(config); err != nil {
		return nil, err
	}

	fconfig = *config
	return config, nil
}

/*
 * scanWeather()
 * Process weather data for a chunk of the search area.
 * Launches a single goroutine that scans the region defined by lat0, lat1, long0, and long1,
 * sending GET requests to each 2.5x2.5km grid square.
 * Sends the output to the main goroutine via a generated channel.
 */

func ScanWeather(lat0 float64, lat1 float64, long0 float64, long1 float64, agg chan Output) {
	for lat := lat0; lat < lat1; lat += 0.02 { //2.5km is roughly 0.02 degrees of latitude/longitude
		for long := long0; long < long1; long += 0.02 {
			url := fmt.Sprintf("https://api.weather.gov/points/%f,%f", lat, long)
			val, err := SendGetRequest(url)
			if err != 0 {
				continue
			}

			dec := json.NewDecoder(val.Body)
			var forecast Response

			dec.Decode(&forecast)

			val, err = SendGetRequest(forecast.Properties.ForecastHourly)
			if err != 0 {
				continue
			}

			var data Contents
			dec = json.NewDecoder(val.Body)

			dec.Decode(&data)

			total := GetScore(data)
			agg <- total
		}
	}
}
