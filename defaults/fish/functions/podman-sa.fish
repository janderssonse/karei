function podman-sa --description "Podman stop all - Complete cleanup of pods, containers, volumes, networks"
    if test "$argv[1]" = "--help" -o "$argv[1]" = "-h"
        echo "Podman stop all - Complete cleanup of pods, containers, volumes, networks"
        echo "-------------------------------------------------------------------"
        echo "Usage:"
        echo "  podman-sa           Stop and remove all pods and containers"
        echo "  podman-sa --prune   Also cleanup unused volumes and networks"
        echo "  podman-sa --help    Show this help"
        return 0
    end
    
    echo "Stopping and removing all Podman resources..."
    
    # Stop and remove containers first, then pods
    podman stop --all 2>/dev/null; or true
    podman rm --all 2>/dev/null; or true
    podman pod stop --all 2>/dev/null; or true
    podman pod rm --all 2>/dev/null; or true
    
    # Optional: cleanup unused volumes and networks
    if test "$argv[1]" = "--prune"
        echo "Pruning unused volumes and networks..."
        podman volume prune -f 2>/dev/null; or true
        podman network prune -f 2>/dev/null; or true
    end
    
    echo "✅ All Podman resources stopped and removed"
end