function code-quality --description "Run code quality checks on current project"
    # Parse arguments
    set -l cmd $argv[1]
    set -l remaining_args $argv[2..-1]
    
    switch $cmd
        case init
            echo "🔧 Initializing code quality for this project..."
            
            # Check if pre-commit config exists
            if test -f .pre-commit-config.yaml
                echo "Pre-commit config already exists"
            else if test -f "$XDG_CONFIG_HOME/karei/templates/pre-commit-config.yaml"
                echo "Copying pre-commit template..."
                cp "$XDG_CONFIG_HOME/karei/templates/pre-commit-config.yaml" .pre-commit-config.yaml
                echo "✅ Created .pre-commit-config.yaml"
            else
                echo "❌ Pre-commit template not found. Run: karei install code-quality"
                return 1
            end
            
            # Initialize pre-commit
            if command -v pre-commit >/dev/null 2>&1
                pre-commit install
                echo "✅ Pre-commit hooks installed"
            else
                echo "❌ pre-commit not found. Run: karei install code-quality"
                return 1
            end
            
        case check run
            echo "🔍 Running code quality checks..."
            
            # Run pre-commit if available
            if test -f .pre-commit-config.yaml && command -v pre-commit >/dev/null 2>&1
                pre-commit run --all-files $remaining_args
            else
                echo "Running available linters..."
                
                # ShellCheck for shell scripts
                if command -v shellcheck >/dev/null 2>&1
                    set -l shell_files (find . -type f -name "*.sh" -o -name "*.bash" 2>/dev/null)
                    if test (count $shell_files) -gt 0
                        echo "📋 ShellCheck:"
                        shellcheck $shell_files
                    end
                end
                
                # Python linting
                if command -v ruff >/dev/null 2>&1
                    set -l python_files (find . -type f -name "*.py" 2>/dev/null)
                    if test (count $python_files) -gt 0
                        echo "🐍 Python (ruff):"
                        ruff check $python_files
                    end
                end
                
                # JavaScript linting
                if command -v eslint >/dev/null 2>&1
                    if test -f package.json
                        echo "📜 JavaScript (eslint):"
                        eslint .
                    end
                end
            end
            
        case fix format
            echo "🔧 Fixing code quality issues..."
            
            # Run pre-commit with auto-fix
            if test -f .pre-commit-config.yaml && command -v pre-commit >/dev/null 2>&1
                pre-commit run --all-files --hook-stage manual $remaining_args
            else
                # Run individual formatters
                
                # Python formatting
                if command -v black >/dev/null 2>&1
                    set -l python_files (find . -type f -name "*.py" 2>/dev/null)
                    if test (count $python_files) -gt 0
                        echo "🐍 Formatting Python files..."
                        black $python_files
                    end
                end
                
                # JavaScript formatting
                if command -v prettier >/dev/null 2>&1
                    if test -f package.json
                        echo "📜 Formatting JavaScript/TypeScript files..."
                        prettier --write "**/*.{js,jsx,ts,tsx,json,md}"
                    end
                end
                
                # Go formatting
                if command -v gofumpt >/dev/null 2>&1
                    set -l go_files (find . -type f -name "*.go" 2>/dev/null)
                    if test (count $go_files) -gt 0
                        echo "🐹 Formatting Go files..."
                        gofumpt -w $go_files
                    end
                end
            end
            
        case shell
            # Quick shell script check
            if test (count $remaining_args) -eq 0
                echo "Usage: code-quality shell <script.sh>"
                return 1
            end
            
            if command -v shellcheck >/dev/null 2>&1
                shellcheck $remaining_args
            else
                echo "❌ ShellCheck not installed. Run: karei install code-quality"
                return 1
            end
            
        case status
            echo "📊 Code Quality Tools Status:"
            echo ""
            
            # Check installed tools
            set -l tools "pre-commit" "shellcheck" "ruff" "black" "eslint" "prettier" "rubocop" "golangci-lint"
            
            for tool in $tools
                if command -v $tool >/dev/null 2>&1
                    echo "✅ $tool: "(command -v $tool)
                else
                    echo "❌ $tool: not installed"
                end
            end
            
            echo ""
            echo "📁 Project Status:"
            test -f .pre-commit-config.yaml && echo "✅ Pre-commit config found" || echo "❌ No pre-commit config"
            test -d .git/hooks && test -f .git/hooks/pre-commit && echo "✅ Pre-commit hooks installed" || echo "❌ Pre-commit hooks not installed"
            
        case help '*'
            echo "Code quality management for Karei"
            echo ""
            echo "Usage:"
            echo "  code-quality init           Initialize code quality for current project"
            echo "  code-quality check          Run all quality checks"
            echo "  code-quality fix            Auto-fix issues where possible"
            echo "  code-quality shell <file>   Check specific shell script"
            echo "  code-quality status         Show installed tools and project status"
            echo "  code-quality help           Show this help"
            echo ""
            echo "Examples:"
            echo "  code-quality init          # Set up pre-commit in current project"
            echo "  code-quality check         # Run all linters"
            echo "  code-quality fix           # Auto-format code"
            echo "  code-quality shell app.sh  # Check specific shell script"
    end
end