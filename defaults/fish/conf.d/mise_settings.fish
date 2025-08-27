# Enable mise for Fish shell with karei configuration
if test -f ~/.local/bin/mise
    # Activate mise for Fish shell (uses XDG-compliant ~/.config/mise/config.toml)
    ~/.local/bin/mise activate fish | source
end

# Mise Java Home setup
function mise_update_java_home
    set -l java_path (mise which java 2>/dev/null)
    if test -n "$java_path"
        set -gx JAVA_HOME (dirname (dirname (realpath $java_path)))
        set -gx JDK_HOME $JAVA_HOME
    end
end

mise_update_java_home

# Softlink Java certs to current Java version
if test -d "$JAVA_HOME/lib/security/cacerts"; or not test -L "$JAVA_HOME/lib/security/cacerts"
    ln -sf /etc/ssl/certs/java/cacerts $JAVA_HOME/lib/security/cacerts
end

# Mise Go environment setup
set -l go_path (mise which go 2>/dev/null)
if test -n "$go_path"
    set -gx GOROOT (dirname (dirname (realpath $go_path)))
    set -gx GOPATH $HOME/go
end