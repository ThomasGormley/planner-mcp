package main

import (
	"context"
	"encoding/json"
	"net/http"
)

func main() {
	http.ListenAndServe("localhost:4006", handler())
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

func handler() http.Handler {
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

			return &ToolResult{
				Content: []TextContent{
					{Type: "text", Text: state},
				},
			}, nil
		},
	}

	mux := http.NewServeMux()
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
		if toolRequest.Name == weatherTool.Name {
			// Create arguments structure
			params := ToolRunParams{
				Name: toolRequest.Name,
				Args: toolRequest.Arguments,
			}

			// Call the tool handler
			res, err := weatherTool.Run(r.Context(), params)
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

	return mux
}
