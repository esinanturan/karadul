package mesh

import (
	"fmt"
	"net"
	"sort"
	"sync"
)

// Router resolves a destination IP to the peer that should receive the packet.
type Router struct {
	mu      sync.RWMutex
	manager *Manager

	// exitNode is the peer used as default gateway (0.0.0.0/0), if any.
	exitNode *Peer
}

// NewRouter creates a Router backed by manager.
func NewRouter(manager *Manager) *Router {
	return &Router{manager: manager}
}

// SetExitNode configures an exit node for default-route traffic.
func (r *Router) SetExitNode(p *Peer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.exitNode = p
}

// RoutePacket returns the peer that should receive a packet destined for dstIP.
//
// Lookup order:
//  1. Direct VIP match (100.64.x.x): return peer by VIP.
//  2. Subnet route: longest-prefix-match across all peer advertised routes.
//  3. Exit node (if set): default route.
func (r *Router) RoutePacket(dstIP net.IP) (*Peer, error) {
	// Rule 1: direct VIP match.
	if p, ok := r.manager.GetPeerByVIP(dstIP); ok {
		if p.GetState() != PeerExpired {
			return p, nil
		}
	}

	// Rule 2: longest prefix match across advertised routes.
	peers := r.manager.ListPeers()
	type candidate struct {
		peer   *Peer
		prefix int // prefix length
	}
	var candidates []candidate

	for _, p := range peers {
		if p.GetState() == PeerExpired {
			continue
		}
		p.mu.RLock()
		routes := make([]*net.IPNet, len(p.Routes))
		copy(routes, p.Routes)
		p.mu.RUnlock()

		for _, route := range routes {
			if route.Contains(dstIP) {
				ones, _ := route.Mask.Size()
				candidates = append(candidates, candidate{p, ones})
			}
		}
	}

	if len(candidates) > 0 {
		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].prefix > candidates[j].prefix
		})
		return candidates[0].peer, nil
	}

	// Rule 3: exit node.
	r.mu.RLock()
	exit := r.exitNode
	r.mu.RUnlock()
	if exit != nil && exit.GetState() != PeerExpired {
		return exit, nil
	}

	return nil, fmt.Errorf("no route to %s", dstIP)
}
