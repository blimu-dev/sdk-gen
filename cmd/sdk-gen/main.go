package main

import (
	"log"
	"os"

	"github.com/spf13/cobra"
	cli "github.com/viniciusdacal/sdk-gen/internal/cli"
)

func main() {
	root := &cobra.Command{
		Use:   "sdk-gen",
		Short: "Generate SDKs from OpenAPI specs",
	}

	root.AddCommand(newGenerateCmd())
	root.AddCommand(newValidateCmd())

	if err := root.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func newGenerateCmd() *cobra.Command {
	var configPath string
	var singleClient string
	var input string
	var typ string
	var outDir string
	var packageName string
	var name string
	var includeTags []string
	var excludeTags []string

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate client SDKs",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.RunGenerate(cli.RunGenerateParams{
				ConfigPath:   configPath,
				SingleClient: singleClient,
				Fallback: cli.FallbackParams{
					Spec:        input,
					Type:        typ,
					OutDir:      outDir,
					PackageName: packageName,
					Name:        name,
					IncludeTags: includeTags,
					ExcludeTags: excludeTags,
				},
			})
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to sdkgen.yaml config")
	cmd.Flags().StringVar(&singleClient, "client", "", "Generate only the named client from config")
	// Fallback single-client flags
	cmd.Flags().StringVar(&input, "input", "", "OpenAPI spec file (yaml/json)")
	cmd.Flags().StringVar(&typ, "type", "", "Client type (e.g., typescript)")
	cmd.Flags().StringVar(&outDir, "out", "", "Output directory")
	cmd.Flags().StringVar(&packageName, "package-name", "", "Package name")
	cmd.Flags().StringVar(&name, "client-name", "", "Client class name")
	cmd.Flags().StringArrayVar(&includeTags, "include-tags", nil, "Regex patterns for tags to include")
	cmd.Flags().StringArrayVar(&excludeTags, "exclude-tags", nil, "Regex patterns for tags to exclude")

	return cmd
}

func newValidateCmd() *cobra.Command {
	var input string
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate an OpenAPI spec",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.RunValidate(input)
		},
	}
	cmd.Flags().StringVar(&input, "input", "", "OpenAPI spec file (yaml/json)")
	_ = cmd.MarkFlagRequired("input")
	return cmd
}
