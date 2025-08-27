function j2y --description "JSON to YAML converter using yq"
    if test "$argv[1]" = "--help" -o "$argv[1]" = "-h"
        echo "JSON to YAML converter"
        echo "--------------------"
        echo "Converts JSON input to YAML format using yq"
        echo ""
        echo "Usage:"
        echo "  j2y <json-string>    Convert JSON string to YAML"
        echo "  j2y <file.json>      Convert JSON file to YAML"
        echo "  cat file.json | j2y  Convert from stdin"
        echo ""
        echo "Examples:"
        echo "  j2y '{\"name\": \"value\"}'"
        echo "  j2y data.json"
        echo "  echo '{\"key\": \"value\"}' | j2y"
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
        yq -P '.'
    else if test -f $argv[1]
        # Read from file  
        yq -P '.' $argv[1]
    else
        # Treat as JSON string
        echo $argv[1] | yq -P '.'
    end
end