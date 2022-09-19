package resources

import (
	"context"
	"errors"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/aws/smithy-go"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/sirupsen/logrus"

	"terraform-provider-iterative/task/aws/client"
	"terraform-provider-iterative/task/common"
)

func NewAutoScalingGroup(client *client.Client, identifier common.Identifier, subnet *DefaultVPCSubnets, launchTemplate *LaunchTemplate, parallelism *uint16, spot common.Spot) *AutoScalingGroup {
	a := &AutoScalingGroup{
		client:     client,
		Identifier: identifier.Long(),
	}
	a.Attributes.Parallelism = parallelism
	a.Attributes.Spot = float64(spot)
	a.Dependencies.DefaultVPCSubnets = subnet
	a.Dependencies.LaunchTemplate = launchTemplate
	return a
}

type AutoScalingGroup struct {
	client     *client.Client
	Identifier string
	Attributes struct {
		Parallelism *uint16
		Spot        float64
		Addresses   []net.IP
		Status      common.Status
		Events      []common.Event
	}
	Dependencies struct {
		DefaultVPCSubnets *DefaultVPCSubnets
		LaunchTemplate    *LaunchTemplate
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

	var subnets []string
	for _, subnet := range a.Dependencies.DefaultVPCSubnets.Resource {
		subnets = append(subnets, aws.ToString(subnet.SubnetId))
	}

	input := autoscaling.CreateAutoScalingGroupInput{
		AutoScalingGroupName: aws.String(a.Identifier),
		DesiredCapacity:      aws.Int32(0),
		MaxSize:              aws.Int32(int32(*a.Attributes.Parallelism)),
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
		VPCZoneIdentifier: aws.String(strings.Join(subnets, ",")),
	}

	if _, err := a.client.Services.AutoScaling.CreateAutoScalingGroup(ctx, &input); err != nil {
		var e smithy.APIError
		if errors.As(err, &e) && e.ErrorCode() == "AlreadyExists" {
			return a.Read(ctx)
		}
		return err
	}

	waitInput := autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []string{a.Identifier},
	}

	if err := autoscaling.NewGroupExistsWaiter(a.client.Services.AutoScaling).Wait(ctx, &waitInput, a.client.Cloud.Timeouts.Create); err != nil {
		return err
	}

	return a.Read(ctx)
}

func (a *AutoScalingGroup) Read(ctx context.Context) error {
	groupsInput := autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []string{a.Identifier},
	}

	groups, err := a.client.Services.AutoScaling.DescribeAutoScalingGroups(ctx, &groupsInput)
	if err != nil {
		return err
	}

	if len(groups.AutoScalingGroups) == 0 {
		return common.NotFoundError
	}

	a.Attributes.Addresses = []net.IP{}
	a.Attributes.Status = common.Status{common.StatusCodeActive: 0}
	if len(groups.AutoScalingGroups[0].Instances) > 0 {
		var instancesInput ec2.DescribeInstancesInput
		for _, instance := range groups.AutoScalingGroups[0].Instances {
			instancesInput.InstanceIds = append(instancesInput.InstanceIds, aws.ToString(instance.InstanceId))
		}

		for instancesPaginator := ec2.NewDescribeInstancesPaginator(a.client.Services.EC2, &instancesInput); instancesPaginator.HasMorePages(); {
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
					logrus.Debug("AutoScaling Group State:", status)
					if status == "running" {
						a.Attributes.Status[common.StatusCodeActive]++
					}
					if address := net.ParseIP(aws.ToString(instance.PublicIpAddress)); address != nil {
						a.Attributes.Addresses = append(a.Attributes.Addresses, address)
					}
				}
			}
		}
	}

	activitiesInput := autoscaling.DescribeScalingActivitiesInput{
		AutoScalingGroupName: aws.String(a.Identifier),
	}

	a.Attributes.Events = []common.Event{}
	for activitiesPaginator := autoscaling.NewDescribeScalingActivitiesPaginator(a.client.Services.AutoScaling, &activitiesInput); activitiesPaginator.HasMorePages(); {
		page, err := activitiesPaginator.NextPage(ctx)
		if err != nil {
			return err
		}

		for _, activity := range page.Activities {
			timeStamp := time.Time{}
			if activity.StartTime != nil {
				timeStamp = *activity.StartTime
			}

			a.Attributes.Events = append(a.Attributes.Events, common.Event{
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
		DesiredCapacity:      aws.Int32(int32(*a.Attributes.Parallelism)),
	}

	if _, err := a.client.Services.AutoScaling.UpdateAutoScalingGroup(ctx, &input); err != nil {
		return err
	}

	return nil
}

func (a *AutoScalingGroup) Delete(ctx context.Context) error {
	input := autoscaling.DeleteAutoScalingGroupInput{
		AutoScalingGroupName: aws.String(a.Identifier),
		ForceDelete:          aws.Bool(true),
	}

	if _, err := a.client.Services.AutoScaling.DeleteAutoScalingGroup(ctx, &input); err != nil {
		var e smithy.APIError
		errors.As(err, &e)
		if errors.As(err, &e) && e.ErrorCode() == "ValidationError" {
			a.Resource = nil
			return nil
		}
		return err
	}

	waitInput := autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []string{a.Identifier},
	}

	if err := autoscaling.NewGroupNotExistsWaiter(a.client.Services.AutoScaling).Wait(ctx, &waitInput, a.client.Cloud.Timeouts.Delete); err != nil {
		return err
	}

	a.Resource = nil
	return nil
}
