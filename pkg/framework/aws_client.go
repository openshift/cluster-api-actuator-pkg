package framework

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"k8s.io/klog"
	"k8s.io/utils/ptr"
)

// AwsClient struct.
type AwsClient struct {
	Svc *ec2.EC2
}

// CreateCapacityReservation Create CapacityReservation.
func (a *AwsClient) CreateCapacityReservation(instanceType string, instancePlatform string, availabilityZone string, instanceCount int64) (string, error) {
	input := &ec2.CreateCapacityReservationInput{
		InstanceType:          aws.String(instanceType),
		InstancePlatform:      aws.String(instancePlatform),
		AvailabilityZone:      aws.String(availabilityZone),
		InstanceCount:         aws.Int64(instanceCount),
		InstanceMatchCriteria: aws.String("targeted"),
		EndDateType:           aws.String("unlimited"),
	}
	result, err := a.Svc.CreateCapacityReservation(input)

	if err != nil {
		return "", fmt.Errorf("error creating capacity reservation: %w", err)
	}

	capacityReservationID := ptr.Deref(result.CapacityReservation.CapacityReservationId, "")
	klog.Infof("The created capacityReservationID is %s", capacityReservationID)

	return capacityReservationID, err
}

// CancelCapacityReservation Cancel a CapacityReservation.
func (a *AwsClient) CancelCapacityReservation(capacityReservationID string) (bool, error) {
	input := &ec2.CancelCapacityReservationInput{
		CapacityReservationId: aws.String(capacityReservationID),
	}
	result, err := a.Svc.CancelCapacityReservation(input)

	return ptr.Deref(result.Return, false), err
}
