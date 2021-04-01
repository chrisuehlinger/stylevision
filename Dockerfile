FROM nvcr.io/nvidia/tensorflow:20.09-tf1-py3


# Supress warnings about missing front-end. As recommended at:
# http://stackoverflow.com/questions/22466255/is-it-possibe-to-answer-dialog-questions-when-installing-under-docker
ARG DEBIAN_FRONTEND=noninteractive
ENV GO111MODULE on

RUN apt-get update -y && \
    apt-get install -y software-properties-common && \
    add-apt-repository ppa:longsleep/golang-backports && \
    apt-get update -y && \
    apt-get install -y \
        python3-numpy python3-scipy pkg-config curl wget golang-go iperf3 nethogs iftop ffmpeg && \
    curl https://bootstrap.pypa.io/get-pip.py -o get-pip.py && \
    python3 get-pip.py --force-reinstall && \
    pip3 install pillow imageio && \
    apt-get clean autoclean && apt-get autoremove -y && rm -rf /var/lib/{apt,dpkg,cache,log}/

WORKDIR /home/tf-transfer

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY ./*.go ./
RUN go build pion-receiver.go && go build pion-sender.go

COPY www www 
COPY ./*.py ./
COPY ./*.sh ./

CMD ["./command.sh"]