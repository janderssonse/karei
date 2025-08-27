# Zoxide integration
status is-interactive; or exit

if command -q zoxide
    zoxide init fish | source
end