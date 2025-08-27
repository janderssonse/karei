# Title

<!-- SPDX-FileCopyrightText: 2025 The Karei Authors -->
<!-- SPDX-License-Identifier: CC0-1.0 -->

## Hexagonal Architecture in Karei

This document describes the new hexagonal (ports and adapters) architecture implemented in Karei to support multiple Linux distributions.

## Architecture Overview

```text
┌─────────────────────────────────────────────────────────┐
│                    Application Layer                    │
│  ┌─────────────────────────────────────────────────────┐ │
│  │           install_service.go                        │ │
│  │   • InstallApplication()                            │ │
│  │   • InstallMultipleApplications()                   │ │
│  │   • GetSystemInfo()                                 │ │
│  └─────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────┐
│                     Domain Layer                        │
│  ┌─────────────────┐  ┌─────────────────┐  ┌───────────┐ │
│  │   Package       │  │   Theme         │  │  System   │ │
│  │   • Name        │  │   • Name        │  │  • Distro │ │
│  │   • Method      │  │   • Colors      │  │  • DE     │ │
│  │   • Source      │  │   • Settings    │  │  • PM     │ │
│  └─────────────────┘  └─────────────────┘  └───────────┘ │
│                                                         │
│  ┌─────────────────────────────────────────────────────┐ │
│  │                    Ports                            │ │
│  │   • PackageInstaller interface                      │ │
│  │   • ThemeApplier interface                         │ │
│  │   • SystemDetector interface                       │ │
│  │   • CommandRunner interface                        │ │
│  │   • FileManager interface                          │ │
│  └─────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────┐
│                   Adapters Layer                        │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
│  │   Ubuntu    │  │   Fedora    │  │   Common    │     │
│  │   Adapter   │  │   Adapter   │  │   Adapters  │     │
│  │             │  │             │  │             │     │
│  │ • APT       │  │ • DNF       │  │ • System    │     │
│  │ • Snap      │  │ • YUM       │  │   Detector  │     │
│  │ • Flatpak   │  │ • RPM       │  │ • Command   │     │
│  │             │  │             │  │   Runner    │     │
│  └─────────────┘  └─────────────┘  │ • File Mgr  │     │
│                                   └─────────────┘     │
└─────────────────────────────────────────────────────────┘
```

## Key Benefits

### 1. Platform Independence
- Core business logic is completely separate from platform-specific code
- Domain entities (Package, Theme, System) work across all distributions
- Application services remain unchanged when adding new distros

### 2. Easy Multi-Distro Support
- Adding Fedora support requires only implementing the adapter interfaces
- System detection automatically chooses the right adapters
- No changes needed to existing Ubuntu functionality

### 3. Excellent Testability
- Mock implementations for all external dependencies
- Domain logic can be tested in isolation
- Integration tests can use real or fake adapters

### 4. Clean Separation of Concerns
- **Domain**: Business entities and rules
- **Application**: Use cases and orchestration  
- **Adapters**: Platform-specific implementations

## Current Implementation Status

### ✅ Completed
- Domain entities (Package, Theme, System)
- Port interfaces (PackageInstaller, SystemDetector, etc.)
- Ubuntu adapter implementation
- Common adapters (SystemDetector, CommandRunner, FileManager)
- Application services
- Mock implementations for testing
- Working example demonstration

### 🔄 Next Steps for Multi-Distro
1. **Create Fedora Adapter**:

```text
   internal/adapters/fedora/package_installer.go
```

2. **Add Distribution Factory**:

   ```go
   func CreatePackageInstaller(distro *domain.Distribution) domain.PackageInstaller {
       switch distro.Family {
       case "debian":
           return ubuntu.NewPackageInstaller(...)
       case "rhel":
           return fedora.NewPackageInstaller(...)
       default:
           return common.NewGenericInstaller(...)
       }
   }
   ```

## Directory Structure

```text
internal/
├── domain/                    # Core business logic
│   ├── package.go            # Package entity and service
│   ├── theme.go              # Theme entity and service  
│   ├── system.go             # System information
│   └── ports.go              # Interface definitions
├── adapters/                 # External implementations
│   ├── ubuntu/               # Ubuntu-specific adapters
│   │   └── package_installer.go
│   ├── fedora/               # Fedora adapters (future)
│   │   └── package_installer.go
│   └── common/               # Shared adapters
│       ├── system_detector.go
│       ├── command_runner.go
│       └── file_manager.go
└── application/              # Use cases
    ├── install_service.go    # Installation orchestration
    └── theme_service.go      # Theme management (future)
```

## Usage Examples

### Basic Installation

```go
// Create adapters based on detected system
commandRunner := common.NewCommandRunner(verbose, dryRun)
fileManager := common.NewFileManager(verbose)
systemDetector := common.NewSystemDetector(commandRunner, fileManager)

// Ubuntu system uses Ubuntu adapter
packageInstaller := ubuntu.NewPackageInstaller(commandRunner, fileManager, verbose, dryRun)

// Create services
packageService := domain.NewPackageService(packageInstaller, systemDetector)
installService := application.NewInstallService(packageService, systemDetector)

// Install application (method automatically determined)
result, err := installService.InstallApplication(ctx, "vim", "vim")
```

### Adding Fedora Support

```go
// internal/adapters/fedora/package_installer.go
func (p *PackageInstaller) installDNF(ctx context.Context, pkg *domain.Package) error {
    if p.dryRun {
        fmt.Printf("DRY RUN: sudo dnf install -y %s\n", pkg.Source)
        return nil
    }
    return p.commandRunner.ExecuteSudo(ctx, "dnf", "install", "-y", pkg.Source)
}
```

### Testing with Mocks

```go
commandRunner := common.NewMockCommandRunner(false)
fileManager := common.NewMockFileManager(false)
fileManager.SetMockFile("/etc/os-release", []byte("ID=ubuntu"))

// Test works with any adapter implementation
installService := application.NewInstallService(packageService, systemDetector)
result, err := installService.InstallApplication(ctx, "test-app", "test-source")
```

## Migration Strategy

The hexagonal architecture has been implemented alongside the existing code. Migration can happen gradually:

1. **Phase 1**: New features use hexagonal architecture
2. **Phase 2**: Gradually migrate existing features  
3. **Phase 3**: Remove old architecture once migration is complete

This approach ensures no disruption to existing functionality while enabling clean multi-distro support.
