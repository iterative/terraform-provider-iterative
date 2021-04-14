package aws

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

//ResourceMachineCreate creates AWS instance
func ResourceMachineCreate(ctx context.Context, d *schema.ResourceData, m interface{}) error {
	userData := d.Get("startup_script").(string)
	instanceName := d.Get("name").(string)
	pairName := d.Id()
	hddSize := d.Get("instance_hdd_size").(int)
	region := getRegion(d.Get("region").(string))
	instanceType := getInstanceType(d.Get("instance_type").(string), d.Get("instance_gpu").(string))
	ami := d.Get("image").(string)
	keyPublic := d.Get("ssh_public").(string)
	securityGroup := d.Get("aws_security_group").(string)
	spot := d.Get("spot").(bool)
	spotPrice := d.Get("spot_price").(float64)
	if ami == "" {
		ami = "iterative-cml"
	}

	svc, err := awsClient(region)

	if err != nil {
		return decodeAWSError(region, err)
	}

	// Image
	imagesRes, err := svc.DescribeImages(&ec2.DescribeImagesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("name"),
				Values: []*string{aws.String(ami)},
			},
			{
				Name:   aws.String("architecture"),
				Values: []*string{aws.String("x86_64")},
			},
		},
	})
	if err != nil {
		return decodeAWSError(region, err)
	}
	if len(imagesRes.Images) == 0 {
		imagesRes, err = svc.DescribeImages(&ec2.DescribeImagesInput{
			Filters: []*ec2.Filter{
				{
					Name:   aws.String("name"),
					Values: []*string{aws.String("*ubuntu/images/hvm-ssd/ubuntu-bionic-18.04*")},
				},
				{
					Name:   aws.String("architecture"),
					Values: []*string{aws.String("x86_64")},
				},
			},
		})

		if err != nil {
			return decodeAWSError(region, err)
		}
		if len(imagesRes.Images) == 0 {
			return errors.New("Nor " + ami + " nor Ubuntu Server 18.04 are available in your region")
		}
	}

	sort.Slice(imagesRes.Images, func(i, j int) bool {
		itime, _ := time.Parse(time.RFC3339, aws.StringValue(imagesRes.Images[i].CreationDate))
		jtime, _ := time.Parse(time.RFC3339, aws.StringValue(imagesRes.Images[j].CreationDate))
		return itime.Unix() > jtime.Unix()
	})

	instanceAmi := *imagesRes.Images[0].ImageId

	// key-pair
	svc.ImportKeyPair(&ec2.ImportKeyPairInput{
		KeyName:           aws.String(pairName),
		PublicKeyMaterial: []byte(keyPublic),
	})

	// securityGroup
	var vpcID, sgID string
	if len(securityGroup) == 0 {
		securityGroup = "iterative"

		vpcsDesc, _ := svc.DescribeVpcs(&ec2.DescribeVpcsInput{})
		if len(vpcsDesc.Vpcs) == 0 {
			return errors.New("no VPCs found")
		}
		vpcID = *vpcsDesc.Vpcs[0].VpcId

		gpResult, err := svc.CreateSecurityGroup(&ec2.CreateSecurityGroupInput{
			GroupName:   aws.String(securityGroup),
			Description: aws.String("Iterative security group"),
			VpcId:       aws.String(vpcID),
		})

		if err == nil {
			ipPermissions := []*ec2.IpPermission{
				(&ec2.IpPermission{}).
					SetIpProtocol("-1").
					SetFromPort(-1).
					SetToPort(-1).
					SetIpRanges([]*ec2.IpRange{
						{CidrIp: aws.String("0.0.0.0/0")},
					}),
			}

			svc.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
				GroupId:       aws.String(*gpResult.GroupId),
				IpPermissions: ipPermissions,
			})

			svc.AuthorizeSecurityGroupEgress(&ec2.AuthorizeSecurityGroupEgressInput{
				GroupId:       aws.String(*gpResult.GroupId),
				IpPermissions: ipPermissions,
			})
		}

		if err != nil {
			decodedError := decodeAWSError(region, err)
			if !strings.Contains(decodedError.Error(), "already exists for VPC") {
				return decodedError
			}
		}
	}

	sgDesc, err := svc.DescribeSecurityGroupsWithContext(ctx, &ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("group-name"),
				Values: []*string{
					aws.String(securityGroup),
					aws.String(strings.Title(securityGroup)),
					aws.String(strings.ToUpper(securityGroup))},
			},
		},
	})
	if err != nil {
		return decodeAWSError(region, err)
	}
	if len(sgDesc.SecurityGroups) == 0 {
		return errors.New("no Security Groups found")
	}

	sgID = *sgDesc.SecurityGroups[0].GroupId
	vpcID = *sgDesc.SecurityGroups[0].VpcId

	subDesc, err := svc.DescribeSubnetsWithContext(ctx, &ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []*string{aws.String(vpcID)},
			},
		},
	})
	if err != nil {
		return decodeAWSError(region, err)
	}
	if len(subDesc.Subnets) == 0 {
		return errors.New("no subnets found")
	}

	blockDeviceMappings := []*ec2.BlockDeviceMapping{
		{
			DeviceName: aws.String("/dev/sda1"),
			Ebs: &ec2.EbsBlockDevice{
				DeleteOnTermination: aws.Bool(true),
				Encrypted:           aws.Bool(false),
				VolumeSize:          aws.Int64(int64(hddSize)),
				VolumeType:          aws.String("gp2"),
			},
		},
	}

	//launch instance
	var instanceID string
	if spot {
		requestSpotInstancesInput := &ec2.RequestSpotInstancesInput{
			LaunchSpecification: &ec2.RequestSpotLaunchSpecification{
				UserData:            aws.String(userData),
				ImageId:             aws.String(instanceAmi),
				KeyName:             aws.String(pairName),
				InstanceType:        aws.String(instanceType),
				SecurityGroupIds:    []*string{aws.String(sgID)},
				SubnetId:            aws.String(*subDesc.Subnets[0].SubnetId),
				BlockDeviceMappings: blockDeviceMappings,
			},
			InstanceCount: aws.Int64(1),
		}

		if spotPrice >= 0 {
			requestSpotInstancesInput.SpotPrice = aws.String(strconv.FormatFloat(spotPrice, 'f', 5, 64))
		}

		spotInstanceRequest, err := svc.RequestSpotInstancesWithContext(ctx, requestSpotInstancesInput)
		if err != nil {
			return decodeAWSError(region, err)
		}

		spotInstanceRequestID := *spotInstanceRequest.SpotInstanceRequests[0].SpotInstanceRequestId
		err = svc.WaitUntilSpotInstanceRequestFulfilled(&ec2.DescribeSpotInstanceRequestsInput{
			SpotInstanceRequestIds: []*string{aws.String(spotInstanceRequestID)},
		})
		if err != nil {
			return decodeAWSError(region, err)
		}
		resolvedSpotInstance, err := svc.DescribeSpotInstanceRequests(&ec2.DescribeSpotInstanceRequestsInput{
			SpotInstanceRequestIds: []*string{aws.String(spotInstanceRequestID)},
		})
		if err != nil {
			return decodeAWSError(region, err)
		}

		instanceID = *resolvedSpotInstance.SpotInstanceRequests[0].InstanceId
	} else {
		runResult, err := svc.RunInstancesWithContext(ctx, &ec2.RunInstancesInput{
			UserData:            aws.String(userData),
			ImageId:             aws.String(instanceAmi),
			KeyName:             aws.String(pairName),
			InstanceType:        aws.String(instanceType),
			MinCount:            aws.Int64(1),
			MaxCount:            aws.Int64(1),
			SecurityGroupIds:    []*string{aws.String(sgID)},
			SubnetId:            aws.String(*subDesc.Subnets[0].SubnetId),
			BlockDeviceMappings: blockDeviceMappings,
		})
		if err != nil {
			return decodeAWSError(region, err)
		}

		instanceID = *runResult.Instances[0].InstanceId
	}

	// Add name to the created instance
	_, err = svc.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{aws.String(instanceID)},
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String(instanceName),
			},
			{
				Key:   aws.String("Id"),
				Value: aws.String(d.Id()),
			},
		},
	})
	if err != nil {
		return decodeAWSError(region, err)
	}

	statusInput := ec2.DescribeInstancesInput{
		InstanceIds: []*string{aws.String(instanceID)},
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("instance-state-name"),
				Values: []*string{aws.String("running")},
			},
		},
	}

	svc.WaitUntilInstanceExistsWithContext(ctx, &statusInput)

	descResult, err := svc.DescribeInstancesWithContext(ctx, &statusInput)
	if err != nil {
		return decodeAWSError(region, err)
	}

	instanceDesc := descResult.Reservations[0].Instances[0]
	d.Set("instance_ip", instanceDesc.PublicIpAddress)
	d.Set("instance_launch_time", instanceDesc.LaunchTime.Format(time.RFC3339))

	return nil
}

//ResourceMachineDelete deletes AWS instance
func ResourceMachineDelete(ctx context.Context, d *schema.ResourceData, m interface{}) error {
	id := aws.String(d.Id())
	region := getRegion(d.Get("region").(string))

	svc, err := awsClient(region)
	if err != nil {
		return decodeAWSError(region, err)
	}

	svc.DeleteKeyPair(&ec2.DeleteKeyPairInput{
		KeyName: id,
	})

	descResult, err := svc.DescribeInstancesWithContext(ctx, &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:Id"),
				Values: []*string{id},
			},
		},
	})
	if len(descResult.Reservations) > 0 && len(descResult.Reservations[0].Instances) > 0 {
		instanceID := *descResult.Reservations[0].Instances[0].InstanceId
		_, err = svc.TerminateInstances(&ec2.TerminateInstancesInput{
			InstanceIds: []*string{
				aws.String(instanceID),
			},
		})
		if err != nil {
			return decodeAWSError(region, err)
		}
	}

	return nil
}

func awsClient(region string) (*ec2.EC2, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
		//Credentials: credentials.NewStaticCredentials(conf.AWS_ACCESS_KEY_ID, conf.AWS_SECRET_ACCESS_KEY, ""),
	})

	svc := ec2.New(sess)
	return svc, err
}

//ImageRegions provider available image regions
var ImageRegions = []string{
	"us-east-2",
	"us-east-1",
	"us-west-1",
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

func getRegion(region string) string {
	instanceRegions := make(map[string]string)
	instanceRegions["us-east"] = "us-east-1"
	instanceRegions["us-west"] = "us-west-1"
	instanceRegions["eu-north"] = "eu-north-1"
	instanceRegions["eu-west"] = "eu-west-1"
	if val, ok := instanceRegions[region]; ok {
		return val
	}

	return region
}

func getInstanceType(instanceType string, instanceGPU string) string {
	instanceTypes := make(map[string]string)
	instanceTypes["m"] = "m5.2xlarge"
	instanceTypes["l"] = "m5.8xlarge"
	instanceTypes["xl"] = "m5.16xlarge"
	instanceTypes["mk80"] = "p2.xlarge"
	instanceTypes["lk80"] = "p2.8xlarge"
	instanceTypes["xlk80"] = "p2.16xlarge"
	instanceTypes["mtesla"] = "p3.xlarge"
	instanceTypes["ltesla"] = "p3.8xlarge"
	instanceTypes["xltesla"] = "p3.16xlarge"

	if val, ok := instanceTypes[instanceType+instanceGPU]; ok {
		return val
	}

	return instanceType
}

var encodedFailureMessagePattern = regexp.MustCompile(`(?i)(.*) Encoded authorization failure message: ([\w-]+) ?( .*)?`)

func decodeAWSError(region string, err error) error {
	sess, erro := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if erro != nil {
		return err
	}

	groups := encodedFailureMessagePattern.FindStringSubmatch(err.Error())
	if len(groups) > 2 {
		svc := sts.New(sess)
		result, erro := svc.DecodeAuthorizationMessage(&sts.DecodeAuthorizationMessageInput{
			EncodedMessage: aws.String(groups[2]),
		})
		if erro != nil {
			return err
		}

		msg := aws.StringValue(result.DecodedMessage)
		return fmt.Errorf("%s Authorization failure message: '%s'%s", groups[1], msg, groups[3])
	}

	return fmt.Errorf("Not able to decode: %s", err.Error())
}
