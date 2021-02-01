aws_region        = "us-east-1"
ssh_public_key    = "" // Make an SSH key pair, put the text of the public key here
short_name = "darger" // a single word name that will be used as a prefix when creating resources
domain_name = "" // The domain name you purchased for your show, e.g. dargervision.xyz
cloudflare_api_key = "" // Your Cloudflare API Token
lets_encrypt_email = "" // The email where you will be notified if your letsencrypt certs expire (no spam)
instance_size = "" // what size VM you want to use. (g4dn.xlarge works well, although if you've set perform_transfer to false and just want to test without the style transfer stuff, you don't need a GPU and even a t31.micro will do)
use_spot = true // Set this to true if you want to use EC2 Spot Instances. GPU's can get expensive, so I definitely recommend keeping this set to true
network_name = "candy" // This should be the name of a folder in /pretrained-networks, but with -network removed (e.g. "darger3" or "starry-night" would work)
model_version = "optimized" // Which version of the model to use. "constant" and "optimized" perform about the same. "trtfp16" takes a while to boot up, but runs about 2x as fast because it uses Tensor Cores
// For various tensorflowy reasons, we have to choose our resolution ahead of time. Whatever you put here, the HTML pages will find out about it and request it via getUserMedia
frame_width = 960
frame_height = 540
perform_transfer = "false" // Set this to "false" if you want to test the system without doing style transfer. Whether true or false, this has to be a string and not a boolean for, uh... reasons