package scaffold

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

var githubShorthand = regexp.MustCompile(`^github\.com/([^/]+)/([^/]+)/(.+)$`)

// Regex to find {{arg N}} usage in templates
var argPattern = regexp.MustCompile(`{{\s*arg\s+(\d+)\s*}}`)

type DSL struct {
	Vars  map[string]string `yaml:"vars"`
	Steps []Step            `yaml:"steps"`
}

type Step struct {
	Mkdir     string      `yaml:"mkdir,omitempty"`
	WriteFile *WriteFile  `yaml:"write_file,omitempty"`
	Run       *RunCommand `yaml:"run,omitempty"`
	When      string      `yaml:"when,omitempty"`
}

type WriteFile struct {
	Path    string `yaml:"path"`
	Content string `yaml:"content"`
}

type RunCommand struct {
	Cmd string `yaml:"cmd"`
	Dir string `yaml:"dir,omitempty"`
}

type Data struct {
	Args []string
	Vars map[string]string
}

func (d Data) Arg(i int) string {
	if i < len(d.Args) {
		return d.Args[i]
	}
	return ""
}

func Render(tmplStr string, data Data) (string, error) {
	t, err := template.New("").Funcs(template.FuncMap{
		"arg": data.Arg,
	}).Parse(tmplStr)
	if err != nil {
		return "", err
	}

	context := map[string]interface{}{
		"Var": data.Vars,
		"Arg": data.Arg,
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, context); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func loadTemplate(templateFile string) ([]byte, error) {
	if matches := githubShorthand.FindStringSubmatch(templateFile); len(matches) == 4 {
		owner, repo, path := matches[1], matches[2], matches[3]
		rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/main/%s", owner, repo, path)
		fmt.Printf("Fetching GitHub template from %s\n", rawURL)

		resp, err := http.Get(rawURL)
		if err != nil {
			return nil, fmt.Errorf("failed to download template from GitHub raw URL: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("failed to download template: status %d", resp.StatusCode)
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read template body: %w", err)
		}
		return data, nil
	}

	if strings.HasPrefix(templateFile, "http://") || strings.HasPrefix(templateFile, "https://") {
		resp, err := http.Get(templateFile)
		if err != nil {
			return nil, fmt.Errorf("failed to download template from URL: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("failed to download template: status %d", resp.StatusCode)
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read template body: %w", err)
		}
		return data, nil
	}

	data, err := os.ReadFile(templateFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file: %w", err)
	}
	return data, nil
}

// findMaxArgIndex scans a string and returns the max arg index used, or -1 if none found
func findMaxArgIndex(s string) int {
	matches := argPattern.FindAllStringSubmatch(s, -1)
	max := -1
	for _, m := range matches {
		var idx int
		fmt.Sscanf(m[1], "%d", &idx)
		if idx > max {
			max = idx
		}
	}
	return max
}

// maxArgInDSL scans all relevant DSL fields and returns the max arg index used
func maxArgInDSL(dsl DSL) int {
	maxIdx := -1

	// Check vars
	for _, v := range dsl.Vars {
		if idx := findMaxArgIndex(v); idx > maxIdx {
			maxIdx = idx
		}
	}

	// Check steps
	for _, step := range dsl.Steps {
		if step.Mkdir != "" {
			if idx := findMaxArgIndex(step.Mkdir); idx > maxIdx {
				maxIdx = idx
			}
		}
		if step.WriteFile != nil {
			if idx := findMaxArgIndex(step.WriteFile.Path); idx > maxIdx {
				maxIdx = idx
			}
			if idx := findMaxArgIndex(step.WriteFile.Content); idx > maxIdx {
				maxIdx = idx
			}
		}
		if step.Run != nil {
			if idx := findMaxArgIndex(step.Run.Cmd); idx > maxIdx {
				maxIdx = idx
			}
			if step.Run.Dir != "" {
				if idx := findMaxArgIndex(step.Run.Dir); idx > maxIdx {
					maxIdx = idx
				}
			}
		}
		if step.When != "" {
			if idx := findMaxArgIndex(step.When); idx > maxIdx {
				maxIdx = idx
			}
		}
	}

	return maxIdx
}

func Run(templateFile string, args []string) error {
	content, err := loadTemplate(templateFile)
	if err != nil {
		return err
	}

	var dsl DSL
	if err := yaml.Unmarshal(content, &dsl); err != nil {
		return fmt.Errorf("failed to parse DSL yaml: %w", err)
	}

	// Validate arg count before running
	maxArg := maxArgInDSL(dsl)
	if maxArg >= 0 && len(args) <= maxArg {
		return fmt.Errorf("not enough arguments: template requires at least %d argument(s), got %d", maxArg+1, len(args))
	}

	data := Data{
		Args: args,
		Vars: make(map[string]string),
	}

	for k, v := range dsl.Vars {
		rendered, err := Render(v, data)
		if err != nil {
			return fmt.Errorf("render var %s: %w", k, err)
		}
		data.Vars[k] = rendered
	}

	for i, step := range dsl.Steps {
		if step.When != "" {
			cond, err := Render(step.When, data)
			if err != nil {
				return fmt.Errorf("render when in step %d: %w", i+1, err)
			}
			if cond != "true" {
				continue
			}
		}

		if step.Mkdir != "" {
			path, err := Render(step.Mkdir, data)
			if err != nil {
				return fmt.Errorf("render mkdir in step %d: %w", i+1, err)
			}
			fmt.Println("mkdir", path)
			if err := os.MkdirAll(path, 0755); err != nil {
				return fmt.Errorf("mkdir step %d: %w", i+1, err)
			}
		}

		if step.WriteFile != nil {
			path, err := Render(step.WriteFile.Path, data)
			if err != nil {
				return fmt.Errorf("render write_file path in step %d: %w", i+1, err)
			}
			content, err := Render(step.WriteFile.Content, data)
			if err != nil {
				return fmt.Errorf("render write_file content in step %d: %w", i+1, err)
			}
			dir := filepath.Dir(path)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("mkdir for write_file step %d: %w", i+1, err)
			}
			fmt.Println("write_file", path)
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return fmt.Errorf("write_file step %d: %w", i+1, err)
			}
		}

		if step.Run != nil {
			cmdStr, err := Render(step.Run.Cmd, data)
			if err != nil {
				return fmt.Errorf("render run cmd in step %d: %w", i+1, err)
			}
			dir := "."
			if step.Run.Dir != "" {
				dir, err = Render(step.Run.Dir, data)
				if err != nil {
					return fmt.Errorf("render run dir in step %d: %w", i+1, err)
				}
			}
			fmt.Println("run", cmdStr)
			cmd := exec.Command("sh", "-c", cmdStr)
			cmd.Dir = dir
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("run step %d failed: %w", i+1, err)
			}
		}
	}

	return nil
}
