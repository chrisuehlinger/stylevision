# Our default security group to access
# the instances over SSH and HTTP
resource "aws_security_group" "security_group" {
  name        = "${var.short_name}_service_sg"
  description = "Used in the terraform"

  # SSH access from anywhere
  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # HTTP access from anywhere
  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # HTTPS access from anywhere
  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # iperf3 access from anywhere
  ingress {
    from_port   = 5201
    to_port     = 5201
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # iperf3 access from anywhere
  ingress {
    from_port   = 5201
    to_port     = 5201
    protocol    = "udp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # RTP access from anywhere
  ingress {
    from_port   = 8000
    to_port     = 60000
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # RTP access from anywhere
  ingress {
    from_port   = 8000
    to_port     = 60000
    protocol    = "udp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # outbound internet access
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    DargervisionShow = var.short_name
    DargervisionResourceType = "sg"
  }
}

resource "aws_iam_role" "role" {
  name = "${var.short_name}-show-role"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF

  tags = {
    DargervisionShow = var.short_name
    DargervisionResourceType = "role"
  }
}

resource "aws_iam_instance_profile" "profile" {
  name = "${var.short_name}-show-profile"
  role = aws_iam_role.role.name
}

resource "aws_iam_role_policy" "policy" {
  name = "${var.short_name}-show-policy"
  role = aws_iam_role.role.id

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": [
        "s3:*"
      ],
      "Effect": "Allow",
      "Resource": [
          "arn:aws:s3:::${var.secrets_bucket_name}",
          "arn:aws:s3:::${var.secrets_bucket_name}/*"
      ]
    }
  ]
}
EOF
}

# data "aws_ami" "ami" {
#   owners      = ["898082745236"]
#   most_recent = true

#   filter {
#     name   = "name"
#     values = ["Deep Learning AMI (Ubuntu 18.04)*"]
#   }

#   filter {
#     name   = "root-device-type"
#     values = ["ebs"]
#   }
# }

data "aws_ami" "ami" {
  owners      = ["self"]
  most_recent = true

  filter {
    name   = "name"
    values = ["darger-amd64"]
  }
}

# data "template_file" "vm_install_script" {
#   template = file("${path.module}/vm-install.sh")

#   vars = {
#     short_name = var.short_name
#     network_name = "candy"
#     model_version = "optimized"
#     frame_width = "1920"
#     frame_height = "1080"
#   }
# }

data "template_file" "startup_script" {
  template = file("${path.module}/startup.sh")

  vars = {
    short_name = var.short_name
    network_name = var.network_name
    model_version = var.model_version
    frame_width = var.frame_width
    frame_height = var.frame_height
    perform_transfer = var.perform_transfer
  }
}

resource "aws_spot_instance_request" "service" {
  count = var.use_spot ? 1 : 0
  instance_type = var.instance_size 
  wait_for_fulfillment = true

  # Lookup the correct AMI based on the region
  # we specified
  ami = data.aws_ami.ami.image_id
  # ami = "ami-02e86b825fe559330"

  availability_zone = "us-east-1c"

  iam_instance_profile = aws_iam_instance_profile.profile.name

  # The name of our SSH keypair we created above.
  key_name = var.ssh_key_pair

  # Our Security group to allow HTTP and SSH access
  vpc_security_group_ids = [aws_security_group.security_group.id]

  user_data = data.template_file.startup_script.rendered

  tags = {
    DargervisionShow = var.short_name
    DargervisionResourceType = "vm"
  }
}

resource "aws_instance" "service" {
  count = var.use_spot ? 0 : 1
  instance_type = var.instance_size

  # Lookup the correct AMI based on the region
  # we specified
  ami = data.aws_ami.ami.image_id
  # ami = "ami-02e86b825fe559330"


  iam_instance_profile = aws_iam_instance_profile.profile.name

  # The name of our SSH keypair we created above.
  key_name = var.ssh_key_pair

  # Our Security group to allow HTTP and SSH access
  vpc_security_group_ids = [aws_security_group.security_group.id]

  user_data = data.template_file.startup_script.rendered

  tags = {
    DargervisionShow = var.short_name
    DargervisionResourceType = "vm"
  }
}

resource "cloudflare_record" "service" {
  zone_id = var.cloudflare_zone_id
  name    = "show.${var.domain_name}"
  type    = "A"
  ttl     = "60"
  value   = var.use_spot ? aws_spot_instance_request.service[0].public_ip : aws_instance.service[0].public_ip
  proxied = false
}