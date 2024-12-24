package framework

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/kms"
	"k8s.io/klog"
	"k8s.io/utils/ptr"
)

// AwsClient struct.
type AwsClient struct {
	svc *ec2.EC2
}

// Init the aws client.
func NewAwsClient(accessKeyID []byte, secureKey []byte, clusterRegion string) *AwsClient {
	awsSession := newAwsSession(accessKeyID, secureKey, clusterRegion)
	aClient := &AwsClient{
		svc: ec2.New(awsSession),
	}

	return aClient
}

// AwsKmsClient struct.
type AwsKmsClient struct {
	kmssvc *kms.KMS
}

// Init the aws kms client.
func NewAwsKmsClient(accessKeyID []byte, secureKey []byte, clusterRegion string) *AwsKmsClient {
	awsSession := newAwsSession(accessKeyID, secureKey, clusterRegion)
	kmsClient := &AwsKmsClient{
		kmssvc: kms.New(awsSession),
	}

	return kmsClient
}

// Create aws backend session connection.
func newAwsSession(accessKeyID []byte, secureKey []byte, clusterRegion string) *session.Session {
	awsConfig := &aws.Config{
		Region: aws.String(clusterRegion),
		Credentials: credentials.NewStaticCredentials(
			string(accessKeyID),
			string(secureKey),
			"",
		),
	}

	return session.Must(session.NewSession(awsConfig))
}

// CreateCapacityReservation Create CapacityReservation.
func (a *AwsClient) CreateCapacityReservation(instanceType string, instancePlatform string, availabilityZone string, instanceCount int64) (string, error) {
	input := &ec2.CreateCapacityReservationInput{
		InstanceType:          aws.String(instanceType),
		InstancePlatform:      aws.String(instancePlatform),
		AvailabilityZone:      aws.String(availabilityZone),
		InstanceCount:         aws.Int64(instanceCount),
		InstanceMatchCriteria: aws.String("targeted"),
		EndDateType:           aws.String("limited"),
		EndDate:               timePtr(time.Now().Add(35 * time.Minute)),
	}
	result, err := a.svc.CreateCapacityReservation(input)

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
	result, err := a.svc.CancelCapacityReservation(input)

	return ptr.Deref(result.Return, false), err
}

// CreatePlacementGroup Create a PlacementGroup.
func (a *AwsClient) CreatePlacementGroup(groupName string, strategy string, partitionCount ...int64) (string, error) {
	var input *ec2.CreatePlacementGroupInput
	if len(partitionCount) > 0 {
		input = &ec2.CreatePlacementGroupInput{
			GroupName:      aws.String(groupName),
			PartitionCount: aws.Int64(partitionCount[0]),
			Strategy:       aws.String(strategy),
		}
	} else {
		input = &ec2.CreatePlacementGroupInput{
			GroupName: aws.String(groupName),
			Strategy:  aws.String(strategy),
		}
	}

	result, err := a.svc.CreatePlacementGroup(input)

	if err != nil {
		return "", fmt.Errorf("error creating placement group: %w", err)
	}

	placementGroupID := ptr.Deref(result.PlacementGroup.GroupId, "")
	klog.Infof("The created placementGroupID is %s", placementGroupID)

	return placementGroupID, nil
}

// DeletePlacementGroup Delete a PlacementGroup.
func (a *AwsClient) DeletePlacementGroup(groupName string) (string, error) {
	input := &ec2.DeletePlacementGroupInput{
		GroupName: aws.String(groupName),
	}
	result, err := a.svc.DeletePlacementGroup(input)

	if err != nil {
		return "", fmt.Errorf("could not delete placement group: %w", err)
	}

	return result.String(), nil
}

// Describes aws customer managed kms key info.
func (akms *AwsKmsClient) DescribeKeyByID(kmsKeyID string) (string, error) {
	input := &kms.DescribeKeyInput{
		KeyId: aws.String(kmsKeyID),
	}
	result, err := akms.kmssvc.DescribeKey(input)

	if err != nil {
		return "", fmt.Errorf("could not get the key: %w", err)
	}

	return result.String(), nil
}

// CreateKey create a key.
func (akms *AwsKmsClient) CreateKey(description string) (string, error) {
	createRes, err := akms.kmssvc.CreateKey(&kms.CreateKeyInput{
		Description: aws.String(description),
	})
	if err != nil {
		klog.Infof("Error creating key %s", err.Error())
		return "", err
	}

	klog.Infof("key created: %s", *createRes.KeyMetadata.Arn)

	return *createRes.KeyMetadata.Arn, nil
}

// DeleteKey delete a key.
func (akms *AwsKmsClient) DeleteKey(key string) error {
	_, err := akms.kmssvc.ScheduleKeyDeletion(&kms.ScheduleKeyDeletionInput{
		KeyId:               aws.String(key),
		PendingWindowInDays: aws.Int64(7),
	})

	return err
}

func timePtr(t time.Time) *time.Time {
	return &t
}
