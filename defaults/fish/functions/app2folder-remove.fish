function app2folder-remove --description "Remove desktop app from GNOME folder"
    if test (count $argv) -ne 2
        set folders (gsettings get org.gnome.desktop.app-folders folder-children | tr -d "[],'")
        echo "Usage: app2folder-remove <desktop_file.desktop> <folder_name>"
        echo "Folders: $folders"
        return 1
    end
    
    set desktop_file $argv[1]
    set folder $argv[2]
    set schema "org.gnome.desktop.app-folders.folder:/org/gnome/desktop/app-folders/folders/$folder/"
    set current_apps (gsettings get "$schema" apps)
    
    if string match -q "*$desktop_file*" $current_apps
        set raw_list (string replace -r "[\[\]']" '' $current_apps)
        set apps_array (string split ',' $raw_list)
        
        set new_apps
        for app in $apps_array
            set app (string trim $app)
            if test "$app" != "$desktop_file" -a -n "$app"
                set new_apps $new_apps "'$app'"
            end
        end
        
        set new_list (string join ',' $new_apps)
        gsettings set "$schema" apps "[$new_list]"
    end
end