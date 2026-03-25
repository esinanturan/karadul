package mesh

import (
	"fmt"
	"net"
	"testing"

	klog "github.com/ersinkoc/karadul/internal/log"
)

// BenchmarkRoutePacket_VIP measures direct VIP lookup (rule 1 in RoutePacket).
func BenchmarkRoutePacket_VIP(b *testing.B) {
	mgr := NewManager(klog.New(nil, klog.LevelError, klog.FormatText), nil)
	defer mgr.Stop()

	var pubKey [32]byte
	pubKey[0] = 1
	vip := net.ParseIP("100.64.0.2")
	mgr.AddOrUpdate(pubKey, "bench-node", "id-bench", vip, "", nil)

	router := NewRouter(mgr)
	target := vip.To4()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := router.RoutePacket(target); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRoutePacket_Subnet measures longest-prefix subnet lookup (rule 2).
func BenchmarkRoutePacket_Subnet(b *testing.B) {
	mgr := NewManager(klog.New(nil, klog.LevelError, klog.FormatText), nil)
	defer mgr.Stop()

	var pubKey [32]byte
	pubKey[0] = 2
	vip := net.ParseIP("100.64.0.3")
	mgr.AddOrUpdate(pubKey, "subnet-node", "id-subnet", vip, "", []string{"192.168.0.0/16"})

	router := NewRouter(mgr)
	target := net.ParseIP("192.168.10.1").To4()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := router.RoutePacket(target); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRoutePacket_ManyPeers measures VIP lookup among 100 peers.
func BenchmarkRoutePacket_ManyPeers(b *testing.B) {
	mgr := NewManager(klog.New(nil, klog.LevelError, klog.FormatText), nil)
	defer mgr.Stop()

	const numPeers = 100
	for i := 0; i < numPeers; i++ {
		var pk [32]byte
		pk[0] = byte(i >> 8)
		pk[1] = byte(i)
		vip := net.ParseIP(fmt.Sprintf("100.64.%d.%d", i/256, i%256))
		mgr.AddOrUpdate(pk, fmt.Sprintf("node-%d", i), fmt.Sprintf("id-%d", i), vip, "", nil)
	}

	router := NewRouter(mgr)
	// Route to the last peer added.
	lastVIP := net.ParseIP(fmt.Sprintf("100.64.%d.%d", (numPeers-1)/256, (numPeers-1)%256)).To4()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := router.RoutePacket(lastVIP); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkManager_AddOrUpdate measures the cost of adding/updating a peer.
func BenchmarkManager_AddOrUpdate(b *testing.B) {
	mgr := NewManager(klog.New(nil, klog.LevelError, klog.FormatText), nil)
	defer mgr.Stop()

	var pk [32]byte
	vip := net.ParseIP("100.64.1.1")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pk[0] = byte(i)
		mgr.AddOrUpdate(pk, "bench", "id", vip, "", nil)
	}
}
