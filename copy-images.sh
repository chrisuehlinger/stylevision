#!/bin/bash -ex

rm -fdr training-images-sample
mkdir training-images-sample

for f in $(ls training-images | head -800000)
do
    echo "$f"
    cp "training-images/${f}" "training-images-sample/${f}" &
done

wait