package task

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	featureHeaderRegex    = regexp.MustCompile(`(?m)^#\s*Feature\s*(\d+):\s*(.+)$`)
	featureIDRegex        = regexp.MustCompile(`\*\*Feature ID:\*\*\s*(F?\d+)`)
	featureStatusRegex    = regexp.MustCompile(`\*\*Status:\*\*\s*(\w+)`)
	taskHeaderRegex       = regexp.MustCompile(`(?m)^###\s*(T\d+):\s*(.+)$`)
	priorityRegex         = regexp.MustCompile(`\*\*Priority:\*\*\s*(P[1-4])`)
	filesToTouchRegex     = regexp.MustCompile(`\*\*Files to Touch:\*\*\s*(.+)`)
	dependenciesRegex     = regexp.MustCompile(`\*\*Dependencies:\*\*\s*(.+)`)
	targetVersionRegex    = regexp.MustCompile(`\*\*Target Version:\*\*\s*(.+)`)
	estimatedDurationRegex = regexp.MustCompile(`\*\*Estimated Duration:\*\*\s*(.+)`)
	estimatedEffortRegex  = regexp.MustCompile(`\*\*Estimated Effort:\*\*\s*(.+)`)
)

// ParseFeature parses a feature file content
func ParseFeature(content, filePath string) (*Feature, error) {
	feature := &Feature{
		FilePath: filePath,
		Status:   StatusNotStarted,
		Priority: PriorityP2,
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

	// Parse feature priority
	if m := priorityRegex.FindStringSubmatch(content); len(m) > 1 {
		feature.Priority = Priority(m[1])
	}

	// Parse target version
	if m := targetVersionRegex.FindStringSubmatch(content); len(m) > 1 {
		feature.TargetVersion = strings.TrimSpace(m[1])
	}

	// Parse estimated duration
	if m := estimatedDurationRegex.FindStringSubmatch(content); len(m) > 1 {
		feature.EstimatedDuration = strings.TrimSpace(m[1])
	}

	// Parse overview section
	feature.Overview = parseSection(content, "## Overview")

	// Parse goals
	feature.Goals = parseListSection(content, "## Goals")

	// Parse performance targets
	feature.PerformanceTarget = parseSection(content, "## Performance Targets")

	// Parse risk assessment
	feature.RiskAssessment = parseSection(content, "## Risk Assessment")

	// Parse tasks
	feature.Tasks = parseTasks(content, feature.ID)

	return feature, nil
}

func parseSection(content, header string) string {
	lines := strings.Split(content, "\n")
	var result strings.Builder
	inSection := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, header) {
			inSection = true
			continue
		}

		if inSection && strings.HasPrefix(trimmed, "## ") {
			break
		}

		if inSection && strings.HasPrefix(trimmed, "### T") {
			break
		}

		if inSection && trimmed != "" {
			if result.Len() > 0 {
				result.WriteString("\n")
			}
			result.WriteString(trimmed)
		}
	}

	return result.String()
}

func parseListSection(content, header string) []string {
	var items []string
	lines := strings.Split(content, "\n")
	inSection := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, header) {
			inSection = true
			continue
		}

		if inSection && strings.HasPrefix(trimmed, "## ") {
			break
		}

		if inSection && (strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ")) {
			item := strings.TrimPrefix(trimmed, "- ")
			item = strings.TrimPrefix(item, "* ")
			if item != "" {
				items = append(items, item)
			}
		}
	}

	return items
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
		if m := estimatedEffortRegex.FindStringSubmatch(taskContent); len(m) > 1 {
			task.EstimatedEffort = strings.TrimSpace(m[1])
		}
		// Parse files to touch (both inline and section formats)
		if m := filesToTouchRegex.FindStringSubmatch(taskContent); len(m) > 1 {
			task.FilesToTouch = parseCommaSeparated(m[1])
		}
		if len(task.FilesToTouch) == 0 {
			task.FilesToTouch = parseTaskListSection(taskContent, "#### Files to Touch")
		}

		// Parse dependencies (both inline and section formats)
		if m := dependenciesRegex.FindStringSubmatch(taskContent); len(m) > 1 {
			task.Dependencies = parseCommaSeparated(m[1])
		}
		if len(task.Dependencies) == 0 {
			task.Dependencies = parseTaskListSection(taskContent, "#### Dependencies")
		}

		// Parse description
		task.Description = parseTaskSubsection(taskContent, "#### Description")

		// Parse technical details
		task.TechnicalDetails = parseTaskSubsection(taskContent, "#### Technical Details")

		// Parse success criteria (both formats)
		task.SuccessCriteria = parseSuccessCriteria(taskContent)
		if len(task.SuccessCriteria) == 0 {
			task.SuccessCriteria = parseTaskListSection(taskContent, "#### Success Criteria")
		}

		tasks = append(tasks, task)
	}

	return tasks
}

func parseTaskSubsection(content, header string) string {
	lines := strings.Split(content, "\n")
	var result strings.Builder
	inSection := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, header) {
			inSection = true
			continue
		}

		if inSection && (strings.HasPrefix(trimmed, "####") || strings.HasPrefix(trimmed, "###") || strings.HasPrefix(trimmed, "**")) {
			break
		}

		if inSection && trimmed != "" && !strings.HasPrefix(trimmed, "- ") {
			if result.Len() > 0 {
				result.WriteString("\n")
			}
			result.WriteString(trimmed)
		}
	}

	return result.String()
}

func parseTaskListSection(content, header string) []string {
	var items []string
	lines := strings.Split(content, "\n")
	inSection := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, header) {
			inSection = true
			continue
		}

		if inSection && (strings.HasPrefix(trimmed, "####") || strings.HasPrefix(trimmed, "###") || strings.HasPrefix(trimmed, "---")) {
			break
		}

		if inSection && (strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ")) {
			item := strings.TrimPrefix(trimmed, "- ")
			item = strings.TrimPrefix(item, "* ")
			item = strings.TrimPrefix(item, "[ ] ")
			item = strings.TrimPrefix(item, "[x] ")
			// Remove parenthetical comments like "(project structure must exist)"
			if idx := strings.Index(item, "("); idx > 0 {
				item = strings.TrimSpace(item[:idx])
			}
			if item != "" && item != "None" && item != "none" {
				// Handle comma-separated items in a single line
				if strings.Contains(item, ",") {
					for _, subItem := range strings.Split(item, ",") {
						subItem = strings.TrimSpace(subItem)
						// Also clean parenthetical from comma-separated items
						if idx := strings.Index(subItem, "("); idx > 0 {
							subItem = strings.TrimSpace(subItem[:idx])
						}
						if subItem != "" {
							// Expand range format like T031-T038
							expanded := expandTaskRange(subItem)
							items = append(items, expanded...)
						}
					}
				} else {
					// Expand range format like T031-T038
					expanded := expandTaskRange(item)
					items = append(items, expanded...)
				}
			}
		}
	}

	return items
}

// expandTaskRange expands range format like "T031-T038" into individual task IDs
func expandTaskRange(item string) []string {
	// Check if it's a range format (e.g., T031-T038)
	rangeRegex := regexp.MustCompile(`^(T)(\d+)-(T)?(\d+)$`)
	if m := rangeRegex.FindStringSubmatch(item); len(m) > 0 {
		prefix := m[1]
		startNum := 0
		endNum := 0
		fmt.Sscanf(m[2], "%d", &startNum)
		fmt.Sscanf(m[4], "%d", &endNum)
		
		if startNum > 0 && endNum > 0 && endNum >= startNum {
			var expanded []string
			for i := startNum; i <= endNum; i++ {
				expanded = append(expanded, fmt.Sprintf("%s%03d", prefix, i))
			}
			return expanded
		}
	}
	return []string{item}
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
