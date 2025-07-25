package main

import (
	"bufio"
	"fmt"

	"github.com/ikawaha/sudachi.go/analysis"
	"github.com/ikawaha/sudachi.go/config"
	"github.com/ikawaha/sudachi.go/dic"
)

// analyzer interface (matching Rust Analysis trait)
type analyzer interface {
	analyze(input string, writer *bufio.Writer) error
}

// createAnalyzer creates the appropriate analyzer (matching Rust analyzer creation)
func createAnalyzer(cliConfig *CLIConfig, cfg *config.Config, dict *dic.SystemDictionary, mode analysis.Mode) (analyzer, error) {
	switch cliConfig.SplitSentences {
	case SplitOnly:
		return newSplitSentencesOnly(dict), nil
	case SplitDefault:
		if cliConfig.Wakati {
			return newAnalyzeSplitted(newWakachiOutput(), dict, cfg, mode, cliConfig.Debug)
		} else {
			return newAnalyzeSplitted(newSimpleOutput(cliConfig.PrintAll), dict, cfg, mode, cliConfig.Debug)
		}
	case SplitNone:
		if cliConfig.Wakati {
			return newAnalyzeNonSplitted(newWakachiOutput(), dict, cfg, mode, cliConfig.Debug)
		} else {
			return newAnalyzeNonSplitted(newSimpleOutput(cliConfig.PrintAll), dict, cfg, mode, cliConfig.Debug)
		}
	default:
		return nil, fmt.Errorf("invalid split sentences mode")
	}
}
