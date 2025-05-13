package main

import (
	"context"
	"fmt"
)

type ToolInputSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Required   []string               `json:"required,omitempty"`
}

type ToolRunParams struct {
	Name string
	Args map[string]any
}

type TextContent struct {
	Type string `json:"type"` // Must be "text"
	// The text content of the message.
	Text string `json:"text"`
}

type ToolResult struct {
	Content []TextContent `json:"content"`
}

type ToolListResult struct {
	Tools []Tool `json:"tools"`
}

type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema ToolInputSchema `json:"inputSchema"`
	Handler     ToolHandlerFunc `json:"-"`
}

type ToolHandlerFunc func(context.Context, ToolRunParams) (*ToolResult, error)

// Helper function to validate if a value matches the expected JSON schema type
func validateType(value interface{}, expectedType string) bool {
	switch expectedType {
	case "string":
		_, ok := value.(string)
		return ok
	case "number":
		_, floatOk := value.(float64)
		_, intOk := value.(int)
		return floatOk || intOk
	case "integer":
		_, ok := value.(int)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "array":
		_, ok := value.([]interface{})
		return ok
	case "object":
		_, ok := value.(map[string]interface{})
		return ok
	case "null":
		return value == nil
	default:
		return false
	}
}

func (t *Tool) Run(ctx context.Context, params ToolRunParams) (*ToolResult, error) {
	input := params.Args
	// Validate required fields
	for _, requiredField := range t.InputSchema.Required {
		if _, exists := input[requiredField]; !exists {
			return nil, fmt.Errorf("missing required field: %s", requiredField)
		}
	}

	// Process and validate properties
	for propName, propSchemaInterface := range t.InputSchema.Properties {
		propValue, exists := input[propName]
		if !exists {
			continue // Skip optional properties that weren't provided
		}

		// Extract type information from the schema
		propSchema, ok := propSchemaInterface.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid schema definition for property: %s", propName)
		}

		// Get the expected type from the schema
		schemaType, ok := propSchema["type"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid type in schema for property: %s", propName)
		}

		// Validate type based on JSON schema types
		if !validateType(propValue, schemaType) {
			return nil, fmt.Errorf("invalid type for %s: expected %s, got %T", propName, schemaType, propValue)
		}

		params.Args[propName] = propValue
	}

	// Run the handler with validated arguments
	return t.Handler(ctx, params)
}
