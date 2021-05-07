#!/bin/bash

# export PION_LOG_TRACE=all
if [[ $OUT_CODEC = "h264" ]]
then
    ./pion-receiver | ffmpeg -hide_banner -loglevel panic -f h264 -r 30 -i - -f rawvideo -vf scale="${FRAME_WIDTH}x${FRAME_HEIGHT}":flags=lanczos -pix_fmt rgb24 -r 30 -
elif [[ $OUT_CODEC = "vp8" ]]
then
    ./pion-receiver | ffmpeg -hide_banner -loglevel panic -f ivf -r 30 -i - -f rawvideo -vf scale="${FRAME_WIDTH}x${FRAME_HEIGHT}":flags=lanczos -pix_fmt rgb24 -r 30 -
fi
