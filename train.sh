#!/bin/bash -x

# export TF_ENABLE_AUTO_MIXED_PRECISION=1
export TF_ENABLE_CUBLAS_TENSOR_OP_MATH_FP32=1
export TF_ENABLE_CUDNN_TENSOR_OP_MATH_FP32=1
export TF_ENABLE_CUDNN_RNN_TENSOR_OP_MATH_FP32=1

# /usr/local/cuda-10.0/NsightCompute-1.0/nv-nsight-cu-cli --csv --metrics tensor_precision_fu_utilization,tensor_int_fu_utilization \
    /usr/bin/python3 train_network.py \
    --epochs "${NUM_EPOCHS}" \
    --checkpoint-iterations 10000 \
    --batch-size 8 \
    --style "examples/${NETWORK_NAME}.jpg" \
    --vgg-path "/var/vgg/imagenet-vgg-verydeep-19.mat" \
    --train-path "/var/training-images" \
    --save-path "/var/pretrained-networks/${NETWORK_NAME}-network" \
    --use-gpu

# tensorboard --logdir=/var/logs/ffwd
# tensorboard --logdir=/var/logs/training