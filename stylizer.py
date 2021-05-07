from argparse import ArgumentParser
from threading import Event, Lock, Thread
import os
import sys
import traceback
import time
from subprocess import Popen, PIPE

import numpy as np
import tensorflow as tf

from pipe_listener import PipeListener

NETWORK_PATH='networks'
SHOULD_PERFORM_TRANSFER=True

def str2bool(v):
    if isinstance(v, bool):
       return v
    if v.lower() in ('yes', 'true', 't', 'y', '1'):
        return True
    elif v.lower() in ('no', 'false', 'f', 'n', '0'):
        return False
    else:
        raise argparse.ArgumentTypeError('Boolean value expected.')

def eprint(*args, **kwargs):
    print(*args, file=sys.stderr, **kwargs)

def build_parser():
    parser = ArgumentParser()

    parser.add_argument('--network-path', type=str,
                        dest='network_path',
                        help='path to network (default %(default)s)',
                        metavar='NETWORK_PATH', default=NETWORK_PATH)

    parser.add_argument('--perform-transfer', type=str2bool,
                        dest='perform_transfer',
                        help='whether or not to perform style transfer',
                        metavar='PERFORM_TRANSFER', default=SHOULD_PERFORM_TRANSFER)

    parser.add_argument('--width', type=int,
                        dest='width',
                        help='frame width',
                        metavar='WIDTH', default=1280)

    parser.add_argument('--height', type=int,
                        dest='height',
                        help='frame height',
                        metavar='HEIGHT', default=720)

    return parser

# Parse the arguments
parser = build_parser()
args = parser.parse_args()


error_event = Event()
new_frame_event = Event()
new_frame_lock = Lock()
new_frame_queue = list()

config = dict()
config['width'] = args.width
config['height'] = args.height

is_first_frame = True
img_placeholder = None
network = None
processStartTime = None


tf_config = tf.compat.v1.ConfigProto()
# tf_config.gpu_options.per_process_gpu_memory_fraction=0.6
tf_config.gpu_options.allow_growth=True
with tf.compat.v1.Session(config=tf_config, graph=tf.Graph()) as sess:

    graph_def = tf.compat.v1.GraphDef()
    with tf.io.gfile.GFile(args.network_path + '/model-' + os.environ['MODEL_VERSION'] + '.pb', "rb") as f:
            graph_def.ParseFromString(f.read())

    if args.perform_transfer:
        blank_frame = np.zeros((1, config['height'], config['width'], 3))
        img_placeholder = tf.compat.v1.placeholder(tf.float32, shape=blank_frame.shape, name='img_placeholder')
        elements = tf.import_graph_def(graph_def, name="", input_map={'input_batch:0': img_placeholder}, return_elements=['myOutput:0'])
        output_node = elements[0]
        output_node = tf.cast(output_node, dtype=tf.uint8, name='castOutput')

        feed_dict = dict()
        feed_dict[img_placeholder] = blank_frame
        eprint('STYLIZER ABOUT TO RUN #1')
        sess.run(output_node, feed_dict=feed_dict)
        eprint('STYLIZER RUN #1 COMPLETE')

    out_process = Popen(f'./pipe-out.sh', shell=True, stdin=PIPE, stderr=sys.stderr)
    pipe_listener = PipeListener(new_frame_event, new_frame_lock, new_frame_queue, config, error_event)
    display_start_time = time.time()
    
    frameCount = 0
    while True:
        new_frame_event.wait()
        new_frame_lock.acquire()
        content_image = new_frame_queue.pop()
        new_frame_event.clear()
        new_frame_lock.release()
        
        prediction = content_image
        if args.perform_transfer:
            feed_dict = dict()
            feed_dict[img_placeholder] = content_image
            # eprint('STYLIZER ABOUT TO RUN')
            prediction = sess.run(output_node, feed_dict=feed_dict)
            # eprint('STYLIZER RUN COMPLETE')
            prediction = prediction


        # eprint('STYLIZER ABOUT TO WRITE')
        out_process.stdin.write(prediction.tobytes())
        # eprint('STYLIZER WRITE COMPLETE')

        if is_first_frame:
            is_first_frame = False
            display_start_time = time.time()
            frameCount = 0
        else:
            frameCount += 1
            rightNow = time.time()
            fps = 1*frameCount / (rightNow - display_start_time)
            eprint('STYL FPS: ' + str(fps))

        
        if error_event.is_set():
            quit()
        
        # time.sleep(1)