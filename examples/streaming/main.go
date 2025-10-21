package main

import (
	"context"
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
		Instructions: "You compose cheerful limericks.",
	})
	if err != nil {
		log.Fatalf("new session: %v", err)
	}
	defer session.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	stream, err := session.RespondStream(ctx, "Write a limerick about coding Go bindings for Swift models.", fundament.WithTemperature(0.7))
	if err != nil {
		log.Fatalf("respond stream: %v", err)
	}

	fmt.Println("Streaming response:")
	for chunk := range stream {
		if chunk.Err != nil {
			log.Fatalf("stream error: %v", chunk.Err)
		}
		fmt.Printf("%s ", chunk.Text)
		if chunk.Final {
			fmt.Println("\n-- end --")
		}
	}
}
