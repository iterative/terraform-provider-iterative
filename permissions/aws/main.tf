terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.3.0"
    }
  }
}

provider "aws" {
  region = "us-west-1"
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

data "aws_region" "current" {}

data "aws_partition" "current" {}

data "aws_caller_identity" "current" {}

data "aws_iam_policy_document" "task" {
  statement {
    actions = [
      "autoscaling:DescribeAutoScalingGroups",
      "autoscaling:DescribeScalingActivities",
    ]
    resources = ["*"]
  }

  statement {
    actions = [
      "autoscaling:CreateAutoScalingGroup",
      "autoscaling:UpdateAutoScalingGroup",
      "autoscaling:DeleteAutoScalingGroup",
    ]
    resources = [
"arn:${data.aws_partition.current.partition}:autoscaling:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:autoScalingGroup:*:autoScalingGroupName/tpi-*",
    ]
  }

  statement {
    actions = [
      "ec2:DescribeAutoScalingGroups",
      "ec2:DescribeImages",
      "ec2:DescribeInstances",
      "ec2:DescribeKeyPairs",
      "ec2:DescribeLaunchTemplates",
      "ec2:DescribeScalingActivities",
      "ec2:DescribeSecurityGroups",
      "ec2:DescribeSubnets",
      "ec2:DescribeVpcs",
    ]
    resources = ["*"]
  }

  statement {
    actions = [
      "ec2:CreateSecurityGroup",
      "ec2:DeleteSecurityGroup",
      "ec2:RevokeSecurityGroupEgress",
      "ec2:RevokeSecurityGroupIngress",
      "ec2:AuthorizeSecurityGroupEgress",
      "ec2:AuthorizeSecurityGroupIngress",
    ]
    resources = ["*"]
  }

  statement {
    actions = [
      "ec2:ImportKeyPair",
      "ec2:DeleteKeyPair",
    ]
    resources = ["*"]
  }

  statement {
    actions = [
      "ec2:CreateLaunchTemplate",
      "ec2:GetLaunchTemplateData",
      "ec2:ModifyLaunchTemplate",
      "ec2:DeleteLaunchTemplate",
    ]
    resources = ["*"]
  }

  statement {
    actions = [
      "ec2:RunInstances",
      "ec2:TerminateInstances",
      "ec2:*"
    ]
    resources = [
      "*",
    ]
  }


  statement {
    actions = [
      "ec2:CreateTags"
    ]
    resources = [
      "*",
    ]
  }

  statement {
    actions = [
      "s3:ListBucket",
      "s3:CreateBucket",
      "s3:DeleteBucket",
      "s3:GetObject",
      "s3:PutObject",
      "s3:DeleteObject",
    ]
    resources = [
      "arn:${data.aws_partition.current.partition}:s3:::tpi-*",
    ]
  }
}

output "aws_access_key_id" {
  value = aws_iam_access_key.task.id
}

output "aws_secret_access_key" {
  value     = aws_iam_access_key.task.secret
  sensitive = true
}