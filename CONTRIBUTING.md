# Contributing to VPS Tools

Thank you for your interest in contributing to VPS Tools! This document provides guidelines for contributing to this project.

## Getting Started

### Prerequisites

- Ubuntu 24.04 or compatible Linux distribution
- Bash 5.0+
- Basic understanding of shell scripting
- Git for version control

### Development Setup

```bash
# Clone the repository
git clone https://github.com/YOUR_USERNAME/vps-tools-bash.git
cd vps-tools-bash

# Make scripts executable
chmod +x *.sh
chmod +x **/*.sh
```

## Code Style Guidelines

### Shell Script Conventions

1. **Shebang and strict mode**
   ```bash
   #!/bin/bash
   set -euo pipefail
   ```

2. **File naming**
   - Use hyphens in filenames: `vps-script-name.sh`
   - Always include `.sh` extension
   - Prefix with `vps-` for consistency

3. **Variable naming**
   - Constants: `UPPERCASE_WITH_UNDERSCORES`
   - Local variables: `lowercase_with_underscores`
   - Always quote variables: `"$variable"`

4. **Function naming**
   ```bash
   # Use lowercase with underscores
   function_name() {
       local variable="value"
       # function body
   }
   ```

5. **Logging functions**
   ```bash
   log_info()     # Blue [INFO] for informational messages
   log_success()  # Green [✓] for success messages
   log_warning()  # Yellow [⚠] for warnings
   log_error()    # Red [✗] for errors
   log_critical() # Red [✗] for critical errors (exits)
   ```

6. **Argument parsing**
   ```bash
   parse_args() {
       for arg in "$@"; do
           case $arg in
               --option=*) OPTION="${arg#*=}" ;;
               --flag) FLAG=true ;;
           esac
       done
   }
   ```

### Documentation

- Add usage comments at the top of each script
- Document all command-line arguments
- Include examples in comments

## Testing

### Manual Testing

1. **Dry-run mode**: Most scripts support `--dry-run`
   ```bash
   sudo bash script.sh --dry-run
   ```

2. **Test on fresh Ubuntu 24.04 VM**
   ```bash
   sudo bash install.sh
   vps-tools help
   ```

3. **Test individual scripts**
   ```bash
   sudo bash monitoring/vps-health-monitor.sh
   ```

### Checklist Before Submitting

- [ ] Script runs without errors (`set -euo pipefail`)
- [ ] Tested on Ubuntu 24.04
- [ ] Documentation updated if needed
- [ ] Follows naming conventions
- [ ] No hardcoded paths (use variables)
- [ ] Includes `--dry-run` option where applicable

## Pull Request Process

1. **Fork the repository**

2. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Make your changes**
   - Follow code style guidelines
   - Test thoroughly
   - Update documentation

4. **Commit with clear messages**
   ```bash
   git commit -m "feat: add new monitoring capability"
   git commit -m "fix: resolve SSH key rotation issue"
   git commit -m "docs: update README with new examples"
   ```

5. **Push and create PR**
   ```bash
   git push origin feature/your-feature-name
   ```

### Commit Message Format

Use conventional commits:
- `feat:` new feature
- `fix:` bug fix
- `docs:` documentation changes
- `refactor:` code refactoring
- `test:` adding tests
- `chore:` maintenance tasks

## Project Structure

```
vps-tools-bash/
├── install.sh              # Installation script
├── vps-build.sh            # Main provisioning script
├── vps-tools-cron.conf     # Cron job configuration
│
├── monitoring/             # Health & status monitoring
├── security/               # Security auditing & hardening
├── docker/                 # Docker/container management
├── maintenance/            # System maintenance
└── orchestration/          # Unified management
```

## Adding New Scripts

1. **Create script in appropriate directory**
2. **Follow the standard template**:
   ```bash
   #!/bin/bash
   set -euo pipefail

   # Script Description
   # Usage: bash script.sh [options]

   readonly SCRIPT_VERSION="1.0"
   readonly TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')
   readonly LOG_DIR="/var/log/vps-tools"
   readonly LOG_FILE="$LOG_DIR/script-name.log"

   # Colors
   RED='\033[0;31m'
   GREEN='\033[0;32m'
   YELLOW='\033[1;33m'
   BLUE='\033[0;34m'
   NC='\033[0m'

   log_info() { echo -e "${BLUE}[INFO]${NC} $*"; }
   log_success() { echo -e "${GREEN}[✓]${NC} $*"; }
   log_warning() { echo -e "${YELLOW}[⚠]${NC} $*"; }
   log_error() { echo -e "${RED}[✗]${NC} $*"; }

   parse_args() {
       for arg in "$@"; do
           case $arg in
               --option=*) OPTION="${arg#*=}" ;;
           esac
       done
   }

   main() {
       parse_args "$@"
       log_info "Script Name v${SCRIPT_VERSION}"
       # Script logic here
   }

   main "$@"
   ```

3. **Update dispatcher** in `install.sh`
4. **Add to README.md** and **USAGE.md**
5. **Add cron entry** if needed

## Reporting Issues

When reporting issues, please include:

1. **Environment info**
   ```bash
   lsb_release -a
   bash --version
   ```

2. **Script output** with `set -x` for debugging
   ```bash
   bash -x script.sh 2>&1 | tee debug.log
   ```

3. **Relevant log files**
   ```bash
   cat /var/log/vps-tools/script-name.log
   ```

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
