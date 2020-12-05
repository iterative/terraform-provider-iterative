package aws

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

//ResourceMachineCreate creates AWS instance
func ResourceMachineCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	instanceName := d.Get("instance_name").(string)
	hddSize := d.Get("instance_hdd_size").(int)
	region := getRegion(d)
	instanceType := getInstanceType(d)

	svc, errClient := awsClient(region)
	if errClient != nil {
		return diag.FromErr(errClient)
	}

	// Image
	ami := d.Get("ami").(string)
	imagesRes, imagesErr := svc.DescribeImages(&ec2.DescribeImagesInput{
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
	if imagesErr != nil {
		return diag.FromErr(imagesErr)
	}
	if len(imagesRes.Images) == 0 {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  ami + " ami not found in region",
		})

		return diags
	}

	sort.Slice(imagesRes.Images, func(i, j int) bool {
		itime, _ := time.Parse(time.RFC3339, aws.StringValue(imagesRes.Images[i].CreationDate))
		jtime, _ := time.Parse(time.RFC3339, aws.StringValue(imagesRes.Images[j].CreationDate))
		return itime.Unix() > jtime.Unix()
	})

	instanceAmi := *imagesRes.Images[0].ImageId
	pairName := instanceName

	// key-pair
	keyPublic := d.Get("key_public").(string)
	svc.ImportKeyPair(&ec2.ImportKeyPairInput{
		KeyName:           aws.String(pairName),
		PublicKeyMaterial: []byte(keyPublic),
	})

	// securityGroup
	var vpcID, sgID string

	securityGroup := d.Get("aws_security_group").(string)
	if len(securityGroup) == 0 {
		securityGroup = "cml"

		vpcsDesc, _ := svc.DescribeVpcs(&ec2.DescribeVpcsInput{})
		vpcID = *vpcsDesc.Vpcs[0].VpcId

		gpResult, ee := svc.CreateSecurityGroup(&ec2.CreateSecurityGroupInput{
			GroupName:   aws.String(securityGroup),
			Description: aws.String("CML security group"),
			VpcId:       aws.String(vpcID),
		})

		if ee == nil {
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
	}

	sgDesc, sgDescErr := svc.DescribeSecurityGroupsWithContext(ctx, &ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("group-name"),
				Values: []*string{aws.String(securityGroup)},
			},
		},
	})
	if sgDescErr != nil {
		return diag.FromErr(sgDescErr)
	}

	sgID = *sgDesc.SecurityGroups[0].GroupId
	vpcID = *sgDesc.SecurityGroups[0].VpcId

	subDesc, _ := svc.DescribeSubnetsWithContext(ctx, &ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []*string{aws.String(vpcID)},
			},
		},
	})

	//launch instance
	runResult, err := svc.RunInstancesWithContext(ctx, &ec2.RunInstancesInput{
		ImageId:          aws.String(instanceAmi),
		KeyName:          aws.String(pairName),
		InstanceType:     aws.String(instanceType),
		MinCount:         aws.Int64(1),
		MaxCount:         aws.Int64(1),
		SecurityGroupIds: []*string{aws.String(sgID)},
		SubnetId:         aws.String(*subDesc.Subnets[0].SubnetId),
		BlockDeviceMappings: []*ec2.BlockDeviceMapping{
			{
				//VirtualName: aws.String("Root"),
				DeviceName: aws.String("/dev/sda1"),
				Ebs: &ec2.EbsBlockDevice{
					DeleteOnTermination: aws.Bool(true),
					Encrypted:           aws.Bool(false),
					//Iops:                aws.Int64(0),
					VolumeSize: aws.Int64(int64(hddSize)),
					VolumeType: aws.String("gp2"),
				},
			},
		},
	})
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Unable to create instance",
			Detail:   fmt.Sprintf("[ERROR] Instance %s of type %s at region %s", instanceName, instanceType, region),
		})

		diags = append(diags, diag.FromErr(err)[0])

		return diags
	}

	instanceID := *runResult.Instances[0].InstanceId

	// Add name to the created instance
	_, errTag := svc.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{aws.String(instanceID)},
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String(instanceName),
			},
		},
	})
	if errTag != nil {
		return diag.FromErr(errTag)
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

	descResult, _ := svc.DescribeInstancesWithContext(ctx, &statusInput)
	instanceDesc := descResult.Reservations[0].Instances[0]

	d.SetId(instanceID)
	d.Set("instance_id", instanceID)
	d.Set("key_name", pairName)

	d.Set("instance_name", instanceName)
	d.Set("instance_ip", instanceDesc.PublicIpAddress)
	d.Set("instance_launch_time", instanceDesc.LaunchTime.Format(time.RFC3339))

	return diags
}

//ResourceMachineDelete deletes AWS instance
func ResourceMachineDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	svc, _ := awsClient(getRegion(d))

	pairName := d.Get("key_name").(string)
	instanceID := d.Get("instance_id").(string)

	svc.DeleteKeyPair(&ec2.DeleteKeyPairInput{
		KeyName: aws.String(pairName),
	})

	input := &ec2.TerminateInstancesInput{
		InstanceIds: []*string{
			aws.String(instanceID),
		},
		DryRun: aws.Bool(false),
	}

	_, err := svc.TerminateInstances(input)

	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func awsClient(region string) (*ec2.EC2, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
		//Credentials: credentials.NewStaticCredentials(conf.AWS_ACCESS_KEY_ID, conf.AWS_SECRET_ACCESS_KEY, ""),
	})

	svc := ec2.New(sess)
	return svc, err
}

func getRegion(d *schema.ResourceData) string {
	instanceRegions := make(map[string]string)
	instanceRegions["us-east"] = "us-east-1"
	instanceRegions["us-west"] = "us-west-1"
	instanceRegions["eu-north"] = "eu-north-1"
	instanceRegions["eu-west"] = "eu-west-1"

	region := d.Get("region").(string)
	if val, ok := instanceRegions[region]; ok {
		region = val
	}

	return region
}

func getInstanceType(d *schema.ResourceData) string {
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

	instanceType := d.Get("instance_type").(string)
	if val, ok := instanceTypes[instanceType+d.Get("instance_gpu").(string)]; ok {
		instanceType = val
	}

	return instanceType
}
