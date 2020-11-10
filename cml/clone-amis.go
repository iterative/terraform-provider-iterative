//usr/bin/env go run $0 "$@"; exit
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

func main() {
	region := "us-west-1"
	amiName := "iterative-cml"
	regions := []string{"us-east-1", "us-east-2", "us-west-2", "eu-central-1", "eu-west-1"}

	sess, sessError := session.NewSession(&aws.Config{
		Region: aws.String(region)},
	)
	if sessError != nil {
		log.Printf("[ERROR] %s", sessError)
		os.Exit(1)
	}

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
	if len(imagesRes.Images) == 0 {
		log.Printf("[ERROR] ami %s not found", amiName)
		os.Exit(1)
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

		copyResult, err := svc.CopyImage(&ec2.CopyImageInput{
			SourceImageId: aws.String(amiID),
			SourceRegion:  aws.String(region),
			Name:          aws.String(amiName),
			Description:   aws.String(amiDesc),
		})
		if err != nil {
			fmt.Println(err)
		}

		svc.WaitUntilImageExists(&ec2.DescribeImagesInput{
			ImageIds: []*string{aws.String(*copyResult.ImageId)},
			Filters: []*ec2.Filter{
				{
					Name:   aws.String("state"),
					Values: []*string{aws.String("available")},
				},
			},
		})

		_, modifyErr := svc.ModifyImageAttribute(&ec2.ModifyImageAttributeInput{
			ImageId: aws.String(*copyResult.ImageId),
			LaunchPermission: &ec2.LaunchPermissionModifications{
				Add: []*ec2.LaunchPermission{
					{
						Group: aws.String("all"),
					},
				},
			},
		})
		if modifyErr != nil {
			fmt.Println(modifyErr)
		}
	}
}
