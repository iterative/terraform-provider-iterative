//usr/bin/env go run $0 "$@"; exit
package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

func main() {
	region := "us-west-1"
	amiName := "iterative-cml"
	regions := []string{"us-east-1", "eu-central-1", "eu-west-1"}

	sess, _ := session.NewSession(&aws.Config{
		Region: aws.String(region)},
	)
	svc := ec2.New(sess)

	amiParams := &ec2.DescribeImagesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("name"),
				Values: []*string{aws.String(amiName)},
			},
			{
				Name:   aws.String("architecture"),
				Values: []*string{aws.String("x86_64")},
			},
		},
	}
	imagesRes, imagesErr := svc.DescribeImages(amiParams)
	if imagesErr != nil {
		diag.FromErr(imagesErr)
	}

	ami := imagesRes.Images[0]
	amiID := *ami.ImageId
	amiDesc := *ami.Description

	for _, value := range regions {
		fmt.Println("Cloning", value)

		sess, _ := session.NewSession(&aws.Config{
			Region: aws.String(value)},
		)

		svc := ec2.New(sess)

		_, err := svc.CopyImage(&ec2.CopyImageInput{
			SourceImageId: aws.String(amiID),
			SourceRegion:  aws.String(region),
			Name:          aws.String(amiName),
			Description:   aws.String(amiDesc),
		})

		if err != nil {
			fmt.Println(err)
		}
	}
}
