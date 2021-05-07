#!/bin/bash -x

close_it_down(){
    docker kill "$(docker ps -q)"
}

trap "close_it_down" SIGINT
trap "close_it_down" SIGTERM

./build.sh

sudo docker system prune -f

# NETWORK_NAME=darger4-big
# NETWORK_NAME=dora-maar-1
NETWORK_NAME=${1:-dora-maar-1}
echo "$NETWORK_NAME"
NUM_EPOCHS=${2:-5}

sudo rm -fdr "pretrained-networks/${NETWORK_NAME}-network"
mkdir "pretrained-networks/${NETWORK_NAME}-network"
{
    printf "Start time: %s\n" "$(date)" ;
    printf "Number of epochs: %s\n" "$NUM_EPOCHS";
    printf "Number of images: %s\n" "$(ls ./training-images-sample | wc -l)";
} > "pretrained-networks/${NETWORK_NAME}-network/training-metadata.txt"
sudo rm -fdr logs/training
mkdir logs/training
while true
do
    time sudo docker run --runtime=nvidia --rm --privileged \
        --volume "$(pwd)/vgg:/var/vgg" \
        --volume "$(pwd)/training-images-sample:/var/training-images" \
        --volume "$(pwd)/pretrained-networks:/var/pretrained-networks" \
        --volume "$(pwd)/logs/training:/var/logs/training" \
        -e "NUM_EPOCHS=${NUM_EPOCHS}" \
        -e "NETWORK_NAME=${NETWORK_NAME}" \
        --network=host \
        --shm-size=1g \
        --ulimit memlock=-1 \
        --memory-swap -1 \
        -it uehreka/stylevision ./train.sh \
    && break
done


printf "Finish time: %s\n" "$(date)" >> "pretrained-networks/${NETWORK_NAME}-network/training-metadata.txt"

sudo chmod 777 "pretrained-networks/${NETWORK_NAME}-network"
for filename in "./pretrained-networks/${NETWORK_NAME}-network/"*; do
    echo "${filename}"
    sudo chmod 777 "${filename}"
done
