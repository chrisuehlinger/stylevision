import threading
import time
import sys
import os
import traceback
from subprocess import Popen, PIPE

import numpy as np

import utils

# os.environ['OPENCV_FFMPEG_CAPTURE_OPTIONS'] = 'protocol_whitelist;file,rtp,udp'

def eprint(*args, **kwargs):
    print(*args, file=sys.stderr, **kwargs)

class PipeListener(threading.Thread):
    def __init__(self, new_frame_event, new_frame_lock, image_queue, config, error_event):
        self.config = config
        # Get all the parallel primitives
        self.new_frame_event = new_frame_event
        self.new_frame_lock = new_frame_lock
        self.image_queue = image_queue
        self.error_event = error_event
        self.in_process = Popen(f'./pipe-in.sh {config["width"]} {config["height"]}', shell=True, stdout=PIPE, stderr=sys.stderr)


        # Start the thread
        threading.Thread.__init__(self)

        self.daemon = True
        self.start()

    def run(self):
        is_first_frame = True
        eprint('Opening input stream...')
        frameCount = 0
        display_start_time = time.time()
        while True:
            in_bytes = self.in_process.stdout.read(self.config['width'] * self.config['height'] * 3)
            if not in_bytes:
                eprint('NO MORE BYTES')
                self.error_event.set()
                break
            new_frame = (
                np
                .frombuffer(in_bytes, np.uint8)
                .reshape([1, self.config['height'], self.config['width'], 3])
            )
            self.new_frame_lock.acquire()
            if len(self.image_queue) > 0:
                self.image_queue[0] = new_frame
            else:
                self.image_queue.append(new_frame)
            self.new_frame_lock.release()
            self.new_frame_event.set()

            if is_first_frame:
                is_first_frame = False
                display_start_time = time.time()
                frameCount = 0
            else:
                frameCount += 1
                rightNow = time.time()
                fps = 1*frameCount / (rightNow - display_start_time)
                eprint('PIPE FPS: ' + str(fps))
