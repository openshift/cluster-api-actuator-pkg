package e2e

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Machine Suite")
}

var _ = BeforeSuite(func() {
	Expect(InitializeSuite()).To(Succeed())
})
