# Stylevision

Realtime Style transfer using WebRTC, ffmpeg and Tensorflow.

## Local Installation

(TODO, but in short, you'll need nvidia-docker, and if you're on Windows you'll need the latest Windows Insider version, bleeding edge Nvidia drivers, WSL2, Docker and Nvidia Docker. Tutorial to come.)

## Using with cloud VMs

If you don't have an Nvidia GPU, you won't be able to do local development with this repo, but you can still mess around by deploying to an EC2 GPU VM and running things from there. Unfortunately though, WebRTC can only run over HTTPS (when not on localhost), so you'll need a domain name, DNS, SSL certificates, the whole shebang. I've got a Terraform/packer workflow that I use a lot when building WebRTC systems that can manage all of that. It takes a bit of setup to get started, but is pretty smooth sailing once that's done.

1. **Get a suitable computer** All these scripts and such are built to run on either Linux, macOS or Windows Subsystem for Linux. Don't try and do this on Windows proper.
2. **Buy a domain and point the nameservers at Cloudflare** This will probably be under "Advanced DNS Settings" or something. Whenever I've used Cloudflare, the nameservers they give me are `liv.ns.cloudflare.com` and `scott.ns.cloudflare.com` although they do have others. After you do this, it can take **up to 48 hours** for the DNS system to start using the new nameservers (although sometimes it only takes an hour or two). So I'd advise you to **do this first**, then take care of the other things while that's happening.
3. **Set up AWS** Get an AWS account. Install the AWS CLI for your OS and log in with your credentials.
4. **Set up Cloudflare** Sign up for a free account. Create an API Token. When assigning permissions, make sure to add the "Zone" + "Edit" permission under the "Zone" category.
5. **Install the other tools** You'll need Terraform and Packer (both free, both from Hashicorp). You may also want Docker and Docker Compose if you're going to use custom containers, but if you're on WSL1 (which doesn't support Docker well) you can skip it.
6. **Fill out the tfvars file** By this point you should have all the keys and such that you'll need. Make a copy of the `terraform.template.tfvars` file, call it `terraform.tfvars` (it will be gitignored because you're about to enter sensitive information into it) and fill in the keys and values you got during the previous steps.
7. **Build an AMI with Packer** Go into `./packer` and run `./build.sh (whatever shortname you put in the tfvars file)`. This will build the AMI you'll be using. This process takes about half an hour, because it pulls down the stylevision docker image so that the layers are cached when your VM starts.
8. **Run terraform once to set it up** In `./terraform`, run the command `terraform apply -auto-approve`. This will set up Cloudflare and an S3 bucket (for storing networks and SSL certs).
9. **Upload the pretrained networks to S3** Log into AWS, find the `(shortname)-secret` bucket and upload the folder `./pretrained-networks` to that bucket (the whole folder itself, not just the contents).
10. **Check if your nameservers are resolving yet** Run the command `dig @8.8.8.8 +short NS YOUR.DOMAIN` and make sure it's pointing at the new nameservers. Wait until it is before moving to step 11.
11. **Fetch the SSL certificates** In `./terraform`, run the command `terraform apply -auto-approve -var run_cert_service=true` to start a VM that will fetch SSL certificates. Then run `terraform apply -auto-approve` to shut it down.
12. **Fire it up** There is an appropriately named script in `./terraform` called `./fireitup.sh` that starts a VM (or replaces it if you've changed anything in your `terraform.tfvars`) and then SSH's into it. Once you've SSH'd in, tap up-arrow on the keyboard, all the handy monitoring commands are already in the bash history. `./status-feed.sh` should be good for monitoring what's going on.
13. **Try it out!** When the words "IN PRODUCTION" show up in the logs, you're good to go. Visit https://show.(your domain name)/camera.html on a device that has a camera, and https://show.(your domain name)/projector.html to view the output.