package converter

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ProjectAnalyzer analyzes project structure and content
type ProjectAnalyzer struct {
	rootDir        string
	maxDepth       int
	excludeDirs    []string
	maxFiles       int
	maxLinesPerFile int
	maxFileSize    int64
}

// AnalysisResult contains the analyzed project information
type AnalysisResult struct {
	ProjectName    string
	FileTree       string
	ReadmeContent  string
	Dependencies   map[string]string
	ConfigFiles    map[string]string
	EntryPoints    map[string]string
	TechStack      []string
	ProjectType    string
	TotalFiles     int
	TotalDirs      int
}

// NewProjectAnalyzer creates a new project analyzer
func NewProjectAnalyzer(rootDir string, maxDepth int, excludeDirs []string) *ProjectAnalyzer {
	if maxDepth <= 0 {
		maxDepth = 3
	}
	if len(excludeDirs) == 0 {
		excludeDirs = []string{
			"node_modules", "vendor", ".git", ".hermes", "dist", "build",
			"bin", "__pycache__", ".venv", "venv", ".idea", ".vscode",
			"target", "coverage", ".next", ".nuxt", "out",
		}
	}
	return &ProjectAnalyzer{
		rootDir:        rootDir,
		maxDepth:       maxDepth,
		excludeDirs:    excludeDirs,
		maxFiles:       100,
		maxLinesPerFile: 200,
		maxFileSize:    50 * 1024, // 50KB
	}
}

// Analyze performs full project analysis
func (a *ProjectAnalyzer) Analyze() (*AnalysisResult, error) {
	result := &AnalysisResult{
		Dependencies: make(map[string]string),
		ConfigFiles:  make(map[string]string),
		EntryPoints:  make(map[string]string),
		TechStack:    []string{},
	}

	// Get project name from directory
	absPath, err := filepath.Abs(a.rootDir)
	if err != nil {
		return nil, err
	}
	result.ProjectName = filepath.Base(absPath)

	// Build file tree
	tree, totalFiles, totalDirs := a.buildFileTree()
	result.FileTree = tree
	result.TotalFiles = totalFiles
	result.TotalDirs = totalDirs

	// Read README
	result.ReadmeContent = a.readReadme()

	// Detect dependencies and tech stack
	a.detectDependencies(result)

	// Find config files
	a.findConfigFiles(result)

	// Find entry points
	a.findEntryPoints(result)

	// Determine project type
	result.ProjectType = a.detectProjectType(result)

	return result, nil
}

func (a *ProjectAnalyzer) buildFileTree() (string, int, int) {
	var sb strings.Builder
	totalFiles := 0
	totalDirs := 0

	filepath.WalkDir(a.rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		relPath, _ := filepath.Rel(a.rootDir, path)
		if relPath == "." {
			return nil
		}

		// Check depth
		depth := strings.Count(relPath, string(filepath.Separator))
		if depth >= a.maxDepth {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Check excluded directories
		for _, excluded := range a.excludeDirs {
			if d.IsDir() && d.Name() == excluded {
				return filepath.SkipDir
			}
		}

		// Build tree line
		indent := strings.Repeat("  ", depth)
		if d.IsDir() {
			sb.WriteString(indent + d.Name() + "/\n")
			totalDirs++
		} else {
			sb.WriteString(indent + d.Name() + "\n")
			totalFiles++
		}

		return nil
	})

	return sb.String(), totalFiles, totalDirs
}

func (a *ProjectAnalyzer) readReadme() string {
	readmeFiles := []string{"README.md", "README.MD", "readme.md", "README", "README.txt"}
	for _, name := range readmeFiles {
		path := filepath.Join(a.rootDir, name)
		if content, err := a.readFileContent(path); err == nil {
			return content
		}
	}
	return ""
}

func (a *ProjectAnalyzer) detectDependencies(result *AnalysisResult) {
	depFiles := map[string]string{
		"package.json":      "Node.js",
		"go.mod":            "Go",
		"requirements.txt":  "Python",
		"Pipfile":           "Python",
		"pyproject.toml":    "Python",
		"Cargo.toml":        "Rust",
		"pom.xml":           "Java/Maven",
		"build.gradle":      "Java/Gradle",
		"composer.json":     "PHP",
		"Gemfile":           "Ruby",
		"pubspec.yaml":      "Dart/Flutter",
		"Package.swift":     "Swift",
		"*.csproj":          ".NET",
	}

	for file, tech := range depFiles {
		if strings.Contains(file, "*") {
			// Handle glob patterns
			pattern := filepath.Join(a.rootDir, file)
			matches, _ := filepath.Glob(pattern)
			if len(matches) > 0 {
				for _, match := range matches {
					if content, err := a.readFileContent(match); err == nil {
						relPath, _ := filepath.Rel(a.rootDir, match)
						result.Dependencies[relPath] = content
						if !contains(result.TechStack, tech) {
							result.TechStack = append(result.TechStack, tech)
						}
					}
				}
			}
		} else {
			path := filepath.Join(a.rootDir, file)
			if content, err := a.readFileContent(path); err == nil {
				result.Dependencies[file] = content
				if !contains(result.TechStack, tech) {
					result.TechStack = append(result.TechStack, tech)
				}
			}
		}
	}
}

func (a *ProjectAnalyzer) findConfigFiles(result *AnalysisResult) {
	configFiles := []string{
		".env.example", ".env.sample", "config.json", "config.yaml", "config.yml",
		"docker-compose.yml", "docker-compose.yaml", "Dockerfile",
		"tsconfig.json", ".eslintrc.js", ".eslintrc.json",
		"webpack.config.js", "vite.config.js", "next.config.js",
		"Makefile", "build.bat", ".goreleaser.yml",
	}

	for _, file := range configFiles {
		path := filepath.Join(a.rootDir, file)
		if content, err := a.readFileContent(path); err == nil {
			result.ConfigFiles[file] = content
		}
	}

	// Check config directory
	configDir := filepath.Join(a.rootDir, "config")
	if info, err := os.Stat(configDir); err == nil && info.IsDir() {
		filepath.WalkDir(configDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if content, err := a.readFileContent(path); err == nil {
				relPath, _ := filepath.Rel(a.rootDir, path)
				result.ConfigFiles[relPath] = content
			}
			return nil
		})
	}
}

func (a *ProjectAnalyzer) findEntryPoints(result *AnalysisResult) {
	entryPoints := []string{
		"main.go", "cmd/main.go",
		"index.js", "index.ts", "src/index.js", "src/index.ts",
		"app.js", "app.ts", "src/app.js", "src/app.ts",
		"main.py", "app.py", "src/main.py",
		"main.rs", "src/main.rs",
		"Main.java", "src/Main.java",
		"Program.cs",
	}

	for _, file := range entryPoints {
		path := filepath.Join(a.rootDir, file)
		if content, err := a.readFileContent(path); err == nil {
			result.EntryPoints[file] = content
		}
	}

	// Check cmd directory for Go projects
	cmdDir := filepath.Join(a.rootDir, "cmd")
	if info, err := os.Stat(cmdDir); err == nil && info.IsDir() {
		filepath.WalkDir(cmdDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if strings.HasSuffix(d.Name(), ".go") {
				if content, err := a.readFileContent(path); err == nil {
					relPath, _ := filepath.Rel(a.rootDir, path)
					result.EntryPoints[relPath] = content
				}
			}
			return nil
		})
	}
}

func (a *ProjectAnalyzer) detectProjectType(result *AnalysisResult) string {
	// Based on tech stack and file patterns
	if contains(result.TechStack, "Go") {
		if _, ok := result.EntryPoints["cmd/main.go"]; ok {
			return "Go CLI Application"
		}
		return "Go Application"
	}
	if contains(result.TechStack, "Node.js") {
		if _, ok := result.ConfigFiles["next.config.js"]; ok {
			return "Next.js Application"
		}
		if _, ok := result.ConfigFiles["vite.config.js"]; ok {
			return "Vite Application"
		}
		return "Node.js Application"
	}
	if contains(result.TechStack, "Python") {
		return "Python Application"
	}
	if contains(result.TechStack, "Rust") {
		return "Rust Application"
	}
	if contains(result.TechStack, "Java/Maven") || contains(result.TechStack, "Java/Gradle") {
		return "Java Application"
	}
	if contains(result.TechStack, ".NET") {
		return ".NET Application"
	}

	return "Software Project"
}

func (a *ProjectAnalyzer) readFileContent(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	// Skip if too large
	if info.Size() > a.maxFileSize {
		return "[File too large - truncated]\n", nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	// Truncate if too many lines
	lines := strings.Split(string(content), "\n")
	if len(lines) > a.maxLinesPerFile {
		lines = lines[:a.maxLinesPerFile]
		return strings.Join(lines, "\n") + "\n\n[... truncated ...]", nil
	}

	return string(content), nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
