package resources

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"

	"terraform-provider-iterative/task/az/client"
	"terraform-provider-iterative/task/common"
)

func NewSecurityGroup(client *client.Client, identifier common.Identifier, resourceGroup *ResourceGroup, firewall common.Firewall) *SecurityGroup {
	s := &SecurityGroup{
		client:     client,
		Identifier: identifier.Long(),
		Attributes: firewall,
	}
	s.Dependencies.ResourceGroup = resourceGroup
	return s
}

type SecurityGroup struct {
	client       *client.Client
	Identifier   string
	Attributes   common.Firewall
	Dependencies struct {
		ResourceGroup *ResourceGroup
	}
	Resource *armnetwork.SecurityGroup
}

func (s *SecurityGroup) Create(ctx context.Context) error {
	var rules []*armnetwork.SecurityRule
	var ingressNets, egressNets []string
	var ingressPorts, egressPorts []string

	if s.Attributes.Egress.Nets == nil {
		egressNets = []string{"*"}
	} else {
		for _, cidrNet := range *s.Attributes.Egress.Nets {
			egressNets = append(egressNets, cidrNet.String())
		}
	}
	if s.Attributes.Ingress.Nets == nil {
		ingressNets = []string{"*"}
	} else {
		for _, cidrNet := range *s.Attributes.Ingress.Nets {
			ingressNets = append(ingressNets, cidrNet.String())
		}
	}
	if s.Attributes.Egress.Ports == nil {
		egressPorts = []string{"*"}
	} else {
		for _, port := range *s.Attributes.Egress.Ports {
			egressPorts = append(egressPorts, strconv.Itoa(int(port)))
		}
	}
	if s.Attributes.Ingress.Ports == nil {
		ingressPorts = []string{"*"}
	} else {
		for _, port := range *s.Attributes.Ingress.Ports {
			ingressPorts = append(ingressPorts, strconv.Itoa(int(port)))
		}
	}

	for netIndex, netValue := range ingressNets {
		for portIndex, portValue := range ingressPorts {
			rules = append(rules, generateSecurityRule(netValue, portValue, uint16(netIndex*len(ingressPorts)+portIndex), armnetwork.SecurityRuleDirectionInbound))
		}
	}
	for netIndex, netValue := range egressNets {
		for portIndex, portValue := range egressPorts {
			rules = append(rules, generateSecurityRule(netValue, portValue, uint16(netIndex*len(egressPorts)+portIndex), armnetwork.SecurityRuleDirectionOutbound))
		}
	}

	poller, err := s.client.Services.SecurityGroups.BeginCreateOrUpdate(ctx, s.Dependencies.ResourceGroup.Identifier, s.Identifier, armnetwork.SecurityGroup{
		Tags:     s.client.Tags,
		Location: to.Ptr(s.client.Region),
		Properties: &armnetwork.SecurityGroupPropertiesFormat{
			SecurityRules: rules,
		},
	}, nil)

	if err != nil {
		return err
	}

	if _, err := poller.PollUntilDone(ctx, nil); err != nil {
		return err
	}

	return s.Read(ctx)
}

func (s *SecurityGroup) Read(ctx context.Context) error {
	response, err := s.client.Services.SecurityGroups.Get(ctx, s.Dependencies.ResourceGroup.Identifier, s.Identifier, nil)
	if err != nil {
		var e *azcore.ResponseError
		if errors.As(err, &e) && e.RawResponse.StatusCode == 404 {
			return common.NotFoundError
		}
		return err
	}

	s.Resource = &response.SecurityGroup
	return nil
}

func (s *SecurityGroup) Update(ctx context.Context) error {
	return common.NotImplementedError
}

func (s *SecurityGroup) Delete(ctx context.Context) error {
	poller, err := s.client.Services.SecurityGroups.BeginDelete(ctx, s.Dependencies.ResourceGroup.Identifier, s.Identifier, nil)
	if err != nil {
		var e *azcore.ResponseError
		if errors.As(err, &e) && e.RawResponse.StatusCode == 404 {
			return nil
		}
		return err
	}

	_, err = poller.PollUntilDone(ctx, nil)
	s.Resource = nil
	return err
}

func generateSecurityRule(net, port string, priority uint16, direction armnetwork.SecurityRuleDirection) *armnetwork.SecurityRule {
	return &armnetwork.SecurityRule{
		Name: to.Ptr(fmt.Sprintf("%s-%d", direction, priority)),
		Properties: &armnetwork.SecurityRulePropertiesFormat{
			Protocol:                 to.Ptr(armnetwork.SecurityRuleProtocolAsterisk),
			SourceAddressPrefix:      to.Ptr("*"),
			SourcePortRange:          to.Ptr("*"),
			DestinationAddressPrefix: to.Ptr(net),
			DestinationPortRange:     to.Ptr(port),
			Access:                   to.Ptr(armnetwork.SecurityRuleAccessAllow),
			Priority:                 to.Ptr(int32(100 + priority)),
			Direction:                &direction,
		},
	}
}
