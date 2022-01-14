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

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var (
	SynthRegions = map[string]string{
		"us-east":  "us-east-1",
		"us-west":  "us-west-1",
		"eu-north": "eu-north-1",
		"eu-west":  "eu-west-1",
	}
)

//ResourceMachineCreate creates AWS instance
func ResourceMachineCreate(ctx context.Context, d *schema.ResourceData, m interface{}) error {
	userData := d.Get("startup_script").(string)
	pairName := d.Id()
	hddSize := d.Get("instance_hdd_size").(int)
	instanceType := getInstanceType(d.Get("instance_type").(string), d.Get("instance_gpu").(string))
	ami := d.Get("image").(string)
	keyPublic := d.Get("ssh_public").(string)
	securityGroup := d.Get("aws_security_group").(string)
	spot := d.Get("spot").(bool)
	spotPrice := d.Get("spot_price").(float64)
	instanceProfile := d.Get("instance_permission_set").(string)
	subnetId := d.Get("aws_subnet_id").(string)

	region := GetRegion(d.Get("region").(string))
	availabilityZone := GetAvailabilityZone(d.Get("region").(string))

	metadata := map[string]string{
		"Name": d.Get("name").(string),
		"Id":   d.Id(),
	}
	for key, value := range d.Get("metadata").(map[string]interface{}) {
		metadata[key] = value.(string)
	}

	if ami == "" {
		ami = "iterative-cml"
	}
	config, err := awsClient(region)
	if err != nil {
		return decodeAWSError(region, err)
	}
	svc := ec2.NewFromConfig(config)

	// Image
	imagesRes, err := svc.DescribeImages(ctx, &ec2.DescribeImagesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("name"),
				Values: []string{ami},
			},
			{
				Name:   aws.String("architecture"),
				Values: []string{"x86_64"},
			},
		},
	})
	if err != nil {
		return decodeAWSError(region, err)
	}
	if len(imagesRes.Images) == 0 {
		imagesRes, err = svc.DescribeImages(ctx, &ec2.DescribeImagesInput{
			Filters: []types.Filter{
				{
					Name:   aws.String("name"),
					Values: []string{"*ubuntu/images/hvm-ssd/ubuntu-bionic-18.04*"},
				},
				{
					Name:   aws.String("architecture"),
					Values: []string{"x86_64"},
				},
				{
					Name:   aws.String("owner-id"),
					Values: []string{"099720109477"},
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
		itime, _ := time.Parse(time.RFC3339, aws.ToString(imagesRes.Images[i].CreationDate))
		jtime, _ := time.Parse(time.RFC3339, aws.ToString(imagesRes.Images[j].CreationDate))
		return itime.Unix() > jtime.Unix()
	})

	instanceAmi := *imagesRes.Images[0].ImageId

	// key-pair
	svc.ImportKeyPair(ctx, &ec2.ImportKeyPairInput{
		KeyName:           aws.String(pairName),
		PublicKeyMaterial: []byte(keyPublic),
		TagSpecifications: resourceTagSpecifications(types.ResourceTypeKeyPair, metadata),
	})

	// securityGroup
	var vpcID, sgID string
	if len(securityGroup) == 0 {
		securityGroup = "iterative"

		vpcsDesc, _ := svc.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{})

		if len(vpcsDesc.Vpcs) == 0 {
			return errors.New("no VPCs found")
		}
		vpcID = *vpcsDesc.Vpcs[0].VpcId

		for _, vpc := range vpcsDesc.Vpcs {
			if *vpc.IsDefault {
				vpcID = *vpc.VpcId
				break
			}
		}

		gpResult, err := svc.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
			GroupName:         aws.String(securityGroup),
			Description:       aws.String("Iterative security group"),
			VpcId:             aws.String(vpcID),
			TagSpecifications: resourceTagSpecifications(types.ResourceTypeSecurityGroup, metadata),
		})

		if err == nil {
			ipPermissions := []types.IpPermission{
				{
					IpProtocol: aws.String("-1"),
					FromPort:   aws.Int32(-1),
					ToPort:     aws.Int32(-1),
					IpRanges: []types.IpRange{
						{CidrIp: aws.String("0.0.0.0/0")},
					},
				},
			}

			svc.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
				GroupId:           aws.String(*gpResult.GroupId),
				IpPermissions:     ipPermissions,
				TagSpecifications: resourceTagSpecifications(types.ResourceTypeSecurityGroupRule, metadata),
			})

			svc.AuthorizeSecurityGroupEgress(ctx, &ec2.AuthorizeSecurityGroupEgressInput{
				GroupId:           aws.String(*gpResult.GroupId),
				IpPermissions:     ipPermissions,
				TagSpecifications: resourceTagSpecifications(types.ResourceTypeSecurityGroupRule, metadata),
			})
		}

		if err != nil {
			decodedError := decodeAWSError(region, err)
			if !strings.Contains(decodedError.Error(), "already exists for VPC") {
				return decodedError
			}
		}
	}

	sgFilters := []types.Filter{}
	if strings.HasPrefix(securityGroup, "sg-") {
		sgFilters = append(sgFilters, types.Filter{
			Name:   aws.String("group-id"),
			Values: []string{securityGroup},
		})
	} else {
		sgFilters = append(sgFilters, types.Filter{
			Name: aws.String("group-name"),
			Values: []string{
				securityGroup,
				strings.Title(securityGroup),
				strings.ToUpper(securityGroup)},
		})
	}

	sgDesc, err := svc.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
		Filters: sgFilters,
	})
	if err != nil {
		return decodeAWSError(region, err)
	}
	if len(sgDesc.SecurityGroups) == 0 {
		return errors.New("no Security Groups found")
	}

	sgID = *sgDesc.SecurityGroups[0].GroupId
	vpcID = *sgDesc.SecurityGroups[0].VpcId

	// default Subnet selection
	subnetOptions := &ec2.DescribeSubnetsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []string{vpcID},
			},
		},
	}
	// use availability zone from user
	if availabilityZone != "" && subnetId == "" {
		subnetOptions.Filters = append(subnetOptions.Filters, types.Filter{
			Name:   aws.String("availability-zone"),
			Values: []string{availabilityZone},
		})
	}
	// use exact subnet-id from user
	if subnetId != "" {
		subnetOptions.Filters = append(subnetOptions.Filters, types.Filter{
			Name:   aws.String("subnet-id"),
			Values: []string{subnetId},
		})
	}
	subDesc, err := svc.DescribeSubnets(ctx, subnetOptions)
	if err != nil {
		return decodeAWSError(region, err)
	}
	if len(subDesc.Subnets) == 0 {
		return errors.New("no Subnet found")
	}
	var subnetID string
	// bypass with user provided ID
	if subnetId == "" {
		for _, subnet := range subDesc.Subnets {
			if *subnet.AvailableIpAddressCount > 0 && *subnet.MapPublicIpOnLaunch {
				subnetID = *subnet.SubnetId
				break
			}
		}
	} else {
		subnetID = subnetId
	}
	if subnetID == "" {
		return errors.New("No subnet found with public IPs available or able to create new public IPs on creation")
	}

	blockDeviceMappings := []types.BlockDeviceMapping{
		{
			DeviceName: aws.String("/dev/sda1"),
			Ebs: &types.EbsBlockDevice{
				DeleteOnTermination: aws.Bool(true),
				Encrypted:           aws.Bool(false),
				VolumeSize:          aws.Int32(int32(hddSize)),
				VolumeType:          types.VolumeType("gp2"),
			},
		},
	}

	//launch instance
	var instanceID string
	if spot {
		requestSpotInstancesInput := &ec2.RequestSpotInstancesInput{
			LaunchSpecification: &types.RequestSpotLaunchSpecification{
				UserData:     aws.String(userData),
				ImageId:      aws.String(instanceAmi),
				KeyName:      aws.String(pairName),
				InstanceType: types.InstanceType(instanceType),
				IamInstanceProfile: &types.IamInstanceProfileSpecification{
					Arn: aws.String(instanceProfile),
				},
				SecurityGroupIds:    []string{sgID},
				SubnetId:            aws.String(subnetID),
				BlockDeviceMappings: blockDeviceMappings,
			},
			InstanceCount:     aws.Int32(1),
			TagSpecifications: resourceTagSpecifications(types.ResourceTypeSpotInstancesRequest, metadata),
		}

		if spotPrice >= 0 {
			requestSpotInstancesInput.SpotPrice = aws.String(strconv.FormatFloat(spotPrice, 'f', 5, 64))
		}

		spotInstanceRequest, err := svc.RequestSpotInstances(ctx, requestSpotInstancesInput)
		if err != nil {
			return decodeAWSError(region, err)
		}

		spotInstanceRequestID := *spotInstanceRequest.SpotInstanceRequests[0].SpotInstanceRequestId
		// 10 minutes as per https://github.com/aws/aws-sdk-go/blob/23db5a4/service/ec2/waiters.go#L426-L459
		spotInstanceRequestFulfilledWaiter := ec2.NewSpotInstanceRequestFulfilledWaiter(svc)
		err = spotInstanceRequestFulfilledWaiter.Wait(ctx, &ec2.DescribeSpotInstanceRequestsInput{
			SpotInstanceRequestIds: []string{spotInstanceRequestID},
		}, 10*time.Minute)
		if err != nil {
			_, cancelError := svc.CancelSpotInstanceRequests(ctx, &ec2.CancelSpotInstanceRequestsInput{
				SpotInstanceRequestIds: []string{spotInstanceRequestID},
			})
			if cancelError != nil {
				err = cancelError
			}
			return decodeAWSError(region, err)
		}

		resolvedSpotInstance, err := svc.DescribeSpotInstanceRequests(ctx, &ec2.DescribeSpotInstanceRequestsInput{
			SpotInstanceRequestIds: []string{spotInstanceRequestID},
		})
		if err != nil {
			return decodeAWSError(region, err)
		}

		instanceID = *resolvedSpotInstance.SpotInstanceRequests[0].InstanceId

		// Add tags to the created instance
		_, err = svc.CreateTags(ctx, &ec2.CreateTagsInput{
			Resources: []string{instanceID},
			Tags:      resourceTagSpecifications(types.ResourceTypeInstance, metadata)[0].Tags,
		})
		if err != nil {
			return decodeAWSError(region, err)
		}
	} else {
		runResult, err := svc.RunInstances(ctx, &ec2.RunInstancesInput{
			UserData:     aws.String(userData),
			ImageId:      aws.String(instanceAmi),
			KeyName:      aws.String(pairName),
			InstanceType: types.InstanceType(instanceType),
			IamInstanceProfile: &types.IamInstanceProfileSpecification{
				Arn: aws.String(instanceProfile),
			},
			MinCount:            aws.Int32(1),
			MaxCount:            aws.Int32(1),
			SecurityGroupIds:    []string{sgID},
			SubnetId:            aws.String(subnetID),
			BlockDeviceMappings: blockDeviceMappings,
			TagSpecifications:   resourceTagSpecifications(types.ResourceTypeInstance, metadata),
		})
		if err != nil {
			return decodeAWSError(region, err)
		}

		instanceID = *runResult.Instances[0].InstanceId
	}

	statusInput := ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
		Filters: []types.Filter{
			{
				Name:   aws.String("instance-state-name"),
				Values: []string{"running"},
			},
		},
	}

	instanceExistsWaiter := ec2.NewInstanceExistsWaiter(svc)
	err = instanceExistsWaiter.Wait(ctx, &statusInput, 10*time.Minute)
	if err != nil {
		return decodeAWSError(region, err)
	}

	descResult, err := svc.DescribeInstances(ctx, &statusInput)
	if err != nil {
		return decodeAWSError(region, err)
	}

	instanceDesc := descResult.Reservations[0].Instances[0]
	var instanceIP string
	if instanceDesc.PublicIpAddress != nil {
		instanceIP = *instanceDesc.PublicIpAddress
	} else {
		instanceIP = *instanceDesc.PrivateIpAddress
	}
	d.Set("instance_ip", instanceIP)
	d.Set("instance_launch_time", instanceDesc.LaunchTime.Format(time.RFC3339))
	d.Set("image", *imagesRes.Images[0].Name)

	return nil
}

//ResourceMachineDelete deletes AWS instance
func ResourceMachineDelete(ctx context.Context, d *schema.ResourceData, m interface{}) error {
	id := aws.String(d.Id())
	region := GetRegion(d.Get("region").(string))

	config, err := awsClient(region)
	if err != nil {
		return decodeAWSError(region, err)
	}
	svc := ec2.NewFromConfig(config)

	svc.DeleteKeyPair(ctx, &ec2.DeleteKeyPairInput{
		KeyName: id,
	})

	descResult, err := svc.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:Id"),
				Values: []string{d.Id()},
			},
		},
	})
	if err != nil {
		return err
	}

	if len(descResult.Reservations) > 0 && len(descResult.Reservations[0].Instances) > 0 {
		instanceID := *descResult.Reservations[0].Instances[0].InstanceId
		_, err = svc.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
			InstanceIds: []string{
				instanceID,
			},
		})
		if err != nil {
			return decodeAWSError(region, err)
		}
	}

	return nil
}

func awsClient(region string) (aws.Config, error) {
	return config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
	)
}

func GetRegion(region string) string {
	if val, ok := SynthRegions[region]; ok {
		return val
	}
	return StripAvailabilityZone(region)
}

func StripAvailabilityZone(region string) string {
	lastChar := region[len(region)-1]
	if lastChar >= 'a' && lastChar <= 'z' {
		return region[:len(region)-1]
	}
	return region
}

func GetAvailabilityZone(region string) string {
	lastChar := region[len(region)-1]
	// no avail-zone with synthetic regions
	if _, ok := SynthRegions[region]; ok {
		return ""
	}
	if lastChar >= 'a' && lastChar <= 'z' {
		return region
	}
	return ""
}

func getInstanceType(instanceType string, instanceGPU string) string {
	instanceTypes := make(map[string]string)
	instanceTypes["m"] = "m5.2xlarge"
	instanceTypes["l"] = "m5.8xlarge"
	instanceTypes["xl"] = "m5.16xlarge"
	instanceTypes["m+k80"] = "p2.xlarge"
	instanceTypes["l+k80"] = "p2.8xlarge"
	instanceTypes["xl+k80"] = "p2.16xlarge"
	instanceTypes["m+v100"] = "p3.xlarge"
	instanceTypes["l+v100"] = "p3.8xlarge"
	instanceTypes["xl+v100"] = "p3.16xlarge"

	if val, ok := instanceTypes[instanceType+"+"+instanceGPU]; ok {
		return val
	} else if val, ok := instanceTypes[instanceType]; ok && instanceGPU == "" {
		return val
	}

	return instanceType
}

var encodedFailureMessagePattern = regexp.MustCompile(`(?i)(.*) Encoded authorization failure message: ([\w-]+) ?( .*)?`)

func decodeAWSError(region string, err error) error {
	config, erro := awsClient(region)
	if erro != nil {
		return erro
	}
	svc := sts.NewFromConfig(config)

	groups := encodedFailureMessagePattern.FindStringSubmatch(err.Error())
	if len(groups) > 2 {
		result, erro := svc.DecodeAuthorizationMessage(context.TODO(), &sts.DecodeAuthorizationMessageInput{
			EncodedMessage: aws.String(groups[2]),
		})
		if erro != nil {
			return err
		}

		msg := aws.ToString(result.DecodedMessage)
		return fmt.Errorf("%s Authorization failure message: '%s'%s", groups[1], msg, groups[3])
	}

	return fmt.Errorf("Not able to decode: %s", err.Error())
}

func resourceTagSpecifications(resourceType types.ResourceType, tags map[string]string) []types.TagSpecification {
	var tagStructs []types.Tag
	for key, value := range tags {
		tagStructs = append(tagStructs, types.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}
	return []types.TagSpecification{
		{
			ResourceType: resourceType,
			Tags:         tagStructs,
		},
	}
}
