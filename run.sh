#!/bin/bash -ex

close_it_down(){
    docker kill "$(docker ps -q)"
}

trap "close_it_down" SIGINT

time docker build -t uehreka/stylevision .

NETWORK_NAME="darger4"
# NETWORK_NAME="candy"
# NETWORK_NAME="starry-night-van-gogh"
# NETWORK_NAME="woman-with-hat-matisse"
# NETWORK_NAME="darger4-big"
# NETWORK_NAME="dora-marr"
# NETWORK_NAME="dora-maar-1"

# MODEL_VERSION="optimized"
MODEL_VERSION="trtfp16"

PERFORM_TRANSFER="true"
# PERFORM_TRANSFER="false"

FRAME_WIDTH=960
FRAME_HEIGHT=540

# sudo rm -fdr logs/ffwd
# mkdir logs/ffwd

# xhost +
if [[ $PERFORM_TRANSFER = true ]]
then
    docker run --gpus all --rm \
        --shm-size=1g \
        --ulimit memlock=-1 \
        --network host \
        -e "DISPLAY=$DISPLAY" \
        -e "ENV=development" \
        --volume "$(pwd)/certs:/var/certs" \
        --volume /tmp/.X11-unix:/tmp/.X11-unix \
        --volume "$(pwd)/pretrained-networks:/var/pretrained-networks" \
        --volume "$(pwd)/results:/var/output" \
        -e "NETWORK_NAME=${NETWORK_NAME}" \
        -e "MODEL_VERSION=${MODEL_VERSION}" \
        -e "FRAME_WIDTH=${FRAME_WIDTH}" \
        -e "FRAME_HEIGHT=${FRAME_HEIGHT}" \
        -e "PERFORM_TRANSFER=${PERFORM_TRANSFER}" \
        -it uehreka/stylevision
else
    docker run --rm \
        --shm-size=1g \
        --ulimit memlock=-1 \
        --network host \
        -e "DISPLAY=$DISPLAY" \
        -e "ENV=development" \
        --volume "$(pwd)/certs:/var/certs" \
        --volume /tmp/.X11-unix:/tmp/.X11-unix \
        --volume "$(pwd)/pretrained-networks:/var/pretrained-networks" \
        --volume "$(pwd)/results:/var/output" \
        -e "NETWORK_NAME=${NETWORK_NAME}" \
        -e "MODEL_VERSION=${MODEL_VERSION}" \
        -e "FRAME_WIDTH=${FRAME_WIDTH}" \
        -e "FRAME_HEIGHT=${FRAME_HEIGHT}" \
        -e "PERFORM_TRANSFER=${PERFORM_TRANSFER}" \
        -it uehreka/stylevision
fi