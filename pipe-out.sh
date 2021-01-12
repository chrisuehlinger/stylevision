#!/bin/bash

# export PION_LOG_TRACE=all

if [[ $ENV = "development" ]]
then
    ffmpeg -hide_banner -f rawvideo -pixel_format rgb24 -video_size "${1}x${2}" -r 30 -i - -an -c:v libx264 -b:v 4M -g 30 -keyint_min 30 -profile:v baseline -pix_fmt yuv420p -r 30 -f h264 - | ./pion-sender-h264
else
    ffmpeg -hide_banner -f rawvideo -pixel_format rgb24 -video_size "${1}x${2}" -r 30 -i - -an -c:v h264_nvenc -b:v 4M -g 30 -keyint_min 30 -profile:v baseline -pix_fmt yuv420p -r 30 -f h264 - | ./pion-sender-h264
fi
