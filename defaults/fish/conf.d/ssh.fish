# SSH key auto-add configuration
# Only run in interactive sessions
status is-interactive; or exit

# Auto-add SSH key if no identities are loaded
if command -q ssh-add
    if string match -q "*has no identities*" (ssh-add -l 2>&1)
        if test -f $HOME/.ssh/id_ed25519
            ssh-add $HOME/.ssh/id_ed25519 2>/dev/null
        else if test -f $HOME/.ssh/id_rsa  
            ssh-add $HOME/.ssh/id_rsa 2>/dev/null
        end
    end
end