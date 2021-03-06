package node_disconnect_nvmf_reject_idle_test

import (
	"e2e-basic/common"
	disconnect_lib "e2e-basic/node_disconnect/lib"

	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	g_nodeToIsolate    = ""
	g_otherNodes       []string
	g_uuid             = ""
	g_disconnectMethod = "REJECT"
)

func lossTest() {
	g_nodeToIsolate, g_otherNodes = disconnect_lib.GetNodes(g_uuid)
	disconnect_lib.LossWhenIdleTest(g_nodeToIsolate, g_otherNodes, g_disconnectMethod, g_uuid)
}

func TestNodeLoss(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Node Loss NVMF reject when idle")
}

var _ = Describe("Mayastor node loss test", func() {
	It("should verify nvmf nexus behaviour when a node becomes inaccessible when no IO is received (iptables REJECT)", func() {
		lossTest()
	})
})

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.LoggerTo(GinkgoWriter, true))
	common.SetupTestEnv()
	g_uuid = disconnect_lib.Setup("loss-test-pvc-nvmf", "mayastor-nvmf-2")
	close(done)
}, 60)

var _ = AfterSuite(func() {
	// NB This only tears down the local structures for talking to the cluster,
	// not the kubernetes cluster itself.
	By("tearing down the test environment")

	// ensure node is reconnected in the event of a test failure
	disconnect_lib.ReconnectNode(g_nodeToIsolate, g_otherNodes, false, g_disconnectMethod)
	disconnect_lib.Teardown("loss-test-pvc-nvmf", "mayastor-nvmf-2")
	common.TeardownTestEnv()
})
