provider "github" {
  owner = var.github_owner
}

data "github_repository" "this" {
  name = var.github_repository
}

resource "github_actions_secret" "gpg_private_key" {
  repository  = data.github_repository.this.name
  secret_name = "GPG_PRIVATE_KEY"
  value       = var.gpg_private_key
}

resource "github_actions_secret" "gpg_passphrase" {
  repository  = data.github_repository.this.name
  secret_name = "PASSPHRASE"
  value       = var.gpg_passphrase
}
