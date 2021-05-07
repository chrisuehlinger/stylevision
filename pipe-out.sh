#!/bin/bash

# export PION_LOG_TRACE=all

if [[ $OUT_CODEC = "h264" ]]
then
    if [[ $ENV = "development" ]]
    then
        ffmpeg -hide_banner -f rawvideo -pixel_format rgb24 -video_size "${FRAME_WIDTH}x${FRAME_HEIGHT}" -r 30 -i - -an -c:v libx264 -b:v 4M -g 30 -keyint_min 30 -profile:v baseline -pix_fmt yuv420p -r 30 -f h264 - | ./pion-sender
        
    else
        ffmpeg -hide_banner -f rawvideo -pixel_format rgb24 -video_size "${FRAME_WIDTH}x${FRAME_HEIGHT}" -r 30 -i - -an -c:v h264_nvenc -b:v 4M -g 30 -keyint_min 30 -profile:v baseline -pix_fmt yuv420p -r 30 -f h264 - | ./pion-sender
    fi
elif [[ $OUT_CODEC = "vp8" ]]
then
    ffmpeg -hide_banner -f rawvideo -pixel_format rgb24 -video_size "${FRAME_WIDTH}x${FRAME_HEIGHT}" -r 30 -i - -an -c:v libvpx -b:v 4M -deadline realtime -pix_fmt yuv420p -r 30 -f ivf - | ./pion-sender
fi
