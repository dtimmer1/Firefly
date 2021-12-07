## Firefly
Firefly is a cyber-physical system designed to facilitate controlled burns for wildfire prevention.

# Usage
To compile the Firefly program provided:

```go build weather_parser.go```

To run:

```./weather_parser [check/scan]```

The locations to be evaluated, as well as many other configuration parameters, are updatable in the ```config.yml``` file.

# Features
Firefly evaluates geographic regions and their potential weather conditions for controlled burns.
By scanning a wide area, Firefly can use the National Weather Service API to narrow down the most likely locations for a safe controlled burn.


Additionally, Firefly is designed for integration with the Paparazzi UAV system, and a sample XML flight plan for Paparazzi is found in ```firefly_flight.xml```. This flight plan programs a unmanned aerial vehicle (UAV) to fly within a geographic location selected by the Firefly driver program, dropping flammable payloads in select locations to ignite from a safe distance. This process can been simulated in the Paparazzi simulation engine.

