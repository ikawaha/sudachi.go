package regression

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ikawaha/sudachi.go/analysis"
)

// AkairousokutoningyoComparisonResult represents a single sentence comparison result
type AkairousokutoningyoComparisonResult struct {
	SentenceIndex int                `json:"sentence_index"`
	Sentence      string             `json:"sentence"`
	Mode          string             `json:"mode"`
	GoMorphemes   []ExpectedMorpheme `json:"go_morphemes"`
	GoldenLines   []string           `json:"golden_lines"`
	IsMatch       bool               `json:"is_match"`
	Differences   []string           `json:"differences,omitempty"`
}

// AkairousokutoningyoComparisonSummary represents the overall comparison results
type AkairousokutoningyoComparisonSummary struct {
	Results        []AkairousokutoningyoComparisonResult `json:"results"`
	TotalSentences int                                   `json:"total_sentences"`
	MatchingCount  map[string]int                        `json:"matching_count"`
	MismatchCount  map[string]int                        `json:"mismatch_count"`
	MatchRate      map[string]float64                    `json:"match_rate"`
	GeneratedAt    string                                `json:"generated_at"`
	TestDuration   string                                `json:"test_duration"`
}

func TestAkairousokutoningyoFullComparison(t *testing.T) {
	startTime := time.Now()

	// Load input sentences
	sentences, err := loadAkairousokutoningyoSentences()
	if err != nil {
		t.Fatalf("Failed to load akairousokutoningyo sentences: %v", err)
	}

	// Create Go tokenizer
	tokenizer, err := createTokenizer()
	if err != nil {
		t.Fatalf("Failed to create Go tokenizer: %v", err)
	}

	summary := &AkairousokutoningyoComparisonSummary{
		Results:        make([]AkairousokutoningyoComparisonResult, 0),
		TotalSentences: len(sentences),
		MatchingCount:  make(map[string]int),
		MismatchCount:  make(map[string]int),
		MatchRate:      make(map[string]float64),
		GeneratedAt:    time.Now().Format("2006-01-02 15:04:05"),
	}

	// Test each mode
	modes := []string{"A", "B", "C"}
	for _, mode := range modes {
		t.Run(fmt.Sprintf("Mode_%s", mode), func(t *testing.T) {
			runAkairousokutoningyoComparisonForMode(t, tokenizer, sentences, mode, summary)
		})
	}

	// Calculate overall statistics
	for _, mode := range modes {
		total := summary.MatchingCount[mode] + summary.MismatchCount[mode]
		if total > 0 {
			summary.MatchRate[mode] = float64(summary.MatchingCount[mode]) / float64(total) * 100
		}
	}

	summary.TestDuration = time.Since(startTime).String()

	// Save detailed comparison report
	reportPath := "akairousokutoningyo_full_comparison_report.json"
	if err := saveAkairousokutoningyoComparisonReport(summary, reportPath); err != nil {
		t.Errorf("Failed to save comparison report: %v", err)
	} else {
		t.Logf("📋 Detailed comparison report saved: %s", reportPath)
	}

	// Print summary
	t.Logf("\n📊 赤いろうそくと人魚全文比較結果サマリー:")
	t.Logf("  総文数: %d", summary.TotalSentences)
	for _, mode := range modes {
		matches := summary.MatchingCount[mode]
		mismatches := summary.MismatchCount[mode]
		rate := summary.MatchRate[mode]
		t.Logf("  モード%s: %d/%d一致 (%.1f%%)", mode, matches, matches+mismatches, rate)

		if mismatches > 0 {
			t.Logf("    ⚠️  %d件の不一致があります", mismatches)
		}
	}
	t.Logf("  実行時間: %s", summary.TestDuration)
}

func runAkairousokutoningyoComparisonForMode(t *testing.T, tokenizer *analysis.Tokenizer, sentences []string, mode string, summary *AkairousokutoningyoComparisonSummary) {
	// Load golden data for this mode
	goldenData, err := loadAkairousokutoningyoGoldenData(mode)
	if err != nil {
		t.Fatalf("Failed to load golden data for mode %s: %v", mode, err)
	}

	matches := 0
	mismatches := 0

	for i, sentence := range sentences {
		// Get Go analysis
		goMorphemes, err := analyzeText(tokenizer, sentence, mode)
		if err != nil {
			t.Errorf("Failed to analyze sentence %d with Go (mode %s): %v", i+1, mode, err)
			continue
		}

		// Get expected golden lines for this sentence
		goldenLines := getGoldenLinesForSentence(goldenData, i)

		// Parse golden lines before comparison
		goldenMorphemes, err := parseGoldenLines(goldenLines)
		if err != nil {
			t.Errorf("Failed to parse golden data for sentence %d: %v", i+1, err)
			continue
		}

		// Compare results using already parsed golden morphemes
		isMatch, differences := compareWithParsedGoldenData(goMorphemes, goldenMorphemes)

		result := AkairousokutoningyoComparisonResult{
			SentenceIndex: i + 1,
			Sentence:      sentence,
			Mode:          mode,
			GoMorphemes:   goMorphemes,
			GoldenLines:   goldenLines,
			IsMatch:       isMatch,
			Differences:   differences,
		}

		summary.Results = append(summary.Results, result)

		if isMatch {
			matches++
		} else {
			mismatches++
			// Log first few mismatches for debugging
			if mismatches <= 3 {
				t.Logf("❌ 文%d不一致 (モード%s): %s", i+1, mode, sentence)
				for _, diff := range differences {
					t.Logf("    %s", diff)
				}
			}
		}
	}

	summary.MatchingCount[mode] = matches
	summary.MismatchCount[mode] = mismatches

	t.Logf("モード%s: %d/%d一致 (%.1f%%)", mode, matches, matches+mismatches,
		float64(matches)/float64(matches+mismatches)*100)
}

func loadAkairousokutoningyoSentences() ([]string, error) {
	file, err := os.Open("../../testdata/sentences_akairousokutoningyo.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to open akairousokutoningyo sentences file: %w", err)
	}
	defer file.Close()

	var sentences []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			sentences = append(sentences, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read akairousokutoningyo sentences: %w", err)
	}

	return sentences, nil
}

func loadAkairousokutoningyoGoldenData(mode string) ([]string, error) {
	filename := fmt.Sprintf("akairousokutoningyo_golden_mode_%s.txt", mode)
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open golden data file %s: %w", filename, err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read golden data: %w", err)
	}

	return lines, nil
}

func saveAkairousokutoningyoComparisonReport(summary *AkairousokutoningyoComparisonSummary, filename string) error {
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal comparison data: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write comparison report: %w", err)
	}

	return nil
}