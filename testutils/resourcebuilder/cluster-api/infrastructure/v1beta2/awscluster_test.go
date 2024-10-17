/*
Copyright 2024 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta2

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capav1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

var _ = Describe("AWSCluster", func() {
	Describe("Build", func() {
		It("should return a default AWSCluster when no options are specified", func() {
			awsCluster := AWSCluster().Build()
			Expect(awsCluster).ToNot(BeNil())
			Expect(awsCluster.TypeMeta.APIVersion).To(Equal("infrastructure.cluster.x-k8s.io/v1beta2"))
			Expect(awsCluster.TypeMeta.Kind).To(Equal("AWSCluster"))
		})
	})

	// Object meta fields.

	Describe("WithAnnotations", func() {
		It("should return the custom value when specified", func() {
			annotations := map[string]string{"key": "value"}
			awsCluster := AWSCluster().WithAnnotations(annotations).Build()
			Expect(awsCluster.Annotations).To(Equal(annotations))
		})
	})

	Describe("WithCreationTimestamp", func() {
		It("should return the custom value when specified", func() {
			timestamp := metav1.Now()
			awsCluster := AWSCluster().WithCreationTimestamp(timestamp).Build()
			Expect(awsCluster.CreationTimestamp).To(Equal(timestamp))
		})
	})

	Describe("WithDeletionTimestamp", func() {
		It("should return the custom value when specified", func() {
			timestamp := metav1.Now()
			awsCluster := AWSCluster().WithDeletionTimestamp(&timestamp).Build()
			Expect(awsCluster.DeletionTimestamp).To(Equal(&timestamp))
		})
	})

	Describe("WithGenerateName", func() {
		It("should return the custom value when specified", func() {
			generateName := "test-"
			awsCluster := AWSCluster().WithGenerateName(generateName).Build()
			Expect(awsCluster.GenerateName).To(Equal(generateName))
		})
	})

	Describe("WithLabels", func() {
		It("should return the custom value when specified", func() {
			labels := map[string]string{"key": "value"}
			awsCluster := AWSCluster().WithLabels(labels).Build()
			Expect(awsCluster.Labels).To(Equal(labels))
		})
	})

	Describe("WithName", func() {
		It("should return the custom value when specified", func() {
			name := "test-aws-cluster"
			awsCluster := AWSCluster().WithName(name).Build()
			Expect(awsCluster.Name).To(Equal(name))
		})
	})

	Describe("WithNamespace", func() {
		It("should return the custom value when specified", func() {
			awsCluster := AWSCluster().WithNamespace("ns").Build()
			Expect(awsCluster.Namespace).To(Equal("ns"))
		})
	})

	// Spec fields.

	Describe("WithAdditionalTags", func() {
		It("should return the custom value when specified", func() {
			tags := capav1.Tags{"key": "value"}
			awsCluster := AWSCluster().WithAdditionalTags(tags).Build()
			Expect(awsCluster.Spec.AdditionalTags).To(Equal(tags))
		})
	})

	Describe("WithBastion", func() {
		It("should return the custom value when specified", func() {
			bastion := capav1.Bastion{AllowedCIDRBlocks: []string{"10.0.0.0/16"}}
			awsCluster := AWSCluster().WithBastion(bastion).Build()
			Expect(awsCluster.Spec.Bastion).To(Equal(bastion))
		})
	})

	Describe("WithControlPlaneEndpoint", func() {
		It("should return the custom value when specified", func() {
			endpoint := clusterv1.APIEndpoint{Host: "example.com", Port: 6443}
			awsCluster := AWSCluster().WithControlPlaneEndpoint(endpoint).Build()
			Expect(awsCluster.Spec.ControlPlaneEndpoint).To(Equal(endpoint))
		})
	})

	Describe("WithControlPlaneLoadBalancer", func() {
		It("should return the custom value when specified", func() {
			lb := &capav1.AWSLoadBalancerSpec{Scheme: &capav1.ELBSchemeInternal}
			awsCluster := AWSCluster().WithControlPlaneLoadBalancer(lb).Build()
			Expect(awsCluster.Spec.ControlPlaneLoadBalancer).To(Equal(lb))
		})
	})

	Describe("WithIdentityRef", func() {
		It("should return the custom value when specified", func() {
			identityRef := &capav1.AWSIdentityReference{Kind: "AWSClusterRoleIdentity", Name: "test-identity"}
			awsCluster := AWSCluster().WithIdentityRef(identityRef).Build()
			Expect(awsCluster.Spec.IdentityRef).To(Equal(identityRef))
		})
	})

	Describe("WithImageLookupBaseOS", func() {
		It("should return the custom value when specified", func() {
			baseOS := "RHEL"
			awsCluster := AWSCluster().WithImageLookupBaseOS(baseOS).Build()
			Expect(awsCluster.Spec.ImageLookupBaseOS).To(Equal(baseOS))
		})
	})

	Describe("WithImageLookupFormat", func() {
		It("should return the custom value when specified", func() {
			awsCluster := AWSCluster().WithImageLookupFormat("{{.BaseOS}}-{{.K88sVersion}}-*").Build()
			Expect(awsCluster.Spec.ImageLookupFormat).To(Equal("{{.BaseOS}}-{{.K88sVersion}}-*"))
		})
	})

	Describe("WithImageLookupOrg", func() {
		It("should return the custom value when specified", func() {
			org := "123456789012"
			awsCluster := AWSCluster().WithImageLookupOrg(org).Build()
			Expect(awsCluster.Spec.ImageLookupOrg).To(Equal(org))
		})
	})

	Describe("WithNetworkSpec", func() {
		It("should return the custom value when specified", func() {
			networkSpec := capav1.NetworkSpec{VPC: capav1.VPCSpec{ID: "vpc-12345"}}
			awsCluster := AWSCluster().WithNetworkSpec(networkSpec).Build()
			Expect(awsCluster.Spec.NetworkSpec).To(Equal(networkSpec))
		})
	})

	Describe("WithPartition", func() {
		It("should return the custom value when specified", func() {
			partition := "aws-cn"
			awsCluster := AWSCluster().WithPartition(partition).Build()
			Expect(awsCluster.Spec.Partition).To(Equal(partition))
		})
	})

	Describe("WithRegion", func() {
		It("should return the custom value when specified", func() {
			region := "us-west-2"
			awsCluster := AWSCluster().WithRegion(region).Build()
			Expect(awsCluster.Spec.Region).To(Equal(region))
		})
	})

	Describe("WithS3Bucket", func() {
		It("should return the custom value when specified", func() {
			s3Bucket := &capav1.S3Bucket{Name: "test-bucket"}
			awsCluster := AWSCluster().WithS3Bucket(s3Bucket).Build()
			Expect(awsCluster.Spec.S3Bucket).To(Equal(s3Bucket))
		})
	})

	Describe("WithSecondaryControlPlaneLoadBalancer", func() {
		It("should return the custom value when specified", func() {
			lb := &capav1.AWSLoadBalancerSpec{Scheme: &capav1.ELBSchemeInternal}
			awsCluster := AWSCluster().WithSecondaryControlPlaneLoadBalancer(lb).Build()
			Expect(awsCluster.Spec.SecondaryControlPlaneLoadBalancer).To(Equal(lb))
		})
	})

	Describe("WithSSHKeyName", func() {
		It("should return the custom value when specified", func() {
			awsCluster := AWSCluster().WithSSHKeyName("kkey").Build()
			Expect(*awsCluster.Spec.SSHKeyName).To(Equal("kkey"))
		})
	})

	// Status fields.

	Describe("WithBastionStatus", func() {
		It("should return the custom value when specified", func() {
			bastionInstance := &capav1.Instance{ID: "i-12345abcdef"}
			awsCluster := AWSCluster().WithBastionStatus(bastionInstance).Build()
			Expect(awsCluster.Status.Bastion).To(Equal(bastionInstance))
		})
	})

	Describe("WithConditions", func() {
		It("should return the custom value when specified", func() {
			conditions := clusterv1.Conditions{{Type: "Ready", Status: "True"}}
			awsCluster := AWSCluster().WithConditions(conditions).Build()
			Expect(awsCluster.Status.Conditions).To(Equal(conditions))
		})
	})

	Describe("WithFailureDomains", func() {
		It("should return the custom value when specified", func() {
			failureDomains := clusterv1.FailureDomains{"us-west-2a": clusterv1.FailureDomainSpec{ControlPlane: true}}
			awsCluster := AWSCluster().WithFailureDomains(failureDomains).Build()
			Expect(awsCluster.Status.FailureDomains).To(Equal(failureDomains))
		})
	})

	Describe("WithNetwork", func() {
		It("should return the custom value when specified", func() {
			network := capav1.NetworkStatus{SecurityGroups: map[capav1.SecurityGroupRole]capav1.SecurityGroup{
				capav1.SecurityGroupControlPlane: {ID: "sg-12345"},
			}}
			awsCluster := AWSCluster().WithNetwork(network).Build()
			Expect(awsCluster.Status.Network).To(Equal(network))
		})
	})

	Describe("WithReady", func() {
		It("should return the custom value when specified", func() {
			ready := true
			awsCluster := AWSCluster().WithReady(ready).Build()
			Expect(awsCluster.Status.Ready).To(Equal(ready))
		})
	})
})
