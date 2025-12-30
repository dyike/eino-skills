// Package main provides a CLI for managing eino skills.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	skill "github.com/dyike/eino-skills/pkg/skill"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	ctx := context.Background()
	cmd := os.Args[1]

	switch cmd {
	case "list":
		listCmd(ctx, os.Args[2:])
	case "create":
		createCmd(ctx, os.Args[2:])
	case "view":
		viewCmd(ctx, os.Args[2:])
	case "validate":
		validateCmd(ctx, os.Args[2:])
	case "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`eino-skills - Manage agent skills

Usage:
  eino-skills <command> [options]

Commands:
  list      List all available skills
  create    Create a new skill from template
  view      View a skill's contents
  validate  Validate a skill's structure

Options:
  --global    Use global skills directory (~/.eino/agent/skills)
  --project   Use project skills directory (.eino/skills)

Examples:
  eino-skills list
  eino-skills list --project
  eino-skills create my-skill
  eino-skills view git-commit
  eino-skills validate ./skills/my-skill`)
}

func listCmd(ctx context.Context, args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	global := fs.Bool("global", false, "List only global skills")
	project := fs.Bool("project", false, "List only project skills")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	loader := skill.NewLoader(
		skill.WithGlobalSkillsDir("~/.claude/skills"),
		skill.WithProjectSkillsDir(".eino/skills"),
	)

	var skills []*skill.Skill
	var err error

	if *project {
		skills, err = loader.LoadAll(ctx)
		// Filter to project only
		filtered := make([]*skill.Skill, 0)
		for _, s := range skills {
			if s.Source == skill.SourceProject {
				filtered = append(filtered, s)
			}
		}
		skills = filtered
	} else if *global {
		skills, err = loader.LoadAll(ctx)
		// Filter to global only
		filtered := make([]*skill.Skill, 0)
		for _, s := range skills {
			if s.Source == skill.SourceGlobal {
				filtered = append(filtered, s)
			}
		}
		skills = filtered
	} else {
		skills, err = loader.LoadAll(ctx)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading skills: %v\n", err)
		os.Exit(1)
	}

	if len(skills) == 0 {
		fmt.Println("No skills found.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(w, "NAME\tSOURCE\tPATH\tDESCRIPTION"); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
		os.Exit(1)
	}
	if _, err := fmt.Fprintln(w, "----\t------\t-----\t-----------"); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
		os.Exit(1)
	}

	for _, s := range skills {
		desc := s.Description
		if len(desc) > 60 {
			desc = desc[:57] + "..."
		}
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.Name, s.Source, s.Path, desc); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
			os.Exit(1)
		}
	}
	if err := w.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "Error flushing output: %v\n", err)
		os.Exit(1)
	}
}

func createCmd(ctx context.Context, args []string) {
	fs := flag.NewFlagSet("create", flag.ExitOnError)
	global := fs.Bool("global", false, "Create in global skills directory")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Usage: eino-skills create <skill-name>\n")
		os.Exit(1)
	}

	name := fs.Arg(0)

	// Determine directory
	var baseDir string
	if *global {
		home, _ := os.UserHomeDir()
		baseDir = filepath.Join(home, ".eino", "agent", "skills")
	} else {
		baseDir = ".eino/skills"
	}

	skillDir := filepath.Join(baseDir, name)

	// Create directory structure
	dirs := []string{
		skillDir,
		filepath.Join(skillDir, "scripts"),
		filepath.Join(skillDir, "references"),
		filepath.Join(skillDir, "assets"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating directory %s: %v\n", dir, err)
			os.Exit(1)
		}
	}

	// Create SKILL.md template
	template := fmt.Sprintf(`---
name: %s
description: "Brief description of what this skill does and when to use it"
---

# %s

## Overview

Describe what this skill does and its main purpose.

## Instructions

### Step 1: [First Step]

Detailed instructions for the first step.

### Step 2: [Second Step]

Detailed instructions for the second step.

## Examples

### Example 1

[Show a concrete example]

## Best Practices

- Practice 1
- Practice 2
- Practice 3

## References

See [references/additional-docs.md](references/additional-docs.md) for more details.
`, name, name)

	skillMDPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillMDPath, []byte(template), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating SKILL.md: %v\n", err)
		os.Exit(1)
	}

	// Create placeholder files
	placeholders := map[string]string{
		filepath.Join(skillDir, "scripts", "example.sh"):            "#!/bin/bash\n# Example script\necho 'Hello from skill script'\n",
		filepath.Join(skillDir, "references", "additional-docs.md"): "# Additional Documentation\n\nAdd detailed reference material here.\n",
		filepath.Join(skillDir, "assets", ".gitkeep"):               "",
	}

	for path, content := range placeholders {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating file %s: %v\n", path, err)
			os.Exit(1)
		}
	}

	fmt.Printf("Created skill '%s' at %s\n", name, skillDir)
	fmt.Println("\nStructure:")
	fmt.Printf("  %s/\n", name)
	fmt.Println("  ├── SKILL.md")
	fmt.Println("  ├── scripts/")
	fmt.Println("  │   └── example.sh")
	fmt.Println("  ├── references/")
	fmt.Println("  │   └── additional-docs.md")
	fmt.Println("  └── assets/")
	fmt.Println("\nNext steps:")
	fmt.Printf("  1. Edit %s to add your instructions\n", skillMDPath)
	fmt.Println("  2. Add scripts, references, and assets as needed")
	fmt.Println("  3. Test with: eino-skills validate", skillDir)
}

func viewCmd(ctx context.Context, args []string) {
	fs := flag.NewFlagSet("view", flag.ExitOnError)
	section := fs.String("section", "", "View specific section")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Usage: eino-skills view <skill-name> [--section <section>]\n")
		os.Exit(1)
	}

	name := fs.Arg(0)
	loader := skill.NewLoader()

	s, err := loader.LoadSkill(ctx, name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading skill '%s': %v\n", name, err)
		os.Exit(1)
	}

	if *section != "" {
		parser := skill.NewParser()
		content := parser.ExtractSection(s.Content, *section)
		if content == "" {
			fmt.Fprintf(os.Stderr, "Section '%s' not found\n", *section)
			os.Exit(1)
		}
		fmt.Println(content)
	} else {
		fmt.Printf("Name: %s\n", s.Name)
		fmt.Printf("Source: %s\n", s.Source)
		fmt.Printf("Path: %s\n", s.Path)
		fmt.Printf("Description: %s\n\n", s.Description)
		fmt.Println("Content:")
		fmt.Println("--------")
		fmt.Println(s.Content)
	}
}

func validateCmd(ctx context.Context, args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Usage: eino-skills validate <skill-path>\n")
		os.Exit(1)
	}

	skillPath := fs.Arg(0)
	skillMDPath := filepath.Join(skillPath, "SKILL.md")

	// Check SKILL.md exists
	if _, err := os.Stat(skillMDPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "❌ SKILL.md not found at %s\n", skillMDPath)
		os.Exit(1)
	}
	fmt.Println("✓ SKILL.md found")

	// Parse and validate
	parser := skill.NewParser()
	fm, content, err := parser.ParseFile(skillMDPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Parse error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ YAML frontmatter valid")

	// Validate frontmatter
	if err := fm.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Frontmatter validation failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Name and description present")

	// Check content length
	if len(content) < 100 {
		fmt.Println("⚠ Content is very short - consider adding more instructions")
	} else {
		fmt.Println("✓ Content length OK")
	}

	// Check for recommended sections
	recommendedSections := []string{"Instructions", "Examples"}
	for _, section := range recommendedSections {
		if sectionContent := parser.ExtractSection(content, section); sectionContent != "" {
			fmt.Printf("✓ '%s' section found\n", section)
		} else {
			fmt.Printf("⚠ '%s' section recommended but not found\n", section)
		}
	}

	fmt.Println("\n✓ Skill validation passed")
}
