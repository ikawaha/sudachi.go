package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultResourceDir = "resources"
	DefaultSettingFile = "sudachi.json"
	DefaultCharDefFile = "char.def"
)

// SurfaceProjection represents the surface projection mode
// Matches Rust: #[derive(Deserialize, Clone, Copy, Debug, Eq, PartialEq, Default)]
type SurfaceProjection int

const (
	Surface SurfaceProjection = iota
	Normalized
	Reading
	Dictionary
	DictionaryAndSurface
	NormalizedAndSurface
	NormalizedNouns
)

// String returns the string representation of the surface projection
func (sp SurfaceProjection) String() string {
	switch sp {
	case Surface:
		return "surface"
	case Normalized:
		return "normalized"
	case Reading:
		return "reading"
	case Dictionary:
		return "dictionary"
	case DictionaryAndSurface:
		return "dictionary_and_surface"
	case NormalizedAndSurface:
		return "normalized_and_surface"
	case NormalizedNouns:
		return "normalized_nouns"
	default:
		return "unknown"
	}
}

// ParseSurfaceProjection parses a string into a SurfaceProjection
// Matches Rust: impl TryFrom<&str> for SurfaceProjection
func ParseSurfaceProjection(s string) (SurfaceProjection, error) {
	switch s {
	case "surface":
		return Surface, nil
	case "normalized":
		return Normalized, nil
	case "reading":
		return Reading, nil
	case "dictionary":
		return Dictionary, nil
	case "dictionary_and_surface":
		return DictionaryAndSurface, nil
	case "normalized_and_surface":
		return NormalizedAndSurface, nil
	case "normalized_nouns":
		return NormalizedNouns, nil
	default:
		return Surface, fmt.Errorf("unknown projection: %s", s)
	}
}

// PathResolver manages multiple root paths for resolving relative paths
// Matches Rust: #[derive(Default, Debug, Clone)] struct PathResolver
type PathResolver struct {
	roots []string
}

// NewPathResolver creates a new PathResolver with the given capacity
// Matches Rust: fn with_capacity(capacity: usize) -> PathResolver
func NewPathResolver(capacity int) PathResolver {
	return PathResolver{
		roots: make([]string, 0, capacity),
	}
}

// Add adds a root path to the resolver
// Matches Rust: fn add<P: Into<PathBuf>>(&mut self, path: P)
func (pr *PathResolver) Add(path string) {
	if !pr.Contains(path) {
		pr.roots = append(pr.roots, path)
	}
}

// Contains checks if a path is already in the resolver
// Matches Rust: fn contains<P: AsRef<Path>>(&self, path: P) -> bool
func (pr *PathResolver) Contains(path string) bool {
	for _, root := range pr.roots {
		if root == path {
			return true
		}
	}
	return false
}

// FirstExisting returns the first existing path from all candidates
// Matches Rust: pub fn first_existing<P: AsRef<Path> + Clone>(&self, path: P) -> Option<PathBuf>
func (pr *PathResolver) FirstExisting(path string) (string, bool) {
	for _, candidate := range pr.AllCandidates(path) {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true
		}
	}
	return "", false
}

// AllCandidates returns all possible candidate paths
// Matches Rust: pub fn all_candidates<'a, P: AsRef<Path> + Clone + 'a>
func (pr *PathResolver) AllCandidates(path string) []string {
	candidates := make([]string, 0, len(pr.roots))
	for _, root := range pr.roots {
		candidates = append(candidates, filepath.Join(root, path))
	}
	return candidates
}

// Roots returns the root paths
// Matches Rust: pub fn roots(&self) -> &[PathBuf]
func (pr *PathResolver) Roots() []string {
	return pr.roots
}

// ResolutionFailure returns an error for path resolution failure
// Matches Rust: pub fn resolution_failure<P: AsRef<Path> + Clone>(&self, path: P) -> ConfigError
func (pr *PathResolver) ResolutionFailure(path string) error {
	candidates := pr.AllCandidates(path)
	return fmt.Errorf("failed to resolve relative path %q, tried: %v", path, candidates)
}

// Config represents the complete configuration
// Matches Rust: pub struct Config
type Config struct {
	resolver                PathResolver
	SystemDict              *string
	UserDicts               []string
	CharacterDefinitionFile string
	ConnectionCostPlugins   []map[string]any
	InputTextPlugins        []map[string]any
	OovProviderPlugins      []map[string]any
	PathRewritePlugins      []map[string]any
	Projection              SurfaceProjection
}

// ConfigBuilder represents the raw configuration from JSON
// Matches Rust: pub struct ConfigBuilder with #[allow(non_snake_case)]
type ConfigBuilder struct {
	PathField                    *string          `json:"path,omitempty"`
	ResourcePathField            *string          `json:"-"`
	RootDirectoryField           *string          `json:"-"`
	SystemDictField              *string          `json:"systemDict,omitempty"`
	SystemDictAlias              *string          `json:"system,omitempty"` // Matches Rust: #[serde(alias = "system")]
	UserDictField                []string         `json:"userDict,omitempty"`
	UserDictAlias                []string         `json:"user,omitempty"` // Matches Rust: #[serde(alias = "user")]
	CharacterDefinitionFileField *string          `json:"characterDefinitionFile,omitempty"`
	ConnectionCostPluginField    []map[string]any `json:"connectionCostPlugin,omitempty"`
	InputTextPluginField         []map[string]any `json:"inputTextPlugin,omitempty"`
	OovProviderPluginField       []map[string]any `json:"oovProviderPlugin,omitempty"`
	PathRewritePluginField       []map[string]any `json:"pathRewritePlugin,omitempty"`
	ProjectionField              *string          `json:"projection,omitempty"`
}

// FromOptFile creates a ConfigBuilder from an optional file path
// Matches Rust: pub fn from_opt_file(config_file: Option<&Path>) -> Result<Self, ConfigError>
func FromOptFile(configFile *string) (*ConfigBuilder, error) {
	if configFile == nil {
		defaultConfig := DefaultConfigLocation()
		return FromFile(defaultConfig)
	}
	return FromFile(*configFile)
}

// FromFile creates a ConfigBuilder from a file
// Matches Rust: pub fn from_file(config_file: &Path) -> Result<Self, ConfigError>
func FromFile(configFile string) (*ConfigBuilder, error) {
	file, err := os.Open(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var builder ConfigBuilder
	if err := json.Unmarshal(data, &builder); err != nil {
		return nil, fmt.Errorf("serialization error: %w", err)
	}

	// Set root directory from config file path (matches Rust behavior)
	if dir := filepath.Dir(configFile); dir != "." {
		builder.RootDirectoryField = &dir
	}

	return &builder, nil
}

// FromBytes creates a ConfigBuilder from byte data
// Matches Rust: pub fn from_bytes(data: &[u8]) -> Result<Self, ConfigError>
func FromBytes(data []byte) (*ConfigBuilder, error) {
	var builder ConfigBuilder
	if err := json.Unmarshal(data, &builder); err != nil {
		return nil, fmt.Errorf("serialization error: %w", err)
	}
	return &builder, nil
}

// Empty creates an empty ConfigBuilder
// Matches Rust: pub fn empty() -> Self
func Empty() *ConfigBuilder {
	builder, _ := FromBytes([]byte("{}"))
	return builder
}

// SystemDict sets the system dictionary path
// Matches Rust: pub fn system_dict(mut self, dict: impl Into<PathBuf>) -> Self
func (b *ConfigBuilder) SystemDict(dict string) *ConfigBuilder {
	b.SystemDictField = &dict
	return b
}

// UserDict adds a user dictionary path
// Matches Rust: pub fn user_dict(mut self, dict: impl Into<PathBuf>) -> Self
func (b *ConfigBuilder) UserDict(dict string) *ConfigBuilder {
	b.UserDictField = append(b.UserDictField, dict)
	return b
}

// ResourcePath sets the resource path
// Matches Rust: pub fn resource_path(mut self, path: impl Into<PathBuf>) -> Self
func (b *ConfigBuilder) ResourcePath(path string) *ConfigBuilder {
	b.ResourcePathField = &path
	return b
}

// RootDirectory sets the root directory
// Matches Rust: pub fn root_directory(mut self, path: impl Into<PathBuf>) -> Self
func (b *ConfigBuilder) RootDirectory(path string) *ConfigBuilder {
	b.RootDirectoryField = &path
	return b
}

// Build creates a Config from the ConfigBuilder
// Matches Rust: pub fn build(self) -> Config
func (b *ConfigBuilder) Build() *Config {
	defaultResourceDir := DefaultResourceDir
	resourceDir := defaultResourceDir
	if b.ResourcePathField != nil {
		resourceDir = *b.ResourcePathField
	}

	resolver := NewPathResolver(3)
	addPath := func(path string) {
		if !resolver.Contains(path) {
			resolver.Add(path)
		}
	}

	if b.PathField != nil {
		addPath(*b.PathField)
	}
	addPath(resourceDir)
	if b.RootDirectoryField != nil {
		addPath(*b.RootDirectoryField)
	}

	charDefFile := DefaultCharDefFile
	if b.CharacterDefinitionFileField != nil {
		charDefFile = *b.CharacterDefinitionFileField
	}

	// Handle alias fields (matching Rust serde aliases)
	systemDict := b.SystemDictField
	if systemDict == nil && b.SystemDictAlias != nil {
		systemDict = b.SystemDictAlias
	}

	userDicts := b.UserDictField
	if len(userDicts) == 0 && len(b.UserDictAlias) > 0 {
		userDicts = b.UserDictAlias
	}

	projection := Surface
	if b.ProjectionField != nil {
		if proj, err := ParseSurfaceProjection(*b.ProjectionField); err == nil {
			projection = proj
		}
	}

	connectionCostPlugins := b.ConnectionCostPluginField
	if connectionCostPlugins == nil {
		connectionCostPlugins = []map[string]any{}
	}

	inputTextPlugins := b.InputTextPluginField
	if inputTextPlugins == nil {
		inputTextPlugins = []map[string]any{}
	}

	oovProviderPlugins := b.OovProviderPluginField
	if oovProviderPlugins == nil {
		oovProviderPlugins = []map[string]any{}
	}

	pathRewritePlugins := b.PathRewritePluginField
	if pathRewritePlugins == nil {
		pathRewritePlugins = []map[string]any{}
	}

	return &Config{
		resolver:                resolver,
		SystemDict:              systemDict,
		UserDicts:               userDicts,
		CharacterDefinitionFile: charDefFile,
		ConnectionCostPlugins:   connectionCostPlugins,
		InputTextPlugins:        inputTextPlugins,
		OovProviderPlugins:      oovProviderPlugins,
		PathRewritePlugins:      pathRewritePlugins,
		Projection:              projection,
	}
}

// Fallback merges with another ConfigBuilder, using the other's values as fallback
// Matches Rust: pub fn fallback(mut self, other: &ConfigBuilder) -> ConfigBuilder
func (b *ConfigBuilder) Fallback(other *ConfigBuilder) *ConfigBuilder {
	if b.PathField == nil {
		b.PathField = other.PathField
	}
	if b.ResourcePathField == nil {
		b.ResourcePathField = other.ResourcePathField
	}
	if b.RootDirectoryField == nil {
		b.RootDirectoryField = other.RootDirectoryField
	}
	if b.SystemDictField == nil {
		b.SystemDictField = other.SystemDictField
	}
	if b.SystemDictAlias == nil {
		b.SystemDictAlias = other.SystemDictAlias
	}
	if b.UserDictField == nil {
		b.UserDictField = other.UserDictField
	}
	if b.UserDictAlias == nil {
		b.UserDictAlias = other.UserDictAlias
	}
	if b.CharacterDefinitionFileField == nil {
		b.CharacterDefinitionFileField = other.CharacterDefinitionFileField
	}
	if b.ConnectionCostPluginField == nil {
		b.ConnectionCostPluginField = other.ConnectionCostPluginField
	}
	if b.InputTextPluginField == nil {
		b.InputTextPluginField = other.InputTextPluginField
	}
	if b.OovProviderPluginField == nil {
		b.OovProviderPluginField = other.OovProviderPluginField
	}
	if b.PathRewritePluginField == nil {
		b.PathRewritePluginField = other.PathRewritePluginField
	}
	if b.ProjectionField == nil {
		b.ProjectionField = other.ProjectionField
	}
	return b
}

// DefaultConfigLocation returns the default config file location
// Matches Rust: pub fn default_config_location() -> PathBuf
func DefaultConfigLocation() string {
	return filepath.Join(DefaultResourceDir, DefaultSettingFile)
}

// New creates a new Config
// Matches Rust: pub fn new(config_file: Option<PathBuf>, resource_dir: Option<PathBuf>, dictionary_path: Option<PathBuf>) -> Result<Self, ConfigError>
func New(configFile, resourceDir, dictionaryPath *string) (*Config, error) {
	// prioritize arg (cli option) > default
	rawConfig, err := FromOptFile(configFile)
	if err != nil {
		return nil, err
	}

	// prioritize arg (cli option) > config file
	if resourceDir != nil {
		rawConfig = rawConfig.ResourcePath(*resourceDir)
	}

	// prioritize arg (cli option) > config file
	if dictionaryPath != nil {
		rawConfig = rawConfig.SystemDict(*dictionaryPath)
	}

	return rawConfig.Build(), nil
}

// WithSystemDict sets the system dictionary path
// Matches Rust: pub fn with_system_dic(mut self, system: impl Into<PathBuf>) -> Config
func (c *Config) WithSystemDict(systemDict string) *Config {
	c.SystemDict = &systemDict
	return c
}

// ResolvePaths resolves special path patterns
// Matches Rust: pub fn resolve_paths(&self, mut path: String) -> Vec<String>
func (c *Config) ResolvePaths(path string) []string {
	if strings.HasPrefix(path, "$exe") {
		// For Go, we'll use the executable directory
		execPath, err := os.Executable()
		if err != nil {
			return []string{path}
		}
		execDir := filepath.Dir(execPath)
		resolved := strings.Replace(path, "$exe", execDir, 1)
		depsPath := filepath.Join(execDir, "deps") + resolved[len(execDir):]
		return []string{depsPath, resolved}
	}
	if strings.HasPrefix(path, "$cfg/") || strings.HasPrefix(path, "$cfg\\") {
		roots := c.resolver.Roots()
		result := make([]string, 0, len(roots))
		relativePath := path[5:] // Remove "$cfg/"
		for _, root := range roots {
			result = append(result, filepath.Join(root, relativePath))
		}
		return result
	}

	return []string{path}
}

// CompletePath resolves a possibly relative path
// Matches Rust: pub fn complete_path<P: AsRef<Path> + Into<PathBuf>>(&self, file_path: P) -> Result<PathBuf, ConfigError>
func (c *Config) CompletePath(filePath string) (string, error) {
	// 1. Absolute paths stay as they are
	if filepath.IsAbs(filePath) {
		return filePath, nil
	}
	// 2. Try to resolve paths with respect to anchors
	if resolved, found := c.resolver.FirstExisting(filePath); found {
		return resolved, nil
	}
	// 3. Try to resolve a path with respect to CWD
	if _, err := os.Stat(filePath); err == nil {
		return filePath, nil
	}
	// 4. Report an error
	return "", c.resolver.ResolutionFailure(filePath)
}

// ResolvedSystemDict returns the resolved system dictionary path
// Matches Rust: pub fn resolved_system_dict(&self) -> Result<PathBuf, ConfigError>
func (c *Config) ResolvedSystemDict() (string, error) {
	if c.SystemDict == nil {
		return "", errors.New("missing required field: systemDict")
	}
	return c.CompletePath(*c.SystemDict)
}

// ResolvedUserDicts returns the resolved user dictionary paths
// Matches Rust: pub fn resolved_user_dicts(&self) -> Result<Vec<PathBuf>, ConfigError>
func (c *Config) ResolvedUserDicts() ([]string, error) {
	resolved := make([]string, 0, len(c.UserDicts))
	for _, userDict := range c.UserDicts {
		path, err := c.CompletePath(userDict)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, path)
	}
	return resolved, nil
}
