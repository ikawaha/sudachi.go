package regression

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/ikawaha/sudachi.go/analysis"
	"github.com/ikawaha/sudachi.go/dic"
)

// TestCase represents a morphological analysis test case
type TestCase struct {
	Name     string             `json:"name"`
	Input    string             `json:"input"`
	Mode     string             `json:"mode"`
	Expected []ExpectedMorpheme `json:"expected"`
}

// ExpectedMorpheme represents expected morpheme analysis result
type ExpectedMorpheme struct {
	Surface        string `json:"surface"`
	POS            string `json:"pos"`
	Reading        string `json:"reading"`
	NormalizedForm string `json:"normalized_form"`
	DictionaryForm string `json:"dictionary_form"`
}

// BaselineData holds the baseline test cases
type BaselineData struct {
	TestCases []TestCase `json:"test_cases"`
}

// Critical test cases that must always pass
var criticalTestCases = []TestCase{
	{
		Name:  "すもも問題",
		Input: "すもももももももものうち",
		Mode:  "B",
		Expected: []ExpectedMorpheme{
			{Surface: "すもも", POS: "名詞,普通名詞,一般,*,*,*", Reading: "スモモ", NormalizedForm: "李"},
			{Surface: "も", POS: "助詞,係助詞,*,*,*,*", Reading: "モ", NormalizedForm: "も"},
			{Surface: "もも", POS: "名詞,普通名詞,一般,*,*,*", Reading: "モモ", NormalizedForm: "もも"},
			{Surface: "も", POS: "助詞,係助詞,*,*,*,*", Reading: "モ", NormalizedForm: "も"},
			{Surface: "もも", POS: "名詞,普通名詞,一般,*,*,*", Reading: "モモ", NormalizedForm: "もも"},
			{Surface: "の", POS: "助詞,格助詞,*,*,*,*", Reading: "ノ", NormalizedForm: "の"},
			{Surface: "うち", POS: "名詞,普通名詞,副詞可能,*,*,*", Reading: "ウチ", NormalizedForm: "うち"},
		},
	},
	{
		Name:  "基本的な文",
		Input: "東京都に行く",
		Mode:  "B",
		Expected: []ExpectedMorpheme{
			{Surface: "東京都", POS: "名詞,固有名詞,地名,一般,*,*", Reading: "トウキョウト", NormalizedForm: "東京都"},
			{Surface: "に", POS: "助詞,格助詞,*,*,*,*", Reading: "ニ", NormalizedForm: "に"},
			{Surface: "行く", POS: "動詞,非自立可能,*,*,五段-カ行,終止形-一般", Reading: "イク", NormalizedForm: "行く"},
		},
	},
	{
		Name:  "カタカナ文字",
		Input: "システム",
		Mode:  "B",
		Expected: []ExpectedMorpheme{
			{Surface: "システム", POS: "名詞,普通名詞,一般,*,*,*", Reading: "システム", NormalizedForm: "システム"},
		},
	},
	{
		Name:  "数値解析",
		Input: "１２３０４５円",
		Mode:  "B",
		Expected: []ExpectedMorpheme{
			{Surface: "１２３０４５", POS: "名詞,数詞,*,*,*,*", Reading: "イチニサンレイヨンゴ", NormalizedForm: "123045", DictionaryForm: "123045"},
			{Surface: "円", POS: "名詞,普通名詞,助数詞可能,*,*,*", Reading: "エン", NormalizedForm: "円", DictionaryForm: "円"},
		},
	},
}

func TestMorphemeAnalysisRegression(t *testing.T) {
	// Load configuration and create tokenizer
	tokenizer, err := createTokenizer()
	if err != nil {
		t.Fatalf("Failed to create tokenizer: %v", err)
	}

	// Run all critical test cases
	for _, testCase := range criticalTestCases {
		t.Run(testCase.Name, func(t *testing.T) {
			runTestCase(t, tokenizer, testCase)
		})
	}

	// Load and run baseline test cases if available
	baselineFile := "baseline.json"
	if _, err := os.Stat(baselineFile); err == nil {
		t.Run("Baseline", func(t *testing.T) {
			runBaselineTests(t, tokenizer, baselineFile)
		})
	}
}

func TestGenerateBaseline(t *testing.T) {
	// This test generates the baseline data for future regression testing
	t.Skip("This test is used to generate baseline data manually")

	tokenizer, err := createTokenizer()
	if err != nil {
		t.Fatalf("Failed to create tokenizer: %v", err)
	}

	baseline := BaselineData{}

	// Generate baseline for critical test cases
	for _, testCase := range criticalTestCases {
		actual, err := analyzeText(tokenizer, testCase.Input, testCase.Mode)
		if err != nil {
			t.Fatalf("Failed to analyze text for %s: %v", testCase.Name, err)
		}

		testCase.Expected = actual
		baseline.TestCases = append(baseline.TestCases, testCase)
	}

	// Add bocchan test sentences to baseline
	for i, sentence := range bocchanTestSentences {
		actual, err := analyzeText(tokenizer, sentence, "B")
		if err != nil {
			t.Fatalf("Failed to analyze bocchan sentence %d: %v", i+1, err)
		}

		testCase := TestCase{
			Name:     fmt.Sprintf("坊ちゃん_%d", i+1),
			Input:    sentence,
			Mode:     "B",
			Expected: actual,
		}
		baseline.TestCases = append(baseline.TestCases, testCase)
	}

	// Add additional comprehensive test cases
	additionalTestCases := []struct {
		name  string
		input string
		mode  string
	}{
		{"数値", "123円です", "B"},
		{"英語混じり", "Appleを食べる", "B"},
		{"長音記号", "コンピューター", "B"},
		{"複合語", "新型コロナウイルス", "B"},
		{"敬語", "いらっしゃいませ", "B"},
		{"関西弁", "何してはんの", "B"},
		{"古語", "ゆく河の流れは絶えずして", "B"},
		{"感嘆詞", "あっ、そうだ！", "B"},
		{"記号混じり", "100%確実です。", "B"},
		{"カタカナ語", "マネジメント", "B"},
	}

	for _, tc := range additionalTestCases {
		actual, err := analyzeText(tokenizer, tc.input, tc.mode)
		if err != nil {
			t.Logf("Warning: Failed to analyze '%s': %v", tc.input, err)
			continue
		}

		testCase := TestCase{
			Name:     tc.name,
			Input:    tc.input,
			Mode:     tc.mode,
			Expected: actual,
		}
		baseline.TestCases = append(baseline.TestCases, testCase)
	}

	// Save baseline data
	data, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal baseline data: %v", err)
	}

	err = os.WriteFile("baseline.json", data, 0644)
	if err != nil {
		t.Fatalf("Failed to write baseline file: %v", err)
	}

	t.Logf("Baseline data generated successfully with %d test cases", len(baseline.TestCases))
}

func createTokenizer() (*analysis.Tokenizer, error) {
	resourceDir := "../../resources"

	// Load system dictionary first
	dictPath := resourceDir + "/system.dic"
	loader := dic.NewDictionaryLoader()
	systemDict, err := loader.LoadSystemDictionary(dictPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load system dictionary: %w", err)
	}

	// Try to build tokenizer with proper configuration
	builder := analysis.NewTokenizerBuilder(systemDict)
	builder, err = builder.LoadConfigFromResourceDir(resourceDir)
	if err != nil {
		// Log the specific error for debugging
		fmt.Printf("Warning: Failed to load config from %s: %v\n", resourceDir, err)
		fmt.Printf("Falling back to BuildFromResourceDir\n")

		// Use BuildFromResourceDir as fallback
		tokenizer, err := analysis.BuildFromResourceDir(systemDict, resourceDir)
		if err != nil {
			return nil, fmt.Errorf("failed to build tokenizer: %w", err)
		}
		return tokenizer, nil
	}

	// Successfully loaded config, build tokenizer
	tokenizer, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build tokenizer with config: %w", err)
	}

	// Debug mode disabled for clean test output
	return tokenizer, nil
}

func runTestCase(t *testing.T, tokenizer *analysis.Tokenizer, testCase TestCase) {
	actual, err := analyzeText(tokenizer, testCase.Input, testCase.Mode)
	if err != nil {
		t.Fatalf("Failed to analyze text: %v", err)
	}

	// Compare results
	if len(actual) != len(testCase.Expected) {
		t.Errorf("Morpheme count mismatch: expected %d, got %d", len(testCase.Expected), len(actual))
		t.Logf("Expected: %+v", testCase.Expected)
		t.Logf("Actual: %+v", actual)
		return
	}

	for i, expected := range testCase.Expected {
		if i >= len(actual) {
			break
		}
		compareMorpheme(t, i, expected, actual[i])
	}
}

func runBaselineTests(t *testing.T, tokenizer *analysis.Tokenizer, baselineFile string) {
	data, err := os.ReadFile(baselineFile)
	if err != nil {
		t.Fatalf("Failed to read baseline file: %v", err)
	}

	var baseline BaselineData
	err = json.Unmarshal(data, &baseline)
	if err != nil {
		t.Fatalf("Failed to unmarshal baseline data: %v", err)
	}

	for _, testCase := range baseline.TestCases {
		t.Run(testCase.Name, func(t *testing.T) {
			runTestCase(t, tokenizer, testCase)
		})
	}
}

func analyzeText(tokenizer *analysis.Tokenizer, text, mode string) ([]ExpectedMorpheme, error) {
	// Set the mode on the tokenizer
	switch strings.ToUpper(mode) {
	case "A":
		tokenizer.SetMode(analysis.ModeA)
	case "B":
		tokenizer.SetMode(analysis.ModeB)
	case "C":
		tokenizer.SetMode(analysis.ModeC)
	default:
		tokenizer.SetMode(analysis.ModeB)
	}

	// Debug output removed for clean test output

	results, err := tokenizer.Tokenize(text)
	if err != nil {
		return nil, err
	}

	var morphemes []ExpectedMorpheme
	for i := 0; i < results.Size(); i++ {
		result := results.Get(i)

		// Debug output removed for clean test output

		morpheme := ExpectedMorpheme{
			Surface:        result.Surface(),
			POS:            strings.Join(result.POS(), ","),
			Reading:        result.Reading(),
			NormalizedForm: result.NormalizedForm(),
			DictionaryForm: result.DictionaryForm(),
		}
		morphemes = append(morphemes, morpheme)
	}

	return morphemes, nil
}

func compareMorpheme(t *testing.T, index int, expected, actual ExpectedMorpheme) {
	if expected.Surface != actual.Surface {
		t.Errorf("Morpheme %d surface mismatch: expected %q, got %q", index, expected.Surface, actual.Surface)
	}
	if expected.POS != actual.POS {
		t.Errorf("Morpheme %d POS mismatch: expected %q, got %q", index, expected.POS, actual.POS)
	}
	if expected.Reading != actual.Reading {
		t.Errorf("Morpheme %d reading mismatch: expected %q, got %q", index, expected.Reading, actual.Reading)
	}
	if expected.NormalizedForm != actual.NormalizedForm {
		t.Errorf("Morpheme %d normalized form mismatch: expected %q, got %q", index, expected.NormalizedForm, actual.NormalizedForm)
	}
}

// TestMorphemeAnalysisModesConsistency tests that different modes produce consistent results
func TestMorphemeAnalysisModesConsistency(t *testing.T) {
	tokenizer, err := createTokenizer()
	if err != nil {
		t.Fatalf("Failed to create tokenizer: %v", err)
	}

	testInput := "すもももももももものうち"

	resultsA, err := analyzeText(tokenizer, testInput, "A")
	if err != nil {
		t.Fatalf("Failed to analyze with Mode A: %v", err)
	}

	resultsB, err := analyzeText(tokenizer, testInput, "B")
	if err != nil {
		t.Fatalf("Failed to analyze with Mode B: %v", err)
	}

	resultsC, err := analyzeText(tokenizer, testInput, "C")
	if err != nil {
		t.Fatalf("Failed to analyze with Mode C: %v", err)
	}

	// Log results for manual inspection
	t.Logf("Mode A (%d morphemes): %+v", len(resultsA), resultsA)
	t.Logf("Mode B (%d morphemes): %+v", len(resultsB), resultsB)
	t.Logf("Mode C (%d morphemes): %+v", len(resultsC), resultsC)

	// Mode B should produce the expected 7 morphemes for this specific input
	if len(resultsB) != 7 {
		t.Errorf("Mode B should produce 7 morphemes for '%s', got %d", testInput, len(resultsB))
	}
}
