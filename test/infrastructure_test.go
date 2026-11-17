package test

import (
	"context"
	"os"
	"sort"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	awstt "github.com/gruntwork-io/terratest/modules/aws"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Expected values, taken from infrastructure/00-params.tf.
const (
	expectedVpcCidr        = "192.168.0.0/16"
	expectedHTTPSubnetCidr = "192.168.1.0/24"
	expectedDBSubnetCidr   = "192.168.2.0/24"
	expectedInstanceType   = "t3.micro"
	expectedHTTPInstances  = 2
	expectedDBInstances    = 3
)

// TestInfrastructure runs against infrastructure that the pipeline has already applied.
// It only reads Terraform outputs and AWS APIs; it never creates or destroys anything.
func TestInfrastructure(t *testing.T) {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "eu-north-1"
	}

	opts := &terraform.Options{TerraformDir: "../infrastructure", NoColor: true}

	// Outputs are maps of instance ID => private IP.
	httpOutput := terraform.OutputMap(t, opts, "http_ip")
	dbOutput := terraform.OutputMap(t, opts, "db_ip")

	ec2Client := awstt.NewEc2Client(t, region)
	httpInstances := describeInstances(t, ec2Client, keys(httpOutput))
	dbInstances := describeInstances(t, ec2Client, keys(dbOutput))

	t.Run("EC2 instances and database are created", func(t *testing.T) {
		assert.Len(t, httpInstances, expectedHTTPInstances, "number of HTTP instances")
		assert.Len(t, dbInstances, expectedDBInstances, "number of DB instances")

		for id, inst := range mergeInstances(httpInstances, dbInstances) {
			assert.Equal(t, types.InstanceStateNameRunning, inst.State.Name, "instance %s state", id)
			assert.Equal(t, expectedInstanceType, string(inst.InstanceType), "instance %s type", id)
		}
		for id, inst := range httpInstances {
			assert.Equal(t, httpOutput[id], aws.ToString(inst.PrivateIpAddress), "HTTP instance %s private IP", id)
		}
		for id, inst := range dbInstances {
			assert.Equal(t, dbOutput[id], aws.ToString(inst.PrivateIpAddress), "DB instance %s private IP", id)
		}
	})

	t.Run("VPC and subnets have the correct CIDR blocks", func(t *testing.T) {
		vpcID := ""
		for id, inst := range mergeInstances(httpInstances, dbInstances) {
			if vpcID == "" {
				vpcID = aws.ToString(inst.VpcId)
			}
			assert.Equal(t, vpcID, aws.ToString(inst.VpcId), "instance %s is in a different VPC", id)
		}
		require.NotEmpty(t, vpcID, "VPC ID")

		vpc := awstt.GetVpcById(t, vpcID, region)
		assert.Equal(t, expectedVpcCidr, aws.ToString(vpc.CidrBlock), "VPC CIDR")

		subnetCidrs := map[string]string{}
		for _, subnet := range awstt.GetSubnetsForVpc(t, vpcID, region) {
			subnetCidrs[subnet.Id] = subnet.CidrBlock
		}
		assert.Len(t, subnetCidrs, 2, "number of subnets in the VPC")
		assert.ElementsMatch(t, []string{expectedHTTPSubnetCidr, expectedDBSubnetCidr}, values(subnetCidrs), "subnet CIDRs")

		for id, inst := range httpInstances {
			assert.Equal(t, expectedHTTPSubnetCidr, subnetCidrs[aws.ToString(inst.SubnetId)], "HTTP instance %s subnet", id)
		}
		for id, inst := range dbInstances {
			assert.Equal(t, expectedDBSubnetCidr, subnetCidrs[aws.ToString(inst.SubnetId)], "DB instance %s subnet", id)
		}
	})

	t.Run("Database is not accessible from the internet", func(t *testing.T) {
		for id, inst := range dbInstances {
			assert.Empty(t, aws.ToString(inst.PublicIpAddress), "DB instance %s must not have a public IP", id)
			assert.Empty(t, aws.ToString(inst.PublicDnsName), "DB instance %s must not have a public DNS name", id)
		}

		addresses, err := ec2Client.DescribeAddresses(context.Background(), &ec2.DescribeAddressesInput{
			Filters: []types.Filter{{Name: aws.String("instance-id"), Values: keys(dbOutput)}},
		})
		require.NoError(t, err)
		assert.Empty(t, addresses.Addresses, "DB instances must not have Elastic IPs")
	})
}

func describeInstances(t *testing.T, client *ec2.Client, ids []string) map[string]types.Instance {
	t.Helper()
	require.NotEmpty(t, ids, "no instance IDs in Terraform output")

	out, err := client.DescribeInstances(context.Background(), &ec2.DescribeInstancesInput{InstanceIds: ids})
	require.NoError(t, err)

	instances := map[string]types.Instance{}
	for _, reservation := range out.Reservations {
		for _, inst := range reservation.Instances {
			instances[aws.ToString(inst.InstanceId)] = inst
		}
	}
	return instances
}

func mergeInstances(maps ...map[string]types.Instance) map[string]types.Instance {
	all := map[string]types.Instance{}
	for _, m := range maps {
		for id, inst := range m {
			all[id] = inst
		}
	}
	return all
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func values(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}
