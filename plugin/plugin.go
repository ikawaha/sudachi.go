package plugin

import (
	"github.com/ikawaha/sudachi.go/dic"
	"github.com/ikawaha/sudachi.go/input"
	"github.com/ikawaha/sudachi.go/lattice"
	"github.com/ikawaha/sudachi.go/types"
)

// Import CreatedWords from types package

// CreatedWords type alias for types.CreatedWords (matching Rust implementation)
type CreatedWords = types.CreatedWords

// InputTextPlugin processes input text during analysis setup
// Matches Rust trait: pub trait InputTextPlugin: Sync + Send
type InputTextPlugin interface {
	// SetUp initializes the plugin with configuration
	// Matches Rust method: fn set_up(&mut self, settings: &Value, config: &Config, grammar: &Grammar) -> SudachiResult<()>
	SetUp(settings map[string]any, resourceDir string, grammar *dic.Grammar) error

	// Rewrite modifies the input buffer during text preprocessing
	// Matches Rust method: fn rewrite(&self, input: &mut InputBuffer) -> SudachiResult<()>
	Rewrite(buffer *input.InputBuffer) error

	// GetName returns the plugin name for identification
	GetName() string
}

// OOVProviderPlugin provides out-of-vocabulary word candidates
// Matches Rust trait: pub trait OovProviderPlugin: Sync + Send
type OOVProviderPlugin interface {
	// SetUp initializes the plugin with configuration
	// Matches Rust method: fn set_up(&mut self, settings: &Value, config: &Config, grammar: &mut Grammar) -> SudachiResult<()>
	SetUp(settings map[string]any, resourceDir string, grammar *dic.Grammar) error

	// ProvideOOV generates OOV nodes at the given character position
	// Matches Rust method: fn provide_oov(&self, input_text: &InputBuffer, offset: usize,
	//                                     other_words: CreatedWords, result: &mut Vec<Node>) -> SudachiResult<usize>
	// Note: Value passing and return updated CreatedWords (matching Rust Copy semantics)
	ProvideOOV(charPos int, buffer *input.InputBuffer, lattice *lattice.Lattice, createdWords CreatedWords) (CreatedWords, error)

	// GetName returns the plugin name for identification
	GetName() string
}

// PathRewritePlugin modifies the analysis path after Viterbi search
// Matches Rust trait: pub trait PathRewritePlugin: Sync + Send
type PathRewritePlugin interface {
	// SetUp initializes the plugin with configuration
	// Matches Rust method: fn set_up(&mut self, settings: &Value, config: &Config, grammar: &Grammar) -> SudachiResult<()>
	SetUp(settings map[string]any, resourceDir string, grammar *dic.Grammar) error

	// Rewrite modifies the optimal path with complete word information access
	// Matches Rust method: fn rewrite(&self, text: &InputBuffer, path: Vec<ResultNode>,
	//                                 lattice: &Lattice) -> SudachiResult<Vec<ResultNode>>
	// Note: Added lattice parameter to match Rust version
	Rewrite(path []*lattice.NodeResult, buffer *input.InputBuffer, lat *lattice.Lattice) ([]*lattice.NodeResult, error)

	// GetName returns the plugin name for identification
	GetName() string
}
