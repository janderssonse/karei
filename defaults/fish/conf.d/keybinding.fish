# Key bindings and dircolors for Fish shell
# Only run in interactive sessions
status is-interactive; or exit

# Accept autosuggestion with Alt+Y
bind -M insert \ey accept-autosuggestion

# Set up dircolors for proper LS_COLORS
# XDG Base Directory compliant approach - check XDG location first
if test -f $XDG_CONFIG_HOME/dircolors
    eval (dircolors -c $XDG_CONFIG_HOME/dircolors)
else if test -f ~/.dir_colors
    eval (dircolors -c ~/.dir_colors)
else if test -f /etc/DIR_COLORS
    eval (dircolors -c /etc/DIR_COLORS)
else
    eval (dircolors -c)
end