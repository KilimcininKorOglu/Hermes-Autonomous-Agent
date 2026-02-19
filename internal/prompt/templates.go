package prompt

// DefaultPromptTemplate is the default PROMPT.md content
const DefaultPromptTemplate = `# Project Instructions

## Overview

This project uses Hermes for AI-powered autonomous application development.

## General Rules

- ***When writing text, you will always follow the grammar rules of the language you are using! For example: if it's Turkish, follow Turkish grammar rules; if it's English, follow English grammar rules, and so on.***
- ***Tables in Markdown files must be aligned!***
- ***Readme files will be emoji-free and professionally designed.***
- ***If the tool you're using is giving an error, don't try the same usage repeatedly! Analyze why the tool is giving the error and apply the correct usage!***
- ***There's no time limit. Relax. Do it slowly but surely.*** 

## Code Style

- **camelCase is mandatory** for all variable, function, and property names;
	✓ userName, getUserData, isActive
	✗ user_name, get_user_data, is_active
- File names: camelCase or kebab-case (according to project standards)
- ***Always use code comments to explain complex logic in your code.***
- ***When creating functions, always include parameter and return type annotations.***
- ***When writing code, always follow the best practices and coding standards of the programming language you are using.***
- ***Writing mock code is strictly forbidden, except for test files!***

## Testing

- Mock code may be used **only in unit tests**
- Mocks are strictly forbidden in production or integration code
- Do not create fake/stub/mock implementations outside of tests

## Error Handling

- Never silently ignore errors
- Always log errors with context
- Use appropriate error types for the language

## Security

- Never hardcode secrets, API keys, or passwords
- Never log sensitive information
- Validate all user inputs

## Commit Messages

Use conventional commits:
- feat(scope): add new feature
- fix(scope): fix bug
- refactor(scope): code refactoring
- test(scope): add tests
- docs(scope): documentation

## Guidelines

1. Follow existing code patterns
2. Write tests for new functionality
3. Use conventional commits
4. Keep changes focused and atomic

## Completion Criteria

Mark task as COMPLETE only when:
- All success criteria in the task are met
- Code compiles without errors
- Tests pass (if applicable)
- No TODO comments left unresolved

## Status Reporting

At the end of each response, output:

` + "```" + `
---HERMES_STATUS---
STATUS: IN_PROGRESS | COMPLETE | BLOCKED | AT_RISK | PAUSED
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
