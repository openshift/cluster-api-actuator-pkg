package framework

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	CAPINamespace = "openshift-cluster-api"
	MAPINamespace = "openshift-machine-api"

	RetryShort  = 1 * time.Second
	RetryMedium = 5 * time.Second
)

var (
	WaitShort    = 1 * time.Minute
	WaitMedium   = 3 * time.Minute
	WaitLong     = 15 * time.Minute
	WaitOverLong = 30 * time.Minute

	ctx = context.Background()
)

// DeleteObjects deletes the objects in the given list.
func DeleteObjects(cl client.Client, objs ...client.Object) {
	for _, o := range objs {
		By(fmt.Sprintf("Deleting %s/%s", o.GetObjectKind().GroupVersionKind().Kind, o.GetName()))
		Expect(cl.Delete(ctx, o)).To(Succeed())
	}
}
