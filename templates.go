package main

var doTemplate = `
terraform {
  required_providers {
    digitalocean = {
      source = "digitalocean/digitalocean"
      version = "1.22.2"
    }
  }
}

variable "do_token" {}
variable "pvt_key" {}
variable "do_ssh_fingerprint" {}

provider "digitalocean" {
  token = "${var.do_token}"
}

{{range .}}
resource "digitalocean_droplet" "{{.Name}}" {
  image = "ubuntu-22-04-x64"
  name = "{{.Name}}"
  region = "{{.Region}}"
  size = "s-1vcpu-1gb"
  private_networking = true
  ssh_keys = [
    "${var.do_ssh_fingerprint}"
  ]
  connection {
    host = self.ipv4_address
    user = "root"
    type = "ssh"
    private_key = file("${var.pvt_key}")
    timeout = "2m"
  }
}

output "{{.Name}}-IP" {
    value = "${digitalocean_droplet.{{.Name}}.ipv4_address}"
}

{{end}}
`

var awsTemplate = `
variable "aws_access_key" {}
variable "aws_secret_key" {}

variable "aws_key_name" {}

# Configure the AWS Provider
provider "aws" {
  access_key = "${var.aws_access_key}"
  secret_key = "${var.aws_secret_key}"
  alias      = "us-east-1"
  region     = "us-east-1"
}

provider "aws" {
  access_key = "${var.aws_access_key}"
  secret_key = "${var.aws_secret_key}"
  alias      = "us-east-2"
  region     = "us-east-2"
}

provider "aws" {
  access_key = "${var.aws_access_key}"
  secret_key = "${var.aws_secret_key}"
  alias      = "us-west-1"
  region     = "us-west-1"
}

provider "aws" {
  access_key = "${var.aws_access_key}"
  secret_key = "${var.aws_secret_key}"
  alias      = "us-west-2"
  region     = "us-west-2"
}

provider "aws" {
  access_key = "${var.aws_access_key}"
  secret_key = "${var.aws_secret_key}"
  alias      = "ap-south-1"
  region     = "ap-south-1"
}

provider "aws" {
  access_key = "${var.aws_access_key}"
  secret_key = "${var.aws_secret_key}"
  alias      = "ap-northeast-1"
  region     = "ap-northeast-1"
}

provider "aws" {
  access_key = "${var.aws_access_key}"
  secret_key = "${var.aws_secret_key}"
  alias      = "ap-northeast-2"
  region     = "ap-northeast-2"
}

provider "aws" {
  access_key = "${var.aws_access_key}"
  secret_key = "${var.aws_secret_key}"
  alias      = "ap-southeast-1"
  region     = "ap-southeast-1"
}

provider "aws" {
  access_key = "${var.aws_access_key}"
  secret_key = "${var.aws_secret_key}"
  alias      = "ap-southeast-2"
  region     = "ap-southeast-2"
}

provider "aws" {
  access_key = "${var.aws_access_key}"
  secret_key = "${var.aws_secret_key}"
  alias      = "ca-central-1"
  region     = "ca-central-1"
}

provider "aws" {
  access_key = "${var.aws_access_key}"
  secret_key = "${var.aws_secret_key}"
  alias      = "eu-central-1"
  region     = "eu-central-1"
}

provider "aws" {
  access_key = "${var.aws_access_key}"
  secret_key = "${var.aws_secret_key}"
  alias      = "eu-west-1"
  region     = "eu-west-1"
}

provider "aws" {
  access_key = "${var.aws_access_key}"
  secret_key = "${var.aws_secret_key}"
  alias      = "eu-west-2"
  region     = "eu-west-2"
}

provider "aws" {
  access_key = "${var.aws_access_key}"
  secret_key = "${var.aws_secret_key}"
  alias      = "eu-west-3"
  region     = "eu-west-3"
}

provider "aws" {
  access_key = "${var.aws_access_key}"
  secret_key = "${var.aws_secret_key}"
  alias      = "sa-east-1"
  region     = "sa-east-1"
}

{{range $region, $name := .AMI}}
data "aws_ami" "{{$name}}" {
  provider = "aws.{{$region}}"

  most_recent = true

  filter {
    name   = "name"
    values = ["amzn2-ami-hvm-2.0.2018*"]
  }

  filter {
    name   = "architecture"
    values = ["x86_64"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }

  filter {
    name   = "root-device-type"
    values = ["ebs"]
  }

  owners = ["amazon"]
}
{{end}}

{{range $region, $name := .SecurityGroup}}
resource "aws_security_group" "{{$name}}" {
  provider = "aws.{{$region}}"

  name        = "{{$name}}"
  description = "Allow all inbound traffic"

  ingress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port       = 0
    to_port         = 0
    protocol        = "-1"
    cidr_blocks     = ["0.0.0.0/0"]
  }
}
{{end}}

{{range .Instances}}
resource "aws_instance" "{{.Name}}" {
  provider = "aws.{{.Region}}"
  ami           = "${data.aws_ami.{{.AMI}}.id}"
  instance_type = "t2.micro"
  key_name = "${var.aws_key_name}"
  security_groups = ["${aws_security_group.{{.SecurityGroup}}.name}"]
}

output "{{.Name}}-IP" {
	value = "${aws_instance.{{.Name}}.public_ip}"
}
{{end}}
`
