# Stylevision

Realtime Style transfer using WebRTC, ffmpeg and Tensorflow.

## Local Installation

You'll need an Nvidia graphics card (and if you want to use the Tensor Core networks, you'll need an RTX card specifically). This is because Tensorflow uses CUDA to execute neural networks really fast, and CUDA is an Nvidia framework that only works on their graphics cards. In theory Tensorflow can also run on OpenCL (which means AMD and possible Intel GPUs), and I'm down to consider adding an OpenCL flag if someone submits a PR that gets it working, but my hunch is that it would be a lot slower.

But don't despair! You can still deploy this thing to the cloud and tinker with it there! (See instructions below)

### Linux

(Note: I haven't tested this myself yet, but given that this is basically a subset of the WSL2 instructions, I expect it will work. File an issue if it doesn't and I'll take a closer look.)

To start you'll need to be using the Nvidia drivers, not the FOSS Nouveau drivers. The instructions for this vary from distro-to-distro, but if you're starting from scratch [Pop!_OS](https://pop.system76.com/) has a version that comes with them pre-installed, which is nice.

1. **Install Nvidia Docker** - Follow [Nvidia's instructions](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/install-guide.html) to install the latest version of nvidia-docker. Those instructions also include steps to install docker itself, if you don't have it already.
2. **Run ./build.sh** - This will build the docker image you'll be using. If it's your first time building the image it'll take up to half an hour on most hardware, but after that if you change something it'll probably only take a second or two.
3. **Run ./run.sh** - This will spin up the docker container and start running the system.
4. **Try it out** - When the words "IN DEVELOPMENT" show up in the logs, you're good to go. Visit https://localhost:8080/camera.html to capture input from your webcam, and https://localhost:8080/projector.html to view the output.

### Windows

This is gonna be a little rough, and these instructions may eventually become obsolete, but this is how I set up my local dev environment.

1. **Install WSL2** - Here are [Microsoft's instructions](https://docs.microsoft.com/en-us/windows/wsl/install-win10).
2. **Install nvidia-docker on WSL2** - Here's [a tutorial](https://medium.com/@dalgibbard/docker-with-gpu-support-in-wsl2-ebbc94251cf5) that worked for me. You'll need the most bleeding edge Windows 10 Insiders Beta, some beta Nvidia drivers, and a version of Docker thats different from the one most WSL Docker tutorials use. If you're not starting from a clean WSL2 install, be careful.
3. **Run ./build.sh** - This will build the docker image you'll be using. If it's your first time building the image it'll take up to half an hour on most hardware, but after that if you change something it'll probably only take a second or two.
4. **Run ./run.sh** - This will spin up the docker container and start running the system.
5. **Try it out** - When the words "IN DEVELOPMENT" show up in the logs, you're good to go. Visit https://localhost:8080/camera.html to capture input from your webcam, and https://localhost:8080/projector.html to view the output.

Due to issues with WSL2 and the Windows networking stack, you may run into difficulty if you try and access Stylevision from another computer in your local network. Hopefully this will improve in the future.

### Things to play around with

The `./run.sh` script has a bunch of settings you can tweak by commenting/uncommenting/editing the lines with environment variables. You can:

- change which of the `./pretrained-networks` is used for the style transfer (`$NETWORK_NAME`)
- turn style transfer on/off so you can test the other parts of the pipeline (`$PERFORM_TRANSFER`)
- change the expected height and width of the footage (these values will be made availabe to camera.html so it can set them as `getUserMedia` constraints) (`$FRAME_WIDTH` and `$FRAME_HEIGHT`)
- change whether to use the version of the network that makes use of Tensor Core (`$MODEL_VERSION`, set it to "optimized" to use the best non-Tensor Core model, or "trtfp16" to use the half-precision Tensor Core model, in the future we may have "trtint8" models, but I haven't gotten them to work yet)

### Training

You'll need a huge training set of images. The one I used has ~80,000 images, but I can't remember where I got it or if I'm allowed to release it. Nevertheless, if you can find a big folder full of images, `./run-prepare-training-images.sh` will crop them to the right size and leave them in a gitignored folder, and `./copy-images.sh` will copy over a selection of them into a "staging folder" in case you only want to use a small slice of them.

To train a network, use `./run-training.sh networkname numepochs`, where networkname is the name of an image in `./examples` (and will also be the name of the folder where the network gets stored) and numepochs is an integer number of epochs.

I usually go for 5 epochs with my 80,000 image training set, and on an RTX 2070 Max-Q that takes about six hours.

## Using with cloud VMs

If you don't have an Nvidia GPU, you won't be able to do local development with this repo, but you can still mess around by deploying to an EC2 GPU VM and running things from there. Unfortunately though, WebRTC can only run over HTTPS (when not on localhost), so you'll need a domain name, DNS, SSL certificates, the whole shebang. I've got a Terraform/packer workflow that I use a lot when building WebRTC systems that can manage all of that. It takes a bit of setup to get started, but is pretty smooth sailing once that's done.

Note: All these bash scripts and such are built to run on either Linux, macOS or Windows Subsystem for Linux. Don't try and do this on Windows proper.

1. **Buy a domain and point the nameservers at Cloudflare** - This will probably be under "Advanced DNS Settings" or something. Whenever I've used Cloudflare, the nameservers they give me are `liv.ns.cloudflare.com` and `scott.ns.cloudflare.com` although they do have others. After you do this, it can take **up to 48 hours** for the DNS system to start using the new nameservers (although sometimes it only takes an hour or two). So I'd advise you to **do this first**, then take care of the other things while that's happening.
2. **Set up AWS** - Get an AWS account. Install the AWS CLI for your OS and log in with your credentials.
3. **Set up Cloudflare** - Sign up for a free account. Create an API Token. When assigning permissions, make sure to add the "Zone" + "Edit" permission under the "Zone" category.
4. **Install the other tools** - You'll need Terraform and Packer (both free, both from Hashicorp). You may also want Docker and Docker Compose if you're going to use custom containers, but if you're on WSL1 (which doesn't support Docker well) you can skip it.
5. **Fill out the tfvars file** - By this point you should have all the keys and such that you'll need. Make a copy of the `terraform.template.tfvars` file, call it `terraform.tfvars` (it will be gitignored because you're about to enter sensitive information into it) and fill in the keys and values you got during the previous steps.
6. **Build an AMI with Packer** - Go into `./packer` and run `./build.sh (whatever shortname you put in the tfvars file)`. This will build the AMI you'll be using. This process takes about half an hour, because it pulls down the stylevision docker image so that the layers are cached when your VM starts.
7. **Run terraform once to set it up** - In `./terraform`, run the command `terraform apply -auto-approve`. This will set up Cloudflare and an S3 bucket (for storing networks and SSL certs).
8. **Upload the pretrained networks to S3** - Log into AWS, find the `(shortname)-secret` bucket and upload the folder `./pretrained-networks` to that bucket (the whole folder itself, not just the contents).
9. **Check if your nameservers are resolving yet** - Run the command `dig @8.8.8.8 +short NS YOUR.DOMAIN` and make sure it's pointing at the new nameservers. Wait until it is before moving to step 10.
10. **Fetch the SSL certificates** - In `./terraform`, run the command `terraform apply -auto-approve -var run_cert_service=true` to start a VM that will fetch SSL certificates. Then run `terraform apply -auto-approve` to shut it down.
11. **Fire it up** - There is an appropriately named script in `./terraform` called `./fireitup.sh` that starts a VM (or replaces it if you've changed anything in your `terraform.tfvars`) and then SSH's into it. Once you've SSH'd in, tap up-arrow on the keyboard, all the handy monitoring commands are already in the bash history. `./status-feed.sh` should be good for monitoring what's going on.
12. **Try it out!** - When the words "IN PRODUCTION" show up in the logs, you're good to go. Visit https://show.(your-domain-name)/camera.html on a device that has a camera, and https://show.(your-domain-name)/projector.html to view the output.