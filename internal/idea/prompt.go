package idea

import (
	"fmt"
	"strings"
)

// BuildPrompt builds the AI prompt for PRD generation
func BuildPrompt(idea, language, additionalContext string) string {
	var sb strings.Builder

	sb.WriteString("You are a senior product manager. Generate a detailed PRD (Product Requirements Document) for the following idea.\n\n")

	sb.WriteString("## Idea\n")
	sb.WriteString(idea)
	sb.WriteString("\n\n")

	if additionalContext != "" {
		sb.WriteString("## Additional Context\n")
		sb.WriteString(additionalContext)
		sb.WriteString("\n\n")
	}

	sb.WriteString(`## Requirements

Generate a comprehensive PRD in Markdown format with the following sections:

1. **Project Overview**
   - Project name (derive from idea)
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

FILE CREATION RULES:
- If creating a file, create it ONLY in .hermes/docs/ directory
- Use filename: .hermes/docs/PRD.md
- Do NOT create files anywhere else

Output ONLY the PRD content in Markdown format. Do not include any explanations or meta-commentary.
Start directly with the project title as a level-1 heading.

`)

	sb.WriteString(fmt.Sprintf("Language: %s\n", getLanguageName(language)))

	if language == "tr" {
		sb.WriteString(`
Note: Write the entire PRD in Turkish. Use Turkish section headers:
- Proje Genel Bakisi
- Ozellikler
- Teknik Gereksinimler
- Fonksiyonel Olmayan Gereksinimler
- Basari Metrikleri
- Zaman Cizelgesi ve Kilometre Taslari
`)
	}

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
