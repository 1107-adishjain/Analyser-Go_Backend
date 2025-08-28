# Golang Axe Core Accessibility Analyzer

A command-line tool that runs accessibility audits on web pages using the axe-core library.

## Features

- Run accessibility audits on any web page
- Save results to JSON file
- Configurable Chrome options (headless mode, timeout)
- Concise summary of accessibility violations

## Requirements

- Go 1.18 or higher
- Chrome/Chromium browser installed

## Usage

```bash
# Basic usage (defaults to Wikipedia homepage)
./accessibility-analyzer

# Specify a URL to audit
./accessibility-analyzer -url https://example.com

# Save results to a file
./accessibility-analyzer -url https://example.com -output results.json

# Run in visible browser mode with longer timeout
./accessibility-analyzer -url https://example.com -headless=false -timeout 120
```

## Command Line Options

- `-url`: URL to audit (default: https://www.wikipedia.org)
- `-output`: Output file path for JSON results (optional)
- `-headless`: Run Chrome in headless mode (default: true)
- `-timeout`: Timeout in seconds (default: 60)

## Understanding Results

The tool provides a summary of accessibility violations found, categorized by impact level:

- 🔴 Critical: Severe accessibility issues that must be fixed
- 🟠 Serious: Major accessibility issues that should be addressed
- 🟡 Moderate: Accessibility issues that could impact some users
- 🔵 Minor: Minor accessibility issues

## Building from Source

```bash
# Clone the repository
git clone https://github.com/yourusername/golang-axe-core.git
cd golang-axe-core

# Build the executable
go build -o accessibility-analyzer

# Run the tool
./accessibility-analyzer
```

## License

This project uses the axe-core library, which is licensed under the MPL-2.0 License.