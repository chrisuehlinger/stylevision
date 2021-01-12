#!/bin/bash -ex
short_name="${1:-darger}"

old_ami_id=$(aws ec2 describe-images --owners self | jq -r ".Images | map(select(.Name == \"${short_name}-amd64\")) | .[0].ImageId")
echo "$old_ami_id"
if [[ $old_ami_id != 'null' ]]; then
  old_snapshot_id=$(aws ec2 describe-images --owners self | jq -r ".Images | map(select(.Name == \"${short_name}-amd64\")) | .[0].BlockDeviceMappings[0].Ebs.SnapshotId")
  aws ec2 deregister-image --image-id "$old_ami_id"
  aws ec2 delete-snapshot --snapshot-id "$old_snapshot_id"
fi
packer build -var "short_name=${short_name}" image.json 