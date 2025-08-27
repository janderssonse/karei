# PATH configuration for Fish
# Simple PATH management using XDG standards

# User local binaries (XDG_BIN_HOME)
fish_add_path $XDG_BIN_HOME

# Snap binaries  
fish_add_path /snap/bin

# Current directory bin (for project-specific tools)
fish_add_path ./bin