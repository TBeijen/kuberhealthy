// Copyright 2018 Comcast Cable Communications Management, LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//     http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"errors"

	log "github.com/sirupsen/logrus"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var ()

const (
	fieldSelectorCordoned = "spec.unschedulable=true"
)

type NodeListResult struct {
	Err      error
	NodeList []v1.Node
}

// listTargetedNodes returns a list of targeted nodes depending on the
// given user inputs on what characteristics make a node qualify for this check.
func listTargetedNodes(ctx context.Context) chan NodeListResult {
	nodesChan := make(chan NodeListResult)

	go func() {
		log.Infoln("Listing qualifying targeted nodes.")

		defer close(nodesChan)

		result := NodeListResult{}

		// Serialize label inputs.
		// This is here just for debugging purposes for now.
		// TODO: Serializing labels has some additional logic to it -- you cannot parse a string of
		// comma-separated values ("a=x,b=y")
		// https://github.com/kubernetes/apimachinery/issues/47
		// https://pkg.go.dev/k8s.io/apimachinery@v0.21.0/pkg/labels#pkg-overview
		// selectors, err := labels.Parse(checkNodeSelectorsEnv)
		// if err != nil {
		// 	log.Errorf("failed to parse node selectors into selector structs: %v", err)
		// }
		// log.Debugln("Parsed selectors:", selectors)

		// Just use a single label / selector for now.
		// List qualifying nodes for a list of targets.
		nodeList, err := nodeClient.List(ctx, metav1.ListOptions{
			LabelSelector: checkNodeSelectorsEnv,
		})
		if err != nil {
			log.Errorf("failed to list nodes in the cluster: %v\n", err)
			result.Err = err
			nodesChan <- result
			return
		}
		if nodeList == nil {
			result.Err = errors.New("received a nil list of nodes: " + err.Error())
			log.Errorf("failed to list nodes in the cluster: %v\n", err)
			nodesChan <- result
			return
		}

		// Return the list of nodes.
		result.NodeList = nodeList.Items
		nodesChan <- result
		return
	}()

	return nodesChan
}

// listUnschedulableNodes returns a list of unschedulable nodes.
func listUnschedulableNodes(ctx context.Context) chan NodeListResult {
	unschedulableNodeChan := make(chan NodeListResult)

	go func() {
		log.Infoln("Listing unschedulable nodes.")

		defer close(unschedulableNodeChan)

		result := NodeListResult{}

		// List qualifying nodes for a list of targets.
		nodeList, err := nodeClient.List(ctx, metav1.ListOptions{
			FieldSelector: fieldSelectorCordoned,
		})
		if err != nil {
			log.Errorf("failed to list nodes in the cluster: %v\n", err)
			result.Err = err
			unschedulableNodeChan <- result
			return
		}
		if nodeList == nil {
			result.Err = errors.New("received a nil list of nodes: " + err.Error())
			log.Errorf("failed to list nodes in the cluster: %v\n", err)
			unschedulableNodeChan <- result
			return
		}

		// Return the list of nodes.
		result.NodeList = nodeList.Items
		unschedulableNodeChan <- result
		return
	}()

	return unschedulableNodeChan
}

// removeUnscheduableNodes looks at a list of targetable nodes and creates a list of
// targetable nodes that are schedulable (this ignores `Unschedulable` nodes and
// `Cordoned` nodes).
// Returns a list of targetable, schedulable nodes.
func removeUnscheduableNodes(nodes, nodesToRemove *[]v1.Node) *[]*v1.Node {
	schedulableNodes := make([]*v1.Node, 0)

	// Look through the list of targted nodes and remove any that are present in the
	// list of nodse to remove.
	for _, n := range *nodes {
		log.Debugln("Node name:", n.Name, n.GetName())
		if !containsNodeName(*nodesToRemove, &n) {
			log.Debugln("Got back node:", n.Name, n.GetName())
			schedulableNodes = append(schedulableNodes, &n)
		}
	}

	if debug {
		log.Debugln("From removeUnscheduableNodes:")
		log.Debugln("Targeted nodes:")
		for _, n := range schedulableNodes {
			log.Debugln(n.Name, n.GetName())
		}
	}

	return &schedulableNodes
}

// removeUnscheduableNodesv2 looks at a list of targetable nodes and creates a list of
// targetable nodes that are schedulable (this ignores `Unschedulable` nodes and
// `Cordoned` nodes).
// Returns a list of targetable, schedulable nodes.
func removeUnscheduableNodesv2(nodes, nodesToRemove *[]v1.Node) []v1.Node {
	// schedulableNodes := make([]v1.Node, 0)

	// // Look through the list of targted nodes and remove any that are present in the
	// // list of nodse to remove.
	// for _, n := range *nodes {
	// 	log.Debugln("Node name:", n.Name, n.GetName())
	// 	if !containsNodeName(*nodesToRemove, &n) {
	// 		log.Debugln("Got back node:", n.Name, n.GetName())
	// 		schedulableNodes = append(schedulableNodes, &n)
	// 	}
	// }

	// Look through the list of targted nodes and remove any that are present in the
	// list of nodse to remove.
	for i, n := range *nodes {
		// log.Debugln("Node name:", n.Name, n.GetName())
		// if !containsNodeName(*nodesToRemove, &n) {
		// 	schedulableNodes = append(schedulableNodes, &n)
		// }
		if containsNodeName(*nodesToRemove, &n) {
			// schedulableNodes = append(schedulableNodes, &n)
			removeNode(nodes, i)
		}
	}

	// if debug {
	// 	log.Debugln("From removeUnscheduableNodes:")
	// 	log.Debugln("Targeted nodes:")
	// 	for _, n := range *nodes {
	// 		log.Debugln(n.Name, n.GetName())
	// 	}
	// }

	return *nodes
}

// removeNode removes a node from the list.
func removeNode(nodes *[]v1.Node, i int) []v1.Node {
	log.Debugln("Removing node", (*nodes)[i].Name, "from list.")
	(*nodes)[i] = (*nodes)[len(*nodes)-1]
	return (*nodes)[:len(*nodes)-1]
}

// containsNodeName returns a boolean value based on whether or not a slice of strings contains
// a string.
func containsNodeName(list []v1.Node, node *v1.Node) bool {
	// log.Debugln("Looking if", node.Name, "exists in list:")

	for _, n := range list {
		// log.Debugln("    ", n.Name)
		// log.Debugf("    ", n.GetName())
		if n.GetName() == node.GetName() {
			return true
		}
	}
	return false
}

// findNodeInSlice looks for a specified node in a slice based on a given name and returns it.
func findNodeInSlice(nodes []*v1.Node, nodeName string) *v1.Node {
	for _, node := range nodes {
		if node.GetName() == nodeName {
			return node
		}
	}
	return nil
}
