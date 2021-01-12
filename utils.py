# Copyright (c) 2016-2017 Shafeen Tejani. Released under GPLv3.
import os

import numpy as np
import scipy
import scipy.misc
import imageio
from os.path import exists

def load_image(image_path, img_size=None):
    assert exists(image_path), "image {} does not exist".format(image_path)
    img = imageio.imread(image_path)
    if (len(img.shape) != 3) or (img.shape[2] != 3):
        img = np.dstack((img, img, img))

    if (img_size is not None):
        img = imageio.imresize(img, img_size)

    img = img.astype("float32")
    return img


def load_image_from_frame(frame, numType='float32'):
    # print frame.format.name
    img = frame.to_ndarray(format="rgb24")
    if (len(img.shape) != 3) or (img.shape[2] != 3):
        img = np.dstack((img, img, img))

    img = img.astype(numType)
    return img

def save_image(img, path):
    imageio.imsave(path, np.clip(img, 0, 255).astype(np.uint8))

def get_files(img_dir):
    files = list_files(img_dir)
    return [os.path.join(img_dir,file) for file in files]

def list_files(in_path):
    files = []
    for (dirpath, dirnames, filenames) in os.walk(in_path):
        files.extend(filenames)
        break

    return files
