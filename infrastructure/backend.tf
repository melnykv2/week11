terraform {
  backend "s3" {
    key          = "week11/terraform.tfstate"
    region       = "eu-north-1"
    use_lockfile = true
    encrypt      = true
  }
}
