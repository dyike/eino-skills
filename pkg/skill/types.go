// Package skill provides types and utilities for loading and managing agent skills.
// Skills follow the Anthropic Agent Skills pattern - folders containing a SKILL.md
// file with YAML frontmatter and markdown instructions that agents can discover
// and load dynamically.
package skill

import (
	"path/filepath"
	"time"
)

// Skill represents a loaded skill with its metadata and content.
type Skill struct {
	// Name is the skill identifier (from YAML frontmatter)
	Name string `json:"name" yaml:"name"`

	// Description describes what the skill does and when to use it
	Description string `json:"description" yaml:"description"`

	// Path is the absolute path to the skill directory
	Path string `json:"path"`

	// Content is the full markdown content (loaded on demand)
	Content string `json:"-"`

	// Files are additional files bundled with the skill
	Files []SkillFile `json:"files,omitempty"`

	// Source indicates where the skill was loaded from
	Source SkillSource `json:"source"`

	// LoadedAt is when the skill was loaded
	LoadedAt time.Time `json:"loaded_at"`
}

// SkillFile represents an additional file bundled with a skill.
type SkillFile struct {
	// RelPath is the path relative to skill directory
	RelPath string `json:"rel_path"`

	// AbsPath is the absolute filesystem path
	AbsPath string `json:"abs_path"`

	// Type indicates the file category
	Type SkillFileType `json:"type"`
}

// SkillFileType categorizes bundled files.
type SkillFileType string

const (
	// FileTypeScript for executable scripts (scripts/)
	FileTypeScript SkillFileType = "script"

	// FileTypeReference for documentation (references/)
	FileTypeReference SkillFileType = "reference"

	// FileTypeAsset for templates, icons, etc (assets/)
	FileTypeAsset SkillFileType = "asset"

	// FileTypeOther for uncategorized files
	FileTypeOther SkillFileType = "other"
)

// SkillSource indicates where a skill was loaded from.
type SkillSource string

const (
	// SourceGlobal for ~/.eino/<agent>/skills/
	SourceGlobal SkillSource = "global"

	// SourceProject for .eino/skills/
	SourceProject SkillSource = "project"

	// SourceBuiltin for built-in skills
	SourceBuiltin SkillSource = "builtin"

	// SourcePlugin for plugin-provided skills
	SourcePlugin SkillSource = "plugin"
)

// Frontmatter represents the YAML frontmatter of a SKILL.md file.
type Frontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`

	// Optional fields
	AllowedTools []string `yaml:"allowed-tools,omitempty"`
	Version      string   `yaml:"version,omitempty"`
	Author       string   `yaml:"author,omitempty"`
	License      string   `yaml:"license,omitempty"`
}

// Validate checks if the frontmatter is valid.
func (f *Frontmatter) Validate() error {
	if f.Name == "" {
		return ErrMissingName
	}
	if len(f.Name) > MaxNameLength {
		return ErrNameTooLong
	}
	if f.Description == "" {
		return ErrMissingDescription
	}
	if len(f.Description) > MaxDescriptionLength {
		return ErrDescriptionTooLong
	}
	return nil
}

// SkillMetadata is the lightweight metadata loaded at startup.
// Only name and description are included to minimize context usage.
type SkillMetadata struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Source      SkillSource `json:"source"`
	Path        string      `json:"path"`
}

// ToMetadata extracts metadata from a full skill.
func (s *Skill) ToMetadata() SkillMetadata {
	return SkillMetadata{
		Name:        s.Name,
		Description: s.Description,
		Source:      s.Source,
		Path:        s.Path,
	}
}

// SkillMDPath returns the path to SKILL.md within the skill directory.
func (s *Skill) SkillMDPath() string {
	return filepath.Join(s.Path, "SKILL.md")
}

// Registry constants and limits.
const (
	// MaxNameLength is the maximum length for skill names
	MaxNameLength = 64

	// MaxDescriptionLength is the maximum length for descriptions
	MaxDescriptionLength = 1024

	// SkillFileName is the required filename for skill definitions
	SkillFileName = "SKILL.md"
)

// Error types for skill validation.
type SkillError struct {
	SkillPath string
	Message   string
	Err       error
}

func (e *SkillError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *SkillError) Unwrap() error {
	return e.Err
}

// Predefined errors.
var (
	ErrMissingName        = &SkillError{Message: "skill name is required"}
	ErrNameTooLong        = &SkillError{Message: "skill name exceeds maximum length"}
	ErrMissingDescription = &SkillError{Message: "skill description is required"}
	ErrDescriptionTooLong = &SkillError{Message: "skill description exceeds maximum length"}
	ErrSkillNotFound      = &SkillError{Message: "skill not found"}
	ErrInvalidFrontmatter = &SkillError{Message: "invalid YAML frontmatter"}
	ErrMissingSkillMD     = &SkillError{Message: "SKILL.md file not found"}
)
