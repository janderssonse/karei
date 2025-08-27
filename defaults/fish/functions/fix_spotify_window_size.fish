function fix_spotify_window_size --description "Fix Spotify window scaling"
    sudo sed -i 's|^Exec=.*|Exec=spotify --force-device-scale-factor=1.5 %U|' /usr/local/share/applications/spotify.desktop
end