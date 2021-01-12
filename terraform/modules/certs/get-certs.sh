#!/bin/bash -ex

sudo add-apt-repository universe
sudo add-apt-repository ppa:certbot/certbot -y
sudo apt-get update
sudo apt-get install -y s3fs certbot

sudo mkdir -p /mnt/secret
# sudo chown 1000:1000 /mnt/secret
sudo s3fs "${short_name}-secret" -o iam_role="${short_name}-certs-role" /mnt/secret


sudo certbot certonly --standalone -m "${lets_encrypt_email}" -n --agree-tos --duplicate \
    -d "show.${domain_name}"

sudo cp -R /etc/letsencrypt /mnt/secret/letsencrypt