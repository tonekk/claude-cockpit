package config

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/google/shlex"
	"github.com/tonekk/claude-cockpit/internal/tmux"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Setup  []string          `yaml:"setup"`
	Server string            `yaml:"server"` // server command, overridden by WD_SERVER_COMMAND / -s
	Env    map[string]string `yaml:"env"`    // server env vars, overridden by WD_ENV_* / -e
}

func Load(projectPath string) (*Config, error) {
	data, err := os.ReadFile(projectPath + "/cockpit.yml")
	if os.IsNotExist(err) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SendSetupAndClaudeToTmux sends setup commands followed by claude command
// If setup fails, shows error message and waits for keypress to close tab
func (c *Config) SendSetupAndClaudeToTmux(claudeCmd string) {
	if len(c.Setup) == 0 {
		// No setup, just run claude
		tmux.SendKeys(claudeCmd)
		return
	}

	setupCmd := strings.Join(c.Setup, " && ")
	// On success: run claude. On failure: show error, wait for key, exit (closes tab)
	fullCmd := fmt.Sprintf("(%s && %s) || (echo '' && echo '❌ Setup failed. Fix your cockpit.yml and try again.' && echo 'Press any key to close this tab...' && read && exit)", setupCmd, claudeCmd)
	tmux.SendKeys(fullCmd)
}

// RunSetupCLI runs setup commands with output to stdout (for CLI usage)
func (c *Config) RunSetupCLI(worktreePath string) error {
	for _, cmdStr := range c.Setup {
		fmt.Printf("→ %s\n", cmdStr)

		output, err := runSetupCommand(cmdStr, worktreePath)
		if err != nil {
			return err
		}

		if output != "" {
			fmt.Print(output)
		}
	}

	return nil
}

func runSetupCommand(cmdStr, worktreePath string) (string, error) {
	parts, err := shlex.Split(cmdStr)
	if err != nil {
		return "", fmt.Errorf("error parsing command `%s`: %w", cmdStr, err)
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Dir = worktreePath
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("error running `%s`:\n%s", cmdStr, stderr.String())
	}

	return stdout.String(), nil
}
