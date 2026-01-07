package converter

import (
	"fmt"
	"strings"
)

// BuildPrompt builds the AI prompt for PRD generation from project analysis
func BuildPrompt(result *AnalysisResult, language string) string {
	var sb strings.Builder

	sb.WriteString("You are a senior product manager and software architect. Analyze this existing project and generate a comprehensive PRD (Product Requirements Document) that describes what this project does.\n\n")

	// Project info
	sb.WriteString("## Project Information\n\n")
	sb.WriteString(fmt.Sprintf("**Project Name:** %s\n", result.ProjectName))
	sb.WriteString(fmt.Sprintf("**Project Type:** %s\n", result.ProjectType))
	sb.WriteString(fmt.Sprintf("**Tech Stack:** %s\n", strings.Join(result.TechStack, ", ")))
	sb.WriteString(fmt.Sprintf("**Total Files:** %d\n", result.TotalFiles))
	sb.WriteString(fmt.Sprintf("**Total Directories:** %d\n\n", result.TotalDirs))

	// File structure
	sb.WriteString("## Project Structure\n\n")
	sb.WriteString("```\n")
	sb.WriteString(result.FileTree)
	sb.WriteString("```\n\n")

	// README content
	if result.ReadmeContent != "" {
		sb.WriteString("## Existing README\n\n")
		sb.WriteString("```markdown\n")
		sb.WriteString(result.ReadmeContent)
		sb.WriteString("\n```\n\n")
	}

	// Dependencies
	if len(result.Dependencies) > 0 {
		sb.WriteString("## Dependencies\n\n")
		for file, content := range result.Dependencies {
			sb.WriteString(fmt.Sprintf("### %s\n\n", file))
			sb.WriteString("```\n")
			sb.WriteString(truncateContent(content, 100))
			sb.WriteString("\n```\n\n")
		}
	}

	// Config files
	if len(result.ConfigFiles) > 0 {
		sb.WriteString("## Configuration Files\n\n")
		for file, content := range result.ConfigFiles {
			sb.WriteString(fmt.Sprintf("### %s\n\n", file))
			sb.WriteString("```\n")
			sb.WriteString(truncateContent(content, 50))
			sb.WriteString("\n```\n\n")
		}
	}

	// Entry points
	if len(result.EntryPoints) > 0 {
		sb.WriteString("## Entry Points / Main Files\n\n")
		for file, content := range result.EntryPoints {
			sb.WriteString(fmt.Sprintf("### %s\n\n", file))
			sb.WriteString("```\n")
			sb.WriteString(truncateContent(content, 100))
			sb.WriteString("\n```\n\n")
		}
	}

	// Instructions
	sb.WriteString(`## Task

Based on the project analysis above, generate a comprehensive PRD that documents this existing project. The PRD should:

1. **Accurately describe** what the project currently does (not what it could do)
2. **Document existing features** based on the code analysis
3. **Identify the architecture** and design patterns used
4. **List actual dependencies** and their purposes
5. **Describe the current state** of the project

## Output Format

Generate a PRD in Markdown format with these sections:

1. **Project Overview**
   - Project name
   - Description (what it does)
   - Target users
   - Current status

2. **Features** (existing, based on code analysis)
   - List each feature found in the codebase
   - Describe what each feature does
   - Note the implementation status

3. **Technical Architecture**
   - Technology stack (with versions if available)
   - Project structure explanation
   - Key components and their responsibilities
   - Data flow

4. **Dependencies**
   - List main dependencies
   - Explain why each is used

5. **Configuration**
   - Environment variables needed
   - Configuration options

6. **API / Commands** (if applicable)
   - List endpoints or CLI commands
   - Describe parameters and usage

7. **Development Setup**
   - Prerequisites
   - Installation steps
   - Build commands
   - Test commands

8. **Current Limitations**
   - Known issues or limitations
   - Areas for improvement

`)

	// Language instruction
	sb.WriteString(fmt.Sprintf("\n**Output Language:** %s\n", getLanguageName(language)))

	if language == "tr" {
		sb.WriteString(`
Note: Write the entire PRD in Turkish. Use Turkish section headers:
- Proje Genel Bakisi
- Ozellikler
- Teknik Mimari
- Bagimliliklar
- Yapilandirma
- API / Komutlar
- Gelistirme Ortami Kurulumu
- Mevcut Kisitlamalar
`)
	}

	sb.WriteString("\nFILE CREATION RULES:\n")
	sb.WriteString("- If creating a file, create it ONLY in .hermes/docs/ directory\n")
	sb.WriteString("- Use filename: .hermes/docs/PRD.md\n")
	sb.WriteString("- Do NOT create files anywhere else\n\n")

	sb.WriteString("Output ONLY the PRD content in Markdown format. Start directly with the project title as a level-1 heading. Do not include any explanations or meta-commentary.\n")

	return sb.String()
}

func getLanguageName(code string) string {
	switch code {
	case "tr":
		return "Turkish"
	case "en":
		return "English"
	default:
		return "English"
	}
}

func truncateContent(content string, maxLines int) string {
	lines := strings.Split(content, "\n")
	if len(lines) <= maxLines {
		return content
	}
	return strings.Join(lines[:maxLines], "\n") + "\n\n[... truncated ...]"
}
