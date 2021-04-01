#!/bin/bash

# export PION_LOG_TRACE=all

./pion-receiver | ffmpeg -hide_banner -loglevel panic -f h264 -r 30 -i - -f rawvideo -vf scale="${1}x${2}":flags=lanczos -pix_fmt rgb24 -r 30 -
