function proxy-enable --description "Enable corporate proxy settings for current session"
    if test -f ~/.config/karei/proxy.conf
        source ~/.config/karei/proxy.conf
        
        # Also set Fish-specific environment variables
        set -gx http_proxy $http_proxy
        set -gx https_proxy $https_proxy
        set -gx ftp_proxy $ftp_proxy
        set -gx no_proxy $no_proxy
        set -gx HTTP_PROXY $HTTP_PROXY
        set -gx HTTPS_PROXY $HTTPS_PROXY
        set -gx FTP_PROXY $FTP_PROXY
        set -gx NO_PROXY $NO_PROXY
        
        echo "✅ Corporate proxy enabled: $http_proxy"
        echo "🚫 No proxy for: $no_proxy"
    else
        echo "❌ No proxy configuration found."
        echo "   Run 'karei install' and select 'Corporate Proxy Support' first."
        return 1
    end
end