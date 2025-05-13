package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const USER_AGENT = "Weather Tool (your-email@example.com)"

// makeNWSRequest makes a request to the NWS API
func makeNWSRequest(url string) (map[string]interface{}, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("User-Agent", USER_AGENT)
	req.Header.Set("Accept", "application/geo+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error! status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %w", err)
	}

	return result, nil
}

// AlertFeature represents a weather alert feature
type AlertFeature struct {
	Properties struct {
		Event    string `json:"event"`
		AreaDesc string `json:"areaDesc"`
		Severity string `json:"severity"`
		Status   string `json:"status"`
		Headline string `json:"headline"`
	} `json:"properties"`
}

// FormatAlert formats an alert feature for display
func FormatAlert(feature AlertFeature) string {
	props := feature.Properties

	event := props.Event
	if event == "" {
		event = "Unknown"
	}

	area := props.AreaDesc
	if area == "" {
		area = "Unknown"
	}

	severity := props.Severity
	if severity == "" {
		severity = "Unknown"
	}

	status := props.Status
	if status == "" {
		status = "Unknown"
	}

	headline := props.Headline
	if headline == "" {
		headline = "No headline"
	}

	return strings.Join([]string{
		fmt.Sprintf("Event: %s", event),
		fmt.Sprintf("Area: %s", area),
		fmt.Sprintf("Severity: %s", severity),
		fmt.Sprintf("Status: %s", status),
		fmt.Sprintf("Headline: %s", headline),
		"---",
	}, "\n")
}

// ForecastPeriod represents a single period in a weather forecast
type ForecastPeriod struct {
	Name            string `json:"name"`
	Temperature     int    `json:"temperature"`
	TemperatureUnit string `json:"temperatureUnit"`
	WindSpeed       string `json:"windSpeed"`
	WindDirection   string `json:"windDirection"`
	ShortForecast   string `json:"shortForecast"`
}

// AlertsResponse represents the response from the alerts endpoint
type AlertsResponse struct {
	Features []AlertFeature `json:"features"`
}

// PointsResponse represents the response from the points endpoint
type PointsResponse struct {
	Properties struct {
		Forecast string `json:"forecast"`
	} `json:"properties"`
}

// ForecastResponse represents the response from the forecast endpoint
type ForecastResponse struct {
	Properties struct {
		Periods []ForecastPeriod `json:"periods"`
	} `json:"properties"`
}

// GetAlerts fetches weather alerts for a given area
func GetAlerts(url string) ([]AlertFeature, error) {
	data, err := makeNWSRequest(url)
	if err != nil {
		return nil, err
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("error re-marshaling data: %w", err)
	}

	var alertsResp AlertsResponse
	if err := json.Unmarshal(jsonData, &alertsResp); err != nil {
		return nil, fmt.Errorf("error parsing alerts response: %w", err)
	}

	return alertsResp.Features, nil
}

// GetForecast fetches forecast data for a given location
func GetForecast(pointsURL string) ([]ForecastPeriod, error) {
	// First get the forecast URL from the points endpoint
	pointsData, err := makeNWSRequest(pointsURL)
	if err != nil {
		return nil, err
	}

	jsonData, err := json.Marshal(pointsData)
	if err != nil {
		return nil, fmt.Errorf("error re-marshaling points data: %w", err)
	}

	var pointsResp PointsResponse
	if err := json.Unmarshal(jsonData, &pointsResp); err != nil {
		return nil, fmt.Errorf("error parsing points response: %w", err)
	}

	if pointsResp.Properties.Forecast == "" {
		return nil, fmt.Errorf("no forecast URL found in points response")
	}

	// Now get the forecast data
	forecastData, err := makeNWSRequest(pointsResp.Properties.Forecast)
	if err != nil {
		return nil, err
	}

	jsonData, err = json.Marshal(forecastData)
	if err != nil {
		return nil, fmt.Errorf("error re-marshaling forecast data: %w", err)
	}

	var forecastResp ForecastResponse
	if err := json.Unmarshal(jsonData, &forecastResp); err != nil {
		return nil, fmt.Errorf("error parsing forecast response: %w", err)
	}

	return forecastResp.Properties.Periods, nil
}
