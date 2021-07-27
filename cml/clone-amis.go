//usr/bin/env go run $0 "$@"; exit
package main

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

func main() {
	region := "us-west-1"
	amiName := "iterative-cml"
	regions := []string{
		"us-east-2",
		"us-east-1",
		"us-west-2",
		"ap-south-1",
		"ap-northeast-3",
		"ap-northeast-2",
		"ap-southeast-1",
		"ap-southeast-2",
		"ap-northeast-1",
		"ca-central-1",
		"eu-central-1",
		"eu-west-1",
		"eu-west-2",
		"eu-west-3",
		"eu-north-1",
		"sa-east-1",
	}

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

	var wg sync.WaitGroup
	for _, destination := range regions {
		wg.Add(1)
		go func(destinationRegion string) {
			fmt.Printf("Cloning image to region %s...\n", destinationRegion)
			if err := cloneImage(region, destinationRegion, *ami.ImageId, *ami.Name, *ami.Description); err != nil {
				fmt.Printf("Error cloning image to region %s\n", destinationRegion)
				fmt.Println(err.Error())
				os.Exit(1)
			}
			fmt.Printf("Image successfully cloned to region %s\n", destinationRegion)
		}(destination)
	}
	wg.Wait()
}

func cloneImage(sourceRegion string, destinationRegion string, imageIdentifier string, imageName string, imageDescription string) error {
	sess, _ := session.NewSession(&aws.Config{
		Region: aws.String(destinationRegion)},
	)

	svc := ec2.New(sess)

	copyResult, err := svc.CopyImage(&ec2.CopyImageInput{
		SourceImageId: aws.String(imageIdentifier),
		SourceRegion:  aws.String(sourceRegion),
		Name:          aws.String(imageName),
		Description:   aws.String(imageDescription),
	})
	if err != nil {
		return err
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
		return err
	}
	return nil
}
