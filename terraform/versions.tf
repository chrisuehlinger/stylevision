terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
    }
    cloudflare = {
      source = "cloudflare/cloudflare"
    }
    template = {
      source = "hashicorp/template"
    }
  }
  required_version = ">= 0.14"
}
