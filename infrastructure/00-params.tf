# parms file for aws ec2 cloud

#### VPC Network
variable "vpc_cidr" {
  type    = string
  default = "192.168.0.0/16"
}

#### HTTP PARAMS
variable "network_http" {
  type = map(string)
  default = {
    subnet_name = "subnet_http"
    cidr        = "192.168.1.0/24"
  }
}

# Set number of instance
variable "http_instance_names" {
  type    = set(string)
  default = ["instance-http-1", "instance-http-2"]
}

#### DB PARAMS
variable "network_db" {
  type = map(string)
  default = {
    subnet_name = "subnet_db"
    cidr        = "192.168.2.0/24"
  }
}

# Set number of instance
variable "db_instance_names" {
  type    = set(string)
  default = ["instance-db-1", "instance-db-2", "instance-db-3"]
}

# Set number of instance
variable "public_key" {
  type        = string
  description = "SSH public key to login into EC2 instance"
  default     = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD3F6tyPEFEzV0LX3X8BsXdMsQz1x2cEikKDEY0aIj41qgxMCP/iteneqXSIFZBp5vizPvaoIR3Um9xK7PGoW8giupGn+EPuxIA4cDM4vzOqOkiMPhz5XK0whEjkVzTo4+S0puvDZuwIsdiW9mxhJc7tgBNL0cYlWSYVkz4G/fslNfRPW5mYAM49f4fhtxPb5ok4Q2Lg9dPKVHO/Bgeu5woMc7RY0p1ej6D4CKFE6lymSDJpW0YHX/wqE9+cfEauh7xZcG0q9t2ta6F6fmX0agvpFyZo8aFbXeUBr7osSCJNgvavWbM/06niWrOvYX2xwWdhXmXSrbX8ZbabVohBK41 email@example.com"
}
