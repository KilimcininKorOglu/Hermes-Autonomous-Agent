package idea

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Question represents an interactive question
type Question struct {
	ID       string
	Text     string
	Required bool
	Default  string
}

// QuestionAnswer holds a question and its answer
type QuestionAnswer struct {
	Question Question
	Answer   string
}

// DefaultQuestions are the questions asked in interactive mode
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

// AskQuestions asks interactive questions and returns answers
func AskQuestions(idea string) ([]QuestionAnswer, error) {
	reader := bufio.NewReader(os.Stdin)
	answers := make([]QuestionAnswer, 0, len(DefaultQuestions))

	fmt.Println("\nI'll ask a few questions for additional context:")

	for _, q := range DefaultQuestions {
		defaultHint := ""
		if q.Default != "" {
			defaultHint = fmt.Sprintf(" [%s]", q.Default)
		}

		fmt.Printf("? %s%s\n> ", q.Text, defaultHint)

		input, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		answer := strings.TrimSpace(input)
		if answer == "" {
			answer = q.Default
		}

		answers = append(answers, QuestionAnswer{
			Question: q,
			Answer:   answer,
		})
	}

	fmt.Println()
	return answers, nil
}

// FormatAnswers formats question answers into additional context string
func FormatAnswers(answers []QuestionAnswer) string {
	var sb strings.Builder

	for _, qa := range answers {
		if qa.Answer != "" {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", qa.Question.Text, qa.Answer))
		}
	}

	return sb.String()
}
