function iso2sd --description "Write ISO file to SD card"
    if test (count $argv) -ne 2
        echo "Usage: iso2sd <input_file> <output_device>"
        echo "Example: iso2sd ~/Downloads/ubuntu-25.04-desktop-amd64.iso /dev/sda"
        echo ""
        echo "Available SD cards:"
        lsblk -d -o NAME | grep -E '^sd[a-z]' | awk '{print "/dev/"$1}'
        return 1
    end
    
    set input_file $argv[1]
    set output_device $argv[2]
    
    sudo dd bs=4M status=progress oflag=sync if="$input_file" of="$output_device"
    sudo eject $output_device
end