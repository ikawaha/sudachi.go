package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ikawaha/sudachi.go/analysis"
	"github.com/ikawaha/sudachi.go/config"
	"github.com/ikawaha/sudachi.go/dic"
	"github.com/ikawaha/sudachi.go/lattice"
	"github.com/ikawaha/sudachi.go/sentence"
)

// SentenceSplitMode represents how to handle sentence splitting (matching Rust)
type SentenceSplitMode int

const (
	SplitDefault SentenceSplitMode = iota // Default - split sentences and analyze
	SplitOnly                             // Only split sentences, no analysis
	SplitNone                             // No sentence splitting, analyze as-is
)

// バージョン情報
var (
	Version = "0.0.1"
	Build   = "dev"
)

func main() {
	config := parseCommandLine()

	if config.Help {
		printUsage()
		return
	}

	if config.Version {
		printVersion()
		return
	}

	if err := run(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(config *CLIConfig) error {
	// Setup input reader (matching Rust exactly)
	var innerReader io.Reader
	if config.File != "" {
		file, err := os.Open(config.File)
		if err != nil {
			return fmt.Errorf("failed to open input file %s: %w", config.File, err)
		}
		defer file.Close()
		innerReader = file
	} else {
		innerReader = os.Stdin
	}
	reader := bufio.NewReader(innerReader)

	// Setup output writer (matching Rust exactly)
	var innerWriter io.Writer = os.Stdout
	if config.OutputFile != "" {
		file, err := os.Create(config.OutputFile)
		if err != nil {
			return fmt.Errorf("failed to open output file %s: %w", config.OutputFile, err)
		}
		defer file.Close()
		innerWriter = file
	}
	writer := bufio.NewWriter(innerWriter)
	defer writer.Flush()

	// Load config (matching Rust Config::new logic)
	cfg, err := createConfig(config)
	if err != nil {
		return fmt.Errorf("failed to load config file: %w", err)
	}

	// Create dictionary (matching Rust JapaneseDictionary::from_cfg)
	dict, err := createDictionary(cfg, config)
	if err != nil {
		return fmt.Errorf("failed to create dictionary: %w", err)
	}

	// Parse mode
	mode, err := parseMode(config.Mode)
	if err != nil {
		return err
	}

	// Create analyzer based on split mode (matching Rust analyzer creation)
	analyzer, err := createAnalyzer(config, cfg, dict, mode)
	if err != nil {
		return fmt.Errorf("failed to create analyzer: %w", err)
	}

	// Process input line by line (matching Rust main loop exactly)
	isStdout := config.OutputFile == ""

	for {
		// Read line (matching Rust reader.read_line)
		n, err := reader.ReadBytes('\n')
		if err != nil && err != io.EOF {
			return fmt.Errorf("readline failed: %w", err)
		}
		if len(n) == 0 {
			break
		}

		// Strip EOL (matching Rust strip_eol function exactly)
		noEol := stripEol(string(n))

		// Analyze (matching Rust analyzer.analyze)
		err = analyzer.analyze(noEol, writer)
		if err != nil {
			return fmt.Errorf("analysis failed: %w", err)
		}

		// Flush if stdout (matching Rust behavior)
		if isStdout {
			writer.Flush()
		}
	}

	return nil
}

// stripEol removes (\r?\n)? pattern at the end of string (exact Rust implementation)
func stripEol(data string) string {
	bytes := []byte(data)
	length := len(bytes)

	if length > 0 && bytes[length-1] == '\n' {
		length--
		bytes = bytes[:length]
		if length > 0 && bytes[length-1] == '\r' {
			length--
			bytes = bytes[:length]
		}
	}

	return string(bytes)
}

// createDictionary creates dictionary (matching Rust JapaneseDictionary::from_cfg)
func createDictionary(cfg *config.Config, cliConfig *CLIConfig) (*dic.SystemDictionary, error) {
	dictPath, err := cfg.ResolvedSystemDict()
	if err != nil {
		return nil, err
	}

	loader := dic.NewDictionaryLoader()
	return loader.LoadSystemDictionary(dictPath)
}

// Removed custom plugin implementation - using existing BuildFromResourceDir function instead

// outputFormatter interface (matching Rust SudachiOutput trait)
type outputFormatter interface {
	write(writer *bufio.Writer, morphemes []*lattice.NodeResult) error
}

// SplitSentencesOnly analyzer implementation (matching Rust SplitSentencesOnly)
type splitSentencesOnly struct {
	// Note: Go implementation doesn't have sentence splitter yet
	// For now, just output raw input
}

func newSplitSentencesOnly(dict *dic.SystemDictionary) analyzer {
	return &splitSentencesOnly{}
}

func (s *splitSentencesOnly) analyze(input string, writer *bufio.Writer) error {
	// Just output the input as-is (placeholder implementation)
	_, err := writer.WriteString(input + "\n")
	return err
}

// AnalyzeSplitted analyzer implementation (matching Rust AnalyzeSplitted)
type analyzeSplitted struct {
	output    outputFormatter
	tokenizer *analysis.Tokenizer
	splitter  *sentence.SentenceSplitter
	mode      analysis.Mode
	debug     bool
}

func newAnalyzeSplitted(output outputFormatter, dict *dic.SystemDictionary, cfg *config.Config, mode analysis.Mode, debug bool) (analyzer, error) {
	// Create tokenizer using the same method as regression tests for compatibility
	builder := analysis.NewTokenizerBuilder(dict)

	// Set debug mode on the builder before building
	builder.SetDebug(debug)

	// Use default resource directory (same as regression tests)
	resourceDir := "resources"

	// Use LoadConfigFromResourceDir method (matching Rust behavior exactly)
	builder, err := builder.LoadConfigFromResourceDir(resourceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load config from resource dir %s: %w", resourceDir, err)
	}

	tokenizer, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build tokenizer: %w", err)
	}

	// Set the tokenization mode (matching Rust implementation)
	tokenizer.SetMode(mode)

	// Set debug mode
	tokenizer.SetDebugMode(debug)

	// Create sentence splitter with checker (matching Rust AnalyzeSplitted::new)
	splitter := sentence.NewSentenceSplitter().WithChecker(dict.LexiconSet())

	return &analyzeSplitted{
		output:    output,
		tokenizer: tokenizer,
		splitter:  splitter,
		mode:      mode,
		debug:     debug,
	}, nil
}

func (a *analyzeSplitted) analyze(input string, writer *bufio.Writer) error {
	// Split sentences and analyze each one (matching Rust AnalyzeSplitted::analyze exactly)
	iter := a.splitter.Split(input)
	for {
		sent, hasMore := iter.Next()
		if !hasMore {
			break
		}

		// Skip empty sentences
		sent = strings.TrimSpace(sent)
		if len(sent) == 0 {
			continue
		}

		// Debug: print what sentence we're processing
		if a.debug {
			fmt.Fprintf(os.Stderr, "Processing sentence: '%s'\n", sent)
		}

		// Analyze this sentence (matching Rust inner.analyze(sent, writer))
		morphemes, err := a.tokenizer.Tokenize(sent)
		if err != nil {
			return fmt.Errorf("tokenization failed: %w", err)
		}

		// Convert MorphemeList to slice of NodeResult
		results := make([]*lattice.NodeResult, morphemes.Size())
		for i := 0; i < morphemes.Size(); i++ {
			results[i] = morphemes.Get(i)
		}

		// Write results for this sentence (including EOS marker)
		err = a.output.write(writer, results)
		if err != nil {
			return err
		}
	}
	return nil
}

// AnalyzeNonSplitted analyzer implementation (matching Rust AnalyzeNonSplitted)
type analyzeNonSplitted struct {
	output    outputFormatter
	tokenizer *analysis.Tokenizer
	mode      analysis.Mode
	debug     bool
}

func newAnalyzeNonSplitted(output outputFormatter, dict *dic.SystemDictionary, cfg *config.Config, mode analysis.Mode, debug bool) (analyzer, error) {
	// Create tokenizer using TokenizerBuilder with config (matching Rust JapaneseDictionary::from_cfg)
	builder := analysis.NewTokenizerBuilder(dict)

	// Use default resource directory (same as regression tests)
	resourceDir := "resources"

	// Set both config and resource directory for plugin loading
	builder.SetConfig(cfg)
	builder.SetResourceDir(resourceDir)

	// Set debug mode on the builder before building
	builder.SetDebug(debug)

	tokenizer, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build tokenizer: %w", err)
	}

	// Set the tokenization mode (matching Rust implementation)
	tokenizer.SetMode(mode)

	// Set debug mode
	tokenizer.SetDebugMode(debug)

	return &analyzeNonSplitted{
		output:    output,
		tokenizer: tokenizer,
		mode:      mode,
		debug:     debug,
	}, nil
}

func (a *analyzeNonSplitted) analyze(input string, writer *bufio.Writer) error {
	morphemes, err := a.tokenizer.Tokenize(input)
	if err != nil {
		return fmt.Errorf("tokenization failed: %w", err)
	}
	// Convert MorphemeList to slice of NodeResult
	results := make([]*lattice.NodeResult, morphemes.Size())
	for i := 0; i < morphemes.Size(); i++ {
		results[i] = morphemes.Get(i)
	}
	return a.output.write(writer, results)
}

// Output formatters (matching Rust output module)

// wakachiOutput outputs only surface forms (matching Rust Wakachi)
type wakachiOutput struct{}

func newWakachiOutput() outputFormatter {
	return &wakachiOutput{}
}

func (w *wakachiOutput) write(writer *bufio.Writer, morphemes []*lattice.NodeResult) error {
	for i, m := range morphemes {
		if i > 0 {
			writer.WriteString(" ")
		}
		writer.WriteString(m.Surface())
	}
	writer.WriteString("\n")
	return nil
}

// simpleOutput outputs detailed morpheme information (matching Rust Simple)
type simpleOutput struct {
	printAll bool
}

func newSimpleOutput(printAll bool) outputFormatter {
	return &simpleOutput{printAll: printAll}
}

func (s *simpleOutput) write(writer *bufio.Writer, morphemes []*lattice.NodeResult) error {
	for _, m := range morphemes {
		// Surface form
		writer.WriteString(m.Surface())
		writer.WriteString("\t")

		// Part of speech (comma-separated) - matching Rust morpheme.part_of_speech()
		pos := m.POS()
		for i, feature := range pos {
			if i > 0 {
				writer.WriteString(",")
			}
			writer.WriteString(feature)
		}

		writer.WriteString("\t")
		// Normalized form - matching Rust morpheme.normalized_form()
		writer.WriteString(m.NormalizedForm())

		if s.printAll {
			// Additional fields for -a option (matching Rust -a output)
			writer.WriteString("\t")
			writer.WriteString(m.DictionaryForm()) // Dictionary form
			writer.WriteString("\t")
			writer.WriteString(m.Reading()) // Reading form
			writer.WriteString("\t")
			writer.WriteString(fmt.Sprintf("%d", m.DictionaryId())) // Dictionary ID
			writer.WriteString("\t")

			// Features array (matching Rust format)
			features := m.Features()
			writer.WriteString("[")
			for i, feature := range features {
				if i > 0 {
					writer.WriteString(", ")
				}
				writer.WriteString(feature)
			}
			writer.WriteString("]")

			// Add OOV marker if morpheme is out-of-vocabulary (matching Rust version)
			if m.IsOOV() {
				writer.WriteString("\t(OOV)")
			}
		}

		writer.WriteString("\n")
	}
	// Add EOS marker exactly like Rust implementation
	writer.WriteString("EOS\n")
	return nil
}

// Removed createPluginEnabledTokenizer - was custom implementation
// Plugins should be loaded at dictionary creation time, not tokenizer creation time

func parseMode(modeStr string) (analysis.Mode, error) {
	switch strings.ToUpper(modeStr) {
	case "A":
		return analysis.ModeA, nil
	case "B":
		return analysis.ModeB, nil
	case "C":
		return analysis.ModeC, nil
	default:
		return analysis.ModeC, fmt.Errorf("invalid mode: allowed values: A, B, C")
	}
}

// Legacy function - removed since we use new output formatter pattern

func printUsage() {
	fmt.Println("Sudachi - 日本語形態素解析器")
	fmt.Println()
	fmt.Println("使用法:")
	fmt.Println("  sudachi [オプション] [入力ファイル]")
	fmt.Println()
	fmt.Println("オプション:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("例:")
	fmt.Println("  echo '東京都に行く' | sudachi")
	fmt.Println("  sudachi -m A -f json input.txt")
	fmt.Println("  sudachi --wakati --output result.txt input.txt")
}

func printVersion() {
	fmt.Printf("Sudachi %s (build %s)\n", Version, Build)
}
