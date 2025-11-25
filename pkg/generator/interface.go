package generator

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/blimu-dev/sdk-gen/pkg/config"
	"github.com/blimu-dev/sdk-gen/pkg/generator/golang"
	"github.com/blimu-dev/sdk-gen/pkg/generator/python"
	"github.com/blimu-dev/sdk-gen/pkg/generator/typescript"
	typescripttypes "github.com/blimu-dev/sdk-gen/pkg/generator/typescript-types"
	"github.com/blimu-dev/sdk-gen/pkg/ir"
	"github.com/blimu-dev/sdk-gen/pkg/openapi"
)

// Generator defines the interface for SDK generators
type Generator interface {
	// Generate creates an SDK from the given configuration and OpenAPI document
	Generate(client config.Client, ir ir.IR) error
	// GetType returns the type identifier for this generator (e.g., "typescript")
	GetType() string
}

// Registry manages available generators
type Registry struct {
	generators map[string]Generator
}

// NewRegistry creates a new generator registry
func NewRegistry() *Registry {
	return &Registry{
		generators: make(map[string]Generator),
	}
}

// Register adds a generator to the registry
func (r *Registry) Register(gen Generator) {
	r.generators[gen.GetType()] = gen
}

// Get retrieves a generator by type
func (r *Registry) Get(genType string) (Generator, bool) {
	gen, exists := r.generators[genType]
	return gen, exists
}

// GetAvailableTypes returns all registered generator types
func (r *Registry) GetAvailableTypes() []string {
	types := make([]string, 0, len(r.generators))
	for t := range r.generators {
		types = append(types, t)
	}
	return types
}

// GenerateOptions contains options for SDK generation
type GenerateOptions struct {
	ConfigPath   string
	SingleClient string
	Fallback     FallbackOptions
}

// FallbackOptions contains fallback options when no config file is provided
type FallbackOptions struct {
	Spec        string
	Type        string
	OutDir      string
	PackageName string
	Name        string
	IncludeTags []string
	ExcludeTags []string
}

// Service provides high-level SDK generation functionality
type Service struct {
	registry *Registry
}

// NewService creates a new generator service with default generators
func NewService() *Service {
	registry := NewRegistry()
	// Register default generators
	registry.Register(typescript.NewTypeScriptGenerator())
	registry.Register(golang.NewGoGenerator())
	registry.Register(python.NewPythonGenerator())
	registry.Register(typescripttypes.NewTypeScriptTypesGenerator())
	return &Service{
		registry: registry,
	}
}

// NewServiceWithRegistry creates a new generator service with a custom registry
func NewServiceWithRegistry(registry *Registry) *Service {
	return &Service{
		registry: registry,
	}
}

// Generate generates SDKs based on the provided options
func (s *Service) Generate(opts GenerateOptions) error {
	var cfg *config.Config
	var err error

	if opts.ConfigPath == "" {
		// Use fallback options to create a config
		if opts.Fallback.Spec == "" || opts.Fallback.Type == "" ||
			opts.Fallback.OutDir == "" || opts.Fallback.PackageName == "" ||
			opts.Fallback.Name == "" {
			return fmt.Errorf("either config path or all fallback options must be provided")
		}
		cfg = &config.Config{
			Spec: opts.Fallback.Spec,
			Clients: []config.Client{
				{
					Type:        opts.Fallback.Type,
					OutDir:      opts.Fallback.OutDir,
					PackageName: opts.Fallback.PackageName,
					Name:        opts.Fallback.Name,
					IncludeTags: opts.Fallback.IncludeTags,
					ExcludeTags: opts.Fallback.ExcludeTags,
				},
			},
		}
	} else {
		cfg, err = config.Load(opts.ConfigPath)
		if err != nil {
			return err
		}
	}

	return s.GenerateFromConfig(cfg, opts.SingleClient)
}

// GenerateFromConfig generates SDKs from a configuration
func (s *Service) GenerateFromConfig(cfg *config.Config, onlyClient string) error {
	// Load and validate OpenAPI document
	doc, err := openapi.LoadDocument(cfg.Spec)
	if err != nil {
		return err
	}

	// Build IR from OpenAPI document
	fullIR, err := s.buildIR(doc)
	if err != nil {
		return err
	}

	// Generate for each client
	for _, client := range cfg.Clients {
		if onlyClient != "" && client.Name != onlyClient {
			continue
		}

		generator, exists := s.registry.Get(client.Type)
		if !exists {
			return fmt.Errorf("unsupported client type: %s", client.Type)
		}

		// Ensure output directory exists before pre-commands
		if err := os.MkdirAll(client.OutDir, 0o755); err != nil {
			return fmt.Errorf("failed to create output directory for client %s: %w", client.Name, err)
		}

		// Execute pre-generation commands if specified
		if err := s.executePreCommands(client); err != nil {
			return fmt.Errorf("pre-generation commands failed for client %s: %w", client.Name, err)
		}

		// Filter IR based on client configuration
		filteredIR, err := s.filterIR(fullIR, client)
		if err != nil {
			return err
		}

		if err := generator.Generate(client, filteredIR); err != nil {
			return err
		}

		// Execute post-generation commands if specified
		if err := s.executePostGenCommands(client); err != nil {
			return fmt.Errorf("post-generation commands failed for client %s: %w", client.Name, err)
		}
	}

	return nil
}

// GetRegistry returns the generator registry
func (s *Service) GetRegistry() *Registry {
	return s.registry
}

// executePreCommands executes the pre-generation command for a client
func (s *Service) executePreCommands(client config.Client) error {
	command := client.GetPreCommand()
	if len(command) == 0 {
		return nil // No command to execute
	}

	return s.executeCommand(command, client.OutDir, "pre-command")
}

// executePostGenCommands executes the post-generation command for a client
func (s *Service) executePostGenCommands(client config.Client) error {
	command := client.GetPostCommand()
	if len(command) == 0 {
		return nil // No command to execute
	}

	return s.executeCommand(command, client.OutDir, "post-command")
}

// executeCommand executes a single command in Docker Compose array format
func (s *Service) executeCommand(command []string, workDir, commandLabel string) error {
	if len(command) == 0 {
		return nil // Skip empty commands
	}

	// Create command with first element as executable and rest as arguments
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Dir = workDir      // Execute in the specified directory
	cmd.Stdout = os.Stdout // Forward stdout to see command output
	cmd.Stderr = os.Stderr // Forward stderr to see errors

	cmdDescription := strings.Join(command, " ")

	// Execute the command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s (%s) failed: %w", commandLabel, cmdDescription, err)
	}

	return nil
}
