variable "cloudflare_api_token" {
  description = "Cloudflare API token"
  type        = string
  sensitive   = true
}

variable "domain_name" {
  description = "Domain name for the website (e.g. example.com)"
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Zone ID associated with your account for API operations"
  type        = string
}

variable "linode_api_token" {
  description = "Linode API token "
  type        = string
  sensitive   = true
}

variable "ip_address" {
  description = "My IP address for SSH firewall settings"
  type        = string
  sensitive   = true
}
