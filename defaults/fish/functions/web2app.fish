function web2app --description "Create desktop launcher for web app"
    if test (count $argv) -ne 3
        echo "Usage: web2app <AppName> <AppURL> <IconURL> (IconURL must be in PNG -- use https://dashboardicons.com)"
        return 1
    end
    
    set app_name $argv[1]
    set app_url $argv[2]
    set icon_url $argv[3]
    set icon_dir "$XDG_DATA_HOME/applications/icons"
    set desktop_file "$XDG_DATA_HOME/applications/$app_name.desktop"
    set icon_path "$icon_dir/$app_name.png"
    
    mkdir -p "$icon_dir"
    
    if not curl -sL -o "$icon_path" "$icon_url"
        echo "Error: Failed to download icon."
        return 1
    end
    
    echo "[Desktop Entry]
Version=1.0
Name=$app_name
Comment=$app_name
Exec=google-chrome --app=\"$app_url\" --name=\"$app_name\" --class=\"$app_name\" --window-size=800,600
Terminal=false
Type=Application
Icon=$icon_path
Categories=GTK;
MimeType=text/html;text/xml;application/xhtml_xml;
StartupNotify=true" > "$desktop_file"
    
    chmod +x "$desktop_file"
end