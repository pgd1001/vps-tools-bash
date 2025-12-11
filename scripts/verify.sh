#!/bin/bash

# Verification script for vps-tools repository structure
echo "=== vps-tools Repository Structure Verification ==="
echo

# Check required files
echo "Checking required files..."
files=(
    "go.mod"
    "main.go"
    "Makefile"
    "README.md"
    ".gitignore"
    ".golangci.yml"
    ".github/workflows/ci.yml"
    "cmd/root.go"
    "cmd/tui.go"
    "tui/main.go"
    "internal/app/app.go"
)

missing_files=()
for file in "${files[@]}"; do
    if [[ -f "$file" ]]; then
        echo "✅ $file"
    else
        echo "❌ $file (missing)"
        missing_files+=("$file")
    fi
done

echo
echo "Checking directory structure..."
dirs=(
    "cmd"
    "internal"
    "internal/app"
    "tui"
    ".github"
    ".github/workflows"
    "bin"
)

missing_dirs=()
for dir in "${dirs[@]}"; do
    if [[ -d "$dir" ]]; then
        echo "✅ $dir/"
    else
        echo "❌ $dir/ (missing)"
        missing_dirs+=("$dir")
    fi
done

echo
echo "=== Summary ==="
if [[ ${#missing_files[@]} -eq 0 && ${#missing_dirs[@]} -eq 0 ]]; then
    echo "🎉 All required files and directories are present!"
    echo
    echo "Next steps:"
    echo "1. Install Go 1.21+ if not already installed"
    echo "2. Run 'make deps' to download dependencies"
    echo "3. Run 'make test' to verify tests pass"
    echo "4. Run 'make build' to build the application"
    echo "5. Run 'make tui' to test the TUI interface"
else
    echo "❌ Some files or directories are missing:"
    [[ ${#missing_files[@]} -gt 0 ]] && printf "  Files: %s\n" "${missing_files[*]}"
    [[ ${#missing_dirs[@]} -gt 0 ]] && printf "  Directories: %s\n" "${missing_dirs[*]}"
fi

echo
echo "=== Go Module Check ==="
if [[ -f "go.mod" ]]; then
    echo "Module name: $(grep '^module' go.mod | cut -d' ' -f2)"
    echo "Go version: $(grep '^go' go.mod | cut -d' ' -f2)"
fi