module "cloudflare" {
  source      = "../cloudflare"
  api_token   = var.cloudflare_api_token
  zone_id     = var.cloudflare_zone_id
  domain_name = var.domain_name
}

module "linode" {
  source     = "../linode"
  api_token  = var.linode_api_token
  ip_address = var.ip_address
}
