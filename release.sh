#!/bin/bash -ex

time docker build -t uehreka/stylevision .

time docker push uehreka/stylevision