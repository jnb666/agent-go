// Package weather implements tools to get the current and forecast weather from https://api.openweathermap.org/
package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"os"
	"time"

	"github.com/jnb666/agent-go/agents"
	"github.com/jnb666/agent-go/llm"
	"github.com/jnb666/agent-go/util"
	log "github.com/sirupsen/logrus"
)

var Tools = []agents.Tool{Current{}, Forecast{}}

// Tool to get current weather - implements agent.Tool interface
type Current struct{}

func (Current) Definition() llm.FunctionDefinition {
	return llm.FunctionDefinition{
		Name:        "get_current_weather",
		Description: "Get the current weather in a given location.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"location": map[string]any{
					"type":        "string",
					"description": `The city name and ISO 3166 country code, e.g. "London,GB" or "New York,US".`,
				},
			},
			"required": []string{"location"},
		},
	}
}

func (Current) Call(ctx context.Context, arg string) string {
	apiKey := os.Getenv("OWM_API_KEY")
	if apiKey == "" {
		return "Error calling get_current_weather - OWM_API_KEY environment variable is not set"
	}
	log.Infof("call get_current_weather(%s)", arg)
	var args struct {
		Location string
	}
	if err := json.Unmarshal([]byte(arg), &args); err != nil {
		return fmt.Sprintf("Error parsing arguments for get_current_weather: %s", err)
	}
	w, err := currentWeather(ctx, args.Location, apiKey)
	if err != nil {
		return "Error: " + err.Error()
	}
	return w.String()
}

// Tool to get weather forecast data - implements agent.Tool interface
type Forecast struct{}

func (Forecast) Definition() llm.FunctionDefinition {
	return llm.FunctionDefinition{
		Name:        "get_weather_forecast",
		Description: "Get the weather forecast in a given location. Returns a list with date and time in local timezone and predicted conditions every 3 hours.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"location": map[string]any{
					"type":        "string",
					"description": `The city name and ISO 3166 country code, e.g. "London,GB" or "New York,US".`,
				},
				"periods": map[string]any{
					"type":        "number",
					"description": `Number of 3 hour periods to look ahead from current time - default 24.`,
				},
			},
			"required": []string{"location"},
		},
	}
}

func (Forecast) Call(ctx context.Context, arg string) string {
	apiKey := os.Getenv("OWM_API_KEY")
	if apiKey == "" {
		return "Error calling get_weather_forecast - OWM_API_KEY environment variable is not set"
	}
	log.Infof("call get_weather_forecast(%s)", arg)
	var args struct {
		Location string
		Periods  float64
	}
	if err := json.Unmarshal([]byte(arg), &args); err != nil {
		return fmt.Sprintf("Error parsing arguments for get_weather_forecast: %s", err)
	}
	if args.Periods == 0 {
		args.Periods = 24
	}
	w, err := weatherForecast(ctx, args.Location, int(args.Periods), apiKey)
	if err != nil {
		return "Error: " + err.Error()
	}
	return w.String()
}

// Current weather API per https://openweathermap.org/current
func currentWeather(ctx context.Context, location string, apiKey string) (w currentWeatherData, err error) {
	locs, err := geocoding(ctx, location, apiKey)
	if err != nil {
		return w, err
	}
	loc := locs[0]
	uri := fmt.Sprintf("https://api.openweathermap.org/data/2.5/weather?lat=%f&lon=%f&appid=%s&units=metric",
		loc.Lat, loc.Lon, apiKey)
	w.Loc = loc
	err = util.Get(ctx, uri, &w)
	if err == nil && len(w.Weather) == 0 {
		err = fmt.Errorf("current weather for %s not found", loc)
	}
	return w, err
}

type currentWeatherData struct {
	weatherData
	Timezone int
	Loc      location
}

func (w currentWeatherData) String() string {
	return fmt.Sprintf("Current weather for %s,%s: %s", w.Loc.Name, w.Loc.Country, w.weatherData)
}

type weatherData struct {
	Dt      int
	Weather []struct {
		Description string
	}
	Main struct {
		Temp       float64
		Feels_Like float64
	}
	Wind struct {
		Speed float64
	}
}

func (w weatherData) String() string {
	s := fmt.Sprintf("%.0f°C - %s", w.Main.Temp, w.Weather[0].Description)
	if w.Main.Feels_Like != 0 && math.Abs(w.Main.Feels_Like-w.Main.Temp) > 1 {
		s += fmt.Sprintf(", feels like %.0f°C", w.Main.Feels_Like)
	}
	if w.Wind.Speed != 0 {
		s += fmt.Sprintf(", wind %.1fm/s", w.Wind.Speed)
	}
	return s
}

// 5 day weather forecast API per https://openweathermap.org/forecast5
func weatherForecast(ctx context.Context, location string, periods int, apiKey string) (w forecastWeatherData, err error) {
	locs, err := geocoding(ctx, location, apiKey)
	if err != nil {
		return w, err
	}
	loc := locs[0]
	uri := fmt.Sprintf("https://api.openweathermap.org/data/2.5/forecast?lat=%f&lon=%f&cnt=%d&appid=%s&units=metric",
		loc.Lat, loc.Lon, periods, apiKey)
	w.Loc = loc
	err = util.Get(ctx, uri, &w)
	if err == nil && len(w.List) == 0 {
		err = fmt.Errorf("weather forecast for %s not found", loc)
	}
	return w, err
}

type forecastWeatherData struct {
	List []weatherData
	Loc  location
	City struct {
		Timezone int
	}
}

func (w forecastWeatherData) String() string {
	s := fmt.Sprintf("Weather forecast for %s,%s:\n", w.Loc.Name, w.Loc.Country)
	for _, r := range w.List {
		s += fmt.Sprintf("- %s: %s\n", localtime(r.Dt, w.City.Timezone), r)
	}
	return s
}

func localtime(dt, timezone int) string {
	t := time.Unix(int64(dt), 0)
	loc := time.FixedZone("", timezone)
	return t.In(loc).Format("Mon 2 Jan 2006 - 3 PM")
}

// Geocoding API per https://openweathermap.org/api/geocoding-api
func geocoding(ctx context.Context, location, apiKey string) (loc []location, err error) {
	if apiKey == "" {
		return nil, fmt.Errorf("openweathermap api key is required")
	}
	uri := fmt.Sprintf("http://api.openweathermap.org/geo/1.0/direct?q=%s&&appid=%s",
		url.QueryEscape(location), apiKey)
	err = util.Get(ctx, uri, &loc)
	if err == nil && len(loc) == 0 {
		err = fmt.Errorf("location %q not found", location)
	}
	return loc, err
}

type location struct {
	Name    string
	Country string
	State   string
	Lat     float64
	Lon     float64
}

func (l location) String() string {
	return fmt.Sprintf("%s,%s", l.Name, l.Country)
}
