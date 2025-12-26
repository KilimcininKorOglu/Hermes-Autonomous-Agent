package prompt

// DefaultPromptTemplate is the default PROMPT.md content
const DefaultPromptTemplate = `# Project Instructions

## Overview

This project uses Hermes for autonomous AI development.

## Guidelines

1. Follow existing code patterns
2. Write tests for new functionality
3. Use conventional commits
4. Keep changes focused and atomic

## Status Reporting

At the end of each response, output:

` + "```" + `
---HERMES_STATUS---
STATUS: IN_PROGRESS | COMPLETE | BLOCKED
EXIT_SIGNAL: false | true
RECOMMENDATION: <next action>
---END_HERMES_STATUS---
` + "```" + `
`

// CreateDefault creates the default prompt if it doesn't exist
func (i *Injector) CreateDefault() error {
	if i.Exists() {
		return nil
	}

	return i.Write(DefaultPromptTemplate)
}

// EnsureExists creates the prompt with default content if missing
func (i *Injector) EnsureExists() error {
	return i.CreateDefault()
}
