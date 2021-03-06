package node_disconnect_lib

import (
	"e2e-basic/common"
	"fmt"
	"os/exec"
	"time"

	. "github.com/onsi/gomega"
)

var (
	defTimeoutSecs           = "90s"
	disconnectionTimeoutSecs = "90s"
	repairTimeoutSecs        = "90s"
)

// disconnect a node from the other nodes in the cluster
func DisconnectNode(vmname string, otherNodes []string, method string) {
	for _, targetIP := range otherNodes {
		cmd := exec.Command("bash", "../lib/io_connect_node.sh", vmname, targetIP, "DISCONNECT", method)
		cmd.Dir = "./"
		_, err := cmd.CombinedOutput()
		Expect(err).ToNot(HaveOccurred())
	}
}

// reconnect a node to the other nodes in the cluster
func ReconnectNode(vmname string, otherNodes []string, checkError bool, method string) {
	for _, targetIP := range otherNodes {
		cmd := exec.Command("bash", "../lib/io_connect_node.sh", vmname, targetIP, "RECONNECT", method)
		cmd.Dir = "./"
		_, err := cmd.CombinedOutput()
		if checkError {
			Expect(err).ToNot(HaveOccurred())
		}
	}
}

// return the node name to isolate and a vector of IP addresses to isolate
func GetNodes(uuid string) (string, []string) {
	nodeList, err := common.GetNodeLocs()
	Expect(err).ToNot(HaveOccurred())

	var nodeToIsolate = ""
	nexusNode, replicas := common.GetMsvNodes(uuid)
	Expect(nexusNode).NotTo(Equal(""))
	fmt.Printf("nexus node is \"%s\"\n", nexusNode)

	var otherAddresses []string

	// find a node which is not the nexus and is a replica
	for _, node := range replicas {
		if node != nexusNode {
			nodeToIsolate = node
			break
		}
	}
	Expect(nodeToIsolate).NotTo(Equal(""))

	// get a list of the other ip addresses in the cluster
	for _, node := range nodeList {
		if node.NodeName != nodeToIsolate {
			otherAddresses = append(otherAddresses, node.IPAddress)
		}
	}
	Expect(len(otherAddresses)).To(BeNumerically(">", 0))

	fmt.Printf("node to isolate is \"%s\"\n", nodeToIsolate)
	return nodeToIsolate, otherAddresses
}

// Run fio against the cluster while a replica is being removed and reconnected to the network
func LossTest(nodeToIsolate string, otherNodes []string, disconnectionMethod string, uuid string) {
	fmt.Printf("running spawned fio\n")
	go common.RunFio("fio", 20)

	time.Sleep(5 * time.Second)
	fmt.Printf("disconnecting \"%s\"\n", nodeToIsolate)

	DisconnectNode(nodeToIsolate, otherNodes, disconnectionMethod)

	fmt.Printf("waiting up to %s for disconnection to affect the nexus\n", disconnectionTimeoutSecs)
	Eventually(func() string {
		return common.GetMsvState(uuid)
	},
		disconnectionTimeoutSecs, // timeout
		"1s",                     // polling interval
	).Should(Equal("degraded"))

	fmt.Printf("volume is in state \"%s\"\n", common.GetMsvState(uuid))

	fmt.Printf("running fio while node is disconnected\n")
	common.RunFio("fio", 20)

	fmt.Printf("reconnecting \"%s\"\n", nodeToIsolate)
	ReconnectNode(nodeToIsolate, otherNodes, true, disconnectionMethod)

	fmt.Printf("running fio when node is reconnected\n")
	common.RunFio("fio", 20)
}

// Remove the replica without running IO and verify that the volume becomes degraded but is still functional
func LossWhenIdleTest(nodeToIsolate string, otherNodes []string, disconnectionMethod string, uuid string) {
	fmt.Printf("disconnecting \"%s\"\n", nodeToIsolate)

	DisconnectNode(nodeToIsolate, otherNodes, disconnectionMethod)

	fmt.Printf("waiting up to %s for disconnection to affect the nexus\n", disconnectionTimeoutSecs)
	Eventually(func() string {
		return common.GetMsvState(uuid)
	},
		disconnectionTimeoutSecs, // timeout
		"1s",                     // polling interval
	).Should(Equal("degraded"))

	fmt.Printf("volume is in state \"%s\"\n", common.GetMsvState(uuid))

	fmt.Printf("running fio while node is disconnected\n")
	common.RunFio("fio", 20)

	fmt.Printf("reconnecting \"%s\"\n", nodeToIsolate)
	ReconnectNode(nodeToIsolate, otherNodes, true, disconnectionMethod)

	fmt.Printf("running fio when node is reconnected\n")
	common.RunFio("fio", 20)
}

// Run fio against the cluster while a replica node is being removed,
// wait for the volume to become degraded, then wait for it to be repaired.
// Run fio against repaired volume, and again after node is reconnected.
func ReplicaReassignTest(nodeToIsolate string, otherNodes []string, disconnectionMethod string, uuid string) {
	// This test needs at least 4 nodes, a refuge node, a mayastor node to isolate, and 2 other mayastor nodes
	Expect(len(otherNodes)).To(BeNumerically(">=", 3))

	fmt.Printf("running spawned fio\n")
	go common.RunFio("fio", 20)

	time.Sleep(5 * time.Second)
	fmt.Printf("disconnecting \"%s\"\n", nodeToIsolate)

	DisconnectNode(nodeToIsolate, otherNodes, disconnectionMethod)

	fmt.Printf("waiting up to %s for disconnection to affect the nexus\n", disconnectionTimeoutSecs)
	Eventually(func() string {
		return common.GetMsvState(uuid)
	},
		disconnectionTimeoutSecs, // timeout
		"1s",                     // polling interval
	).Should(Equal("degraded"))

	fmt.Printf("volume is in state \"%s\"\n", common.GetMsvState(uuid))

	fmt.Printf("waiting up to %s for the volume to be repaired\n", repairTimeoutSecs)
	Eventually(func() string {
		return common.GetMsvState(uuid)
	},
		repairTimeoutSecs, // timeout
		"1s",              // polling interval
	).Should(Equal("healthy"))

	fmt.Printf("volume is in state \"%s\"\n", common.GetMsvState(uuid))

	fmt.Printf("running fio while node is disconnected\n")
	common.RunFio("fio", 20)

	fmt.Printf("reconnecting \"%s\"\n", nodeToIsolate)
	ReconnectNode(nodeToIsolate, otherNodes, true, disconnectionMethod)

	fmt.Printf("running fio when node is reconnected\n")
	common.RunFio("fio", 20)
}

// Common steps required when setting up the test
func Setup(pvc_name string, storage_class_name string) string {
	uuid := common.MkPVC(fmt.Sprintf(pvc_name), storage_class_name)

	CreateFioOnRefugeNode("fio", pvc_name)

	fmt.Printf("waiting for fio\n")
	Eventually(func() bool {
		return common.FioReadyPod()
	},
		defTimeoutSecs, // timeout
		"1s",           // polling interval
	).Should(Equal(true))
	return uuid
}

// Common steps required when tearing down the test
func Teardown(pvcName string, storageClassName string) {

	fmt.Printf("removing fio pod\n")
	err := common.DeletePod("fio")
	Expect(err).ToNot(HaveOccurred())

	fmt.Printf("removing pvc\n")
	common.RmPVC(fmt.Sprintf(pvcName), storageClassName)
}

// Deploy an instance of fio on a node labelled as "podrefuge"
func CreateFioOnRefugeNode(podName string, vol_claim_name string) {
	podObj := common.CreateFioPodDef(podName, vol_claim_name)
	common.ApplyNodeSelectorToPodObject(podObj, "openebs.io/podrefuge", "true")
	_, err := common.CreatePod(podObj)
	Expect(err).ToNot(HaveOccurred())
}
