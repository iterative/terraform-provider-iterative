package resources

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// makeTagSlice creates an `[]ec2/types.Tag` slice of structs from the given
// `name` and `map[string]string` of tags, using the former to create or
// overwrite the `Name` tag in the latter.
// See also https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/Using_Tags.html
func makeTagSlice(name string, tags map[string]string) []types.Tag {
	if tags == nil {
		tags = make(map[string]string)
	}

	tags["Name"] = name

	var result []types.Tag
	for key, value := range tags {
		result = append(result, types.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}

	return result
}
