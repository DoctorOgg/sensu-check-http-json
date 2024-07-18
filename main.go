package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/itchyny/gojq"
	"github.com/sensu-community/sensu-plugin-sdk/sensu"
	"github.com/sensu/sensu-go/types"
)

// Config holds the configuration options for the Sensu plugin.
type Config struct {
	sensu.PluginConfig
	Debug              bool   // Enable debug mode
	Expression         string // Expression for comparing result of query
	InsecureSkipVerify bool   // Skip TLS certificate verification (not recommended!)
	Query              string // Query for extracting value from JSON
	Timeout            int    // Request timeout in seconds
	URL                string // URL to test
}

var (
	config = Config{
		PluginConfig: sensu.PluginConfig{
			Name:     "sensu-check-http-json",
			Short:    "HTTP JSON Sensu Check",
			Keyspace: "sensu.io/plugins/sensu-check-http-json/config",
		},
	}

	options = []*sensu.PluginConfigOption{
		{
			Path:      "debug",
			Env:       "SENSU_CHECK_DEBUG",
			Argument:  "debug",
			Shorthand: "d",
			Default:   false,
			Usage:     "Enable debug mode",
			Value:     &config.Debug,
		},
		{
			Path:      "expression",
			Env:       "SENSU_CHECK_EXPRESSION",
			Argument:  "expression",
			Shorthand: "e",
			Default:   "",
			Usage:     "Expression for comparing result of query",
			Value:     &config.Expression,
		},
		{
			Path:      "insecure-skip-verify",
			Env:       "SENSU_CHECK_INSECURE_SKIP_VERIFY",
			Argument:  "insecure-skip-verify",
			Shorthand: "i",
			Default:   false,
			Usage:     "Skip TLS certificate verification (not recommended!)",
			Value:     &config.InsecureSkipVerify,
		},
		{
			Path:      "query",
			Env:       "SENSU_CHECK_QUERY",
			Argument:  "query",
			Shorthand: "q",
			Default:   "",
			Usage:     "Query for extracting value from JSON",
			Value:     &config.Query,
		},
		{
			Path:      "timeout",
			Env:       "SENSU_CHECK_TIMEOUT",
			Argument:  "timeout",
			Shorthand: "T",
			Default:   15,
			Usage:     "Request timeout in seconds",
			Value:     &config.Timeout,
		},
		{
			Path:      "url",
			Env:       "SENSU_CHECK_URL",
			Argument:  "url",
			Shorthand: "u",
			Default:   "http://localhost:80/",
			Usage:     "URL to test",
			Value:     &config.URL,
		},
	}
)

func main() {
	check := sensu.NewGoCheck(&config.PluginConfig, options, checkArgs, executeCheck, false)
	// Call Execute without expecting a return value
	check.Execute()
	// After Execute, if you need to handle specific cases, you can do so here
	// For example, you might want to log a message or set an exit status based on internal logic
}

// checkArgs checks if essential configuration parameters are provided.
func checkArgs(event *types.Event) (int, error) {
	if config.Expression == "" {
		return sensu.CheckStateWarning, fmt.Errorf("expression is required")
	}
	if config.Query == "" {
		return sensu.CheckStateWarning, fmt.Errorf("query is required")
	}
	if config.URL == "" {
		return sensu.CheckStateWarning, fmt.Errorf("url is required")
	}
	return sensu.CheckStateOK, nil
}

// executeCheck implements the actual check logic using encoding/json.
func executeCheck(event *types.Event) (int, error) {
	// Make HTTP GET request to URL
	client := &http.Client{Timeout: time.Duration(config.Timeout) * time.Second}
	resp, err := client.Get(config.URL)
	if err != nil {
		return sensu.CheckStateCritical, fmt.Errorf("failed to fetch URL: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return sensu.CheckStateCritical, fmt.Errorf("failed to read response body: %v", err)
	}

	// Parse JSON response into a map[string]interface{}
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return sensu.CheckStateCritical, fmt.Errorf("failed to unmarshal JSON response: %v", err)
	}

	// Debug log the parsed JSON data if debug mode is enabled
	if config.Debug {
		fmt.Println("Parsed JSON data:", data)
	}

	// Prepare query with gojq
	query, err := gojq.Parse(config.Query)
	if err != nil {
		return sensu.CheckStateCritical, fmt.Errorf("failed to parse jq query: %v", err)
	}

	// Use gojq to evaluate query against data
	iter := query.Run(data)
	var extractedValue float64
	valueExtracted := false
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}

		// Convert the extracted value to a float64 for comparison (assuming numeric comparison)
		strValue := fmt.Sprintf("%v", v)
		if strValue == "null" {
			continue
		}

		value, err := strconv.ParseFloat(strings.TrimSpace(strValue), 64)
		if err != nil {
			return sensu.CheckStateCritical, fmt.Errorf("failed to parse extracted value: %v", err)
		}
		extractedValue = value
		valueExtracted = true
	}

	if !valueExtracted {
		return sensu.CheckStateCritical, fmt.Errorf("no valid value extracted")
	}

	// Evaluate the comparison expression
	switch {
	case config.Expression == "":
		// If no expression is provided, assume success
		return sensu.CheckStateOK, nil
	case startsWith(config.Expression, ">"):
		valueToCompare, err := strconv.ParseFloat(strings.TrimSpace(config.Expression[1:]), 64)
		if err != nil {
			fmt.Printf("failed to parse expression value (%s): %v", config.Query, err)
			return sensu.CheckStateCritical, nil
		}
		if !(extractedValue > valueToCompare) {
			fmt.Printf("expression check failed (%s): %.2f <= %.2f", config.Query, extractedValue, valueToCompare)
			return sensu.CheckStateCritical, nil
		}
	case startsWith(config.Expression, "<"):
		valueToCompare, err := strconv.ParseFloat(strings.TrimSpace(config.Expression[1:]), 64)
		if err != nil {
			fmt.Printf("failed to parse expression value (%s): %v", config.Query, err)
			return sensu.CheckStateCritical, nil
		}
		if !(extractedValue < valueToCompare) {
			fmt.Printf("expression check failed (%s): %.2f >= %.2f", config.Query, extractedValue, valueToCompare)
			return sensu.CheckStateCritical, nil
		}
	default:
		// Handle other comparison operators as needed
		fmt.Printf("unsupported expression (%s) : %s", config.Query, config.Expression)
		return sensu.CheckStateWarning, nil
	}
	fmt.Printf("expression check passed (%s): %.2f %s", config.Query, extractedValue, config.Expression)
	return sensu.CheckStateOK, nil
}

// startsWith checks if a string starts with a specified prefix.
func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[0:len(prefix)] == prefix
}

// extractValue extracts the value from JSON based on the provided query.
func extractValue(data map[string]interface{}, query string) (float64, error) {
	// Example implementation: you can implement your logic to parse 'query' and extract the appropriate value
	// Here we assume that 'query' is a simple dot-separated path, e.g., "key1.key2"

	keys := strings.Split(query, ".")
	var value interface{} = data
	for _, key := range keys {
		if val, ok := value.(map[string]interface{}); ok {
			value = val[key]
		} else {
			return 0, fmt.Errorf("invalid query path")
		}
	}

	// Convert the value to a float64 for comparison
	switch v := value.(type) {
	case float64:
		return v, nil
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, fmt.Errorf("failed to convert string to float: %v", err)
		}
		return f, nil
	default:
		return 0, fmt.Errorf("unsupported value type")
	}
}
