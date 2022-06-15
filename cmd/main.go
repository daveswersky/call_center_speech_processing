package main

import (
	"log"
	"os"
	"context"

	// Blank-import the function package so the init() runs
	spch "example.com/speech_analysis"
	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
)

func main() {
	// Use PORT environment variable, or default to 8080.
	port := "8080"
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}
	
	funcframework.RegisterEventFunctionContext(context.Background(), "/", spch.Process_transcript) 
	
	if err := funcframework.Start(port); err != nil {
		log.Fatalf("funcframework.Start: %v\n", err)
	}
}
