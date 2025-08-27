function show_security_warnings --description "Display security warnings from installed security tools"
    # Only run if security tools are installed and logs exist
    if not test -d /var/log/karei-security
        return
    end

    set warnings ""
    set warning_count 0

    # Lynis warnings (red) - check last audit
    if test -f /var/log/karei-security/lynis.log
        set lynis_warn (grep '\[WRN\]' /var/log/karei-security/lynis.log 2>/dev/null | wc -l)
        if test "$lynis_warn" -gt 0
            set warnings "$warnings\n\033[31m⚠ Lynis: $lynis_warn warnings found! Run: karei security audit\033[0m"
            set warning_count (math $warning_count + 1)
        end
    end

    # chkrootkit infections (red)
    if test -f /var/log/karei-security/chkrootkit.log
        set chk_warn (grep -i 'INFECTED\|Possible' /var/log/karei-security/chkrootkit.log 2>/dev/null | wc -l)
        if test "$chk_warn" -gt 0
            set warnings "$warnings\n\033[31m⚠ chkrootkit: Potential threats detected! Run: karei security scan\033[0m"
            set warning_count (math $warning_count + 1)
        end
    end

    # rkhunter warnings (yellow)
    if test -f /var/log/karei-security/rkhunter.log
        set rkh_warn (grep -i 'Warning\|Found\|\[ Warning \]' /var/log/karei-security/rkhunter.log 2>/dev/null | wc -l)
        if test "$rkh_warn" -gt 0
            set warnings "$warnings\n\033[33m⚠ rkhunter: $rkh_warn issues detected! Run: karei security check\033[0m"
            set warning_count (math $warning_count + 1)
        end
    end

    # ClamAV infected files (red) - check scan summary
    if test -f /var/log/karei-security/clamav.log
        set clam_infected (tail -5 /var/log/karei-security/clamav.log 2>/dev/null | grep -E "Infected files: [1-9]")
        if test -n "$clam_infected"
            set infected_count (echo $clam_infected | grep -o '[0-9]\+')
            set warnings "$warnings\n\033[31m⚠ ClamAV: $infected_count infected files found! Run: karei security scan\033[0m"
            set warning_count (math $warning_count + 1)
        end
    end

    # Display warnings or success message
    if test $warning_count -gt 0
        echo -e "\n\033[1;31mSecurity Warnings ($warning_count):\033[0m$warnings"
        echo -e "\033[36mRun 'karei security' for management options\033[0m\n"
    else if test -f /var/log/karei-security/lynis.log
        # Only show green message if security tools are actually installed and have run
        echo -e "\033[32m✓ No security warnings detected\033[0m"
    end
end