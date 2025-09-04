<!-- SPDX-FileCopyrightText: 2025 The Karei Authors -->
<!-- SPDX-License-Identifier: CC0-1.0 -->

# Karei Extensions

This document catalogs community-developed extensions and customizations for the Karei Ubuntu development environment automation system.

## Theme Extensions

### [Dark and Light Mode Theme Switch](https://github.com/florentdestremau/karei-darkmode-switch)

**Description**: Enables automatic theme switching between user-selected dark and light themes when toggling GNOME's light/dark mode preference.

**Features**:
- Automatic theme detection and switching
- Integration with GNOME settings
- Support for all built-in Karei themes
- Preserves user theme preferences

**Installation**: Follow the repository instructions for integrating with your Karei installation.

**Compatibility**: All Karei versions with GNOME desktop environment support.

## Contributing Extensions

### Extension Development Guidelines

Community extensions should adhere to the following standards to ensure compatibility and quality:

#### Technical Requirements

1. **XDG Base Directory Compliance**: Use standard XDG environment variables for configuration and data storage
2. **Non-destructive Installation**: Extensions must not modify core Karei files
3. **Atomic Operations**: All configuration changes should be reversible and atomic
4. **Error Handling**: Implement proper error handling with clear user feedback
5. **Logging Integration**: Use Karei logging patterns for consistency

#### Security Standards

1. **Input Validation**: Sanitize all user inputs and external data sources
2. **Privilege Minimization**: Use minimal required permissions
3. **Secure Defaults**: Implement conservative default configurations
4. **HTTPS Requirements**: All external downloads must use secure protocols

#### Documentation Requirements

1. **Installation Instructions**: Clear setup and integration procedures
2. **Configuration Documentation**: All configuration options
3. **Compatibility Matrix**: Supported Karei versions and environments
4. **Troubleshooting Guide**: Common issues and resolution procedures

### Extension Categories

#### Theme Extensions
- Custom color schemes and visual modifications
- Application-specific theme integrations
- Dynamic theme switching mechanisms

#### Application Packages
- Additional software installation scripts
- Enterprise application integrations
- Development tool extensions

#### Configuration Overrides
- Alternative configurations for existing tools
- Workplace-specific customizations
- Regional or compliance-specific modifications

### Submission Process

To have your extension listed in this document:

1. **Repository Setup**: Create a public repository with complete documentation
2. **Testing**: Validate compatibility with current Karei release
3. **Documentation**: Include all required documentation sections
4. **Security Review**: Ensure adherence to security standards
5. **Pull Request**: Submit a pull request to add your extension to this list

### Extension Template

Use this template for new extension documentation:

```markdown
### [Extension Name](repository-url)

**Description**: Brief description of extension functionality and purpose.

**Features**:
- Feature 1 description
- Feature 2 description
- Feature 3 description

**Installation**: Link to installation instructions or brief setup steps.

**Compatibility**: Supported Karei versions and system requirements.

**Maintainer**: Extension maintainer contact information (optional).
```

## Support and Community

### Getting Help

- **Karei Core Issues**: Report issues with core Karei functionality to the main repository
- **Extension Issues**: Contact extension maintainers directly for extension-specific problems
- **General Support**: Use discussions for general questions and community support

### Community Guidelines

- Maintain respectful and professional communication
- Provide constructive feedback and support to other community members
- Follow security best practices in all contributions
- Document changes and maintain compatibility information

## Extension Registry

*Additional community extensions will be added to this section as they become available.*

For the most current list of extensions and community contributions, check the Karei repository discussions and community forums.
