# Security warnings integration
# Show security warnings on shell start if security tools are installed

# Only show warnings if security tools directory exists and it's an interactive shell
if status is-interactive && test -d /var/log/karei-security
    # Show security warnings (but not on every single command, just shell start)
    show_security_warnings
end