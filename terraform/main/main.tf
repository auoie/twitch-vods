# Create a CSR and generate a CA certificate
resource "tls_private_key" "key" {
  algorithm = "RSA"
}

resource "tls_cert_request" "request" {
  private_key_pem = tls_private_key.key.private_key_pem
}

module "cloudflare" {
  source           = "../cloudflare"
  api_token        = var.cloudflare_api_token
  domain_name      = var.domain_name
  server_ipv4      = module.linode.server_ip4
  origin_ca_key    = var.cloudflare_origin_ca_key
  cert_request_pem = tls_cert_request.request.cert_request_pem
  account_id       = var.cloudflare_account_id
}

module "linode" {
  source     = "../linode"
  api_token  = var.linode_api_token
  ip_address = var.ip_address
}

output "certificate" {
  value = module.cloudflare.cloudflare_origin_ca_certificate
}

output "key" {
  sensitive = true
  value     = tls_cert_request.request.private_key_pem
}
