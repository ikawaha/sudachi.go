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

type SurfaceProjectionString = string

const (
	SurfaceString              SurfaceProjectionString = "surface"
	NormalizedString           SurfaceProjectionString = "normalized"
	ReadingString              SurfaceProjectionString = "reading"
	DictionaryString           SurfaceProjectionString = "dictionary"
	DictionaryAndSurfaceString SurfaceProjectionString = "dictionary_and_surface"
	NormalizedAndSurfaceString SurfaceProjectionString = "normalized_and_surface"
	NormalizedNounsString      SurfaceProjectionString = "normalized_nouns"
	UnknownString              SurfaceProjectionString = "unknown"
)

// String returns the string representation of the surface projection
func (sp SurfaceProjection) String() string {
	switch sp {
	case Surface:
		return SurfaceString
	case Normalized:
		return NormalizedString
	case Reading:
		return ReadingString
	case Dictionary:
		return DictionaryString
	case DictionaryAndSurface:
		return DictionaryAndSurfaceString
	case NormalizedAndSurface:
		return NormalizedAndSurfaceString
	case NormalizedNouns:
		return NormalizedNounsString
	default:
		return UnknownString
	}
}

// ParseSurfaceProjection parses a string into a SurfaceProjection
func ParseSurfaceProjection(s string) (SurfaceProjection, error) {
	switch s {
	case SurfaceString:
		return Surface, nil
	case NormalizedString:
		return Normalized, nil
	case ReadingString:
		return Reading, nil
	case DictionaryString:
		return Dictionary, nil
	case DictionaryAndSurfaceString:
		return DictionaryAndSurface, nil
	case NormalizedAndSurfaceString:
		return NormalizedAndSurface, nil
	case NormalizedNounsString:
		return NormalizedNouns, nil
	default:
		return Surface, fmt.Errorf("unknown projection: %s", s)
	}
}

// PathResolver manages multiple root paths for resolving relative paths
type PathResolver struct {
	roots []string
}

// NewPathResolver creates a new PathResolver with the given capacity
func NewPathResolver(capacity int) *PathResolver {
	return &PathResolver{
		roots: make([]string, 0, capacity),
	}
}

// Add adds a root path to the resolver
func (pr *PathResolver) Add(path string) {
	if !pr.Contains(path) {
		pr.roots = append(pr.roots, path)
	}
}

// Contains checks if a path is already in the resolver
func (pr *PathResolver) Contains(path string) bool {
	for _, root := range pr.roots {
		if root == path {
			return true
		}
	}
	return false
}

// FirstExisting returns the first existing path from all candidates
func (pr *PathResolver) FirstExisting(path string) (string, bool) {
	for _, candidate := range pr.AllCandidates(path) {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true
		}
	}
	return "", false
}

// AllCandidates returns all possible candidate paths
func (pr *PathResolver) AllCandidates(path string) []string {
	candidates := make([]string, 0, len(pr.roots))
	for _, root := range pr.roots {
		candidates = append(candidates, filepath.Join(root, path))
	}
	return candidates
}

// Roots returns the root paths
func (pr *PathResolver) Roots() []string {
	return pr.roots
}

// ResolutionFailure returns an error for path resolution failure
func (pr *PathResolver) ResolutionFailure(path string) error {
	candidates := pr.AllCandidates(path)
	return fmt.Errorf("failed to resolve relative path %s: tried: %v", path, candidates)
}

// Config represents the complete configuration
type Config struct {
	resolver                *PathResolver
	SystemDict              string
	UserDicts               []string
	CharacterDefinitionFile string
	ConnectionCostPlugins   []map[string]any
	InputTextPlugins        []map[string]any
	OovProviderPlugins      []map[string]any
	PathRewritePlugins      []map[string]any
	Projection              SurfaceProjection
}

// Builder represents the raw configuration from JSON
type Builder struct {
	Path                    string           `json:"path,omitempty"`
	ResourcePath            string           `json:"-"`
	RootDirectory           string           `json:"-"`
	SystemDict              string           `json:"systemDict,omitempty"`
	UserDict                []string         `json:"userDict,omitempty"`
	CharacterDefinitionFile string           `json:"characterDefinitionFile,omitempty"`
	ConnectionCostPlugin    []map[string]any `json:"connectionCostPlugin,omitempty"`
	InputTextPlugin         []map[string]any `json:"inputTextPlugin,omitempty"`
	OovProviderPlugin       []map[string]any `json:"oovProviderPlugin,omitempty"`
	PathRewritePlugin       []map[string]any `json:"pathRewritePlugin,omitempty"`
	Projection              string           `json:"projection,omitempty"`
}

// FromFile creates a Builder from a file
func FromFile(configFile string) (*Builder, error) {
	file, err := os.Open(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var builder Builder
	if err := json.Unmarshal(data, &builder); err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	// Set root directory from config file path
	if dir := filepath.Dir(configFile); dir != "." {
		builder.RootDirectory = dir
	}

	return &builder, nil
}

// FromOptFile creates a Builder from an optional file path
func FromOptFile(configFile *string) (*Builder, error) {
	if configFile == nil {
		defaultConfig := DefaultConfigLocation()
		return FromFile(defaultConfig)
	}
	return FromFile(*configFile)
}

// SetSystemDict sets the system dictionary path
func (b *Builder) SetSystemDict(dict string) *Builder {
	b.SystemDict = dict
	return b
}

// SetUserDict adds a user dictionary path
func (b *Builder) SetUserDict(dict string) *Builder {
	b.UserDict = append(b.UserDict, dict)
	return b
}

// SetResourcePath sets the resource path
func (b *Builder) SetResourcePath(path string) *Builder {
	b.ResourcePath = path
	return b
}

// SetRootDirectory sets the root directory
func (b *Builder) SetRootDirectory(path string) *Builder {
	b.RootDirectory = path
	return b
}

// Build creates a Config from the Builder
func (b *Builder) Build() (*Config, error) {
	resourceDir := DefaultResourceDir
	if b.ResourcePath != "" {
		resourceDir = b.ResourcePath
	}
	resolver := NewPathResolver(3)
	if b.Path != "" {
		resolver.Add(b.Path)
	}
	resolver.Add(resourceDir)
	if b.RootDirectory != "" {
		resolver.Add(b.RootDirectory)
	}
	charDefFile := DefaultCharDefFile
	if b.CharacterDefinitionFile != "" {
		charDefFile = b.CharacterDefinitionFile
	}
	projection := Surface
	if b.Projection != "" {
		var err error
		projection, err = ParseSurfaceProjection(b.Projection)
		if err != nil {
			return nil, fmt.Errorf("config error: %w", err)
		}
	}
	return &Config{
		resolver:                resolver,
		SystemDict:              b.SystemDict,
		UserDicts:               b.UserDict,
		CharacterDefinitionFile: charDefFile,
		ConnectionCostPlugins:   b.ConnectionCostPlugin,
		InputTextPlugins:        b.InputTextPlugin,
		OovProviderPlugins:      b.OovProviderPlugin,
		PathRewritePlugins:      b.PathRewritePlugin,
		Projection:              projection,
	}, nil
}

// Fallback merges with another Builder, using the other's values as fallback
func (b *Builder) Fallback(other *Builder) *Builder {
	if b.Path == "" {
		b.Path = other.Path
	}
	if b.ResourcePath == "" {
		b.ResourcePath = other.ResourcePath
	}
	if b.RootDirectory == "" {
		b.RootDirectory = other.RootDirectory
	}
	if b.SystemDict == "" {
		b.SystemDict = other.SystemDict
	}
	if b.UserDict == nil {
		b.UserDict = other.UserDict
	}
	if b.CharacterDefinitionFile == "" {
		b.CharacterDefinitionFile = other.CharacterDefinitionFile
	}
	if b.ConnectionCostPlugin == nil {
		b.ConnectionCostPlugin = other.ConnectionCostPlugin
	}
	if b.InputTextPlugin == nil {
		b.InputTextPlugin = other.InputTextPlugin
	}
	if b.OovProviderPlugin == nil {
		b.OovProviderPlugin = other.OovProviderPlugin
	}
	if b.PathRewritePlugin == nil {
		b.PathRewritePlugin = other.PathRewritePlugin
	}
	if b.Projection == "" {
		b.Projection = other.Projection
	}
	return b
}

// DefaultConfigLocation returns the default config file location
func DefaultConfigLocation() string {
	return filepath.Join(DefaultResourceDir, DefaultSettingFile)
}

// New creates a new Config
func New(configFile, resourceDir, dictionaryPath *string) (*Config, error) {
	builder, err := FromOptFile(configFile)
	if err != nil {
		return nil, err
	}
	if resourceDir != nil {
		builder.SetResourcePath(*resourceDir)
	}
	if dictionaryPath != nil {
		builder.SetSystemDict(*dictionaryPath)
	}
	return builder.Build()
}

// WithSystemDict sets the system dictionary path
func (c *Config) WithSystemDict(systemDict string) *Config {
	c.SystemDict = systemDict
	return c
}

// ResolvePaths resolves special path patterns
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
// 1. Absolute paths stay as they are
// 2. Paths are resolved wrt to anchors, returning the first existing one
// 3. Paths are checked wrt to CWD
// 4. If all fail, return an error with all candidate paths listed
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
func (c *Config) ResolvedSystemDict() (string, error) {
	if c.SystemDict == "" {
		return "", errors.New("missing required field: systemDict")
	}
	return c.CompletePath(c.SystemDict)
}

// ResolvedUserDicts returns the resolved user dictionary paths
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
