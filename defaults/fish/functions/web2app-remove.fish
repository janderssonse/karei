function web2app-remove --description "Remove web app desktop launcher"
    if test (count $argv) -ne 1
        echo "Usage: web2app-remove <AppName>"
        return 1
    end
    
    set app_name $argv[1]
    set icon_dir "$XDG_DATA_HOME/applications/icons"
    set desktop_file "$XDG_DATA_HOME/applications/$app_name.desktop"
    set icon_path "$icon_dir/$app_name.png"
    
    rm "$desktop_file"
    rm "$icon_path"
end