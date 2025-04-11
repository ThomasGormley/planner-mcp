package main

import (
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

	return mux
}
