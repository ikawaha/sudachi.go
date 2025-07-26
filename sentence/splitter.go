package sentence

import (
	"github.com/ikawaha/sudachi.go/dic"
)

// SentenceIter implements SentenceIterator interface (matching Rust SentenceIter)
type SentenceIter struct {
	splitter *SentenceSplitter
	checker  *NonBreakChecker
	data     string
	position int
}

// Next returns the next sentence and whether there are more sentences (matching Rust Iterator::next)
func (si *SentenceIter) Next() (string, bool) {
	if si.position >= len(si.data) {
		return "", false
	}

	slice := si.data[si.position:]
	eosResult, err := si.splitter.detector.GetEOS(slice, si.checker)
	if err != nil {
		// If error, return the rest of the data as a sentence
		sentence := si.data[si.position:]
		si.position = len(si.data)
		return sentence, len(sentence) > 0
	}

	var end int
	if eosResult < 0 {
		// No EOS found, return the rest of the data
		end = len(si.data)
	} else {
		end = si.position + eosResult
	}

	sentence := si.data[si.position:end]
	si.position = end

	return sentence, len(sentence) > 0
}

// SentenceSplitter represents a sentence splitter (matching Rust SentenceSplitter exactly)
type SentenceSplitter struct {
	detector *SentenceDetector
	checker  *NonBreakChecker
}

// NewSentenceSplitter creates a new SentenceSplitter (matching Rust SentenceSplitter::new)
func NewSentenceSplitter() *SentenceSplitter {
	return &SentenceSplitter{
		detector: NewSentenceDetector(),
		checker:  nil,
	}
}

// NewSentenceSplitterWithLimit creates a new SentenceSplitter with limit (matching Rust SentenceSplitter::with_limit)
func NewSentenceSplitterWithLimit(limit int) *SentenceSplitter {
	return &SentenceSplitter{
		detector: NewSentenceDetectorWithLimit(limit),
		checker:  nil,
	}
}

// WithChecker adds a NonBreakChecker to the splitter (matching Rust SentenceSplitter::with_checker)
func (ss *SentenceSplitter) WithChecker(lexicon *dic.LexiconSet) *SentenceSplitter {
	checker := NewNonBreakChecker(lexicon)
	return &SentenceSplitter{
		detector: ss.detector,
		checker:  checker,
	}
}

// Split implements SplitSentences interface (matching Rust SplitSentences::split)
func (ss *SentenceSplitter) Split(data string) SentenceIterator {
	return &SentenceIter{
		splitter: ss,
		checker:  ss.checker,
		data:     data,
		position: 0,
	}
}
