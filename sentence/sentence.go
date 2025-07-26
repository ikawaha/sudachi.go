package sentence

// SplitSentences interface matching Rust SplitSentences trait
type SplitSentences interface {
	Split(data string) SentenceIterator
}

// SentenceIterator interface for iterating over sentences (matching Rust SentenceIter)
type SentenceIterator interface {
	Next() (string, bool)
}

// SentenceRange represents a sentence range with byte positions (matching Rust Range<usize>)
type SentenceRange struct {
	Start int
	End   int
}
