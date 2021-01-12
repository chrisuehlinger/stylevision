import os
import math
from PIL import Image

in_folder = '/var/input-images'
out_folder = '/var/output-images'
image_list = os.listdir(in_folder)
total_images = len(image_list)
i = 0
for filename in image_list:
    i += 1
    print filename + ': image #' + str(i) + ' of ' + str(total_images)
    img = Image.open(in_folder + '/' + filename)

    width, height = img.size
    smallestSide = min(width, height)
    scalingFactor = float(256) / float(smallestSide)

    width = int(math.ceil(width*scalingFactor))
    height = int(math.ceil(height*scalingFactor))
    print (width, height)
    img = img.resize((width, height))

    xOffset = (width-256)/2
    yOffset = (height-256)/2
    print (xOffset, yOffset)
    img = img.crop((xOffset,yOffset,xOffset+256,yOffset+256))

    out_path = out_folder + '/' + filename
    print 'Saving... ' + out_path
    img.save(out_path)