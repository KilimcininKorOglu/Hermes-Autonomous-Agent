# Phase 13: Idea to PRD Generation

**Version:** 1.1.0

## Overview

A new command that generates a detailed PRD (Product Requirements Document) from a simple one-liner idea or short description.

## Use Cases

```bash
# Simple idea
hermes idea "e-commerce website"

# Detailed idea
hermes idea "real-time chat application with React and Node.js"

# Specify output file
hermes idea "task management app" --output .hermes/docs/PRD.md

# Dry-run mode
hermes idea "blog platform" --dry-run

# Interactive mode (asks additional questions)
hermes idea "CRM system" --interactive
```

## CLI Command

### Command Structure

```
hermes idea <idea> [flags]
```

### Flags

| Flag            | Short | Default                 | Description                           |
|-----------------|-------|-------------------------|---------------------------------------|
| `--output`      | `-o`  | `.hermes/docs/PRD.md`   | Output file path                      |
| `--dry-run`     |       | false                   | Preview without writing file          |
| `--interactive` | `-i`  | false                   | Interactive mode (additional questions)|
| `--language`    | `-l`  | `en`                    | PRD language (en/tr)                  |
| `--timeout`     |       | 600                     | AI timeout in seconds                 |
| `--debug`       |       | false                   | Enable debug output                   |

## Technical Design

### File Structure

```
internal/
├── cmd/
│   └── idea.go              # CLI command
└── idea/
    ├── generator.go         # PRD generator
    ├── prompt.go            # AI prompt templates
    ├── questions.go         # Interactive questions
    └── templates.go         # PRD templates
```

### Generator Interface

```go
// internal/idea/generator.go
package idea

type Generator struct {
    ai       ai.Provider
    config   *config.Config
    logger   *ui.Logger
}

type GenerateOptions struct {
    Idea        string
    Output      string
    DryRun      bool
    Interactive bool
    Language    string
    Timeout     int
}

type GenerateResult struct {
    PRDContent  string
    FilePath    string
    TokensUsed  int
    Duration    time.Duration
}

func NewGenerator(ai ai.Provider, cfg *config.Config, logger *ui.Logger) *Generator

func (g *Generator) Generate(ctx context.Context, opts GenerateOptions) (*GenerateResult, error)

func (g *Generator) AskQuestions(idea string) ([]QuestionAnswer, error)
```

### AI Prompt Template

```go
// internal/idea/prompt.go
package idea

const PRDGenerationPrompt = `You are a senior product manager. Generate a detailed PRD (Product Requirements Document) for the following idea.

## Idea
{{.Idea}}

{{if .AdditionalContext}}
## Additional Context
{{.AdditionalContext}}
{{end}}

## Requirements

Generate a comprehensive PRD in Markdown format with the following sections:

1. **Project Overview**
   - Project name
   - Brief description
   - Target audience
   - Key objectives

2. **Features**
   - List 3-6 main features
   - Each feature should have:
     - Clear name
     - Description
     - User stories (2-3 per feature)
     - Acceptance criteria

3. **Technical Requirements**
   - Technology stack recommendations
   - Architecture overview
   - Integration requirements
   - Performance requirements

4. **Non-Functional Requirements**
   - Security requirements
   - Scalability considerations
   - Accessibility requirements

5. **Success Metrics**
   - KPIs
   - Success criteria

6. **Timeline & Milestones**
   - Phase breakdown
   - Estimated timeline

## Output Format

Output ONLY the PRD content in Markdown format. Do not include any explanations or meta-commentary.
Language: {{.Language}}
`
```

### Interactive Questions

```go
// internal/idea/questions.go
package idea

type Question struct {
    ID       string
    Text     string
    Required bool
    Default  string
}

var DefaultQuestions = []Question{
    {
        ID:       "target_audience",
        Text:     "Who is the target audience?",
        Required: false,
        Default:  "",
    },
    {
        ID:       "tech_stack",
        Text:     "Any preferred technology stack? (e.g., React, Go, PostgreSQL)",
        Required: false,
        Default:  "",
    },
    {
        ID:       "scale",
        Text:     "Expected scale? (small/medium/large/enterprise)",
        Required: false,
        Default:  "medium",
    },
    {
        ID:       "timeline",
        Text:     "Expected timeline? (e.g., 2 weeks, 1 month, 3 months)",
        Required: false,
        Default:  "",
    },
    {
        ID:       "priority_features",
        Text:     "Any must-have features?",
        Required: false,
        Default:  "",
    },
}

type QuestionAnswer struct {
    Question Question
    Answer   string
}

func (g *Generator) AskQuestions(idea string) ([]QuestionAnswer, error) {
    // Ask questions interactively from terminal
    // Use bufio.Scanner
}
```

### CLI Command Implementation

```go
// internal/cmd/idea.go
package cmd

import (
    "context"
    "fmt"
    "os"
    "path/filepath"
    "time"

    "github.com/spf13/cobra"
    "hermes/internal/ai"
    "hermes/internal/config"
    "hermes/internal/idea"
    "hermes/internal/ui"
)

func NewIdeaCmd() *cobra.Command {
    var (
        output      string
        dryRun      bool
        interactive bool
        language    string
        timeout     int
        debug       bool
    )

    cmd := &cobra.Command{
        Use:   "idea <description>",
        Short: "Generate PRD from idea",
        Long:  "Generate a detailed Product Requirements Document from a simple idea or description",
        Args:  cobra.MinimumNArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            ideaText := strings.Join(args, " ")
            return ideaExecute(ideaText, output, dryRun, interactive, language, timeout, debug)
        },
    }

    cmd.Flags().StringVarP(&output, "output", "o", ".hermes/docs/PRD.md", "Output file path")
    cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview without writing file")
    cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Interactive mode with additional questions")
    cmd.Flags().StringVarP(&language, "language", "l", "en", "PRD language (en/tr)")
    cmd.Flags().IntVar(&timeout, "timeout", 600, "AI timeout in seconds")
    cmd.Flags().BoolVar(&debug, "debug", false, "Enable debug output")

    return cmd
}

func ideaExecute(ideaText, output string, dryRun, interactive bool, language string, timeout int, debug bool) error {
    // 1. Load config
    cfg, err := config.Load()
    if err != nil {
        return err
    }

    // 2. Create logger
    logger := ui.NewLogger(debug)

    // 3. Get AI provider
    provider, err := ai.GetProvider(cfg.AI.Planning)
    if err != nil {
        return err
    }

    // 4. Create generator
    gen := idea.NewGenerator(provider, cfg, logger)

    // 5. Interactive mode
    var additionalContext string
    if interactive {
        answers, err := gen.AskQuestions(ideaText)
        if err != nil {
            return err
        }
        additionalContext = formatAnswers(answers)
    }

    // 6. Generate PRD
    ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
    defer cancel()

    result, err := gen.Generate(ctx, idea.GenerateOptions{
        Idea:              ideaText,
        Output:            output,
        DryRun:            dryRun,
        Interactive:       interactive,
        Language:          language,
        Timeout:           timeout,
        AdditionalContext: additionalContext,
    })
    if err != nil {
        return err
    }

    // 7. Show result
    if dryRun {
        fmt.Println("=== PRD Preview ===")
        fmt.Println(result.PRDContent)
        fmt.Println("===================")
        fmt.Printf("\nWould be written to: %s\n", result.FilePath)
    } else {
        logger.Success("PRD generated: %s", result.FilePath)
        logger.Info("Tokens used: %d", result.TokensUsed)
        logger.Info("Duration: %s", result.Duration)
        
        fmt.Println("\nNext steps:")
        fmt.Printf("  1. Review: cat %s\n", result.FilePath)
        fmt.Printf("  2. Parse:  hermes prd %s\n", result.FilePath)
    }

    return nil
}
```

## PRD Output Format

Generated PRD will follow this format:

```markdown
# Project Name - Product Requirements Document

## 1. Project Overview

### 1.1 Description
Brief description of the project...

### 1.2 Target Audience
- Primary: ...
- Secondary: ...

### 1.3 Key Objectives
- Objective 1
- Objective 2
- Objective 3

## 2. Features

### Feature 1: Feature Name
**Description:** Detailed description...

**User Stories:**
- As a [user type], I want to [action] so that [benefit]
- As a [user type], I want to [action] so that [benefit]

**Acceptance Criteria:**
- Criterion 1
- Criterion 2
- Criterion 3

### Feature 2: Feature Name
...

## 3. Technical Requirements

### 3.1 Technology Stack
- Frontend: ...
- Backend: ...
- Database: ...
- Infrastructure: ...

### 3.2 Architecture
Overview of system architecture...

### 3.3 Integrations
- Integration 1
- Integration 2

### 3.4 Performance Requirements
- Response time: < X ms
- Concurrent users: X
- Uptime: X%

## 4. Non-Functional Requirements

### 4.1 Security
- Requirement 1
- Requirement 2

### 4.2 Scalability
- Consideration 1
- Consideration 2

### 4.3 Accessibility
- WCAG 2.1 AA compliance
- ...

## 5. Success Metrics

### 5.1 KPIs
| Metric | Target | Measurement |
|--------|--------|-------------|
| ...    | ...    | ...         |

### 5.2 Success Criteria
- Criterion 1
- Criterion 2

## 6. Timeline & Milestones

### Phase 1: Foundation (Week 1-2)
- Milestone 1
- Milestone 2

### Phase 2: Core Features (Week 3-4)
- Milestone 3
- Milestone 4

### Phase 3: Polish & Launch (Week 5-6)
- Milestone 5
- Milestone 6
```

## Workflow

```
+------------------+
|   hermes idea    |
|   "e-commerce"   |
+--------+---------+
         |
         v
+------------------+
| Interactive?     |
+--------+---------+
    |         |
   Yes        No
    |         |
    v         |
+------------------+
| Ask questions    |
| - Target audience|
| - Tech stack     |
| - Scale          |
+--------+---------+
         |
         v
+------------------+
| Send to AI       |
| (Claude/Droid/   |
|  Gemini)         |
+--------+---------+
         |
         v
+------------------+
| Generate PRD     |
| Markdown         |
+--------+---------+
         |
         v
+------------------+
| Dry-run?         |
+--------+---------+
    |         |
   Yes        No
    |         |
    v         v
+----------+ +------------------+
| Display  | | Write to file    |
| on screen| | .hermes/docs/    |
+----------+ | PRD.md           |
             +--------+---------+
                      |
                      v
             +------------------+
             | Show next steps  |
             +------------------+
```

## Language Support

### English (en)

Default language. All section headers and content in English.

### Turkish (tr)

```bash
hermes idea "e-commerce website" --language tr
```

Turkish PRD output:

- Proje Genel Bakisi
- Ozellikler
- Teknik Gereksinimler
- Fonksiyonel Olmayan Gereksinimler
- Basari Metrikleri
- Zaman Cizelgesi ve Kilometre Taslari

## Error Handling

| Error                   | Message                                        |
|-------------------------|------------------------------------------------|
| AI not found            | "AI provider not found (install claude/droid)" |
| Timeout                 | "PRD generation timed out after X seconds"     |
| File write error        | "Failed to write PRD: <error>"                 |
| .hermes dir missing     | "Run 'hermes init' first"                      |
| Empty idea              | "Idea description is required"                 |

## Tests

```go
// internal/idea/generator_test.go

func TestGenerator_Generate(t *testing.T) {
    // Test with mock AI provider
}

func TestGenerator_AskQuestions(t *testing.T) {
    // Test with stdin mock
}

func TestPRDFormat(t *testing.T) {
    // Validate output format
}
```

## main.go Integration

```go
// cmd/hermes/main.go

func init() {
    rootCmd.AddCommand(cmd.NewIdeaCmd())
}
```

## Config Support

```json
{
  "ai": {
    "planning": "claude",
    "ideaTimeout": 600
  },
  "idea": {
    "defaultLanguage": "en",
    "defaultOutput": ".hermes/docs/PRD.md",
    "interactive": false
  }
}
```

## Example Usage

### Simple Usage

```bash
$ hermes idea "task management application with team collaboration"

 _   _
| | | | ___ _ __ _ __ ___   ___  ___
| |_| |/ _ \ '__| '_ ` _ \ / _ \/ __|
|  _  |  __/ |  | | | | | |  __/\__ \
|_| |_|\___|_|  |_| |_| |_|\___||___/

      AI-Powered Application Development

Idea to PRD Generator
=====================

Idea: task management application with team collaboration
AI: claude
Language: en

Generating PRD...

[SUCCESS] PRD generated: .hermes/docs/PRD.md

Next steps:
  1. Review: cat .hermes/docs/PRD.md
  2. Parse:  hermes prd .hermes/docs/PRD.md
```

### Interactive Usage

```bash
$ hermes idea "CRM system" --interactive

Idea to PRD Generator
=====================

Idea: CRM system

I'll ask a few questions for additional context:

? Who is the target audience? [Small businesses]
> Mid-size B2B companies

? Any preferred technology stack? [Not specified]
> React, Node.js, PostgreSQL

? Expected scale? [medium]
> large

? Expected timeline? [Not specified]
> 3 months

? Any must-have features? [Not specified]
> Customer tracking, sales pipeline, reporting

Generating PRD...

[SUCCESS] PRD generated: .hermes/docs/PRD.md
```

## Migration Notes

This feature is additive to the existing system, no breaking changes.

### New Files

- `internal/cmd/idea.go`
- `internal/idea/generator.go`
- `internal/idea/prompt.go`
- `internal/idea/questions.go`
- `internal/idea/templates.go`
- `internal/idea/generator_test.go`

### Updated Files

- `cmd/hermes/main.go` - Add new command
- `internal/config/config.go` - New config fields

## Version

This feature will be released in **v1.1.0**.

### Changelog

```markdown
## [1.1.0] - YYYY-MM-DD

### Added
- `hermes idea` command to generate PRD from simple idea
- Interactive mode with additional questions
- Multi-language PRD support (en/tr)
- Dry-run mode for PRD preview
```
