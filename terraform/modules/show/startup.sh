#!/bin/bash -ex
cat <<EOF > /var/stylevision-options.json;
{
    "NETWORK_NAME":"${network_name}",
    "MODEL_VERSION":"${model_version}",
    "FRAME_WIDTH":${frame_width},
    "FRAME_HEIGHT":${frame_height},
    "PERFORM_TRANSFER":${perform_transfer}
}
EOF