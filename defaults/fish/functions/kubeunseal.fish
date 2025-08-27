function kubeunseal --description "Unseal Kubernetes sealed secrets using kubeseal"
    if test "$argv[1]" = "--help" -o "$argv[1]" = "-h"
        echo "Kubernetes sealed secret unsealer"
        echo "--------------------------------"
        echo "Decrypts a sealed secret using kubeseal and cluster private key"
        echo ""
        echo "Usage:"
        echo "  kubeunseal <sealed-secret-file>     Unseal the sealed secret"
        echo "  kubeunseal --help                   Show this help"
        echo ""
        echo "Example:"
        echo "  kubeunseal overlays/resources/sealed-secret.yaml"
        echo ""
        echo "Requirements: kubectl context must be set to target cluster"
        return 0
    end
    
    # Check dependencies (already installed by Karei)
    if not command -q kubectl
        echo "✗ kubectl is required but not installed"
        return 1
    end
    
    if not command -q kubeseal
        echo "✗ kubeseal is required but not installed"
        return 1
    end
    
    if test (count $argv) -eq 0
        echo "✗ No sealed secret file provided"
        echo "Usage: kubeunseal <sealed-secret-file>"
        return 1
    end
    
    set sealed_file $argv[1]
    
    if not test -f $sealed_file
        echo "✗ File not found: $sealed_file"
        return 1
    end
    
    # Create temporary files for secure processing
    set file_name (basename $sealed_file)
    set secret_key (mktemp -t sealed-secrets-key.XXXXXX.yaml)
    set unsealed_file (mktemp -t unsealed-$file_name.XXXXXX.yaml)
    
    echo "▸ Retrieving sealed-secrets private key..."
    
    # Get the sealed-secrets private key from cluster
    if kubectl get secret -n sealed-secrets -l sealedsecrets.bitnami.com/sealed-secrets-key --output yaml > $secret_key 2>/dev/null
        echo "✓ Retrieved sealed-secrets private key"
    else
        echo "✗ Failed to retrieve sealed-secrets private key"
        echo "▪ Ensure you're connected to the correct cluster and have access to sealed-secrets namespace"
        # Safe cleanup - only remove if files are in temp directory
        set temp_dir_pattern "$TMPDIR*" "/tmp/*" "/var/tmp/*"
        for pattern in $temp_dir_pattern
            if string match -q $pattern $secret_key; and test -f $secret_key
                rm -f $secret_key
                break
            end
        end
        for pattern in $temp_dir_pattern
            if string match -q $pattern $unsealed_file; and test -f $unsealed_file
                rm -f $unsealed_file
                break
            end
        end
        return 1
    end
    
    echo "▸ Unsealing secret: $sealed_file"
    
    # Unseal the secret
    if kubeseal \
        --controller-name=sealed-secrets \
        --controller-namespace=sealed-secrets \
        --recovery-unseal \
        --recovery-private-key $secret_key \
        --format yaml \
        < $sealed_file \
        > $unsealed_file 2>/dev/null
        
        # Verify the unsealed file was actually created and has content
        if test -s $unsealed_file
            echo "✓ Successfully unsealed to: $unsealed_file"
            echo "⚠ Unsealed secret is in temporary file"
            echo "▪ Copy to desired location: cp $unsealed_file ./unsealed-$file_name"
            echo "▪ View content: cat $unsealed_file"
            echo "◈ Temporary file will be cleaned up automatically on system restart"
        else
            echo "✗ Unsealing failed - no output generated"
            set exit_code 1
        end
    else
        echo "✗ Failed to unseal secret"
        echo "▪ Check that the sealed secret is valid and matches the cluster"
        set exit_code 1
    end
    
    # Always cleanup secret key (but leave unsealed file for user to access)
    # Safe cleanup - only remove if file is in temp directory
    set temp_dir_pattern "$TMPDIR*" "/tmp/*" "/var/tmp/*"
    for pattern in $temp_dir_pattern
        if string match -q $pattern $secret_key; and test -f $secret_key
            rm -f $secret_key
            break
        end
    end
    
    return $exit_code
end