package skill

import (
	"testing"
)

func TestExtractTOC(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name     string
		body     string
		expected string
	}{
		{
			name: "simple hierarchy",
			body: `# Introduction
This is intro content.

## Getting Started
Some content here.

### Installation
Install instructions.

## Configuration
Config content.`,
			expected: `# Introduction
  ## Getting Started
    ### Installation
  ## Configuration`,
		},
		{
			name: "all heading levels",
			body: `# Level 1
## Level 2
### Level 3
#### Level 4
##### Level 5
###### Level 6`,
			expected: `# Level 1
  ## Level 2
    ### Level 3
      #### Level 4
        ##### Level 5
          ###### Level 6`,
		},
		{
			name: "mixed indentation with content",
			body: `# Main Title

Some paragraph text.

## Section 1

More content.

### Subsection 1.1

Content here.

### Subsection 1.2

More content.

## Section 2

Final content.`,
			expected: `# Main Title
  ## Section 1
    ### Subsection 1.1
    ### Subsection 1.2
  ## Section 2`,
		},
		{
			name:     "empty body",
			body:     "",
			expected: "",
		},
		{
			name: "no headings",
			body: `This is just regular text.
No headings here.
Just content.`,
			expected: "",
		},
		{
			name: "headings with leading spaces",
			body: `  # Title with spaces
    ## Another title`,
			expected: `# Title with spaces
  ## Another title`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.ExtractTOC(tt.body)
			if result != tt.expected {
				t.Errorf("ExtractTOC() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExtractSection(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name     string
		body     string
		heading  string
		expected string
	}{
		{
			name: "extract top-level section",
			body: `# Introduction
This is the intro.

# Getting Started
Here's how to get started.

# Configuration
Config details here.`,
			heading: "Getting Started",
			expected: `# Getting Started
Here's how to get started.`,
		},
		{
			name: "extract nested section",
			body: `# Main Title

## Subsection 1
Content for subsection 1.

### Nested Subsection
Nested content.

## Subsection 2
Content for subsection 2.`,
			heading: "Subsection 1",
			expected: `## Subsection 1
Content for subsection 1.

### Nested Subsection
Nested content.`,
		},
		{
			name: "extract deeply nested section",
			body: `# Level 1

## Level 2

### Level 3
This is level 3 content.

#### Level 4
This is level 4 content.

### Another Level 3
Different content.`,
			heading: "Level 3",
			expected: `### Level 3
This is level 3 content.

#### Level 4
This is level 4 content.`,
		},
		{
			name: "section not found",
			body: `# Introduction
Content here.

# Configuration
Config content.`,
			heading:  "NonExistent",
			expected: "",
		},
		{
			name: "case insensitive heading match",
			body: `# Introduction
Intro content.

# Getting Started
Getting started content.`,
			heading: "getting started",
			expected: `# Getting Started
Getting started content.`,
		},
		{
			name:     "empty body",
			body:     "",
			heading:  "Anything",
			expected: "",
		},
		{
			name: "section at the end",
			body: `# First Section
First content.

# Last Section
Last content.`,
			heading: "Last Section",
			expected: `# Last Section
Last content.`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.ExtractSection(tt.body, tt.heading)
			if result != tt.expected {
				t.Errorf("ExtractSection() = %q, want %q", result, tt.expected)
			}
		})
	}
}
