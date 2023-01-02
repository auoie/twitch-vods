terraform {
  required_providers {
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 3.0"
    }
  }
}

variable "api_token" {
  description = "Cloudflare API token"
  type        = string
  sensitive   = true
}

variable "domain_name" {
  description = "Domain name for the website (e.g. example.com)"
  type        = string
}

variable "zone_id" {
  description = "Zone ID associated with your account for API operations"
  type = string
}

provider "cloudflare" {
  api_token = var.api_token
}

resource "cloudflare_record" "www" {
  zone_id = var.zone_id
  name    = "www"
  value   = var.domain_name
  type    = "CNAME"
  proxied = true
}

resource "cloudflare_record" "root" {
  zone_id = var.zone_id
  name    = var.domain_name
  value   = "203.0.113.10"
  type    = "A"
  proxied = true
}
