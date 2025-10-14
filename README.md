# scaffold

A simple, flexible Go-based project scaffolder that uses a YAML DSL to define scaffold templates.  
Supports local files, HTTP URLs, and GitHub shorthand URLs for fetching scaffold templates.

---

## Features

- Define scaffold steps in YAML:
  - Create directories
  - Write templated files
  - Run shell commands
  - Conditional execution (`when`)
- Pass arguments to templates (`arg` function)
- Load scaffold templates from:
  - Local files
  - Any HTTP/HTTPS URL
  - GitHub shorthand URLs (`github.com/owner/repo/path/to/template.yaml`) — auto converts to raw URL
- Easily extendable with Go `text/template` syntax

---

## Installation

```bash
git clone https://github.com/toshsan/scaffold.git
cd scaffold
go mod tidy
go build -o scaffold ./cmd/scaffold
````

---

## Usage

```bash
./scaffold <template> [args...]
```

* `<template>` can be:

  * Local file path (`templates/example.yaml`)
  * HTTP/HTTPS URL (`https://example.com/template.yaml`)
  * GitHub shorthand (`github.com/owner/repo/path/to/template.yaml`)

* `[args...]` are positional arguments available inside templates via `{{arg 0}}`, `{{arg 1}}`, etc.

---

## Example

Run scaffold using local template:

```bash
./scaffold templates/example.yaml MyProject Alice
```

Run scaffold using GitHub shorthand URL (auto fetch raw file from `main` branch):

```bash
./scaffold github.com/toshsan/scaffold/templates/example.yaml MyProject Alice
```

---

## Template DSL Example (`templates/example.yaml`)

```yaml
vars:
  project: "{{arg 0}}"
  author: "{{arg 1}}"
  EnableGit: "true"

steps:
  - mkdir: "{{.Var.project}}"

  - write_file:
      path: "{{.Var.project}}/main.go"
      content: |
        package main

        import "fmt"

        func main() {
            fmt.Println("Hello, {{.Var.author}}!")
        }

  - run:
      cmd: "go mod init github.com/{{.Var.author}}/{{.Var.project}}"
      dir: "{{.Var.project}}"

  - run:
      cmd: "git init"
      dir: "{{.Var.project}}"
      when: "{{.Var.EnableGit}}"
```

---

## Template Syntax

* Use Go's `text/template` syntax in all template strings.
* Available template data:

  * `.Arg` — function to access arguments, e.g., `{{arg 0}}`
  * `.Var` — map of variables defined in `vars`
* Supported step keys:

  * `mkdir`: directory to create
  * `write_file`:

    * `path`: file path to write
    * `content`: file content
  * `run`:

    * `cmd`: shell command to run
    * `dir` (optional): working directory
  * `when` (optional): condition, step runs only if condition evaluates to `"true"`

---

## License

MIT © [Your Name]

---

## Contributions

Contributions, issues, and feature requests are welcome!
Feel free to check [issues page](https://github.com/toshsan/scaffold/issues).

---

## Author

[Your Name] - [[your.email@example.com](mailto:your.email@example.com)]
[GitHub](https://github.com/toshsan)
