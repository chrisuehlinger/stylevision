variable "aws_region" {
  description = "AWS region to launch servers."
}

variable "ssh_public_key" {
  description = "The public key you'll use to access the resources"
}

variable "short_name" {
  description = "The name of the show."
}

variable "domain_name" {
  description = "The domain name for your show."
}

variable "cloudflare_api_key" {
  description = "API Key from Cloudflare"
}

variable "lets_encrypt_email" {
  description = "Email address to use with LetsEncrypt"
}

variable "run_cert_service" {
  type    = bool
  default = false
}

variable "run_show" {
  type    = bool
  default = false
}

variable "instance_size" {
  default = "t4g.micro"
}

variable "use_spot" {
  type    = bool
  default = false
}

variable "network_name" {
}

variable "model_version" {
}

variable "frame_width" {
}

variable "frame_height" {
}

variable "perform_transfer" {
}