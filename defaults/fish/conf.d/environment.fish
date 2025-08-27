# Environment variables
set -gx DOCKER_HOST unix:///run/user/1000/podman/podman.sock
set -gx TERM xterm-256color
set -gx LANG en_US.UTF-8
set -gx BAT_THEME TwoDark
set -gx NODE_OPTIONS --use-openssl-ca

# XDG Base Directory
set -gx XDG_CONFIG_HOME $HOME/.config
set -gx XDG_DATA_HOME $HOME/.local/share
set -gx XDG_CACHE_HOME $HOME/.cache
set -gx XDG_STATE_HOME $HOME/.local/state
set -gx XDG_RUNTIME_DIR /run/user/1000
set -gx SSH_AUTH_SOCK "$XDG_RUNTIME_DIR/ssh-agent.socket"

# Unofficial but commonly used XDG extension for user binaries
set -gx XDG_BIN_HOME $HOME/.local/bin

# Karei path
set -gx KAREI_PATH $XDG_DATA_HOME/karei

# XDG State Directory usage for history files
set -gx LESSHISTFILE $XDG_STATE_HOME/less/history
set -gx NODE_REPL_HISTORY $XDG_STATE_HOME/node_repl_history

# User limits
ulimit -S 70000
ulimit -f unlimited