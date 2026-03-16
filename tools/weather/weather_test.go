package weather

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jnb666/agent-go/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var apiKey = os.Getenv("OWM_API_KEY")

func TestCurrentWeather(t *testing.T) {
	tool := Current{}
	t.Log(util.Pretty(tool.Definition()))

	resp := tool.Call(context.Background(), `{"location":"London,GB"}`)
	t.Log(resp)
	assert.Contains(t, resp, "Current weather for London,GB")
}

func TestWeatherForecast(t *testing.T) {
	tool := Forecast{}
	t.Log(util.Pretty(tool.Definition()))

	resp := tool.Call(context.Background(), `{"location":"London,GB","periods":4}`)
	t.Log(resp)
	assert.Contains(t, resp, "Weather forecast for London,GB")
	assert.Equal(t, 5, strings.Count(resp, "\n"))
}

func TestGeocodingAPI(t *testing.T) {
	loc, err := geocoding(context.Background(), "New York,US", apiKey)
	require.NoError(t, err)
	t.Logf("%#v", loc)
	require.Equal(t, 1, len(loc))
	assert.Equal(t, "New York", loc[0].Name)
	assert.Equal(t, "US", loc[0].Country)
}

func TestCurrentAPI(t *testing.T) {
	w, err := currentWeather(context.Background(), "New York,US", apiKey)
	require.NoError(t, err)
	t.Log(w)
	assert.NotEmpty(t, w.weatherData)
}

func TestForecastAPI(t *testing.T) {
	w, err := weatherForecast(context.Background(), "New York,US", 8, apiKey)
	require.NoError(t, err)
	for _, entry := range w.List {
		t.Log(entry)
	}
	assert.Equal(t, 8, len(w.List))
}
