# Universal Checker

A high-performance universal account checker that supports multiple configuration formats and automatic proxy management.

## Features

- 🔧 **Multiple Config Formats**: Support for OpenBullet (.opk), SilverBullet (.svb), and Loli (.loli) configurations
- 🌐 **Auto Proxy Scraping**: Automatically scrapes and validates SOCKS4, SOCKS5, HTTP, and HTTPS proxies
- ⚡ **High Performance**: Optimized for high CPM (Checks Per Minute) with concurrent workers
- 📊 **Live Statistics**: Real-time display of checking progress and performance metrics
- 🎯 **Drag & Drop**: Easy-to-use interface with drag-and-drop config file support
- 💾 **Flexible Output**: Save results with customizable output formats and directories

## Installation

```bash
# Clone the repository
git clone https://github.com/Aiu-xD/stuffer-go.git
cd stuffer-go

# Install dependencies
go mod download

# Build the application
make build
```

## Usage

### Command Line

```bash
# Basic usage
./universal-checker -c config.opk -l combos.txt

# With custom workers and auto-proxy scraping
./universal-checker -c config.opk -l combos.txt -w 200 --auto-scrape

# With manual proxy list
./universal-checker -c config.opk -l combos.txt -p proxies.txt --auto-scrape=false

# Multiple configs
./universal-checker -c config1.opk -c config2.svb -l combos.txt
```

### Drag & Drop

Simply drag and drop your config files and combo lists onto the executable. The application will automatically detect file types.

### Flags

- `-c, --configs`: Config file paths (supports .opk, .svb, .loli)
- `-l, --combos`: Combo list file path
- `-p, --proxies`: Proxy list file path
- `-o, --output`: Output directory for results (default: "results")
- `-w, --workers`: Maximum number of workers (default: 100)
- `--auto-scrape`: Automatically scrape proxies (default: true)
- `--valid-only`: Save only valid results (default: true)
- `--request-timeout`: Request timeout in milliseconds (default: 30000)
- `--proxy-timeout`: Proxy validation timeout in milliseconds (default: 5000)

## Project Structure

```
stuffer-go/
├── cmd/                    # Command line interfaces
│   ├── main.go            # Main CLI entry point
│   ├── gui/               # GUI application
│   ├── test-mode/         # Testing mode
│   └── global-mode/       # Global mode
├── internal/              # Internal packages
│   ├── checker/           # Core checker logic
│   ├── config/            # Config parsing
│   ├── logger/            # Logging utilities
│   └── proxy/             # Proxy management
├── pkg/                   # Public packages
│   ├── httpclient/        # HTTP client wrapper
│   ├── types/             # Type definitions
│   └── utils/             # Utility functions
├── configs/               # Sample configurations
├── data/                  # Data files
│   ├── combos/           # Combo lists
│   └── proxies/          # Proxy lists
└── tests/                # Test files
```

## Configuration

The checker supports three configuration formats:

### OpenBullet (.opk)
Standard OpenBullet configuration format with full block support.

### SilverBullet (.svb)
SilverBullet configuration format with enhanced features.

### Loli (.loli)
LoliScript configuration format with custom syntax support.

## Development

```bash
# Run tests
make test

# Run linter
make lint

# Build all binaries
make build-all

# Clean build artifacts
make clean
```

## Requirements

- Go 1.24.0 or higher
- Linux/macOS/Windows

## Dependencies

- [azuretls-client](https://github.com/Noooste/azuretls-client) - Advanced HTTP client with TLS fingerprinting
- [cobra](https://github.com/spf13/cobra) - CLI framework
- [fyne](https://fyne.io/) - GUI framework

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Disclaimer

This tool is for educational purposes only. Use responsibly and in accordance with all applicable laws and terms of service.
