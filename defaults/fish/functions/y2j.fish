function y2j --description "YAML to JSON converter using yq"
    if test "$argv[1]" = "--help" -o "$argv[1]" = "-h"
        echo "YAML to JSON converter"
        echo "--------------------"
        echo "Converts YAML input to JSON format using yq"
        echo ""
        echo "Usage:"
        echo "  y2j <yaml-string>    Convert YAML string to JSON"
        echo "  y2j <file.yaml>      Convert YAML file to JSON"
        echo "  cat file.yaml | y2j  Convert from stdin"
        echo ""
        echo "Examples:"
        echo "  y2j 'name: value'"
        echo "  y2j config.yaml"
        echo "  echo 'key: value' | y2j"
        return 0
    end
    
    if not command -q yq
        echo "❌ yq is required but not installed"
        echo "💡 Install with: sudo snap install yq"
        return 1
    end
    
    # Handle input sources
    if test (count $argv) -eq 0
        # Read from stdin
        yq -o json '.'
    else if test -f $argv[1]
        # Read from file
        yq -o json '.' $argv[1]
    else
        # Treat as YAML string
        echo $argv[1] | yq -o json '.'
    end
end