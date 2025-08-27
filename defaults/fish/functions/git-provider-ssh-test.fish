function git-provider-ssh-test --description "Test Git providers SSH connectivity (ports 22 and 443)"
    echo "🧪 Testing Git providers SSH connectivity..."
    echo ""
    
    # Test GitHub
    echo "GitHub connectivity:"
    echo "  Standard SSH (port 22):"
    if timeout 10 ssh -T -o ConnectTimeout=5 -o BatchMode=yes git@github.com 2>/dev/null | grep -q "successfully authenticated"
        echo "    ✅ Port 22: Working"
    else
        echo "    ❌ Port 22: Failed or blocked"
    end
    
    echo "  SSH over HTTPS (port 443):"
    if timeout 10 ssh -T -p 443 -o ConnectTimeout=5 -o BatchMode=yes git@ssh.github.com 2>/dev/null | grep -q "successfully authenticated"
        echo "    ✅ Port 443: Working"
    else
        echo "    ❌ Port 443: Failed or blocked"
    end
    
    # Test GitLab
    echo ""
    echo "GitLab connectivity:"
    echo "  Standard SSH (port 22):"
    if timeout 10 ssh -T -o ConnectTimeout=5 -o BatchMode=yes git@gitlab.com 2>/dev/null | grep -q "Welcome to GitLab"
        echo "    ✅ Port 22: Working"
    else
        echo "    ❌ Port 22: Failed or blocked"
    end
    
    echo "  SSH over HTTPS (port 443):"
    if timeout 10 ssh -T -p 443 -o ConnectTimeout=5 -o BatchMode=yes git@altssh.gitlab.com 2>/dev/null | grep -q "Welcome to GitLab"
        echo "    ✅ Port 443: Working"
    else
        echo "    ❌ Port 443: Failed or blocked"
    end
    
    # Test Bitbucket
    echo ""
    echo "Bitbucket connectivity:"
    echo "  Standard SSH (port 22):"
    if timeout 10 ssh -T -o ConnectTimeout=5 -o BatchMode=yes git@bitbucket.org 2>/dev/null | grep -q "authenticated"
        echo "    ✅ Port 22: Working"
    else
        echo "    ❌ Port 22: Failed or blocked"
    end
    
    echo "  SSH over HTTPS (port 443):"
    if timeout 10 ssh -T -p 443 -o ConnectTimeout=5 -o BatchMode=yes git@altssh.bitbucket.org 2>/dev/null | grep -q "authenticated"
        echo "    ✅ Port 443: Working"
    else
        echo "    ❌ Port 443: Failed or blocked"
    end
    
    # Check current configuration
    echo ""
    echo "Current configuration:"
    if grep -q "ssh.github.com\|altssh.gitlab.com\|altssh.bitbucket.org" ~/.ssh/user.config 2>/dev/null
        echo "✅ Git SSH over HTTPS: ENABLED"
        if grep -q "ProxyCommand" ~/.ssh/user.config 2>/dev/null
            echo "✅ SSH proxy tunneling: ENABLED"
        end
    else
        echo "❌ Git SSH over HTTPS: DISABLED"
    end
end