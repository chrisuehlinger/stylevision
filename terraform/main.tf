terraform {
  required_version = ">= 0.14"
}

provider "aws" {
  region = var.aws_region
}

provider "cloudflare" {
  email = "chris.uehlinger@gmail.com"
  api_token = var.cloudflare_api_key
}

resource "cloudflare_zone" "dns_zone" {
    zone = var.domain_name
}

resource "cloudflare_zone_settings_override" "show_cdn_settings" {
    zone_id = cloudflare_zone.dns_zone.id
    settings {
      ssl = "flexible"
    }
}

resource "aws_key_pair" "key_pair" {
  key_name   = "${var.short_name}-key"
  public_key = var.ssh_public_key
}

resource "aws_s3_bucket" "secrets" {
  bucket = "${var.short_name}-secret"
  acl    = "private"
  force_destroy = true
}

module "certs" {
  source = "./modules/certs"
  count = var.run_cert_service ? 1 : 0
  secrets_bucket_name = aws_s3_bucket.secrets.bucket
  short_name = var.short_name
  cloudflare_zone_id = cloudflare_zone.dns_zone.id
  domain_name = var.domain_name
  lets_encrypt_email = var.lets_encrypt_email
  ssh_key_pair = aws_key_pair.key_pair.key_name
  use_spot = var.use_spot
}

module "show_service" {
  source = "./modules/show"
  count = var.run_show ? 1 : 0
  secrets_bucket_name = aws_s3_bucket.secrets.bucket
  short_name = var.short_name
  cloudflare_zone_id = cloudflare_zone.dns_zone.id
  domain_name = var.domain_name
  ssh_key_pair = aws_key_pair.key_pair.key_name
  instance_size = var.instance_size
  use_spot = var.use_spot
  network_name = var.network_name
  model_version = var.model_version
  frame_width = var.frame_width
  frame_height = var.frame_height
  perform_transfer = var.perform_transfer
}