package regression

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ikawaha/sudachi.go/analysis"
)

// RustGoComparisonResult represents a comparison result between Rust and Go implementations
type RustGoComparisonResult struct {
	Sentence      string             `json:"sentence"`
	Mode          string             `json:"mode"`
	GoMorphemes   []ExpectedMorpheme `json:"go_morphemes"`
	RustMorphemes []ExpectedMorpheme `json:"rust_morphemes"`
	IsMatch       bool               `json:"is_match"`
	Differences   []string           `json:"differences,omitempty"`
}

// ComparisonSummary represents the overall comparison results
type ComparisonSummary struct {
	Results        []RustGoComparisonResult `json:"results"`
	TotalSentences int                      `json:"total_sentences"`
	MatchingCount  int                      `json:"matching_count"`
	MismatchCount  int                      `json:"mismatch_count"`
	MatchRate      float64                  `json:"match_rate"`
	GeneratedAt    string                   `json:"generated_at"`
}

func TestRustGoComparisonBocchan(t *testing.T) {
	// Check if Rust CLI is available
	rustPath := "../../sudachi.rs/target/release/sudachi"
	if _, err := os.Stat(rustPath); os.IsNotExist(err) {
		t.Fatalf("rustPath %s does not exist", rustPath)
		return
	}

	// Create Go tokenizer
	tokenizer, err := createTokenizer()
	if err != nil {
		t.Fatalf("Failed to create Go tokenizer: %v", err)
	}

	t.Run("CompareSelectedSentences", func(t *testing.T) {
		runRustGoComparison(t, tokenizer, rustPath, bocchanTestSentences)
	})
}

func TestGenerateRustGoComparisonReport(t *testing.T) {
	// This test generates a detailed comparison report
	t.Skip("This test is used to generate comparison reports manually")

	rustPath := "../../sudachi.rs/target/release/sudachi"
	if _, err := os.Stat(rustPath); os.IsNotExist(err) {
		t.Fatalf("rustPath %s does not exist", rustPath)
	}

	tokenizer, err := createTokenizer()
	if err != nil {
		t.Fatalf("Failed to create Go tokenizer: %v", err)
	}

	summary := generateComparisonSummary(t, tokenizer, rustPath, bocchanTestSentences)

	// Save comparison report
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal comparison data: %v", err)
	}

	err = os.WriteFile("rust_go_comparison_report.json", data, 0644)
	if err != nil {
		t.Fatalf("Failed to write comparison report: %v", err)
	}

	t.Logf("Generated comparison report: %d/%d sentences match (%.1f%%)",
		summary.MatchingCount, summary.TotalSentences, summary.MatchRate)
}

func runRustGoComparison(t *testing.T, tokenizer *analysis.Tokenizer, rustPath string, sentences []string) {
	mismatches := 0
	matches := 0

	for i, sentence := range sentences {
		t.Run(fmt.Sprintf("Sentence_%d", i+1), func(t *testing.T) {
			// Get Go analysis
			goResult, err := analyzeText(tokenizer, sentence, "B")
			if err != nil {
				t.Fatalf("Failed to analyze with Go: %v", err)
			}

			// Get Rust analysis
			rustResult, err := analyzeWithRust(rustPath, sentence, "B")
			if err != nil {
				t.Fatalf("Failed to analyze with Rust: %v", err)
			}

			// Compare results
			isMatch, differences := compareMorphemeResults(goResult, rustResult)

			if isMatch {
				matches++
				t.Logf("✅ MATCH: %s", sentence)
			} else {
				mismatches++
				t.Errorf("❌ MISMATCH: %s", sentence)
				t.Logf("Go (%d morphemes): %s", len(goResult), formatMorphemes(goResult))
				t.Logf("Rust (%d morphemes): %s", len(rustResult), formatMorphemes(rustResult))

				for _, diff := range differences {
					t.Logf("  Difference: %s", diff)
				}
			}
		})
	}

	// Summary
	total := matches + mismatches
	matchRate := float64(matches) / float64(total) * 100
	t.Logf("\n📊 Comparison Summary:")
	t.Logf("  Total sentences: %d", total)
	t.Logf("  Matches: %d", matches)
	t.Logf("  Mismatches: %d", mismatches)
	t.Logf("  Match rate: %.1f%%", matchRate)

	if mismatches > 0 {
		t.Logf("\n⚠️  Found %d mismatches. Consider updating test expectations or investigating differences.", mismatches)
	}
}

func generateComparisonSummary(t *testing.T, tokenizer *analysis.Tokenizer, rustPath string, sentences []string) *ComparisonSummary {
	summary := &ComparisonSummary{
		Results:        make([]RustGoComparisonResult, 0),
		TotalSentences: len(sentences),
		GeneratedAt:    "auto-generated",
	}

	for _, sentence := range sentences {
		// Get Go analysis
		goResult, err := analyzeText(tokenizer, sentence, "B")
		if err != nil {
			t.Logf("Failed to analyze with Go: %v", err)
			continue
		}

		// Get Rust analysis
		rustResult, err := analyzeWithRust(rustPath, sentence, "B")
		if err != nil {
			t.Logf("Failed to analyze with Rust: %v", err)
			continue
		}

		// Compare results
		isMatch, differences := compareMorphemeResults(goResult, rustResult)

		result := RustGoComparisonResult{
			Sentence:      sentence,
			Mode:          "B",
			GoMorphemes:   goResult,
			RustMorphemes: rustResult,
			IsMatch:       isMatch,
			Differences:   differences,
		}

		summary.Results = append(summary.Results, result)

		if isMatch {
			summary.MatchingCount++
		} else {
			summary.MismatchCount++
		}
	}

	summary.MatchRate = float64(summary.MatchingCount) / float64(summary.TotalSentences) * 100
	return summary
}

func analyzeWithRust(rustPath, text, mode string) ([]ExpectedMorpheme, error) {
	// Create absolute path
	absPath, err := filepath.Abs(rustPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Prepare command
	cmd := exec.Command(absPath)
	// Always specify mode explicitly to ensure consistent behavior
	cmd.Args = append(cmd.Args, "-m", mode, "--all")

	// Set working directory to sudachi.rs
	rustDir := filepath.Dir(filepath.Dir(absPath))
	cmd.Dir = rustDir

	// Provide input
	cmd.Stdin = strings.NewReader(text)

	// Execute and capture output
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute Rust CLI: %w", err)
	}

	// Parse output
	return parseRustOutput(string(output))
}

func parseRustOutput(output string) ([]ExpectedMorpheme, error) {
	var morphemes []ExpectedMorpheme

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || line == "EOS" {
			continue
		}

		// Skip debug output from Rust
		if strings.HasPrefix(line, "DEBUG(Rust):") {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) < 7 {
			// If we don't have all 7 fields from --all option, skip
			continue
		}

		morpheme := ExpectedMorpheme{
			Surface:        parts[0], // 1. 表層形
			POS:            parts[1], // 2. 品詞
			NormalizedForm: parts[2], // 3. 正規化形
			DictionaryForm: parts[3], // 4. 辞書形
			Reading:        parts[4], // 5. 読み形（正確に取得）
		}

		morphemes = append(morphemes, morpheme)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse Rust output: %w", err)
	}

	return morphemes, nil
}

func compareMorphemeResults(goResult, rustResult []ExpectedMorpheme) (bool, []string) {
	var differences []string

	// Check count first
	if len(goResult) != len(rustResult) {
		differences = append(differences, fmt.Sprintf("Count mismatch: Go=%d, Rust=%d", len(goResult), len(rustResult)))
	}

	// Compare individual morphemes
	maxLen := len(goResult)
	if len(rustResult) > maxLen {
		maxLen = len(rustResult)
	}

	for i := 0; i < maxLen; i++ {
		var goMorpheme, rustMorpheme ExpectedMorpheme

		if i < len(goResult) {
			goMorpheme = goResult[i]
		}
		if i < len(rustResult) {
			rustMorpheme = rustResult[i]
		}

		if i >= len(goResult) {
			differences = append(differences, fmt.Sprintf("[%d] Missing in Go: %s", i, rustMorpheme.Surface))
			continue
		}
		if i >= len(rustResult) {
			differences = append(differences, fmt.Sprintf("[%d] Missing in Rust: %s", i, goMorpheme.Surface))
			continue
		}

		// Compare all fields comprehensively
		if goMorpheme.Surface != rustMorpheme.Surface {
			differences = append(differences, fmt.Sprintf("[%d] Surface: Go='%s', Rust='%s'", i, goMorpheme.Surface, rustMorpheme.Surface))
		}

		if goMorpheme.POS != rustMorpheme.POS {
			differences = append(differences, fmt.Sprintf("[%d] POS: Go='%s', Rust='%s'", i, goMorpheme.POS, rustMorpheme.POS))
		}

		if goMorpheme.NormalizedForm != rustMorpheme.NormalizedForm {
			differences = append(differences, fmt.Sprintf("[%d] NormalizedForm: Go='%s', Rust='%s'", i, goMorpheme.NormalizedForm, rustMorpheme.NormalizedForm))
		}

		// 追加：辞書形の比較
		if goMorpheme.DictionaryForm != rustMorpheme.DictionaryForm {
			differences = append(differences, fmt.Sprintf("[%d] DictionaryForm: Go='%s', Rust='%s'", i, goMorpheme.DictionaryForm, rustMorpheme.DictionaryForm))
		}

		// 重要：読み形の比較（これまで欠落していた）
		if goMorpheme.Reading != rustMorpheme.Reading {
			differences = append(differences, fmt.Sprintf("[%d] Reading: Go='%s', Rust='%s'", i, goMorpheme.Reading, rustMorpheme.Reading))
		}
	}

	return len(differences) == 0, differences
}

func formatMorphemes(morphemes []ExpectedMorpheme) string {
	var parts []string
	for _, m := range morphemes {
		pos := strings.Split(m.POS, ",")
		mainPOS := pos[0]
		if len(pos) > 1 && pos[1] != "*" {
			mainPOS += "," + pos[1]
		}
		parts = append(parts, fmt.Sprintf("%s(%s)", m.Surface, mainPOS))
	}
	return strings.Join(parts, " ")
}
