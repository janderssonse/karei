# Fish prompt configuration
# Starship prompt initialization with fallback

# Disable fish greeting
set fish_greeting

# Initialize Starship prompt if available
if command -q starship
    starship init fish | source
else
    # Fallback prompt if Starship is not available
    function fish_prompt
        set_color cyan
        echo -n (prompt_pwd)
        set_color normal
        
        # Git branch if in git repo
        if git rev-parse --is-inside-work-tree >/dev/null 2>&1
            set_color purple
            echo -n " ("(git branch --show-current)")"
            set_color normal
        end
        
        echo -n " ❯ "
    end
end