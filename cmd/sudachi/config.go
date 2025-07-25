package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ikawaha/sudachi.go/config"
)

// CLI設定 (matching Rust CLI exactly)
type CLIConfig struct {
	File           string            // Input file (positional argument)
	ConfigFile     string            // -r, --config-file
	ResourceDir    string            // -p, --resource_dir
	Mode           string            // -m, --mode
	OutputFile     string            // -o, --output
	PrintAll       bool              // -a, --all
	Wakati         bool              // -w, --wakati
	Debug          bool              // -d, --debug
	DictionaryPath string            // -l, --dict
	SplitSentences SentenceSplitMode // --split-sentences
	Help           bool              // --help
	Version        bool              // --version
}

func parseCommandLine() *CLIConfig {
	config := &CLIConfig{
		Mode:           "C",          // Default mode (matching Rust)
		SplitSentences: SplitDefault, // Default split mode (matching Rust "yes")
	}

	// Define flags exactly matching Rust CLI
	flag.StringVar(&config.ConfigFile, "r", "", "Path to the setting file in JSON format")
	flag.StringVar(&config.ConfigFile, "config-file", "", "Path to the setting file in JSON format")
	flag.StringVar(&config.ResourceDir, "p", "", "Path to the root directory of resources")
	flag.StringVar(&config.ResourceDir, "resource_dir", "", "Path to the root directory of resources")
	flag.StringVar(&config.Mode, "m", "C", "Split unit: \"A\" (short), \"B\" (middle), or \"C\" (Named Entity)")
	flag.StringVar(&config.Mode, "mode", "C", "Split unit: \"A\" (short), \"B\" (middle), or \"C\" (Named Entity)")
	flag.StringVar(&config.OutputFile, "o", "", "Output text file: If not present, use stdout")
	flag.StringVar(&config.OutputFile, "output", "", "Output text file: If not present, use stdout")
	flag.BoolVar(&config.PrintAll, "a", false, "Prints all fields")
	flag.BoolVar(&config.PrintAll, "all", false, "Prints all fields")
	flag.BoolVar(&config.Wakati, "w", false, "Outputs only surface form")
	flag.BoolVar(&config.Wakati, "wakati", false, "Outputs only surface form")
	flag.BoolVar(&config.Debug, "d", false, "Debug mode: Print the debug information")
	flag.BoolVar(&config.Debug, "debug", false, "Debug mode: Print the debug information")
	flag.StringVar(&config.DictionaryPath, "l", "", "Path to sudachi dictionary")
	flag.StringVar(&config.DictionaryPath, "dict", "", "Path to sudachi dictionary")

	// Sentence splitting option (matching Rust)
	var splitSentencesStr string
	flag.StringVar(&splitSentencesStr, "split-sentences", "yes", "How to split sentences: \"yes\", \"default\", \"no\", \"none\", \"only\"")

	flag.BoolVar(&config.Help, "help", false, "Show help")
	flag.BoolVar(&config.Help, "h", false, "Show help")
	flag.BoolVar(&config.Version, "version", false, "Show version")

	flag.Parse()

	// Parse sentence split mode
	switch splitSentencesStr {
	case "yes", "default":
		config.SplitSentences = SplitDefault
	case "no", "none":
		config.SplitSentences = SplitNone
	case "only":
		config.SplitSentences = SplitOnly
	default:
		fmt.Fprintf(os.Stderr, "invalid sentence split mode: %s\n", splitSentencesStr)
		os.Exit(1)
	}

	// Handle positional argument (input file)
	if flag.NArg() > 0 {
		config.File = flag.Arg(0)
	}

	return config
}

// createConfig creates configuration (matching Rust Config::new)
func createConfig(cliConfig *CLIConfig) (*config.Config, error) {
	var configFilePath *string
	if cliConfig.ConfigFile != "" {
		configFilePath = &cliConfig.ConfigFile
	}

	var resourceDir *string
	if cliConfig.ResourceDir != "" {
		resourceDir = &cliConfig.ResourceDir
	}

	var dictPath *string
	if cliConfig.DictionaryPath != "" {
		dictPath = &cliConfig.DictionaryPath
	}

	return config.New(configFilePath, resourceDir, dictPath)
}
