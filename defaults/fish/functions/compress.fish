function compress --description "Create tar.gz archive from directory"
    if test (count $argv) -ne 1
        echo "Usage: compress <directory>"
        return 1
    end
    
    set target (string replace -r '/$' '' $argv[1])
    tar -czf "$target.tar.gz" "$target"
end