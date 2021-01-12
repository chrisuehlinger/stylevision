# Copyright (c) 2016-2017 Shafeen Tejani. Released under GPLv3.

import os

import numpy as np
import scipy
import scipy.misc
from os.path import exists
from sys import stdout

import math
import utils
from fast_style_transfer import FastStyleTransfer
from argparse import ArgumentParser
import tensorflow as tf
from tensorflow.python.framework import graph_util
from tensorflow.python.tools.optimize_for_inference_lib import optimize_for_inference
import tensorflow.contrib.tensorrt as trt
# from tensorflow.python.compiler.tensorrt import trt_convert as trt

CONTENT_WEIGHT = 1
STYLE_WEIGHT = 5
TV_WEIGHT = 1e-6
LEARNING_RATE = 1e-3
NUM_EPOCHS=5
BATCH_SIZE=4
VGG_PATH = 'imagenet-vgg-verydeep-19.mat'
CHECKPOINT_ITERATIONS = 100
SAVE_PATH = 'network'

def build_parser():
    parser = ArgumentParser()

    parser.add_argument('--style', type=str,
                        dest='style', help='style image path',
                        metavar='STYLE', required=True)

    parser.add_argument('--train-path', type=str,
                        dest='train_path', help='path to training images folder',
                        metavar='TRAIN_PATH', required=True)

    parser.add_argument('--save-path', type=str,
                        dest='save_path',
                        help='directory to save network (default %(default)s)',
                        metavar='SAVE_PATH', default=SAVE_PATH)

    parser.add_argument('--epochs', type=int,
                        dest='epochs', help='num epochs',
                        metavar='EPOCHS', default=NUM_EPOCHS)

    parser.add_argument('--batch-size', type=int,
                        dest='batch_size', help='batch size',
                        metavar='BATCH_SIZE', default=BATCH_SIZE)

    parser.add_argument('--checkpoint-iterations', type=int,
                        dest='checkpoint_iterations', help='checkpoint frequency',
                        metavar='CHECKPOINT_ITERATIONS',
                        default=CHECKPOINT_ITERATIONS)

    parser.add_argument('--vgg-path', type=str,
                        dest='vgg_path',
                        help='path to VGG19 network (default %(default)s)',
                        metavar='VGG_PATH', default=VGG_PATH)

    parser.add_argument('--content-weight', type=float,
                        dest='content_weight',
                        help='content weight (default %(default)s)',
                        metavar='CONTENT_WEIGHT', default=CONTENT_WEIGHT)

    parser.add_argument('--style-weight', type=float,
                        dest='style_weight',
                        help='style weight (default %(default)s)',
                        metavar='STYLE_WEIGHT', default=STYLE_WEIGHT)

    parser.add_argument('--tv-weight', type=float,
                        dest='tv_weight',
                        help='total variation regularization weight (default %(default)s)',
                        metavar='TV_WEIGHT', default=TV_WEIGHT)

    parser.add_argument('--learning-rate', type=float,
                        dest='learning_rate',
                        help='learning rate (default %(default)s)',
                        metavar='LEARNING_RATE', default=LEARNING_RATE)

    parser.add_argument('--use-gpu', dest='use_gpu', help='run on a GPU', action='store_true')
    parser.set_defaults(use_gpu=False)

    return parser

def check_opts(opts):
    assert exists(opts.style), "style path not found!"
    assert exists(opts.train_path), "train path not found!"

    assert exists(opts.vgg_path), "vgg network not found!"
    assert exists(opts.save_path), "save path not found!"
    assert opts.epochs > 0
    assert opts.batch_size > 0
    assert opts.checkpoint_iterations > 0
    assert os.path.exists(opts.vgg_path)
    assert opts.content_weight >= 0
    assert opts.style_weight >= 0
    assert opts.tv_weight >= 0
    assert opts.learning_rate >= 0


def main():
    print('START')
    parser = build_parser()
    options = parser.parse_args()
    check_opts(options)

    style_image = utils.load_image(options.style)
    style_image = np.ndarray.reshape(style_image, (1,) + style_image.shape)

    content_targets = utils.get_files(options.train_path)
    content_shape = utils.load_image(content_targets[0]).shape

    device = '/gpu:0' if options.use_gpu else '/cpu:0'

    config = tf.compat.v1.ConfigProto()
    # config.gpu_options.per_process_gpu_memory_fraction=0.8
    config.gpu_options.allow_growth=True
    with tf.Session(config=config, graph=tf.Graph()) as sess:

        print('SETTING UP STYLE TRANSFER...')
        style_transfer = FastStyleTransfer(
            vgg_path=options.vgg_path,
            style_image=style_image,
            content_shape=content_shape,
            content_weight=options.content_weight,
            style_weight=options.style_weight,
            tv_weight=options.tv_weight,
            batch_size=options.batch_size,
            device=device,
            sess=sess)
        print('DONE SETTING UP STYLE TRANSFER')

        final_losses = None
        for iteration, network, first_image, losses in style_transfer.train(
            content_training_images=content_targets,
            learning_rate=options.learning_rate,
            epochs=options.epochs,
            checkpoint_iterations=options.checkpoint_iterations,
            sess=sess
        ):
            print_losses(losses)
            
            with open(options.save_path + '/training-metadata.txt', 'a') as f:
                log_losses(losses, iteration, f)

            # saver = tf.compat.v1.train.Saver()
            # if (iteration % 100 == 0):
            #     saver.save(network, options.save_path + '/fast_style_network.ckpt')

            # saver.save(network, options.save_path + '/fast_style_network.ckpt')

        output_graph = graph_util.convert_variables_to_constants(sess, sess.graph.as_graph_def(), ['myOutput'])
        tf.io.write_graph(output_graph, '/var/tf-logs', options.save_path + '/model-constant.pb', as_text=False)

        output_graph = optimize_for_inference(output_graph, ['input_batch'], ['myOutput'], tf.float32.as_datatype_enum)
        tf.io.write_graph(output_graph, '/var/tf-logs', options.save_path + '/model-optimized.pb', as_text=False)
        
        output_graph = trt.create_inference_graph(output_graph, ['myOutput'], precision_mode="FP16", is_dynamic_op=True, max_batch_size=8, max_workspace_size_bytes=(1 << 32))
        tf.io.write_graph(output_graph, '/var/tf-logs', options.save_path + '/model-trtfp16.pb', as_text=False)
        # converter = trt.TrtGraphConverter(
        #     input_graph_def=output_graph,
        #     nodes_blacklist=['myOutput'],
        #     precision_mode="INT8",
        #     is_dynamic_op=True,
        #     use_calibration=True,
        #     max_batch_size=1,
        #     max_workspace_size_bytes=(1 << 32))


        # def input_map_fn():
        #     width = 1280
        #     height = 720
        #     blank_frame = tf.eye((1, height, width, 3))
        #     # img_placeholder = tf.compat.v1.placeholder(tf.float32, shape=blank_frame.shape, name='img_placeholder')
        #     return {
        #         'input_batch:0': blank_frame
        #     }
        # output_graph = converter.convert()
        # output_graph = converter.calibrate(
        #     fetch_names=['myOutput'],
        #     num_runs=1,
        #     input_map_fn=input_map_fn)
        # tf.io.write_graph(output_graph, '/var/tf-logs', options.save_path + '/model-trtint8.pb', as_text=False)
        
        tf.summary.FileWriter('/var/logs/ffwd', output_graph)


def log_losses(losses, iteration, f):
    f.write('\nIteration %d\n' % iteration)
    f.write('  content loss: %g\n' % losses['content'])
    f.write('    style loss: %g\n' % losses['style'])
    f.write('       tv loss: %g\n' % losses['total_variation'])
    f.write('    total loss: %g\n\n' % losses['total'])

def print_losses(losses):
    stdout.write('  content loss: %g\n' % losses['content'])
    stdout.write('    style loss: %g\n' % losses['style'])
    stdout.write('       tv loss: %g\n' % losses['total_variation'])
    stdout.write('    total loss: %g\n' % losses['total'])


if __name__ == '__main__':
    main()
