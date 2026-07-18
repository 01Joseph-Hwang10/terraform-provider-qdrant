variable "github_owner" {
  description = "The owner of the GitHub repository"
  type        = string
}

variable "github_repository" {
  description = "The name of the GitHub repository"
  type        = string
  default     = "terraform-provider-qdrant"
}

variable "gpg_private_key" {
  description = "GPG private key for signing releases"
  type        = string
  sensitive   = true
}

variable "gpg_passphrase" {
  description = "Passphrase for the GPG private key"
  type        = string
  sensitive   = true
}

