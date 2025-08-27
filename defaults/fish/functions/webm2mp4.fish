function webm2mp4 --description "Convert webm files to mp4 with ffmpeg"
    if test (count $argv) -ne 1
        echo "Usage: webm2mp4 <input.webm>"
        return 1
    end
    
    set input_file $argv[1]
    set output_file (string replace -r '\.webm$' '.mp4' $input_file)
    
    ffmpeg -i "$input_file" -c:v libx264 -preset slow -crf 22 -c:a aac -b:a 192k "$output_file"
end