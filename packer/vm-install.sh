#!/bin/bash -ex

short_name=$1

echo 'APT::Periodic::Update-Package-Lists "1";' | tee /etc/apt/apt.conf.d/20auto-upgrades
echo 'APT::Periodic::Unattended-Upgrade "0";' | tee -a /etc/apt/apt.conf.d/20auto-upgrades

echo -e "tail -f /var/log/cloud-init-output.log" > /home/ubuntu/.bash_history

add-apt-repository universe
apt-get update

while sudo fuser /var/{lib/{dpkg,apt/lists},cache/apt/archives}/lock >/dev/null 2>&1; do
    echo "Waiting for dpkg lock..."
    sleep 1
done

apt-get install -y s3fs tmux jq nethogs iperf3 iftop

mkdir -p /mnt/secret
chown 1000:1000 /mnt/secret
echo "s3fs#${short_name}-secret /mnt/secret fuse _netdev,rw,uid=1000,gid=1000,allow_other,iam_role=${short_name}-show-role" | tee -a /etc/fstab

cat <<'EOF' > /home/ubuntu/boot.sh;
#!/bin/bash -x

close_it_down(){
    docker kill "$(docker ps -q)"
}

trap "close_it_down" SIGINT
trap "close_it_down" SIGTERM

docker pull uehreka/stylevision:latest
docker system prune -f
export LD_LIBRARY_PATH="/usr/lib64/openmpi/lib/:/usr/local/cuda/lib64:/usr/local/lib:/usr/lib:/usr/local/cuda/extras/CUPTI/lib64:/usr/local/mpi/lib:/lib/:/usr/local/cuda/lib64:/usr/local/lib:/usr/lib:/usr/local/cuda/extras/CUPTI/lib64:/opt/amazon/openmpi/lib:/usr/local/cuda/lib:/opt/amazon/efa/lib:/usr/local/mpi/lib:/usr/lib64/openmpi/lib/:/usr/local/cuda/lib64:/usr/local/lib:/usr/lib:/usr/local/cuda/extras/CUPTI/lib64:/usr/local/mpi/lib:/lib/:"
nvidia-smi
cat /var/darger-options.json
uptime
if [[ $(jq -r '.PERFORM_TRANSFER' < /var/darger-options.json) = true ]]
then
    echo "YES NVIDIA DOCKER"
    docker run --gpus all --rm \
        --shm-size=1g \
        --ulimit memlock=-1 \
        --network host \
        --volume '/mnt/secret/letsencrypt:/etc/letsencrypt' \
        --volume '/mnt/secret/pretrained-networks:/var/pretrained-networks' \
        -e "ENV=production" \
        -e "NETWORK_NAME=$(jq -r '.NETWORK_NAME' < /var/darger-options.json)" \
        -e "MODEL_VERSION=$(jq -r '.MODEL_VERSION' < /var/darger-options.json)" \
        -e "FRAME_WIDTH=$(jq -r '.FRAME_WIDTH' < /var/darger-options.json)" \
        -e "FRAME_HEIGHT=$(jq -r '.FRAME_HEIGHT' < /var/darger-options.json)" \
        -e "PERFORM_TRANSFER=$(jq -r '.PERFORM_TRANSFER' < /var/darger-options.json)" \
        -t uehreka/stylevision &
else
    echo "NO NVIDIA DOCKER"
    docker run --rm \
        --shm-size=1g \
        --ulimit memlock=-1 \
        --network host \
        --volume '/mnt/secret/letsencrypt:/etc/letsencrypt' \
        --volume '/mnt/secret/pretrained-networks:/var/pretrained-networks' \
        -e "ENV=production" \
        -e "NETWORK_NAME=$(jq -r '.NETWORK_NAME' < /var/darger-options.json)" \
        -e "MODEL_VERSION=$(jq -r '.MODEL_VERSION' < /var/darger-options.json)" \
        -e "FRAME_WIDTH=$(jq -r '.FRAME_WIDTH' < /var/darger-options.json)" \
        -e "FRAME_HEIGHT=$(jq -r '.FRAME_HEIGHT' < /var/darger-options.json)" \
        -e "PERFORM_TRANSFER=$(jq -r '.PERFORM_TRANSFER' < /var/darger-options.json)" \
        -t uehreka/stylevision &
fi

wait
EOF
chmod a+x /home/ubuntu/boot.sh;

cat <<EOF > /etc/systemd/system/show.service;
[Unit]
Description=Show
Requires=containerd.service
After=cloud-final.service

[Service]
WorkingDirectory=/home/ubuntu
Restart=never
ExecStart=/home/ubuntu/boot.sh

[Install]
WantedBy=cloud-init.target
EOF
sudo systemctl enable show;

cat <<EOF > /home/ubuntu/status.sh;
#!/bin/bash
journalctl -u show.service -b -a
EOF
chmod a+x /home/ubuntu/status.sh;

cat <<EOF > /home/ubuntu/status-feed.sh;
#!/bin/bash
journalctl -u show.service -b -a -f
EOF
chmod a+x /home/ubuntu/status-feed.sh;

cat <<EOF > /home/ubuntu/monitor.sh;
#!/bin/bash
tmux new-session  'htop' \; split-window './status-feed.sh'
EOF
chmod a+x /home/ubuntu/monitor.sh;
{
    echo -e "./status.sh\n"
    echo -e "./status-feed.sh\n"
    echo -e "./monitor.sh\n"
    echo -e "tmux new-session 'htop' \\; split-window -l 20 './status-feed.sh' \\; split-window -hl 80 'nvidia-smi -l 1'\n"
    echo -e "tmux new-session 'htop' \\; split-window -l 20 './status-feed.sh' \\; split-window -hl 80 'sudo iftop'"
} > /home/ubuntu/.bash_history


docker pull uehreka/stylevision:latest

uptime