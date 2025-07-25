package regression

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/ikawaha/sudachi.go/analysis"
)

// BocchanTestResult represents the analysis result for a sentence
type BocchanTestResult struct {
	Sentence   string             `json:"sentence"`
	Mode       string             `json:"mode"`
	Morphemes  []ExpectedMorpheme `json:"morphemes"`
	MorphCount int                `json:"morph_count"`
}

// BocchanBaseline represents the complete baseline for bocchan test
type BocchanBaseline struct {
	TestSentences  []BocchanTestResult `json:"test_sentences"`
	TotalSentences int                 `json:"total_sentences"`
	GeneratedAt    string              `json:"generated_at"`
}

// Representative sentences from bocchan for regression testing
var bocchanTestSentences = []string{
	"親譲りの無鉄砲で小供の時から損ばかりしている。",
	"小学校に居る時分学校の二階から飛び降りて一週間ほど腰を抜かした事がある。",
	"なぜそんな無闇をしたと聞く人があるかも知れぬ。",
	"別段深い理由でもない。",
	"弱虫やーい。",
	"と囃したからである。",
	"これは命より大事な栗だ。",
	"勘太郎は無論弱虫である。",
	"弱虫の癖に四つ目垣を乗りこえて、栗を盗みにくる。",
	"そんなら君の指を切ってみろと注文したから、何だ指ぐらいこの通りだと右の手の親指の甲をはすに切り込んだ。",
	"大抵は十三四人｜漬ってるがたまには誰も居ない事がある。",
}

func TestBocchanAnalysisRegression(t *testing.T) {
	tokenizer, err := createTokenizer()
	if err != nil {
		t.Fatalf("Failed to create tokenizer: %v", err)
	}

	// Load baseline if it exists
	baselineFile := "bocchan_baseline.json"
	if _, err := os.Stat(baselineFile); err == nil {
		t.Run("CompareWithBaseline", func(t *testing.T) {
			runBocchanBaselineComparison(t, tokenizer, baselineFile)
		})
	} else {
		t.Logf("Baseline file %s not found, skipping baseline comparison", baselineFile)
	}

	// Run analysis on test sentences for manual inspection
	t.Run("AnalyzeTestSentences", func(t *testing.T) {
		runBocchanTestSentences(t, tokenizer)
	})
}

func TestGenerateBocchanBaseline(t *testing.T) {
	// This test generates baseline data for bocchan sentences
	t.Skip("This test is used to generate baseline data manually")

	tokenizer, err := createTokenizer()
	if err != nil {
		t.Fatalf("Failed to create tokenizer: %v", err)
	}

	baseline := BocchanBaseline{
		TestSentences:  make([]BocchanTestResult, 0),
		TotalSentences: len(bocchanTestSentences),
		GeneratedAt:    "auto-generated",
	}

	// Generate baseline for all test sentences in Mode B
	for _, sentence := range bocchanTestSentences {
		result, err := analyzeText(tokenizer, sentence, "B")
		if err != nil {
			t.Fatalf("Failed to analyze sentence '%s': %v", sentence, err)
		}

		testResult := BocchanTestResult{
			Sentence:   sentence,
			Mode:       "B",
			Morphemes:  result,
			MorphCount: len(result),
		}
		baseline.TestSentences = append(baseline.TestSentences, testResult)
	}

	// Save baseline
	data, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal baseline data: %v", err)
	}

	err = os.WriteFile("bocchan_baseline.json", data, 0644)
	if err != nil {
		t.Fatalf("Failed to write baseline file: %v", err)
	}

	t.Logf("Generated baseline for %d sentences", len(bocchanTestSentences))
}

func TestBocchanFullFileAnalysis(t *testing.T) {
	// This test analyzes a subset of the full bocchan file for performance monitoring
	t.Skip("This test is for performance monitoring and should be run manually")

	tokenizer, err := createTokenizer()
	if err != nil {
		t.Fatalf("Failed to create tokenizer: %v", err)
	}

	file, err := os.Open("../../testdata/sentences_bocchan.txt")
	if err != nil {
		t.Fatalf("Failed to open sentences_bocchan.txt: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	processedCount := 0
	totalMorphemes := 0
	maxSentencesToProcess := 100 // Limit for performance

	for scanner.Scan() && processedCount < maxSentencesToProcess {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 {
			continue
		}

		results, err := analyzeText(tokenizer, line, "B")
		if err != nil {
			t.Logf("Failed to analyze line '%s': %v", line, err)
			continue
		}

		processedCount++
		totalMorphemes += len(results)

		// Log some interesting sentences
		if len(results) > 20 {
			t.Logf("Long sentence (%d morphemes): %s", len(results), line)
		}
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("Error reading file: %v", err)
	}

	avgMorphemes := float64(totalMorphemes) / float64(processedCount)
	t.Logf("Processed %d sentences, total %d morphemes, average %.1f morphemes per sentence",
		processedCount, totalMorphemes, avgMorphemes)
}

func runBocchanBaselineComparison(t *testing.T, tokenizer *analysis.Tokenizer, baselineFile string) {
	data, err := os.ReadFile(baselineFile)
	if err != nil {
		t.Fatalf("Failed to read baseline file: %v", err)
	}

	var baseline BocchanBaseline
	err = json.Unmarshal(data, &baseline)
	if err != nil {
		t.Fatalf("Failed to unmarshal baseline data: %v", err)
	}

	for i, expected := range baseline.TestSentences {
		t.Run(fmt.Sprintf("Sentence_%d", i+1), func(t *testing.T) {
			actual, err := analyzeText(tokenizer, expected.Sentence, expected.Mode)
			if err != nil {
				t.Fatalf("Failed to analyze sentence: %v", err)
			}

			// Compare morpheme count first
			if len(actual) != expected.MorphCount {
				t.Errorf("Morpheme count mismatch for '%s': expected %d, got %d",
					expected.Sentence, expected.MorphCount, len(actual))
				t.Logf("Expected: %d morphemes", expected.MorphCount)
				t.Logf("Actual: %d morphemes", len(actual))
				for i, m := range actual {
					t.Logf("  [%d] %s (%s)", i, m.Surface, strings.Join(strings.Split(m.POS, ",")[:2], ","))
				}
				return
			}

			// Compare individual morphemes
			for i, expectedMorpheme := range expected.Morphemes {
				if i >= len(actual) {
					break
				}
				actualMorpheme := actual[i]

				if actualMorpheme.Surface != expectedMorpheme.Surface {
					t.Errorf("Morpheme %d surface mismatch: expected '%s', got '%s'",
						i, expectedMorpheme.Surface, actualMorpheme.Surface)
				}

				if actualMorpheme.POS != expectedMorpheme.POS {
					t.Errorf("Morpheme %d POS mismatch: expected '%s', got '%s'",
						i, expectedMorpheme.POS, actualMorpheme.POS)
				}

				if actualMorpheme.Reading != expectedMorpheme.Reading {
					t.Errorf("Morpheme %d reading mismatch: expected '%s', got '%s'",
						i, expectedMorpheme.Reading, actualMorpheme.Reading)
				}
			}
		})
	}
}

func runBocchanTestSentences(t *testing.T, tokenizer *analysis.Tokenizer) {
	for i, sentence := range bocchanTestSentences {
		t.Run(fmt.Sprintf("TestSentence_%d", i+1), func(t *testing.T) {
			result, err := analyzeText(tokenizer, sentence, "B")
			if err != nil {
				t.Fatalf("Failed to analyze sentence: %v", err)
			}

			t.Logf("Sentence: %s", sentence)
			t.Logf("Morphemes (%d): ", len(result))
			for j, morpheme := range result {
				posParts := strings.Split(morpheme.POS, ",")
				mainPOS := posParts[0]
				if len(posParts) > 1 {
					mainPOS += "," + posParts[1]
				}
				t.Logf("  [%d] %s (%s)", j, morpheme.Surface, mainPOS)
			}

			// Ensure we have reasonable morpheme counts (basic sanity check)
			if len(result) == 0 {
				t.Error("No morphemes found - this seems wrong")
			}
			if len(result) > len(sentence) {
				t.Errorf("Too many morphemes (%d) for sentence length (%d)", len(result), len(sentence))
			}
		})
	}
}
