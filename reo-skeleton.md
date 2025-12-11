Below is a clean, production-ready repo skeleton with:

Go module setup

Cobra CLI scaffold

Bubble Tea “Hello TUI”

Makefile

GitHub Actions CI

Linting ready

You can paste this directly into a fresh repo and your AI coding tool can extend it PR-by-PR.

✅ Repo Structure (initial)
vps-tools/
├── cmd/
│   ├── root.go
│   └── tui.go
├── internal/
│   └── app/
│       └── app.go
├── tui/
│   └── main.go
├── .github/
│   └── workflows/
│       └── ci.yml
├── go.mod
├── main.go
├── Makefile
└── README.md

✅ 1. go.mod
module github.com/yourorg/vps-tools

go 1.21

require (
	github.com/charmbracelet/bubbletea v0.26.2
	github.com/charmbracelet/lipgloss v0.10.0
	github.com/spf13/cobra v1.8.0
)

✅ 2. main.go (CLI entrypoint)
package main

import "github.com/yourorg/vps-tools/cmd"

func main() {
	cmd.Execute()
}

✅ 3. cmd/root.go (Cobra root)
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "vps-tools",
	Short: "Modern terminal tool for managing Linux servers",
	Long:  "vps-tools replaces legacy bash scripts with a fast, secure, structured terminal application.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {}

✅ 4. cmd/tui.go (launches Bubble Tea)
package cmd

import (
	"github.com/spf13/cobra"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yourorg/vps-tools/tui"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the interactive terminal UI",
	RunE: func(cmd *cobra.Command, args []string) error {
		p := tea.NewProgram(tui.NewModel())
		_, err := p.Run()
		return err
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}

✅ 5. tui/main.go (Minimal Bubble Tea app)

This is your visual validation that everything works.

package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	ready bool
}

func NewModel() model {
	return model{}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m model) View() string {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205"))

	return style.Render("vps-tools TUI is running\n\nPress Q to quit\n")
}

✅ 6. internal/app/app.go (core placeholder)
package app

type App struct {
	Name string
}

func New() *App {
	return &App{
		Name: "vps-tools",
	}
}

✅ 7. Makefile
APP_NAME=vps-tools

build:
	go build -o bin/$(APP_NAME)

run:
	go run .

tui:
	go run . tui

test:
	go test ./...

fmt:
	gofmt -w .

lint:
	golangci-lint run

clean:
	rm -rf bin

✅ 8. GitHub Actions CI

.github/workflows/ci.yml

name: CI

on:
  push:
  pull_request:

jobs:
  build:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Install deps
        run: go mod download

      - name: Format
        run: gofmt -w .

      - name: Test
        run: go test ./...

      - name: Build
        run: go build ./...

✅ 9. README.md (starter)
# vps-tools

Modern terminal application for managing Linux servers.

## Features (Planned)
- SSH orchestration
- Job scheduling
- Inventory management
- Interactive TUI
- Audit logging
- Secure secrets handling

## Run CLI


go run .


## Run TUI


go run . tui

✅ Verify Locally

From repo root:

go mod tidy
make run
make tui


You should see:

vps-tools TUI is running
Press Q to quit


If that works, your base architecture is correct.

✅ What This Unlocks

You now have:

Proper Go module structure

CLI + TUI dual-mode app

CI pipeline

Styling engine ready

Clean insertion points for:

SSH layer

Job engine

Inventory store

Security & secrets