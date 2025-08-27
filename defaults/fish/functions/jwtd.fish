function jwtd --description "JWT decoder - Decode JSON Web Tokens"
    if test "$argv[1]" = "--help" -o "$argv[1]" = "-h"
        echo "JWT decoder - Decode JSON Web Tokens"
        echo "-----------------------------------"
        echo "Decodes header, payload, and signature of a JWT token"
        echo ""
        echo "Usage:"
        echo "  jwtd <token>     Decode JWT token"
        echo "  jwtd --help      Show this help"
        echo ""
        echo "Example:"
        echo "  jwtd eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.signature"
        return 0
    end
    
    if not command -q jq
        echo "❌ jq is required but not installed"
        return 1
    end
    
    if test (count $argv) -eq 0
        echo "❌ No JWT token provided"
        echo "Usage: jwtd <token>"
        return 1
    end
    
    set token $argv[1]
    
    # Validate JWT format (3 parts separated by dots)
    if test (echo $token | tr '.' '\n' | wc -l) -ne 3
        echo "❌ Invalid JWT format. Expected 3 parts separated by dots."
        return 1
    end
    
    echo "🔓 JWT Token Decoded:"
    echo "===================="
    
    # Use jq to build proper JSON structure
    echo $token | jq -R 'split(".") | {
        header: (.[0] | @base64d | fromjson),
        payload: (.[1] | @base64d | fromjson), 
        signature: .[2]
    }' 2>/dev/null; or begin
        echo "❌ Failed to decode JWT token. Invalid base64 or JSON format."
        return 1
    end
end