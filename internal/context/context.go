package context

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Step represents a progress step with timestamp
type Step struct {
	Timestamp string
	Message   string
}

// Context represents a saved Claude context file
type Context struct {
	Name      string // directory basename (was IssueID)
	Title     string
	Branch    string
	WorkDone  string
	KeyFiles  []string
	Steps     []Step
	FilePath  string
}

// stepRegex matches lines like "- [2024-01-09 14:30] message"
var stepRegex = regexp.MustCompile(`^-\s*\[([^\]]+)\]\s*(.+)$`)

// Load reads a context file from .claude/contexts/{name}.md
func Load(repoPath, name string) (*Context, error) {
	filePath := filepath.Join(repoPath, ".claude", "contexts", name+".md")
	return LoadFromFile(filePath, name)
}

// LoadFromFile reads a context file from a specific path
func LoadFromFile(filePath, name string) (*Context, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	ctx := &Context{
		Name:     name,
		FilePath: filePath,
	}

	scanner := bufio.NewScanner(file)
	var currentSection string
	var sectionContent strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		// Check for section headers
		if strings.HasPrefix(line, "# ") {
			// Main title
			ctx.Title = strings.TrimPrefix(line, "# ")
			continue
		}

		if strings.HasPrefix(line, "## ") {
			// Save previous section content
			saveSection(ctx, currentSection, sectionContent.String())
			sectionContent.Reset()

			currentSection = strings.TrimPrefix(line, "## ")
			continue
		}

		// Accumulate content for current section
		if currentSection != "" {
			sectionContent.WriteString(line)
			sectionContent.WriteString("\n")
		}
	}

	// Save last section
	saveSection(ctx, currentSection, sectionContent.String())

	return ctx, scanner.Err()
}

func saveSection(ctx *Context, section, content string) {
	content = strings.TrimSpace(content)
	if content == "" {
		return
	}

	switch section {
	case "Branch":
		ctx.Branch = content
	case "Work Done":
		ctx.WorkDone = content
	case "Key Files":
		// Parse bullet list
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			line = strings.TrimPrefix(line, "- ")
			line = strings.TrimPrefix(line, "* ")
			if line != "" {
				ctx.KeyFiles = append(ctx.KeyFiles, line)
			}
		}
	case "Steps":
		// Parse steps with timestamps: - [YYYY-MM-DD HH:MM] message
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if matches := stepRegex.FindStringSubmatch(line); matches != nil {
				ctx.Steps = append(ctx.Steps, Step{
					Timestamp: matches[1],
					Message:   matches[2],
				})
			}
		}
	}
}

// Exists checks if a context file exists for the given name
func Exists(repoPath, name string) bool {
	filePath := filepath.Join(repoPath, ".claude", "contexts", name+".md")
	_, err := os.Stat(filePath)
	return err == nil
}

// ListAll returns all context file names in the repo
func ListAll(repoPath string) ([]string, error) {
	dir := filepath.Join(repoPath, ".claude", "contexts")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			name := strings.TrimSuffix(entry.Name(), ".md")
			names = append(names, name)
		}
	}

	return names, nil
}
