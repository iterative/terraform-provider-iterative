package resources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"

	"terraform-provider-iterative/task/az/client"
	"terraform-provider-iterative/task/common"
)

func NewSecurityGroup(client *client.Client, identifier common.Identifier, resourceGroup *ResourceGroup, firewall common.Firewall) *SecurityGroup {
	s := new(SecurityGroup)
	s.Client = client
	s.Identifier = identifier.Long()
	s.Attributes = firewall
	s.Dependencies.ResourceGroup = resourceGroup
	return s
}

type SecurityGroup struct {
	Client       *client.Client
	Identifier   string
	Attributes   common.Firewall
	Dependencies struct {
		*ResourceGroup
	}
	Resource *network.SecurityGroup
}

func (s *SecurityGroup) Create(ctx context.Context) error {
	var rules []network.SecurityRule
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
			rules = append(rules, generateSecurityRule(netValue, portValue, uint16(netIndex*len(ingressPorts)+portIndex), network.SecurityRuleDirectionInbound))
		}
	}
	for netIndex, netValue := range egressNets {
		for portIndex, portValue := range egressPorts {
			rules = append(rules, generateSecurityRule(netValue, portValue, uint16(netIndex*len(egressPorts)+portIndex), network.SecurityRuleDirectionOutbound))
		}
	}

	securityGroupCreateFuture, err := s.Client.Services.SecurityGroups.CreateOrUpdate(ctx, s.Dependencies.ResourceGroup.Identifier, s.Identifier, network.SecurityGroup{
		Tags:     s.Client.Tags,
		Location: to.StringPtr(s.Client.Region),
		SecurityGroupPropertiesFormat: &network.SecurityGroupPropertiesFormat{
			SecurityRules: &rules,
		},
	})

	if err != nil {
		return err
	}

	if err := securityGroupCreateFuture.WaitForCompletionRef(ctx, s.Client.Services.SecurityGroups.Client); err != nil {
		return err
	}

	return s.Read(ctx)
}

func (s *SecurityGroup) Read(ctx context.Context) error {
	securityGroup, err := s.Client.Services.SecurityGroups.Get(ctx, s.Dependencies.ResourceGroup.Identifier, s.Identifier, "")
	if err != nil {
		if err.(autorest.DetailedError).StatusCode == 404 {
			return common.NotFoundError
		}
		return err
	}

	s.Resource = &securityGroup
	return nil
}

func (s *SecurityGroup) Update(ctx context.Context) error {
	return common.NotImplementedError
}

func (s *SecurityGroup) Delete(ctx context.Context) error {
	groupDeleteFuture, err := s.Client.Services.SecurityGroups.Delete(ctx, s.Dependencies.ResourceGroup.Identifier, s.Identifier)
	if err != nil {
		if err.(autorest.DetailedError).StatusCode == 404 {
			return nil
		}
		return err
	}

	err = groupDeleteFuture.WaitForCompletionRef(ctx, s.Client.Services.Groups.Client)
	s.Resource = nil
	return err
}

func generateSecurityRule(net, port string, priority uint16, direction network.SecurityRuleDirection) network.SecurityRule {
	return network.SecurityRule{
		Name: to.StringPtr(fmt.Sprintf("%s-%d", direction, priority)),
		SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
			Protocol:                 network.SecurityRuleProtocolAsterisk,
			SourceAddressPrefix:      to.StringPtr("*"),
			SourcePortRange:          to.StringPtr("*"),
			DestinationAddressPrefix: to.StringPtr(net),
			DestinationPortRange:     to.StringPtr(port),
			Access:                   network.SecurityRuleAccessAllow,
			Priority:                 to.Int32Ptr(int32(100 + priority)),
			Direction:                direction,
		},
	}
}
