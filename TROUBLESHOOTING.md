# Troubleshooting

## Common Issues

### Installation Failures

**Problem**: Installation script fails

- Check system requirements (Ubuntu 24.04+)
- Verify internet connection
- Run with verbose output: `karei --verbose install`

### Theme Not Applied

**Problem**: Theme changes don't appear

- Restart your GNOME session
- Check if application supports theming
- Run: `karei theme list` to see available themes

### Application Missing

**Problem**: Installed app not found

- Check installation logs: `karei logs install`
- Verify PATH includes `~/.local/bin`
- Re-run: `karei install --packages [app]`

### Performance Issues

**Problem**: System running slowly after installation

- Consider minimal installation without desktop apps
- Check disk space: `df -h`
- Review installed packages: `karei list`

## Log Files

```bash
# View installation logs
karei logs install

# View theme operation logs  
karei logs theme

# View application-specific logs
karei logs [application-name]
```

## Recovery Procedures

### Restore from Backup

```bash
# List available backups
ls ~/.local/share/karei/backups/

# Restore specific backup
~/.local/share/karei/backups/[timestamp]/restore.sh
```

### Reset to Defaults

```bash
# Reset configuration
karei reset

# Complete reinstall
karei uninstall --all
rm ~/.local/bin/karei
# Then reinstall
```

### Manual Cleanup

If Karei is completely broken:

```bash
# Remove Karei files
rm -rf ~/.config/karei
rm -rf ~/.local/share/karei
rm -rf ~/.cache/karei
rm ~/.local/bin/karei

# Remove installed packages (careful!)
# Check what Karei installed first
cat ~/.local/share/karei/installed-packages.txt
```

## Getting Help

For help:
1. Check existing documentation in `docs/`
2. Review the source code
3. Check GitHub issues for similar problems
