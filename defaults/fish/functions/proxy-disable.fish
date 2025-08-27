function proxy-disable --description "Disable all proxy settings for current session"
    # Unset all proxy environment variables
    set -e http_proxy
    set -e https_proxy
    set -e ftp_proxy
    set -e no_proxy
    set -e HTTP_PROXY
    set -e HTTPS_PROXY
    set -e FTP_PROXY
    set -e NO_PROXY
    
    echo "🔓 All proxy settings disabled for current session"
    echo "   APT, Git, NPM, Maven, and Snap proxy configs remain unchanged"
    echo "   Use 'proxy-enable' to re-enable proxy for this session"
end