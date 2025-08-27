function fix_fkeys --description "Fix external keyboard F-keys to be default"
    echo 2 | sudo tee /sys/module/hid_apple/parameters/fnmode
end