terraform {
  required_providers {
    linode = {
      source  = "linode/linode"
      version = "1.29.4"
    }
  }
}

variable "api_token" {
  type        = string
  sensitive   = true
  description = "API token for Linode"
}

variable "ip_address" {
  type        = string
  description = "My IP address for SSH firewall settings since in order to deploy the website I have to manually configure stuff in the server"
}

provider "linode" {
  token = var.api_token
}

resource "linode_firewall" "firewall" {
  label          = "instance-firewall"
  inbound_policy = "DROP"
  inbound {
    label    = "allow-ssh"
    action   = "ACCEPT"
    protocol = "TCP"
    ports    = "22"
    ipv4     = ["${var.ip_address}/32"]
  }
  outbound_policy = "ACCEPT"
  linodes         = [linode_instance.instance.id]
}

resource "linode_instance" "instance" {
  image           = "linode/ubuntu22.04"
  region          = "us-west"
  type            = "g6-nanode-1"
  authorized_keys = [trimspace(file("~/.ssh/id_ed25519.pub"))]
}
