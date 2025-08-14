package main

import (
	"log"

	"github.com/blimu-dev/sdk-gen/pkg/config"
	"github.com/blimu-dev/sdk-gen/pkg/generator"
	"github.com/blimu-dev/sdk-gen/pkg/generator/typescript"
)

func main() {
	// Example 1: Using a configuration file
	err := generator.GenerateFromConfig("./sdkgen.yaml")
	if err != nil {
		log.Fatalf("Failed to generate from config: %v", err)
	}

	// Example 2: Generate only a specific client from config
	err = generator.GenerateFromConfig("./sdkgen.yaml", "my-client")
	if err != nil {
		log.Fatalf("Failed to generate specific client: %v", err)
	}

	// Example 3: Using the service directly with custom registry
	registry := generator.NewRegistry()
	registry.Register(typescript.NewTypeScriptGenerator())
	// You could register additional custom generators here

	service := generator.NewServiceWithRegistry(registry)

	cfg := &config.Config{
		Spec: "https://api.example.com/openapi.json",
		Clients: []config.Client{
			{
				Type:        "typescript",
				OutDir:      "./client-a",
				PackageName: "client-a",
				Name:        "ClientA",
				IncludeTags: []string{"public"},
			},
			{
				Type:        "typescript",
				OutDir:      "./client-b",
				PackageName: "client-b",
				Name:        "ClientB",
				IncludeTags: []string{"admin"},
			},
		},
	}

	err = service.GenerateFromConfig(cfg, "")
	if err != nil {
		log.Fatalf("Failed to generate with service: %v", err)
	}

	// Example 4: Validate a spec before generating
	err = generator.ValidateSpec("./openapi.yaml")
	if err != nil {
		log.Fatalf("OpenAPI spec is invalid: %v", err)
	}

	log.Println("All examples completed successfully!")
}
