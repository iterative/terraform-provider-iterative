package iterative

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/teris-io/shortid"
)

func resourceMachine() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceMachineCreate,
		ReadContext:   resourceMachineRead,
		//UpdateContext: resourceMachineUpdate,s
		DeleteContext: resourceMachineDelete,
		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "us-west-2",
			},
			"instance_ami": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "ami-e7527ed7",
			},
			"instance_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "t2.micro",
			},
			"instance_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"instance_ip": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"instance_launch_time": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"key_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"key_public": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "",
			},
			"key_private": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"aws_security_group": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "",
			},
		},
	}
}

func resourceMachineCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	svc, errClient := awsClient(d)
	if errClient != nil {
		return diag.FromErr(errClient)
	}

	sid, err := shortid.New(1, shortid.DefaultABC, 2342)
	id, _ := sid.Generate()

	instanceAmi := d.Get("instance_ami").(string)
	instanceType := d.Get("instance_type").(string)
	keyPublic := d.Get("key_public").(string)

	securityGroup := d.Get("aws_security_group").(string)

	pairName := "cml_" + id
	var keyMaterial string

	// key-pair
	if len(keyPublic) != 0 {
		_, errImportKeyPair := svc.ImportKeyPair(&ec2.ImportKeyPairInput{
			KeyName:           aws.String(pairName),
			PublicKeyMaterial: []byte(keyPublic),
		})
		if errImportKeyPair != nil {
			return diag.FromErr(errImportKeyPair)
		}

	} else {
		keyResult, err := svc.CreateKeyPair(&ec2.CreateKeyPairInput{
			KeyName: aws.String(pairName),
		})
		if err != nil {
			return diag.FromErr(err)
		}
		keyMaterial = *keyResult.KeyMaterial
	}

	if len(securityGroup) == 0 {
		securityGroup = "cml"

		vpcsDesc, _ := svc.DescribeVpcs(&ec2.DescribeVpcsInput{})
		vpc := vpcsDesc.Vpcs[0]
		vpcID := *vpc.VpcId

		gpResult, ee := svc.CreateSecurityGroup(&ec2.CreateSecurityGroupInput{
			GroupName:   aws.String(securityGroup),
			Description: aws.String("CML security group"),
			VpcId:       aws.String(vpcID),
		})

		svc.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
			GroupId: aws.String(*gpResult.GroupId),
			IpPermissions: []*ec2.IpPermission{
				(&ec2.IpPermission{}).
					SetIpProtocol("-1").
					SetFromPort(-1).
					SetToPort(-1).
					SetIpRanges([]*ec2.IpRange{
						{CidrIp: aws.String("0.0.0.0/0")},
					}),
			},
		})

		svc.AuthorizeSecurityGroupEgress(&ec2.AuthorizeSecurityGroupEgressInput{
			GroupId: aws.String(*gpResult.GroupId),
			IpPermissions: []*ec2.IpPermission{
				(&ec2.IpPermission{}).
					SetIpProtocol("-1").
					SetFromPort(-1).
					SetToPort(-1).
					SetIpRanges([]*ec2.IpRange{
						{CidrIp: aws.String("0.0.0.0/0")},
					}),
			},
		})
		if ee != nil {
			return diag.FromErr(err)
		}
	}

	runResult, err := svc.RunInstancesWithContext(ctx, &ec2.RunInstancesInput{
		ImageId:      aws.String(instanceAmi),
		KeyName:      aws.String(pairName),
		InstanceType: aws.String(instanceType),
		MinCount:     aws.Int64(1),
		MaxCount:     aws.Int64(1),
		SecurityGroups: []*string{
			aws.String(securityGroup),
		},
		//CpuOptions:   instanceOpts.CpuOptions,
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// Add tags to the created instance
	_, errtag := svc.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{runResult.Instances[0].InstanceId},
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String("cml"),
			},
		},
	})
	if errtag != nil {
		return diag.FromErr(errtag)
	}

	instance := *runResult.Instances[0]
	instanceID := *instance.InstanceId

	instanceIds := make([]*string, 1)
	instanceIds[0] = &instanceID
	statusInput := ec2.DescribeInstancesInput{
		InstanceIds: instanceIds,
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
	d.Set("instance_ip", instanceDesc.PublicIpAddress)
	d.Set("instance_launch_time", instanceDesc.LaunchTime.Format(time.RFC3339))
	d.Set("key_name", pairName)
	d.Set("key_privates", keyMaterial)

	return diags
}

func resourceMachineRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return nil
}

func resourceMachineUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return nil
}

func resourceMachineDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	svc, _ := awsClient(d)

	instanceID := d.Get("instance_id").(string)
	pairName := d.Get("key_name").(string)

	_, erro := svc.DeleteKeyPair(&ec2.DeleteKeyPairInput{
		KeyName: aws.String(pairName),
	})
	if erro != nil {
		diag.FromErr(erro)
	}

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

func awsClient(d *schema.ResourceData) (*ec2.EC2, error) {
	region := d.Get("region").(string)
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region)},
	)
	svc := ec2.New(sess)

	return svc, err
}
