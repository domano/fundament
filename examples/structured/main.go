package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/domano/fundament"
)

func main() {
	availability, err := fundament.CheckAvailability()
	if err != nil {
		log.Fatalf("check availability: %v", err)
	}
	if availability.State != fundament.AvailabilityReady {
		log.Fatalf("system language model unavailable: %s", availability)
	}

	session, err := fundament.NewSession(fundament.SessionOptions{
		Instructions: "You produce structured travel recommendations.",
	})
	if err != nil {
		log.Fatalf("new session: %v", err)
	}
	defer session.Close()

	schemaDefinition := map[string]any{
		"name":        "TravelPlan",
		"description": "A short trip plan with destination, highlights, and packing list.",
		"properties": []map[string]any{
			{
				"name": "destination",
				"schema": map[string]any{
					"type":        "string",
					"description": "City and country for the trip.",
				},
			},
			{
				"name": "highlights",
				"schema": map[string]any{
					"type":            "array",
					"description":     "A list of must-see highlights.",
					"minimumElements": 2,
					"maximumElements": 4,
					"items": map[string]any{
						"type":        "string",
						"description": "Concise highlight.",
					},
				},
			},
			{
				"name": "packingList",
				"schema": map[string]any{
					"type":        "array",
					"description": "Packing list tailored to the itinerary.",
					"items": map[string]any{
						"type":        "string",
						"description": "Single packing list item.",
					},
				},
			},
		},
	}

	schemaBytes, err := json.Marshal(schemaDefinition)
	if err != nil {
		log.Fatalf("marshal schema: %v", err)
	}
	schema, err := fundament.SchemaFromRawJSON(schemaBytes)
	if err != nil {
		log.Fatalf("schema: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	res, err := session.RespondStructured(ctx, "Plan a 2-day trip to Kyoto in autumn", schema)
	if err != nil {
		log.Fatalf("respond structured: %v", err)
	}

	fmt.Println(string(res.JSON))
}
