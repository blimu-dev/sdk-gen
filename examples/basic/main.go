package main

import (
	"log"

	sdkgen "github.com/blimu-dev/sdk-gen"
)

func main() {
	// Example 1: Generate TypeScript SDK with minimal configuration
	err := sdkgen.GenerateTypeScriptSDK(
		"https://petstore3.swagger.io/api/v3/openapi.json", // OpenAPI spec
		"./generated-sdk", // Output directory
		"petstore-client", // Package name
		"PetStoreClient",  // Client class name
	)
	if err != nil {
		log.Fatalf("Failed to generate SDK: %v", err)
	}

	log.Println("TypeScript SDK generated successfully!")

	// Example 2: Generate with more options
	err = sdkgen.GenerateSDK(sdkgen.GenerateSDKOptions{
		Spec:        "./openapi.yaml",
		Type:        "typescript",
		OutDir:      "./my-sdk",
		PackageName: "my-api-client",
		Name:        "MyAPIClient",
		IncludeTags: []string{"users", "orders"}, // Only include these tags
		ExcludeTags: []string{"internal"},        // Exclude these tags
	})
	if err != nil {
		log.Fatalf("Failed to generate SDK: %v", err)
	}

	log.Println("Custom SDK generated successfully!")
}
