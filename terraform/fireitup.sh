#!/bin/bash -ex

terraform apply -auto-approve -var run_show=true
sleep 10
ssh -oStrictHostKeyChecking=no -oConnectionAttempts=10 "ubuntu@$(terraform output -raw show_service_ip)"