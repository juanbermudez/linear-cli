# Linear CLI

A JSON-first command-line interface for Linear, designed for AI agents.

## Features

- **JSON-first output** - Structured responses for AI agents
- **--human flag** - Readable tables for terminal use
- **Error hints** - Helpful guidance in error responses
- **Secure auth** - System keychain credential storage
- **24-hour caching** - Reduced API calls for common data
- **Stdin support** - Non-interactive setup for automation

## Installation

### Using Go

```bash
go install github.com/juanbermudez/linear-cli/cmd/linear@latest
```

### From Source

```bash
git clone https://github.com/juanbermudez/linear-cli.git
cd linear-cli
make build
make install
```

## Quick Start

```bash
# Interactive setup
linear config setup

# Or login directly
linear auth login --api-key lin_api_xxxxx --team ENG

# Verify configuration
linear whoami
```

## Usage

### Issues

```bash
# List issues
linear issue list --team ENG --human
linear issue list --team ENG --state started

# Create issue
linear issue create --title "Fix login bug" --team ENG --priority 2

# View issue
linear issue view ENG-123

# Update issue
linear issue update ENG-123 --state "Done"

# Search issues
linear issue search "authentication bug"

# Create relationships
linear issue relate ENG-123 ENG-456 --blocks
```

### Projects

```bash
# List projects
linear project list --human

# Create project with document
linear project create --name "Q1 Feature" --team ENG --with-doc

# View project
linear project view PROJECT_ID

# Add milestone
linear project milestone create PROJECT_ID --name "Phase 1" --target-date 2025-02-15
```

### Documents

```bash
# Create document
linear document create --title "PRD: Feature X" --project PROJECT_ID

# List documents
linear document list --project PROJECT_ID

# Search documents
linear document search "authentication"
```

### Initiatives

```bash
# Create initiative
linear initiative create --name "Q1 Platform Improvements" --status Active

# Add project to initiative
linear initiative project-add INIT_ID PROJECT_ID
```

### Workflows

```bash
# List workflow states
linear workflow list --team ENG

# Force refresh cache
linear workflow cache --team ENG
```

## Output Modes

**JSON (default)** - Machine-readable output:
```bash
linear issue list --team ENG
# {"issues": {"nodes": [{"id": "...", "identifier": "ENG-123", ...}]}}
```

**Human-readable** - Formatted for terminal:
```bash
linear issue list --team ENG --human
# ENG-123  Fix login bug  In Progress  JD  2 hours ago
```

**Error with hints** - Guidance for AI agents:
```json
{
  "success": false,
  "error": {
    "code": "MISSING_TEAM",
    "message": "Team is required",
    "hint": "Specify a team using --team flag or set a default team",
    "usage": ["linear issue list --team ENG"]
  }
}
```

## Configuration

Configuration is stored in `.linear.toml`:

```toml
team_key = "ENG"
```

API key is stored securely in your system keychain.

Environment variables override config:
- `LINEAR_API_KEY` - API key
- `LINEAR_TEAM_KEY` - Default team

## Documentation

Full documentation: https://juanbermudez.github.io/linear-cli

## Development

```bash
# Build
make build

# Run tests
make test

# Install locally
make install
```

## License

MIT
