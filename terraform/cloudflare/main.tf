terraform {
  required_providers {
    cloudflare = {
      source                = "cloudflare/cloudflare"
      version               = "~> 3.0"
      configuration_aliases = [cloudflare.abomination]
    }
    random = {
      source  = "hashicorp/random"
      version = "3.4.3"
    }
  }
}

variable "account_id" {
  type        = string
  description = "Cloudflare account ID"
}

variable "api_token" {
  description = "Cloudflare API token"
  type        = string
  sensitive   = true
}

variable "origin_ca_key" {
  description = "Cloudflare origin CA key, found at the bottom of the API tokens page"
  type        = string
  sensitive   = true
}

variable "domain_name" {
  description = "Domain name for the website (e.g. example.com)"
  type        = string
}

variable "server_ipv4" {
  description = "Server IP address"
  type        = string
}

variable "cert_request_pem" {
  description = "Certificate request PEM for cloudflare origin CA certificate to enable authenticated origin pulls"
  type        = string
}

provider "cloudflare" {
  api_token = var.api_token
}

provider "cloudflare" {
  alias                = "abomination"
  api_user_service_key = var.origin_ca_key
}

resource "cloudflare_zone" "zone" {
  account_id = var.account_id
  zone       = var.domain_name
}


resource "cloudflare_pages_domain" "apex_domain" {
  account_id   = var.account_id
  project_name = cloudflare_pages_project.pages_project.name
  domain       = var.domain_name
}

resource "random_pet" "page_project_name" {
}

resource "cloudflare_pages_project" "pages_project" {
  account_id        = var.account_id
  name              = random_pet.page_project_name.id
  production_branch = "main"
  lifecycle {
    replace_triggered_by = [
      random_pet.page_project_name
    ]
  }
}

resource "cloudflare_record" "root" {
  zone_id = cloudflare_zone.zone.id
  name    = var.domain_name
  value   = cloudflare_pages_project.pages_project.subdomain
  type    = "CNAME"
  proxied = true
}

resource "cloudflare_record" "api" {
  zone_id = cloudflare_zone.zone.id
  name    = "api"
  value   = var.server_ipv4
  type    = "A"
  proxied = true
}

resource "cloudflare_origin_ca_certificate" "origin_certificate" {
  provider           = cloudflare.abomination
  hostnames          = [var.domain_name, "*.${var.domain_name}"]
  request_type       = "origin-rsa"
  requested_validity = "5475"
  csr                = var.cert_request_pem
}

resource "cloudflare_zone_settings_override" "site-settings" {
  zone_id = cloudflare_zone.zone.id
  settings {
    ssl                      = "strict"
    automatic_https_rewrites = "on"
    tls_client_auth          = "on"
  }
}

resource "cloudflare_authenticated_origin_pulls" "aop" {
  zone_id = cloudflare_zone.zone.id
  enabled = true
}

output "cloudflare_origin_ca_certificate" {
  description = "Cloudflare's origin certificate"
  value       = cloudflare_origin_ca_certificate.origin_certificate.certificate
}

output "cloudflare_account_id" {
  description = "Cloudflare account ID"
  value       = cloudflare_zone.zone.account_id
}
