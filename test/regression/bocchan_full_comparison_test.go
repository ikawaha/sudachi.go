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

// BocchanComparisonResult represents a single sentence comparison result
type BocchanComparisonResult struct {
	SentenceIndex int                `json:"sentence_index"`
	Sentence      string             `json:"sentence"`
	Mode          string             `json:"mode"`
	GoMorphemes   []ExpectedMorpheme `json:"go_morphemes"`
	GoldenLines   []string           `json:"golden_lines"`
	IsMatch       bool               `json:"is_match"`
	Differences   []string           `json:"differences,omitempty"`
}

// BocchanComparisonSummary represents the overall comparison results
type BocchanComparisonSummary struct {
	Results        []BocchanComparisonResult `json:"results"`
	TotalSentences int                       `json:"total_sentences"`
	MatchingCount  map[string]int            `json:"matching_count"`
	MismatchCount  map[string]int            `json:"mismatch_count"`
	MatchRate      map[string]float64        `json:"match_rate"`
	GeneratedAt    string                    `json:"generated_at"`
	TestDuration   string                    `json:"test_duration"`
}

func TestBocchanFullComparison(t *testing.T) {
	startTime := time.Now()

	// Load input sentences
	sentences, err := loadBocchanSentences()
	if err != nil {
		t.Fatalf("Failed to load bocchan sentences: %v", err)
	}

	// Create Go tokenizer
	tokenizer, err := createTokenizer()
	if err != nil {
		t.Fatalf("Failed to create Go tokenizer: %v", err)
	}

	summary := &BocchanComparisonSummary{
		Results:        make([]BocchanComparisonResult, 0),
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
			runBocchanComparisonForMode(t, tokenizer, sentences, mode, summary)
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
	reportPath := "bocchan_full_comparison_report.json"
	if err := saveBocchanComparisonReport(summary, reportPath); err != nil {
		t.Errorf("Failed to save comparison report: %v", err)
	} else {
		t.Logf("📋 Detailed comparison report saved: %s", reportPath)
	}

	// Print summary
	t.Logf("\n📊 坊っちゃん全文比較結果サマリー:")
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

func runBocchanComparisonForMode(t *testing.T, tokenizer *analysis.Tokenizer, sentences []string, mode string, summary *BocchanComparisonSummary) {
	// Load golden data for this mode
	goldenData, err := loadGoldenData(mode)
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

		// Compare results
		isMatch, differences := compareWithGoldenData(goMorphemes, goldenLines)

		result := BocchanComparisonResult{
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

func loadBocchanSentences() ([]string, error) {
	file, err := os.Open("../../testdata/sentences_bocchan.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to open bocchan sentences file: %w", err)
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
		return nil, fmt.Errorf("failed to read bocchan sentences: %w", err)
	}

	return sentences, nil
}

func loadGoldenData(mode string) ([]string, error) {
	filename := fmt.Sprintf("bocchan_golden_mode_%s.txt", mode)
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

func getGoldenLinesForSentence(goldenData []string, sentenceIndex int) []string {
	var sentenceLines []string
	currentSentence := 0
	inSentence := false

	for _, line := range goldenData {
		line = strings.TrimSpace(line)

		if line == "EOS" {
			if inSentence {
				if currentSentence == sentenceIndex {
					return sentenceLines
				}
				currentSentence++
				sentenceLines = nil
				inSentence = false
			}
			continue
		}

		if line == "" {
			continue
		}

		// Start collecting lines for the current sentence
		if !inSentence {
			inSentence = true
		}

		if currentSentence == sentenceIndex {
			sentenceLines = append(sentenceLines, line)
		}
	}

	// Handle case where file doesn't end with EOS
	if currentSentence == sentenceIndex && len(sentenceLines) > 0 {
		return sentenceLines
	}

	return nil
}

func compareWithGoldenData(goMorphemes []ExpectedMorpheme, goldenLines []string) (bool, []string) {
	var differences []string

	// Parse golden lines into expected morphemes
	goldenMorphemes, err := parseGoldenLines(goldenLines)
	if err != nil {
		differences = append(differences, fmt.Sprintf("Failed to parse golden data: %v", err))
		return false, differences
	}

	// Check count
	if len(goMorphemes) != len(goldenMorphemes) {
		differences = append(differences, fmt.Sprintf("Count mismatch: Go=%d, Golden=%d", len(goMorphemes), len(goldenMorphemes)))
	}

	// Compare individual morphemes
	maxLen := len(goMorphemes)
	if len(goldenMorphemes) > maxLen {
		maxLen = len(goldenMorphemes)
	}

	for i := 0; i < maxLen; i++ {
		if i >= len(goMorphemes) {
			differences = append(differences, fmt.Sprintf("[%d] Missing in Go: %s", i, goldenMorphemes[i].Surface))
			continue
		}
		if i >= len(goldenMorphemes) {
			differences = append(differences, fmt.Sprintf("[%d] Extra in Go: %s", i, goMorphemes[i].Surface))
			continue
		}

		goM := goMorphemes[i]
		goldenM := goldenMorphemes[i]

		if goM.Surface != goldenM.Surface {
			differences = append(differences, fmt.Sprintf("[%d] Surface: Go='%s', Golden='%s'", i, goM.Surface, goldenM.Surface))
		}
		if goM.POS != goldenM.POS {
			differences = append(differences, fmt.Sprintf("[%d] POS: Go='%s', Golden='%s'", i, goM.POS, goldenM.POS))
		}
		if goM.NormalizedForm != goldenM.NormalizedForm {
			differences = append(differences, fmt.Sprintf("[%d] NormalizedForm: Go='%s', Golden='%s'", i, goM.NormalizedForm, goldenM.NormalizedForm))
		}
		if goM.DictionaryForm != goldenM.DictionaryForm {
			differences = append(differences, fmt.Sprintf("[%d] DictionaryForm: Go='%s', Golden='%s'", i, goM.DictionaryForm, goldenM.DictionaryForm))
		}
		if goM.Reading != goldenM.Reading {
			differences = append(differences, fmt.Sprintf("[%d] Reading: Go='%s', Golden='%s'", i, goM.Reading, goldenM.Reading))
		}
	}

	return len(differences) == 0, differences
}

func parseGoldenLines(lines []string) ([]ExpectedMorpheme, error) {
	var morphemes []ExpectedMorpheme

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line == "EOS" {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) < 5 {
			continue // Skip malformed lines
		}

		morpheme := ExpectedMorpheme{
			Surface:        parts[0], // 表層形
			POS:            parts[1], // 品詞
			NormalizedForm: parts[2], // 正規化形
			DictionaryForm: parts[3], // 辞書形
			Reading:        parts[4], // 読み形
		}

		morphemes = append(morphemes, morpheme)
	}

	return morphemes, nil
}

func saveBocchanComparisonReport(summary *BocchanComparisonSummary, filename string) error {
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal comparison data: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write comparison report: %w", err)
	}

	return nil
}
