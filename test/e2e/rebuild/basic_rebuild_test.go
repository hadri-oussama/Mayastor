package basic_rebuild_test

import (
	"e2e-basic/common"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"time"
)

var (
	pvcName      = "rebuild-test-pvc"
	storageClass = "mayastor-nvmf"
)

func basicRebuildTest() {
	pvc, err := common.GetPVC(pvcName)
	Expect(err).To(BeNil())
	Expect(pvc).ToNot(BeNil())

	uuid := string(pvc.ObjectMeta.UID)
	repl, err := common.GetNumReplicas(uuid)
	Expect(err).To(BeNil())
	Expect(repl).Should(BeEquivalentTo(1))

	// Add another child which should kick off a rebuild.
	common.UpdateNumReplicas(uuid, 2)
	repl, err = common.GetNumReplicas(uuid)
	Expect(err).To(BeNil())
	Expect(repl).Should(BeEquivalentTo(2))

	timeout := "90s"
	pollPeriod := "1s"

	// Wait for the added child to show up.
	Eventually(func() int { return common.GetNumChildren(uuid) }, timeout, pollPeriod).Should(BeEquivalentTo(2))

	getChildrenFunc := func(uuid string) []common.NexusChild {
		children, err := common.GetChildren(uuid)
		if err != nil {
			panic("Failed to get children")
		}
		Expect(len(children)).Should(BeEquivalentTo(2))
		return children
	}

	// Check the added child and nexus are both degraded.
	Eventually(func() string { return getChildrenFunc(uuid)[1].State }, timeout, pollPeriod).Should(BeEquivalentTo("CHILD_DEGRADED"))
	Eventually(func() (string, error) { return common.GetNexusState(uuid) }, timeout, pollPeriod).Should(BeEquivalentTo("NEXUS_DEGRADED"))

	// Check everything eventually goes healthy following a rebuild.
	Eventually(func() string { return getChildrenFunc(uuid)[0].State }, timeout, pollPeriod).Should(BeEquivalentTo("CHILD_ONLINE"))
	Eventually(func() string { return getChildrenFunc(uuid)[1].State }, timeout, pollPeriod).Should(BeEquivalentTo("CHILD_ONLINE"))
	Eventually(func() (string, error) { return common.GetNexusState(uuid) }, timeout, pollPeriod).Should(BeEquivalentTo("NEXUS_ONLINE"))
}

func TestRebuild(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Rebuild Test Suite")
}

var _ = Describe("Mayastor rebuild test", func() {
	It("should run a rebuild job to completion", func() {
		basicRebuildTest()
	})
})

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.LoggerTo(GinkgoWriter, true))
	common.SetupTestEnv()
	common.MkPVC(pvcName, storageClass)
	CreateDummyPod()
	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	DestroyDummyPod()
	common.RmPVC(pvcName, storageClass)
	common.TeardownTestEnv()
})

const ApplicationPod = "fio.yaml"

// CreateDummyPod deploys a pod which mounts the PVC but doesn't do anything with it.
// This is to allow a nexus entry to be created in the MSV (because a nexus is only created on volume publish).
func CreateDummyPod() {
	common.ApplyDeployYaml(ApplicationPod)
	// Allow some time for the PVC to be mounted.
	time.Sleep(3 * time.Second)
}

// DestroyDummyPod destroys the dummy pod.
func DestroyDummyPod() {
	common.DeleteDeployYaml(ApplicationPod)
}
