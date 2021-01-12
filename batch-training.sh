#!/bin/bash -x


# NETS_TO_TRAIN=(darger3 darger4 candy rain-princess-aframov)
NETS_TO_TRAIN=(darger3)

for i in "${NETS_TO_TRAIN[@]}"; do
    echo "$i";
    ./run-training.sh "$i" 10;
done