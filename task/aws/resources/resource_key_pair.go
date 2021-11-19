package resources

import (
	"context"
	"errors"
	"strings"

	"github.com/aws/smithy-go"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"terraform-provider-iterative/task/aws/client"
	"terraform-provider-iterative/task/common"
	"terraform-provider-iterative/task/common/ssh"
)

func NewKeyPair(client *client.Client, identifier common.Identifier) *KeyPair {
	k := new(KeyPair)
	k.Client = client
	k.Identifier = identifier.Long()
	return k
}

type KeyPair struct {
	Client     *client.Client
	Identifier string
	Attributes ssh.DeterministicSSHKeyPair
	Resource   *types.KeyPairInfo
}

func (k *KeyPair) Create(ctx context.Context) error {
	keyPair, err := k.Client.GetKeyPair(ctx)
	if err != nil {
		return err
	}
	k.Attributes = *keyPair

	publicKey, err := k.Attributes.PublicString()
	if err != nil {
		return err
	}

	input := ec2.ImportKeyPairInput{
		KeyName:           aws.String(k.Identifier),
		PublicKeyMaterial: []byte(strings.TrimSpace(publicKey) + " host\n"),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeKeyPair,
				Tags:         makeTagSlice(k.Identifier, k.Client.Tags),
			},
		},
	}

	pair, err := k.Client.Services.EC2.ImportKeyPair(ctx, &input)
	if err != nil {
		var e smithy.APIError
		if errors.As(err, &e) && e.ErrorCode() == "InvalidKeyPair.Duplicate" {
			return k.Read(ctx)
		}
		return err
	}

	waitInput := ec2.DescribeKeyPairsInput{
		KeyPairIds: []string{aws.ToString(pair.KeyPairId)},
	}

	if err := ec2.NewKeyPairExistsWaiter(k.Client.Services.EC2).Wait(ctx, &waitInput, k.Client.Cloud.Timeouts.Create); err != nil {
		return err
	}

	return k.Read(ctx)
}

func (k *KeyPair) Read(ctx context.Context) error {
	pair, err := k.Client.GetKeyPair(ctx)
	if err != nil {
		return err
	}
	k.Attributes = *pair

	input := ec2.DescribeKeyPairsInput{
		KeyNames: []string{k.Identifier},
	}

	pairs, err := k.Client.Services.EC2.DescribeKeyPairs(ctx, &input)
	if err != nil {
		var e smithy.APIError
		if errors.As(err, &e) && e.ErrorCode() == "InvalidKeyPair.NotFound" {
			return common.NotFoundError
		}
	}

	k.Resource = &pairs.KeyPairs[0]
	return nil
}

func (k *KeyPair) Update(ctx context.Context) error {
	return common.NotImplementedError
}

func (k *KeyPair) Delete(ctx context.Context) error {
	input := ec2.DeleteKeyPairInput{
		KeyName: aws.String(k.Identifier),
	}

	if _, err := k.Client.Services.EC2.DeleteKeyPair(ctx, &input); err != nil {
		return err
	}

	k.Resource = nil
	return nil
}
