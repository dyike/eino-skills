package skill

import (
	"fmt"
	"testing"
)

// TestTOCTokenSavings demonstrates the token savings from using TOC
func TestTOCTokenSavings(t *testing.T) {
	sampleSkill := `# Introduction
This is a comprehensive guide to using our git commit skill.
It helps you write better commit messages and maintain consistency.

## Why Use This Skill
- Consistency across the team
- Better git history
- Automated changelog generation
- Improved code review process

# Instructions

## Prerequisites
Before using this skill, ensure you have:
- Git installed and configured
- A repository with staged or unstaged changes
- Understanding of conventional commit format

## How to Use
Follow these steps carefully:
1. Stage your changes using git add
2. Run the skill with appropriate parameters
3. Review the generated message
4. Commit if satisfied

### Advanced Usage
You can customize the commit message format by:
- Using conventional commit types (feat, fix, docs, etc.)
- Adding scope information to contextualize changes
- Including breaking change markers for API changes
- Referencing issue numbers for traceability

# Examples

## Simple Example
Here's a basic usage for a single file change:
git add src/main.go
/commit

Expected output:
feat: add user authentication

## Complex Example
For multi-file changes across different scopes:
git add src/auth/
git add tests/auth/
/commit --scope=auth --type=feat

Expected output:
feat(auth): implement OAuth2 authentication flow

- Add OAuth2 client configuration
- Implement token refresh mechanism
- Add integration tests

# Best Practices
- Always review the generated message before committing
- Use meaningful commit types that reflect the change
- Keep messages concise but descriptive
- Follow the conventional commit specification
- Reference related issues when applicable

# Troubleshooting

## Common Issues
- Message too generic: Ensure you have meaningful changes staged
- Wrong commit type: Review the conventional commit types
- Missing scope: Add scope for better organization`

	parser := NewParser()

	// Test 1: Full content
	fullContent := sampleSkill
	fullTokens := estimateTokens(fullContent)

	// Test 2: TOC only
	toc := parser.ExtractTOC(sampleSkill)
	tocTokens := estimateTokens(toc)

	// Test 3: Single section
	instructionsSection := parser.ExtractSection(sampleSkill, "Instructions")
	sectionTokens := estimateTokens(instructionsSection)

	// Test 4: Multiple sections (typical workflow)
	examplesSection := parser.ExtractSection(sampleSkill, "Examples")
	multiSectionTokens := tocTokens + sectionTokens + estimateTokens(examplesSection)

	t.Logf("\n=== Token Usage Comparison ===")
	t.Logf("Full Content:        ~%d tokens (baseline)", fullTokens)
	t.Logf("TOC Only:            ~%d tokens (%.1f%% savings)", tocTokens, savingsPercent(fullTokens, tocTokens))
	t.Logf("Single Section:      ~%d tokens (%.1f%% savings)", sectionTokens, savingsPercent(fullTokens, sectionTokens))
	t.Logf("TOC + 2 Sections:    ~%d tokens (%.1f%% savings)", multiSectionTokens, savingsPercent(fullTokens, multiSectionTokens))
	t.Logf("\n=== Recommended Workflow ===")
	t.Logf("1. view_skill(name='git-commit', toc=true)")
	t.Logf("   → Returns TOC (~%d tokens)", tocTokens)
	t.Logf("2. view_skill(name='git-commit', section='Instructions')")
	t.Logf("   → Returns only needed section (~%d tokens)", sectionTokens)
	t.Logf("3. Total: ~%d tokens vs %d tokens (%.1f%% savings)",
		tocTokens+sectionTokens, fullTokens, savingsPercent(fullTokens, tocTokens+sectionTokens))

	// Verify significant savings
	if tocTokens >= fullTokens/2 {
		t.Errorf("TOC should save at least 50%% tokens, got %.1f%%", savingsPercent(fullTokens, tocTokens))
	}
}

// estimateTokens provides a rough estimate of tokens
// Using conservative estimate: 1 token ≈ 3.5 characters
func estimateTokens(text string) int {
	return len(text) / 4
}

// savingsPercent calculates the percentage of tokens saved
func savingsPercent(original, new int) float64 {
	if original == 0 {
		return 0
	}
	return float64(original-new) / float64(original) * 100
}

// ExampleParser_ExtractTOC demonstrates TOC extraction
func ExampleParser_ExtractTOC() {
	parser := NewParser()
	body := `# Getting Started
This is the introduction.

## Installation
Install the tool.

### Prerequisites
You need Go 1.21+.

## Configuration
Configure your environment.

# Advanced Topics
More advanced content here.`

	toc := parser.ExtractTOC(body)
	fmt.Println(toc)
	// Output:
	// # Getting Started
	//   ## Installation
	//     ### Prerequisites
	//   ## Configuration
	// # Advanced Topics
}

// ExampleParser_ExtractSection demonstrates section extraction
func ExampleParser_ExtractSection() {
	parser := NewParser()
	body := `# Introduction
Welcome to the guide.

# Installation
Follow these steps to install.

## Step 1
Download the binary.

## Step 2
Run the installer.

# Configuration
Set up your config file.`

	section := parser.ExtractSection(body, "Installation")
	fmt.Println(section)
	// Output:
	// # Installation
	// Follow these steps to install.
	//
	// ## Step 1
	// Download the binary.
	//
	// ## Step 2
	// Run the installer.
}
