package context

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// Context represents a saved Claude context file
type Context struct {
	IssueID   string
	Title     string
	Branch    string
	WorkDone  string
	KeyFiles  []string
	FilePath  string
}

// Load reads a context file from .claude/contexts/{issueID}.md
func Load(repoPath, issueID string) (*Context, error) {
	filePath := filepath.Join(repoPath, ".claude", "contexts", issueID+".md")
	return LoadFromFile(filePath, issueID)
}

// LoadFromFile reads a context file from a specific path
func LoadFromFile(filePath, issueID string) (*Context, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	ctx := &Context{
		IssueID:  issueID,
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
	}
}

// Exists checks if a context file exists for the given issue ID
func Exists(repoPath, issueID string) bool {
	filePath := filepath.Join(repoPath, ".claude", "contexts", issueID+".md")
	_, err := os.Stat(filePath)
	return err == nil
}

// ListAll returns all context files in the repo
func ListAll(repoPath string) ([]string, error) {
	dir := filepath.Join(repoPath, ".claude", "contexts")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var issueIDs []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			issueID := strings.TrimSuffix(entry.Name(), ".md")
			issueIDs = append(issueIDs, issueID)
		}
	}

	return issueIDs, nil
}
