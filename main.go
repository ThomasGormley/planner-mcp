package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

func main() {
	weatherTool := Tool{
		Name:        "get-forecast",
		Description: "Get weather alerts for a state",
		InputSchema: ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"state": map[string]interface{}{
					"type":        "string",
					"description": "The state to get weather alerts for",
				},
			},
			Required: []string{"state"},
		},
		Handler: func(ctx context.Context, args ToolRunParams) (*ToolResult, error) {
			state := args.Args["state"].(string)
			weatherURL := "https://api.weather.gov/alerts?area=" + state

			data, err := GetAlerts(weatherURL)

			if err != nil {
				slog.Error("error getting alerts", "Err", err)
				return &ToolResult{
					Content: []TextContent{
						{Type: "text", Text: "Failed to retrieve alerts data"},
					},
				}, nil
			}

			if len(data) == 0 {
				return &ToolResult{
					Content: []TextContent{
						{Type: "text", Text: "No active alerts for " + state},
					},
				}, nil
			}

			formattedAlerts := make([]string, len(data))
			for i, f := range data {
				formattedAlerts[i] = FormatAlert(f)
			}

			// Build a properly formatted alert text with state name and all formatted alerts
			alertText := fmt.Sprintf("Active alerts for %s:\n\n%s", state, strings.Join(formattedAlerts, "\n"))

			return &ToolResult{
				Content: []TextContent{
					{Type: "text", Text: alertText},
				},
			}, nil
		},
	}

	http.ListenAndServe("localhost:4006", handle(HandleMcpParams{Tools: []Tool{weatherTool}}))
}

type HandleMcpParams struct {
	Tools []Tool
}

func handle(hmp HandleMcpParams) http.Handler {

	mux := http.NewServeMux()
	// Add a health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	mux.Handle("/mcp/", http.StripPrefix("/mcp", handleMcp(hmp.Tools...)))

	return mux
}

type Capabilities struct {
	Logging map[string]interface{} `json:"logging"`
	Prompts struct {
		ListChanged bool `json:"listChanged"`
	} `json:"prompts"`
	Resources struct {
		Subscribe   bool `json:"subscribe"`
		ListChanged bool `json:"listChanged"`
	} `json:"resources"`
	Tools struct {
		ListChanged bool `json:"listChanged"`
	} `json:"tools"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Result struct {
	ProtocolVersion string       `json:"protocolVersion"`
	Capabilities    Capabilities `json:"capabilities"`
	ServerInfo      ServerInfo   `json:"serverInfo"`
}

type ServerInitialization struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  Result `json:"result"`
}

func handleMcp(tools ...Tool) http.Handler {
	mux := http.NewServeMux()
	slog.Info("mounting mcp mux")
	toolMap := make(map[string]Tool)
	for _, t := range tools {
		toolMap[t.Name] = t
	}

	mux.HandleFunc("/mcp-health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	initialize := ServerInitialization{
		JSONRPC: "2.0",
		ID:      1,
		Result: Result{
			ProtocolVersion: "2024-11-05",
			Capabilities:    Capabilities{},
			ServerInfo: ServerInfo{
				Name:    "hermes-planner",
				Version: "0.0.1",
			},
		},
	}

	mux.HandleFunc("/initialize", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		err := json.NewEncoder(w).Encode(initialize)
		if err != nil {
			w.WriteHeader(500)
			return
		}
		return
	})

	mux.HandleFunc("/tools/call", func(w http.ResponseWriter, r *http.Request) {
		// Parse the incoming JSON request
		var toolRequest struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}

		if err := json.NewDecoder(r.Body).Decode(&toolRequest); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request format"})
			return
		}

		// Check if the requested tool is our weatherTool
		if tool, ok := toolMap[toolRequest.Name]; ok {
			// Create arguments structure
			params := ToolRunParams{
				Name: toolRequest.Name,
				Args: toolRequest.Arguments,
			}

			// Call the tool handler
			res, err := tool.Run(r.Context(), params)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}

			// Return a successful response
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "success",
				"result": res.Content,
			})
		} else {
			// Tool not found
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Tool not found: " + toolRequest.Name})
		}
	})

	mux.HandleFunc("/tools/list", func(w http.ResponseWriter, r *http.Request) {
		toolsList := ToolListResult{tools}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(&toolsList); err != nil {
			slog.Error("Failed to encode tools list", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to generate tools list"})
			return
		}
	})

	return mux
}
