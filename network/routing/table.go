/*
 * The Clear BSD License
 *
 * Copyright (c) 2019 Insolar Technologies
 *
 * All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without modification, are permitted (subject to the limitations in the disclaimer below) provided that the following conditions are met:
 *
 *  Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.
 *  Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.
 *  Neither the name of Insolar Technologies nor the names of its contributors may be used to endorse or promote products derived from this software without specific prior written permission.
 *
 * NO EXPRESS OR IMPLIED LICENSES TO ANY PARTY'S PATENT RIGHTS ARE GRANTED BY THIS LICENSE. THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 *
 */

package routing

import (
	"strconv"

	"github.com/insolar/insolar/core"
	"github.com/insolar/insolar/log"
	"github.com/insolar/insolar/network"
	"github.com/insolar/insolar/network/transport/host"
	"github.com/pkg/errors"
)

type Table struct {
	NodeKeeper network.NodeKeeper
}

func (t *Table) isLocalNode(core.RecordRef) bool {
	return true
}

func (t *Table) resolveRemoteNode(ref core.RecordRef) (*host.Host, error) {
	return nil, errors.New("not implemented")
}

func (t *Table) addRemoteHost(h *host.Host) {
	log.Warn("not implemented")
}

// Resolve NodeID -> ShortID, Address. Can initiate network requests.
func (t *Table) Resolve(ref core.RecordRef) (*host.Host, error) {
	if t.isLocalNode(ref) {
		node := t.NodeKeeper.GetActiveNode(ref)
		if node == nil {
			return nil, errors.New("no such local node with NodeID: " + ref.String())
		}
		return host.NewHostNS(node.PhysicalAddress(), node.ID(), node.ShortID())
	}
	return t.resolveRemoteNode(ref)
}

// ResolveS ShortID -> NodeID, Address for node inside current globe.
func (t *Table) ResolveS(id core.ShortNodeID) (*host.Host, error) {
	node := t.NodeKeeper.GetActiveNodeByShortID(id)
	if node == nil {
		return nil, errors.New("no such local node with ShortID: " + strconv.FormatUint(uint64(id), 10))
	}
	return host.NewHostNS(node.PhysicalAddress(), node.ID(), node.ShortID())
}

// AddToKnownHosts add host to routing table.
func (t *Table) AddToKnownHosts(h *host.Host) {
	if t.isLocalNode(h.NodeID) {
		// we should already have this node in NodeNetwork active list, do nothing
		return
	}
	t.addRemoteHost(h)
}

// GetRandomNodes get a specified number of random nodes. Returns less if there are not enough nodes in network.
func (t *Table) GetRandomNodes(count int) []host.Host {
	// TODO: this workaround returns all nodes
	nodes := t.NodeKeeper.GetActiveNodes()
	result := make([]host.Host, 0)
	for _, n := range nodes {
		address, err := host.NewAddress(n.PhysicalAddress())
		if err != nil {
			log.Error(err)
			continue
		}
		result = append(result, host.Host{NodeID: n.ID(), Address: address})
	}

	// TODO: original implementation
	/*
		// not so random for now
		nodes := t.NodeKeeper.GetActiveNodes()
		//return nodes
		resultCount := count
		if count > len(nodes) {
			resultCount = len(nodes)
		}
		result := make([]host.Host, 0)
		for i := 0; i < resultCount; i++ {
			address, err := host.NewAddress(nodes[i].PhysicalAddress())
			if err != nil {
				log.Error(err)
				continue
			}
			h := host.Host{NodeID: nodes[i].ID(), Address: address}
			result = append(result, h)
		}
	*/
	return result
}

// Rebalance recreate shards of routing table with known hosts according to new partition policy.
func (t *Table) Rebalance(network.PartitionPolicy) {
	log.Warn("not implemented")
}

func (t *Table) Inject(nodeKeeper network.NodeKeeper) {
	t.NodeKeeper = nodeKeeper
}
