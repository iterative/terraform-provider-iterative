package resources

import (
	"context"
	"errors"
	"net"
	"strconv"
	"time"

	"github.com/aws/smithy-go"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"terraform-provider-iterative/task/aws/client"
	"terraform-provider-iterative/task/universal"
)

func NewAutoScalingGroup(client *client.Client, identifier string, subnet *DefaultVPCSubnet, launchTemplate *LaunchTemplate, parallelism uint16, spot float64) *AutoScalingGroup {
	a := new(AutoScalingGroup)
	a.Client = client
	a.Identifier = universal.NormalizeIdentifier(identifier, true)
	a.Attributes.Parallelism = parallelism
	a.Attributes.Spot = spot
	a.Dependencies.DefaultVPCSubnet = subnet
	a.Dependencies.LaunchTemplate = launchTemplate
	return a
}

type AutoScalingGroup struct {
	Client     *client.Client
	Identifier string
	Attributes struct {
		Parallelism uint16
		Spot        float64
		Addresses   []net.IP
		Status      map[string]int
		Events      []universal.Event
	}
	Dependencies struct {
		*DefaultVPCSubnet
		*LaunchTemplate
	}
	Resource *types.AutoScalingGroup
}

func (a *AutoScalingGroup) Create(ctx context.Context) error {
	var spotPrice string
	var onDemandPercentage int32 = 100
	switch spot := a.Attributes.Spot; {
	case spot > 0:
		spotPrice = strconv.FormatFloat(float64(spot), 'f', 5, 64)
		fallthrough
	case spot == 0:
		onDemandPercentage = 0
	}

	input := autoscaling.CreateAutoScalingGroupInput{
		AutoScalingGroupName: aws.String(a.Identifier),
		DesiredCapacity:      aws.Int32(int32(a.Attributes.Parallelism)),
		MaxSize:              aws.Int32(int32(a.Attributes.Parallelism)),
		MinSize:              aws.Int32(0),
		MixedInstancesPolicy: &types.MixedInstancesPolicy{
			InstancesDistribution: &types.InstancesDistribution{
				OnDemandBaseCapacity:                aws.Int32(0),
				OnDemandPercentageAboveBaseCapacity: aws.Int32(onDemandPercentage),
				SpotAllocationStrategy:              aws.String("lowest-price"),
				SpotMaxPrice:                        aws.String(spotPrice),
			},
			LaunchTemplate: &types.LaunchTemplate{
				LaunchTemplateSpecification: &types.LaunchTemplateSpecification{
					LaunchTemplateName: aws.String(a.Dependencies.LaunchTemplate.Identifier),
					Version:            aws.String("$Latest"),
				},
			},
		},
		VPCZoneIdentifier: a.Dependencies.DefaultVPCSubnet.Resource.SubnetId,
	}

	if _, err := a.Client.Services.AutoScaling.CreateAutoScalingGroup(ctx, &input); err != nil {
		var e smithy.APIError
		if errors.As(err, &e) && e.ErrorCode() == "AlreadyExists" {
			return a.Read(ctx)
		}
		return err
	}

	waitInput := autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []string{a.Identifier},
	}

	if err := autoscaling.NewGroupInServiceWaiter(a.Client.Services.AutoScaling).Wait(ctx, &waitInput, a.Client.Cloud.Timeouts.Create); err != nil {
		return err
	}

	return a.Read(ctx)
}

func (a *AutoScalingGroup) Read(ctx context.Context) error {
	groupsInput := autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []string{a.Identifier},
	}

	groups, err := a.Client.Services.AutoScaling.DescribeAutoScalingGroups(ctx, &groupsInput)
	if err != nil {
		return err
	}

	if len(groups.AutoScalingGroups) == 0 {
		return universal.NotFoundError
	}

	var instancesInput ec2.DescribeInstancesInput
	for _, instance := range groups.AutoScalingGroups[0].Instances {
		instancesInput.InstanceIds = append(instancesInput.InstanceIds, aws.ToString(instance.InstanceId))
	}

	a.Attributes.Addresses = []net.IP{}
	a.Attributes.Status = make(map[string]int)
	for instancesPaginator := ec2.NewDescribeInstancesPaginator(a.Client.Services.EC2, &instancesInput); instancesPaginator.HasMorePages(); {
		page, err := instancesPaginator.NextPage(ctx)
		if err != nil {
			return err
		}

		for _, reservation := range page.Reservations {
			for _, instance := range reservation.Instances {
				status := string(instance.State.Name)
				if instance.StateReason != nil {
					status += " " + aws.ToString(instance.StateReason.Message)
				}
				a.Attributes.Status[status]++
				a.Attributes.Addresses = append(a.Attributes.Addresses, net.ParseIP(aws.ToString(instance.PublicIpAddress)))
			}
		}
	}

	activitiesInput := autoscaling.DescribeScalingActivitiesInput{
		AutoScalingGroupName: aws.String(a.Identifier),
	}

	a.Attributes.Events = []universal.Event{}
	for activitiesPaginator := autoscaling.NewDescribeScalingActivitiesPaginator(a.Client.Services.AutoScaling, &activitiesInput); activitiesPaginator.HasMorePages(); {
		page, err := activitiesPaginator.NextPage(ctx)
		if err != nil {
			return err
		}

		for _, activity := range page.Activities {
			timeStamp := time.Time{}
			if activity.StartTime != nil {
				timeStamp = *activity.StartTime
			}

			a.Attributes.Events = append(a.Attributes.Events, universal.Event{
				Time: timeStamp,
				Code: string(activity.StatusCode),
				Description: []string{
					aws.ToString(activity.Cause),
					aws.ToString(activity.Description),
					aws.ToString(activity.Details),
					aws.ToString(activity.StatusMessage),
				},
			})
		}
	}

	a.Resource = &groups.AutoScalingGroups[0]
	return nil
}

func (a *AutoScalingGroup) Update(ctx context.Context) error {
	input := autoscaling.UpdateAutoScalingGroupInput{
		AutoScalingGroupName: aws.String(a.Identifier),
		DesiredCapacity:      aws.Int32(int32(a.Attributes.Parallelism)),
	}

	if _, err := a.Client.Services.AutoScaling.UpdateAutoScalingGroup(ctx, &input); err != nil {
		return err
	}

	return nil
}

func (a *AutoScalingGroup) Delete(ctx context.Context) error {
	input := autoscaling.DeleteAutoScalingGroupInput{
		AutoScalingGroupName: aws.String(a.Identifier),
		ForceDelete:          aws.Bool(true),
	}

	if _, err := a.Client.Services.AutoScaling.DeleteAutoScalingGroup(ctx, &input); err != nil {
		var e smithy.APIError
		if errors.As(err, &e) && e.ErrorCode() == "ValidationError" {
			a.Resource = nil
			return nil
		}
		return err
	}

	waitInput := ec2.DescribeInstancesInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: []string{a.Dependencies.LaunchTemplate.Identifier},
			},
		},
	}

	if err := ec2.NewInstanceTerminatedWaiter(a.Client.Services.EC2).Wait(ctx, &waitInput, a.Client.Cloud.Timeouts.Delete); err != nil {
		return err
	}

	a.Resource = nil
	return nil
}
