function proxy-status --description "Show current proxy configuration and status"
    echo "▸ Proxy Configuration Status"
    echo "==============================="
    
    # Check if proxy config file exists
    if test -f ~/.config/karei/proxy.conf
        echo "▫ Config file: ~/.config/karei/proxy.conf ✓"
    else
        echo "▫ Config file: Not configured ✗"
        echo "   Run 'karei install' and select 'Corporate Proxy Support'"
        return
    end
    
    # Check current session proxy status
    if test -n "$http_proxy"
        echo ""
        echo "◈ Current Session: ENABLED"
        echo "   HTTP Proxy:  $http_proxy"
        echo "   HTTPS Proxy: $https_proxy"
        echo "   No Proxy:    $no_proxy"
    else
        echo ""
        echo "◈ Current Session: DISABLED"
        echo "   Use 'proxy-enable' to enable for this session"
    end
    
    # Check system-level configurations
    echo ""
    echo "▸ System Configuration Status:"
    
    # Check APT proxy
    if test -f /etc/apt/apt.conf.d/95karei-proxy
        echo "   APT:    Configured ✓"
    else
        echo "   APT:    Not configured ✗"
    end
    
    # Check Git proxy
    set git_proxy (git config --global --get http.proxy 2>/dev/null)
    if test -n "$git_proxy"
        echo "   Git:    Configured ✓ ($git_proxy)"
    else
        echo "   Git:    Not configured ✗"
    end
    
    # Check NPM proxy
    if command -v npm >/dev/null 2>&1
        set npm_proxy (npm config get proxy 2>/dev/null)
        if test "$npm_proxy" != "null" -a -n "$npm_proxy"
            echo "   NPM:    Configured ✓ ($npm_proxy)"
        else
            echo "   NPM:    Not configured ✗"
        end
    else
        echo "   NPM:    Not installed"
    end
    
    # Check Maven proxy
    if test -f ~/.m2/settings.xml
        if grep -q "karei-proxy" ~/.m2/settings.xml
            echo "   Maven:  Configured ✓"
        else
            echo "   Maven:  Configuration exists but not Karei-managed"
        end
    else
        echo "   Maven:  Not configured ✗"
    end
    
    echo ""
    echo "▪ Management Commands:"
    echo "   proxy-enable    # Enable proxy for current session"
    echo "   proxy-disable   # Disable proxy for current session"
    echo "   karei proxy    # Reconfigure proxy settings"
end