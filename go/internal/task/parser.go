package task

import (
	"regexp"
	"strings"
)

var (
	featureHeaderRegex = regexp.MustCompile(`(?m)^#\s*Feature\s*(\d+):\s*(.+)$`)
	featureIDRegex     = regexp.MustCompile(`\*\*Feature ID:\*\*\s*(F?\d+)`)
	featureStatusRegex = regexp.MustCompile(`\*\*Status:\*\*\s*(\w+)`)
	taskHeaderRegex    = regexp.MustCompile(`(?m)^###\s*(T\d+):\s*(.+)$`)
	priorityRegex      = regexp.MustCompile(`\*\*Priority:\*\*\s*(P[1-4])`)
	filesToTouchRegex  = regexp.MustCompile(`\*\*Files to Touch:\*\*\s*(.+)`)
	dependenciesRegex  = regexp.MustCompile(`\*\*Dependencies:\*\*\s*(.+)`)
)

// ParseFeature parses a feature file content
func ParseFeature(content, filePath string) (*Feature, error) {
	feature := &Feature{
		FilePath: filePath,
		Status:   StatusNotStarted,
	}

	// Parse feature header (# Feature N: Name)
	if m := featureHeaderRegex.FindStringSubmatch(content); len(m) > 2 {
		feature.Name = strings.TrimSpace(m[2])
	}

	// Parse feature ID (**Feature ID:** FXXX)
	if m := featureIDRegex.FindStringSubmatch(content); len(m) > 1 {
		id := m[1]
		if !strings.HasPrefix(id, "F") {
			id = "F" + id
		}
		feature.ID = id
	}

	// Parse feature status
	if m := featureStatusRegex.FindStringSubmatch(content); len(m) > 1 {
		feature.Status = Status(m[1])
	}

	// Parse tasks
	feature.Tasks = parseTasks(content, feature.ID)

	return feature, nil
}

func parseTasks(content, featureID string) []Task {
	var tasks []Task

	// Find all task headers
	taskMatches := taskHeaderRegex.FindAllStringSubmatchIndex(content, -1)
	if len(taskMatches) == 0 {
		return tasks
	}

	for i, match := range taskMatches {
		taskID := content[match[2]:match[3]]
		taskName := strings.TrimSpace(content[match[4]:match[5]])

		// Get task content (until next task or end)
		startIdx := match[1]
		var endIdx int
		if i < len(taskMatches)-1 {
			endIdx = taskMatches[i+1][0]
		} else {
			endIdx = len(content)
		}
		taskContent := content[startIdx:endIdx]

		task := Task{
			ID:        taskID,
			Name:      taskName,
			FeatureID: featureID,
			Status:    StatusNotStarted,
			Priority:  PriorityP2,
		}

		// Parse task attributes
		if m := featureStatusRegex.FindStringSubmatch(taskContent); len(m) > 1 {
			task.Status = Status(m[1])
		}
		if m := priorityRegex.FindStringSubmatch(taskContent); len(m) > 1 {
			task.Priority = Priority(m[1])
		}
		if m := filesToTouchRegex.FindStringSubmatch(taskContent); len(m) > 1 {
			task.FilesToTouch = parseCommaSeparated(m[1])
		}
		if m := dependenciesRegex.FindStringSubmatch(taskContent); len(m) > 1 {
			task.Dependencies = parseCommaSeparated(m[1])
		}

		// Parse success criteria
		task.SuccessCriteria = parseSuccessCriteria(taskContent)

		tasks = append(tasks, task)
	}

	return tasks
}

func parseCommaSeparated(s string) []string {
	var items []string
	for _, item := range strings.Split(s, ",") {
		item = strings.TrimSpace(item)
		if item != "" && item != "None" && item != "none" {
			items = append(items, item)
		}
	}
	return items
}

func parseSuccessCriteria(content string) []string {
	var criteria []string
	lines := strings.Split(content, "\n")
	inCriteria := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.Contains(line, "**Success Criteria:**") {
			inCriteria = true
			continue
		}

		// Stop at next section or task
		if inCriteria && (strings.HasPrefix(trimmed, "**") || strings.HasPrefix(trimmed, "###")) {
			break
		}

		if inCriteria && strings.HasPrefix(trimmed, "-") {
			criterion := strings.TrimPrefix(trimmed, "- ")
			criterion = strings.TrimPrefix(criterion, "* ")
			if criterion != "" {
				criteria = append(criteria, criterion)
			}
		}
	}

	return criteria
}
