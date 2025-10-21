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
		Instructions: "You are a concise assistant that answers in one sentence.",
	})
	if err != nil {
		log.Fatalf("new session: %v", err)
	}
	defer session.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := session.Respond(ctx, "Explain what Foundation Models are on macOS.")
	if err != nil {
		log.Fatalf("respond: %v", err)
	}

	fmt.Println("Assistant:", response.Text)
}
