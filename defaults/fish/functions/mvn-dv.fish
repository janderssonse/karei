function mvn-dv --description "Maven dependency version scanner - Check for available updates"
    if test "$argv[1]" = "--help" -o "$argv[1]" = "-h"
        echo "Maven dependency version scanner"
        echo "------------------------------"
        echo "Scans for available dependency, plugin, and property updates"
        echo ""
        echo "Usage:"
        echo "  mvn-dv           Run all dependency scans"
        echo "  mvn-dv deps      Check dependency updates only"
        echo "  mvn-dv plugins   Check plugin updates only" 
        echo "  mvn-dv props     Check property updates only"
        return 0
    end
    
    if not test -f pom.xml
        echo "❌ No pom.xml found. Run from Maven project root."
        return 1
    end
    
    switch "$argv[1]"
        case "deps"
            echo "🔍 Checking dependency updates..."
            mvn versions:display-dependency-updates
        case "plugins"
            echo "🔍 Checking plugin updates..."
            mvn versions:display-plugin-updates
        case "props"
            echo "🔍 Checking property updates..."
            mvn versions:display-property-updates
        case "*"
            echo "🔍 Checking all Maven updates..."
            mvn versions:display-dependency-updates
            mvn versions:display-plugin-updates  
            mvn versions:display-property-updates
    end
end