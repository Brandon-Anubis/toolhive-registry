// Package main provides the registry builder CLI tool
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/stacklok/toolhive-registry/pkg/registry"
	"github.com/stacklok/toolhive-registry/pkg/types"
)

var (
	// Version information (set during build)
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "registry-builder",
	Short: "Build and manage the ToolHive registry",
	Long: `registry-builder is a tool for building and managing the ToolHive registry.
It converts modular YAML registry entries into various output formats
including ToolHive JSON and upstream MCP Registry formats.`,
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the registry from YAML files",
	Long: `Build the registry by loading all YAML files from the registry directory
and generating output in the specified format.

Supported formats:
  - toolhive: ToolHive JSON format (default)
  - mcp-registry: Upstream MCP Registry format (future)
  - all: Build all supported formats`,
	RunE: runBuild,
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate registry entries",
	Long:  `Validate all registry entries without building the output files.`,
	RunE:  runValidate,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registry entries",
	Long:  `List all registry entries found in the registry directory.`,
	RunE:  runList,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(*cobra.Command, []string) {
		fmt.Printf("registry-builder %s\n", version)
		fmt.Printf("  commit: %s\n", commit)
		fmt.Printf("  built:  %s\n", date)
	},
}

var (
	registryPath string
	outputDir    string
	outputFormat string
	verbose      bool
)

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&registryPath, "registry", "r", "registry", "Path to the registry directory")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

	// Build command flags
	buildCmd.Flags().StringVarP(&outputDir, "output-dir", "o", "build", "Output directory for built registry files")
	buildCmd.Flags().StringVarP(&outputFormat, "format", "f", "toolhive", "Output format (toolhive, mcp-registry, all)")

	// Add commands
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runBuild(_ *cobra.Command, _ []string) error {
	if verbose {
		log.Printf("Building registry from %s", registryPath)
	}

	// Create loader
	loader := registry.NewLoader(registryPath)

	// Load all entries
	if err := loader.LoadAll(); err != nil {
		return fmt.Errorf("failed to load registry entries: %w", err)
	}

	entries := loader.GetEntries()
	if verbose {
		log.Printf("Loaded %d registry entries", len(entries))
	}

	// Count image and remote servers
	imageCount := 0
	remoteCount := 0
	for _, entry := range entries {
		if entry.IsImage() {
			imageCount++
		} else if entry.IsRemote() {
			remoteCount++
		}
	}

	// Determine which formats to build
	formats := determineFormats(outputFormat)

	// Build each format
	var builtFormats []string
	for _, format := range formats {
		if err := buildFormat(loader, format, outputDir); err != nil {
			return fmt.Errorf("failed to build %s format: %w", format, err)
		}
		builtFormats = append(builtFormats, format)
	}

	fmt.Printf("✓ Successfully built registry with %d entries\n", len(entries))
	if imageCount > 0 || remoteCount > 0 {
		fmt.Printf("  - %d container-based servers\n", imageCount)
		fmt.Printf("  - %d remote servers\n", remoteCount)
	}
	fmt.Printf("  Formats: %s\n", strings.Join(builtFormats, ", "))
	fmt.Printf("  Output directory: %s\n", outputDir)

	return nil
}

func determineFormats(format string) []string {
	switch strings.ToLower(format) {
	case "all":
		// Return all supported formats
		// For now, just toolhive, but will expand to include mcp-registry
		return []string{"toolhive"}
	case "mcp-registry", "mcp":
		// Future: Upstream MCP Registry format
		fmt.Println("Note: MCP Registry format support is planned for a future release")
		fmt.Println("This will generate output compatible with https://github.com/modelcontextprotocol/registry")
		return []string{}
	case "toolhive":
		fallthrough
	default:
		return []string{"toolhive"}
	}
}

func buildFormat(loader *registry.Loader, format string, outputDir string) error {
	switch format {
	case "toolhive":
		return buildToolhiveFormat(loader, outputDir)
	case "mcp-registry":
		// Future implementation
		return fmt.Errorf("MCP Registry format not yet implemented")
	default:
		return fmt.Errorf("unknown format: %s", format)
	}
}

func buildToolhiveFormat(loader *registry.Loader, outputDir string) error {
	// Create builder
	builder := registry.NewBuilder(loader)

	// Validate against schema
	if err := builder.ValidateAgainstSchema(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write JSON output
	outputPath := filepath.Join(outputDir, "registry.json")
	if err := builder.WriteJSON(outputPath); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	if verbose {
		log.Printf("Written ToolHive format to %s", outputPath)
	}

	return nil
}

// Future: buildMCPRegistryFormat function will be added here
// func buildMCPRegistryFormat(loader *registry.Loader, outputDir string) error {
//     // Implementation for upstream MCP Registry format
//     // This will create output compatible with the MCP Registry service:
//     // https://github.com/modelcontextprotocol/registry
//     // The format will evolve as the upstream standard evolves
// }

func runValidate(_ *cobra.Command, _ []string) error {
	if verbose {
		log.Printf("Validating registry entries in %s", registryPath)
	}

	// Create loader
	loader := registry.NewLoader(registryPath)

	// Load all entries
	if err := loader.LoadAll(); err != nil {
		return fmt.Errorf("failed to load registry entries: %w", err)
	}

	entries := loader.GetEntries()

	// Create builder for validation
	builder := registry.NewBuilder(loader)

	// Validate against schema
	if err := builder.ValidateAgainstSchema(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Count image and remote servers
	imageCount := 0
	remoteCount := 0
	for _, entry := range entries {
		if entry.IsImage() {
			imageCount++
		} else if entry.IsRemote() {
			remoteCount++
		}
	}

	fmt.Printf("✓ All %d registry entries are valid\n", len(entries))
	if imageCount > 0 && remoteCount > 0 {
		fmt.Printf("  - %d container-based servers\n", imageCount)
		fmt.Printf("  - %d remote servers\n", remoteCount)
	}

	if verbose {
		fmt.Println("\nValidated entries:")
		for _, entry := range loader.GetSortedEntries() {
			serverType := "Container"
			if entry.IsRemote() {
				serverType = "Remote"
			}
			fmt.Printf("  - %s [%s]: %s\n", entry.GetName(), serverType, entry.GetDescription())
		}
	}

	return nil
}

func runList(_ *cobra.Command, _ []string) error {
	// Create loader
	loader := registry.NewLoader(registryPath)

	// Load all entries
	if err := loader.LoadAll(); err != nil {
		return fmt.Errorf("failed to load registry entries: %w", err)
	}

	entries := loader.GetSortedEntries()

	fmt.Printf("Found %d registry entries:\n\n", len(entries))

	// Separate image and remote servers for display
	var imageServers, remoteServers []*types.RegistryEntry
	for _, entry := range entries {
		if entry.IsRemote() {
			remoteServers = append(remoteServers, entry)
		} else {
			imageServers = append(imageServers, entry)
		}
	}

	// Display image-based servers
	if len(imageServers) > 0 {
		fmt.Println("=== Container-based MCP Servers ===")
		for _, entry := range imageServers {
			displayEntry(entry, verbose)
		}
	}

	// Display remote servers
	if len(remoteServers) > 0 {
		if len(imageServers) > 0 {
			fmt.Println()
		}
		fmt.Println("=== Remote MCP Servers ===")
		for _, entry := range remoteServers {
			displayEntry(entry, verbose)
		}
	}

	return nil
}

func displayEntry(entry *types.RegistryEntry, verbose bool) {
	status := entry.GetStatus()
	if status == "" {
		status = "Active"
	}

	tier := entry.GetTier()
	if tier == "" {
		tier = "Community"
	}

	// Display differently based on type
	if entry.IsImage() {
		fmt.Printf("%-30s [%s/%s] %s\n", entry.GetName(), tier, status, entry.Image)
	} else if entry.IsRemote() {
		fmt.Printf("%-30s [%s/%s] %s\n", entry.GetName(), tier, status, entry.URL)
	}

	if verbose {
		fmt.Printf("  Type:        %s\n", getServerType(entry))
		fmt.Printf("  Description: %s\n", entry.GetDescription())
		fmt.Printf("  Transport:   %s\n", entry.GetTransport())

		tools := entry.GetTools()
		if len(tools) > 0 {
			fmt.Printf("  Tools:       %d available\n", len(tools))
		}

		if entry.IsImage() && entry.ImageMetadata.RepositoryURL != "" {
			fmt.Printf("  Repository:  %s\n", entry.ImageMetadata.RepositoryURL)
		} else if entry.IsRemote() && entry.RemoteServerMetadata.RepositoryURL != "" {
			fmt.Printf("  Repository:  %s\n", entry.RemoteServerMetadata.RepositoryURL)
		}

		if entry.License != "" {
			fmt.Printf("  License:     %s\n", entry.License)
		}

		if len(entry.Examples) > 0 {
			fmt.Printf("  Examples:    %d available\n", len(entry.Examples))
		}

		// Show remote-specific info
		if entry.IsRemote() {
			if entry.OAuthConfig != nil {
				fmt.Printf("  Auth:        OAuth/OIDC configured\n")
			}
			if len(entry.Headers) > 0 {
				fmt.Printf("  Headers:     %d configured\n", len(entry.Headers))
			}
		}

		fmt.Println()
	}
}

func getServerType(entry *types.RegistryEntry) string {
	if entry.IsImage() {
		return "Container"
	} else if entry.IsRemote() {
		return "Remote"
	}
	return "Unknown"
}
