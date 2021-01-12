#!/bin/bash -ex

cat <<EOF > www/constraints.json;
{
    "width": {
        "exact": $FRAME_WIDTH
    },
    "height": {
        "exact": $FRAME_HEIGHT
    },
    "frameRate": {
        "exact": 30
    } 
}
EOF

time /usr/bin/python3 stylizer.py \
    --network-path "/var/pretrained-networks/${NETWORK_NAME}-network" \
    --width "$FRAME_WIDTH" \
    --height "$FRAME_HEIGHT" \
    --perform-transfer "$PERFORM_TRANSFER"
