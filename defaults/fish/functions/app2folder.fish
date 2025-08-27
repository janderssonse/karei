function app2folder --description "Move desktop app to GNOME folder"
    if test (count $argv) -ne 2
        set folders (gsettings get org.gnome.desktop.app-folders folder-children | tr -d "[],'")
        echo "Usage: app2folder <desktop_file.desktop> <folder_name>"
        echo "Folders: $folders"
        return 1
    end
    
    set desktop_file $argv[1]
    set folder $argv[2]
    set schema "org.gnome.desktop.app-folders.folder:/org/gnome/desktop/app-folders/folders/$folder/"
    set current_apps (gsettings get "$schema" apps)
    
    if not string match -q "*$desktop_file*" $current_apps
        set trimmed (string replace -r '^\[' '' $current_apps | string replace -r '\]$' '')
        gsettings set "$schema" apps "[$trimmed, '$desktop_file']"
    end
end