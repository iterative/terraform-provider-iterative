package resources

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"

	"terraform-provider-iterative/task/common"
	"terraform-provider-iterative/task/gcp/client"
)

type FirewallRuleDirection string

const (
	FirewallRuleDirectionIngress FirewallRuleDirection = "INGRESS"
	FirewallRuleDirectionEgress  FirewallRuleDirection = "EGRESS"
)

type FirewallRuleAction string

const (
	FirewallRuleActionDeny  FirewallRuleAction = "DENY"
	FirewallRuleActionAllow FirewallRuleAction = "ALLOW"
)

func NewFirewallRule(client *client.Client, identifier common.Identifier, defaultNetwork *DefaultNetwork, rule common.FirewallRule, direction FirewallRuleDirection, action FirewallRuleAction, priority uint16) *FirewallRule {
	f := new(FirewallRule)
	f.Client = client
	f.Identifier = fmt.Sprintf("%s-%s%d", identifier.Long(), strings.ToLower(string(direction[0:1])), priority)
	f.Attributes.Rule = rule
	f.Attributes.Direction = direction
	f.Attributes.Action = action
	f.Attributes.Priority = priority
	f.Dependencies.DefaultNetwork = defaultNetwork
	return f
}

type FirewallRule struct {
	Client     *client.Client
	Identifier string
	Attributes struct {
		Rule      common.FirewallRule
		Direction FirewallRuleDirection
		Action    FirewallRuleAction
		Priority  uint16
	}
	Dependencies struct {
		*DefaultNetwork
	}
	Resource *compute.Firewall
}

func (f *FirewallRule) Create(ctx context.Context) error {
	var nets []string
	if f.Attributes.Rule.Nets != nil {
		for _, block := range *f.Attributes.Rule.Nets {
			nets = append(nets, block.String())
		}
	}

	var ports []string
	if f.Attributes.Rule.Ports != nil {
		for _, port := range *f.Attributes.Rule.Ports {
			ports = append(ports, strconv.Itoa(int(port)))
		}
	}

	definition := compute.Firewall{
		Name:       f.Identifier,
		Network:    f.Dependencies.DefaultNetwork.Resource.SelfLink,
		Priority:   int64(f.Attributes.Priority),
		TargetTags: []string{f.Identifier},
	}

	switch f.Attributes.Direction {
	case FirewallRuleDirectionIngress:
		definition.Direction = "INGRESS"
		definition.SourceRanges = nets
	case FirewallRuleDirectionEgress:
		definition.Direction = "EGRESS"
		definition.DestinationRanges = nets
	}

	switch f.Attributes.Action {
	case FirewallRuleActionAllow:
		definition.Allowed = []*compute.FirewallAllowed{
			{
				IPProtocol: "tcp",
				Ports:      ports,
			},
			{
				IPProtocol: "udp",
				Ports:      ports,
			},
		}
	case FirewallRuleActionDeny:
		definition.Denied = []*compute.FirewallDenied{
			{
				IPProtocol: "tcp",
				Ports:      ports,
			},
			{
				IPProtocol: "udp",
				Ports:      ports,
			},
		}
	}

	insertOperation, err := f.Client.Services.Compute.Firewalls.Insert(f.Client.Credentials.ProjectID, &definition).Do()
	if err != nil {
		if strings.HasSuffix(err.Error(), "alreadyExists") {
			return f.Read(ctx)
		}
		return err
	}

	getOperationCall := f.Client.Services.Compute.GlobalOperations.Get(f.Client.Credentials.ProjectID, insertOperation.Name)
	_, err = waitForOperation(ctx, f.Client.Cloud.Timeouts.Create, 2*time.Second, 32*time.Second, getOperationCall.Do)
	if err != nil {
		return err
	}

	return nil
}

func (f *FirewallRule) Read(ctx context.Context) error {
	firewall, err := f.Client.Services.Compute.Firewalls.Get(f.Client.Credentials.ProjectID, f.Identifier).Do()
	if err != nil {
		var e *googleapi.Error
		if errors.As(err, &e) && e.Code == 404 {
			return common.NotFoundError
		}
		return err
	}

	f.Resource = firewall
	return nil
}

func (f *FirewallRule) Update(ctx context.Context) error {
	return common.NotImplementedError
}

func (f *FirewallRule) Delete(ctx context.Context) error {
	deleteOperationCall := f.Client.Services.Compute.Firewalls.Delete(f.Client.Credentials.ProjectID, f.Identifier)
	_, err := waitForOperation(ctx, f.Client.Cloud.Timeouts.Delete, 2*time.Second, 32*time.Second, deleteOperationCall.Do)
	if err != nil {
		var e *googleapi.Error
		if !errors.As(err, &e) || e.Code != 404 {
			return err
		}
	}

	f.Resource = nil
	return nil
}
