#!/bin/bash -ex

close_it_down(){
    docker kill $(docker ps -q)
}

trap "close_it_down" SIGINT
trap "close_it_down" SIGTERM

./build.sh

rm -fdr training-images
mkdir training-images

time docker run --runtime=nvidia --rm \
    --volume /path/to/imagefolder:/var/input-images \
    --volume $(pwd)/training-images:/var/output-images \
    -it uehreka/darger-tf-style ./prepare-training-images.sh

sudo chmod 777 training-images
for filename in ./training-images/*; do
    echo ${filename}
    sudo chmod 777 ${filename}
done