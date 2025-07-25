package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/ikawaha/sudachi.go/lattice"
)

// OutputFormatter インターフェース
type OutputFormatter interface {
	Format(writer io.Writer, results *lattice.MorphemeList) error
}

// SimpleFormatter - MeCab形式の出力
type SimpleFormatter struct {
	printAll bool
}

func NewSimpleFormatter(printAll bool) *SimpleFormatter {
	return &SimpleFormatter{printAll: printAll}
}

func (f *SimpleFormatter) Format(writer io.Writer, results *lattice.MorphemeList) error {
	for i := 0; i < results.Size(); i++ {
		morpheme := results.Get(i)
		if morpheme == nil {
			continue
		}

		// 基本情報: 表層形 \t 品詞 \t 正規化形（Rust版に合わせて）
		surface := morpheme.Surface()
		pos := strings.Join(morpheme.POS(), ",")
		normalizedForm := morpheme.NormalizedForm()

		if f.printAll {
			// Rust版と同じ形式: 表層形	品詞	正規化形	辞書形	読み	数値	配列
			// 3列目: NormalizedForm, 4列目: DictionaryForm
			normalizedForm := morpheme.NormalizedForm()
			dictionaryForm := morpheme.DictionaryForm()
			readingAll := morpheme.Reading()
			features := morpheme.Features()

			// Format features array like Rust version
			featuresStr := "[]"
			if len(features) > 0 {
				featuresStr = fmt.Sprintf("[%s]", strings.Join(features, ", "))
			}

			// Get dictionary ID (matching Rust version's output format)
			dictionaryId := morpheme.DictionaryId()
			oovSuffix := ""
			if morpheme.IsOOV() {
				oovSuffix = "\t(OOV)"
			}

			_, err := fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%d\t%s%s\n",
				surface, pos, normalizedForm, dictionaryForm, readingAll, dictionaryId, featuresStr, oovSuffix)
			if err != nil {
				return err
			}
		} else {
			// Rust版と同じ形式: 表層形 \t 品詞 \t 正規化形
			_, err := fmt.Fprintf(writer, "%s\t%s\t%s\n", surface, pos, normalizedForm)
			if err != nil {
				return err
			}
		}
	}

	// 文末マーカー
	_, err := fmt.Fprintln(writer, "EOS")
	return err
}

// JSONFormatter - JSON形式の出力
type JSONFormatter struct{}

func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{}
}

type JSONMorpheme struct {
	Surface        string   `json:"surface"`
	POS            []string `json:"pos"`
	Begin          uint16   `json:"begin"`
	End            uint16   `json:"end"`
	Length         uint16   `json:"length"`
	NormalizedForm string   `json:"normalized_form"`
	DictionaryForm string   `json:"dictionary_form"`
	Reading        string   `json:"reading"`
	Features       []string `json:"features,omitempty"`
}

func (f *JSONFormatter) Format(writer io.Writer, results *lattice.MorphemeList) error {
	morphemes := make([]JSONMorpheme, 0, results.Size())

	for i := 0; i < results.Size(); i++ {
		morpheme := results.Get(i)
		if morpheme == nil {
			continue
		}

		jsonMorpheme := JSONMorpheme{
			Surface:        morpheme.Surface(),
			POS:            morpheme.POS(),
			Begin:          morpheme.Begin(),
			End:            morpheme.End(),
			Length:         morpheme.Length(),
			NormalizedForm: morpheme.NormalizedForm(),
			DictionaryForm: morpheme.DictionaryForm(),
			Reading:        morpheme.Reading(),
			Features:       morpheme.Features(),
		}
		morphemes = append(morphemes, jsonMorpheme)
	}

	encoder := json.NewEncoder(writer)
	return encoder.Encode(morphemes)
}

// WakatiFormatter - わかち書き出力
type WakatiFormatter struct {
	separator string
}

func NewWakatiFormatter() *WakatiFormatter {
	return &WakatiFormatter{separator: " "}
}

func (f *WakatiFormatter) Format(writer io.Writer, results *lattice.MorphemeList) error {
	surfaces := make([]string, 0, results.Size())

	for i := 0; i < results.Size(); i++ {
		morpheme := results.Get(i)
		if morpheme == nil {
			continue
		}
		surfaces = append(surfaces, morpheme.Surface())
	}

	if len(surfaces) > 0 {
		_, err := fmt.Fprintln(writer, strings.Join(surfaces, f.separator))
		return err
	}

	// 空の場合も改行を出力
	_, err := fmt.Fprintln(writer)
	return err
}
