package main

import (
	"log"

	sdkgen "github.com/blimu-dev/sdk-gen"
)

func main() {
	// Example 1: Generate Go SDK with minimal configuration
	err := sdkgen.GenerateGoSDK(
		"https://petstore3.swagger.io/api/v3/openapi.json", // OpenAPI spec
		"./generated-go-sdk",                               // Output directory
		"github.com/example/petstore-client",               // Go module name
		"PetStoreClient",                                   // Client name
	)
	if err != nil {
		log.Fatalf("Failed to generate Go SDK: %v", err)
	}

	log.Println("Go SDK generated successfully!")

	// Example 2: Generate with more options using the general SDK function
	err = sdkgen.GenerateSDK(sdkgen.GenerateSDKOptions{
		Spec:        "./openapi.yaml",
		Type:        "go",
		OutDir:      "./my-go-sdk",
		PackageName: "github.com/myorg/my-api-client",
		Name:        "MyAPIClient",
		IncludeTags: []string{"users", "orders"}, // Only include these tags
		ExcludeTags: []string{"internal"},        // Exclude these tags
	})
	if err != nil {
		log.Fatalf("Failed to generate custom Go SDK: %v", err)
	}

	log.Println("Custom Go SDK generated successfully!")
}
