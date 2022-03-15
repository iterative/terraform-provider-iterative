terraform {
  required_providers {
    aws = { source = "hashicorp/aws", version = "~> 4.3.0" }
  }
}

variable "aws_region" {
  description = "Name of the Amazon Web Services region to use"
}

provider "aws" {
  region = var.aws_region
}

resource "aws_iam_user" "task" {
  name = "task"
}

resource "aws_iam_access_key" "task" {
  user = aws_iam_user.task.name
}

resource "aws_iam_user_policy" "task" {
  name   = aws_iam_user.task.name
  user   = aws_iam_user.task.name
  policy = data.aws_iam_policy_document.task.json
}

data "aws_iam_policy_document" "task" {
  statement {
    actions = [
      "autoscaling:CreateAutoScalingGroup",
      "autoscaling:DeleteAutoScalingGroup",
      "autoscaling:DescribeAutoScalingGroups",
      "autoscaling:DescribeScalingActivities",
      "autoscaling:UpdateAutoScalingGroup",
      "ec2:AuthorizeSecurityGroupEgress",
      "ec2:AuthorizeSecurityGroupIngress",
      "ec2:CancelSpotInstanceRequests",
      "ec2:CreateKeyPair",
      "ec2:CreateLaunchTemplate",
      "ec2:CreateSecurityGroup",
      "ec2:CreateTags",
      "ec2:DeleteKeyPair",
      "ec2:DeleteLaunchTemplate",
      "ec2:DeleteSecurityGroup",
      "ec2:DescribeAutoScalingGroups",
      "ec2:DescribeImages",
      "ec2:DescribeInstances",
      "ec2:DescribeKeyPairs",
      "ec2:DescribeLaunchTemplates",
      "ec2:DescribeScalingActivities",
      "ec2:DescribeSecurityGroups",
      "ec2:DescribeSpotInstanceRequests",
      "ec2:DescribeSubnets",
      "ec2:DescribeVpcs",
      "ec2:GetLaunchTemplateData",
      "ec2:ImportKeyPair",
      "ec2:ModifyImageAttribute",
      "ec2:ModifyLaunchTemplate",
      "ec2:RequestSpotInstances",
      "ec2:RevokeSecurityGroupEgress",
      "ec2:RevokeSecurityGroupIngress",
      "ec2:RunInstances",
      "ec2:TerminateInstances",
      "s3:CreateBucket",
      "s3:DeleteBucket",
      "s3:DeleteObject",
      "s3:GetObject",
      "s3:ListBucket",
      "s3:PutObject",
    ]
    resources = ["*"]
  }
}

output "aws_access_key_id" {
  value = aws_iam_access_key.task.id
}

output "aws_secret_access_key" {
  value     = aws_iam_access_key.task.secret
  sensitive = true
}
