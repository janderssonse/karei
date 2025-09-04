# Karei Documentation

This directory contains documentation for Karei in multiple formats.

## Documentation Hierarchy

### 1. Quick Help (Terminal)

**Purpose**: Brief overview for immediate use
**Access**: `karei --help`, `karei <invalid-command>`
**Content**: Essential commands, quick start, pointers to detailed docs

### 2. Terminal Documentation  

**Purpose**: Complete reference that stays in sync with installed version
**Access**: `karei help <command>`, `karei help examples`
**Content**: Detailed command documentation, complete workflows

### 3. Manual Pages

**Purpose**: Unix standard documentation system
**Access**: `man karei`, `karei help man`
**Content**: Complete reference in traditional Unix man page format

### 4. Web Documentation  

**Purpose**: Searchable, linkable, complete guide
**Access**: Web documentation (roadmap)
**Content**: Full documentation with deep links and search

## File Organization

```text
docs/
├── README.md                    # This file - documentation overview
├── karei.1.md                   # Main manual page source
├── HEXAGONAL_ARCHITECTURE.md    # System architecture design
├── tui-architecture.md          # TUI implementation details
├── configuration-patterns.md    # Configuration management patterns
├── installation-structure.md    # Installation flow documentation
├── SSH_CONFIGURATION.md         # SSH setup and configuration
├── GIT_CONFIGURATION.md         # Git setup and configuration
├── EXTENSIONS.md                # Community extensions catalog
├── PROXY_SUPPORT.md             # Proxy configuration guide
├── SECURITY_ASSESSMENT.md       # Security audit and assessment
└── CLI_OUTPUT.md                # CLI output formatting reference
```

## Documentation Principles

### Help vs Documentation
- **Help**: Brief, immediate sense of what the tool does and how to start
- **Documentation**: Complete details, guides, troubleshooting

### Accessibility
- **Terminal docs**: Work offline, stay in sync, fast access
- **Web docs**: Searchable, linkable, inclusive format
- **Man pages**: Unix standard, familiar to users

### Consistency
- Same information presented appropriately for each format
- Consistent examples and workflows across all documentation
- Clear cross-references between formats

## Generating Documentation

### Manual Pages

Generate man pages from markdown:

```bash
./scripts/docs/manpage.sh
```

### Web Documentation

Web documentation generation is on the roadmap.

### Install Man Pages

Install generated man pages to system:

```bash
sudo cp generated/manpages/karei.1.gz /usr/local/share/man/man1/
sudo mandb
```

## Maintenance

- Keep terminal help brief and focused
- Update man pages when commands change  
- Ensure examples work and outputs are current
- Test documentation hierarchy flows
- Maintain deep links for web documentation

## Writing Guidelines

### Help Text
- Lead with examples showing value
- Keep to 10-15 lines maximum
- Point to detailed documentation
- Show only most common usage

### Command Documentation  
- Start with synopsis and description
- Include detailed examples with output
- Cover all options and edge cases
- Provide troubleshooting section

### Web Documentation
- Make content searchable and skimmable
- Include copy-paste ready examples
- Link to related commands and concepts
- Provide direct links to specific topics
