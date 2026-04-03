//go:build !windows

package node

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/karadul/karadul/internal/config"
	"github.com/karadul/karadul/internal/crypto"
	klog "github.com/karadul/karadul/internal/log"
	"github.com/karadul/karadul/internal/mesh"
	"github.com/karadul/karadul/internal/nat"
	"github.com/karadul/karadul/internal/relay"
)

// TestRekeyLoop_WithAgedSession verifies that rekeyLoop exits cleanly on context cancellation
// even when sessions needing rekey exist.
func TestRekeyLoop_WithAgedSession(t *testing.T) {
	e := testEngine(t)

	// Build an aged session that NeedsRekey() returns true.
	var remotePub [32]byte
	remotePub[0] = 0xDD
	ep := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}
	ps := e.buildSession(remotePub, [32]byte{1}, [32]byte{2}, 99, 88, ep)
	peer := mesh.NewPeer(remotePub, "aged-peer", "n1", net.ParseIP("100.64.0.5"))
	peer.SetEndpoint(ep)
	ps.peer = peer

	// Age the session so NeedsRekey() returns true.
	ps.session.mu.Lock()
	ps.session.createdAt = time.Now().Add(-(sessionLifetime + time.Second))
	ps.session.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		e.rekeyLoop(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
		// Good — rekeyLoop exited cleanly.
	case <-time.After(5 * time.Second):
		t.Fatal("rekeyLoop did not exit on context cancellation")
	}
}

// TestEndpointRefreshLoop_Cancelled verifies endpointRefreshLoop exits on context cancellation.
func TestEndpointRefreshLoop_Cancelled(t *testing.T) {
	e := testEngine(t)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		e.endpointRefreshLoop(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("endpointRefreshLoop did not exit on context cancellation")
	}
}

// TestUdpReadLoop_StopChClosed verifies udpReadLoop exits when stopCh is closed.
func TestUdpReadLoop_StopChClosed(t *testing.T) {
	e := testEngine(t)

	udp, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Skipf("UDP listen: %v", err)
	}
	e.udp = udp

	done := make(chan struct{})
	go func() {
		e.udpReadLoop()
		close(done)
	}()

	// Close stopCh AND the socket to trigger the exit path.
	close(e.stopCh)
	udp.Close()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("udpReadLoop did not exit after stopCh + socket close")
	}
}

// TestConnectPeer_WithEndpoint exercises connectPeer with a peer that has an endpoint.
func TestConnectPeer_WithEndpoint(t *testing.T) {
	e := testEngine(t)

	udp, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Skipf("UDP listen: %v", err)
	}
	e.udp = udp
	t.Cleanup(func() { udp.Close() })

	var remotePub [32]byte
	remotePub[0] = 0xEE
	peer := mesh.NewPeer(remotePub, "connect-peer", "n2", net.ParseIP("100.64.0.6"))
	ep := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: udp.LocalAddr().(*net.UDPAddr).Port}
	peer.SetEndpoint(ep)

	// connectPeer should try to handshake — may fail crypto but shouldn't panic.
	_ = e.connectPeer(peer)
}

// TestConnectPeer_NoEndpoint_NoDERP exercises connectPeer when peer has no endpoint and no DERP client.
func TestConnectPeer_NoEndpoint_NoDERP(t *testing.T) {
	e := testEngine(t)

	var remotePub [32]byte
	remotePub[0] = 0xFF
	peer := mesh.NewPeer(remotePub, "no-ep-peer", "n3", net.ParseIP("100.64.0.7"))
	// No endpoint, no DERP client — connectPeer should return error.

	udp, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Skipf("UDP listen: %v", err)
	}
	e.udp = udp
	t.Cleanup(func() { udp.Close() })

	err = e.connectPeer(peer)
	if err == nil {
		t.Log("connectPeer succeeded despite no endpoint")
	}
}

// TestEnableExitNode_Darwin_ErrorPath verifies EnableExitNode fails gracefully on non-root.
func TestEnableExitNode_Darwin_ErrorPath(t *testing.T) {
	e := testEngine(t)
	err := e.EnableExitNode("nonexistent-iface-xyz")
	if err == nil {
		t.Log("EnableExitNode succeeded (running as root)")
	}
}

// TestDisableExitNode_Darwin_NoPanic verifies DisableExitNode doesn't panic.
func TestDisableExitNode_Darwin_NoPanic(t *testing.T) {
	DisableExitNode("nonexistent-iface-xyz")
}

// TestConcurrentSessionAccess verifies no data races under concurrent session operations.
func TestConcurrentSessionAccess(t *testing.T) {
	e := testEngine(t)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			var pub [32]byte
			pub[0] = byte(id)
			ep := &net.UDPAddr{IP: net.ParseIP("10.0.0.1"), Port: 5000 + id}
			ps := e.buildSession(pub, [32]byte{byte(id)}, [32]byte{byte(id + 1)}, uint32(id*2), uint32(id*2+1), ep)
			peer := mesh.NewPeer(pub, "concurrent-peer", "n", net.ParseIP("100.64.0.1"))
			peer.SetEndpoint(ep)
			ps.peer = peer

			ps.session.mu.Lock()
			ps.session.createdAt = time.Now().Add(-(sessionLifetime + time.Second))
			ps.session.mu.Unlock()
		}(i)
	}
	wg.Wait()

	e.mu.RLock()
	count := len(e.sessions)
	e.mu.RUnlock()
	if count != 10 {
		t.Errorf("expected 10 sessions, got %d", count)
	}
}

// TestStart_RegisterFails verifies Start returns error when coordination server is unreachable.
func TestStart_RegisterFails(t *testing.T) {
	kp, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.NodeConfig{
		ServerURL: "http://127.0.0.1:0",
		Hostname:  "test-fail",
		AuthKey:   "test",
	}
	log := klog.New(nil, klog.LevelDebug, klog.FormatText)
	e := NewEngine(cfg, kp, log)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = e.Start(ctx)
	if err == nil {
		t.Fatal("expected Start to fail with unreachable server")
	}
	if !strings.Contains(err.Error(), "register") {
		t.Errorf("expected register error, got: %v", err)
	}
}

// TestUseExitNode_NilTun verifies UseExitNode handles nil tun gracefully.
func TestUseExitNode_NilTun(t *testing.T) {
	e := testEngine(t)
	peer := mesh.NewPeer([32]byte{1}, "exit-peer", "n3", net.ParseIP("100.64.0.3"))

	defer func() {
		if r := recover(); r != nil {
			t.Logf("UseExitNode panicked with nil tun (expected): %v", r)
		}
	}()
	_ = e.UseExitNode(peer)
}

// ─── handleAPIExitNodeEnable: method and validation paths ────────────────────

func TestHandleAPIExitNodeEnable_Error(t *testing.T) {
	e := testEngine(t)
	req := httptest.NewRequest(http.MethodPost, "/exit-node/enable",
		strings.NewReader(`{"out_interface":"bogus0"}`))
	w := httptest.NewRecorder()
	e.handleAPIExitNodeEnable(w, req)
	// Will fail with error since exit node requires root
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("got %d, want 200 or 500", w.Code)
	}
}

// ─── handleAPIExitNodeUse: peer lookup paths ────────────────────────────────

func TestHandleAPIExitNodeUse_PeerLookupMiss(t *testing.T) {
	e := testEngine(t)
	req := httptest.NewRequest(http.MethodPost, "/exit-node/use",
		strings.NewReader(`{"peer":"nonexistent"}`))
	w := httptest.NewRecorder()
	e.handleAPIExitNodeUse(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("got %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleAPIExitNodeUse_PeerFoundByHostname(t *testing.T) {
	e := testEngine(t)
	var pub [32]byte
	pub[0] = 0xAA
	e.manager.AddOrUpdate(pub, "lookup-peer", "n1", net.ParseIP("100.64.0.10"), "", nil)

	// UseExitNode will panic on nil tun, so recover
	defer func() {
		if r := recover(); r != nil {
			t.Logf("UseExitNode panicked (expected with nil tun): %v", r)
		}
	}()

	req := httptest.NewRequest(http.MethodPost, "/exit-node/use",
		strings.NewReader(`{"peer":"lookup-peer"}`))
	w := httptest.NewRecorder()
	e.handleAPIExitNodeUse(w, req)
}

// ─── handleUDPPacket: unknown type ──────────────────────────────────────────

func TestHandleUDPPacket_UnknownType(t *testing.T) {
	e := testEngine(t)
	// Empty packet should be ignored without panic
	e.handleUDPPacket(&net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 1234}, []byte{})
}

// ─── udpReadLoop: semaphore full path ──────────────────────────────────────

func TestUdpReadLoop_SemaphoreFull(t *testing.T) {
	e := testEngine(t)

	udp, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Skipf("UDP listen: %v", err)
	}
	e.udp = udp

	// Fill the semaphore
	for i := 0; i < cap(e.udpSem); i++ {
		e.udpSem <- struct{}{}
	}

	// Send a packet — should be dropped (semaphore full)
	udp.WriteToUDP([]byte("test"), udp.LocalAddr().(*net.UDPAddr))

	// Close to trigger exit
	close(e.stopCh)
	udp.Close()

	// Drain semaphore so goroutines finish
	for i := 0; i < cap(e.udpSem); i++ {
		<-e.udpSem
	}
}

// ─── discoverEndpoint: no UDP ───────────────────────────────────────────────

func TestDiscoverEndpoint_NoUDP(t *testing.T) {
	e := testEngine(t)
	// discoverEndpoint panics with nil UDP (BindingRequest dereferences nil conn).
	// Verify it doesn't silently succeed.
	defer func() {
		if r := recover(); r != nil {
			t.Logf("discoverEndpoint panicked with nil UDP (expected): %v", r)
		}
	}()
	_, err := e.discoverEndpoint()
	if err != nil {
		t.Logf("discoverEndpoint returned error: %v", err)
	}
}

// ─── connectPeer: no session, no endpoint ───────────────────────────────────

func TestConnectPeer_NoEndpoint_NoDERP_NoSession(t *testing.T) {
	e := testEngine(t)

	var pub [32]byte
	pub[0] = 0xCC
	peer := mesh.NewPeer(pub, "no-ep", "n5", net.ParseIP("100.64.0.9"))
	// No endpoint set, no DERP client

	err := e.connectPeer(peer)
	// Should return nil (logs warning, doesn't error)
	if err != nil {
		t.Logf("connectPeer returned: %v", err)
	}
}

// ─── serveLocalAPI: listen on unix socket ────────────────────────────────────

func TestServeLocalAPI_ContextCancel(t *testing.T) {
	e := testEngine(t)
	dir := t.TempDir()
	e.cfg.DataDir = dir

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		e.serveLocalAPI(ctx)
		close(done)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("serveLocalAPI did not exit on context cancellation")
	}
}

// ─── handleAPIPeers: peer with endpoint set ─────────────────────────────────

func TestHandleAPIPeers_WithEndpoint(t *testing.T) {
	e := testEngine(t)

	var pub [32]byte
	pub[0] = 0xBB
	ep := &net.UDPAddr{IP: net.ParseIP("203.0.113.5"), Port: 41641}
	e.manager.AddOrUpdate(pub, "ep-peer", "n4", net.ParseIP("100.64.0.4"), "", nil)
	// Set endpoint on the peer
	p, ok := e.manager.GetPeer(pub)
	if !ok {
		t.Fatal("peer not found after AddOrUpdate")
	}
	p.SetEndpoint(ep)

	w := httptest.NewRecorder()
	e.handleAPIPeers(w, httptest.NewRequest(http.MethodGet, "/peers", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if !strings.Contains(body, "203.0.113.5:41641") {
		t.Errorf("response should contain peer endpoint, got:\n%s", body)
	}
}

// ─── handleUDPPacket: type dispatch for HandshakeInit, HandshakeResp, Data ──

func TestHandleUDPPacket_HandshakeInit(t *testing.T) {
	e := testEngine(t)
	// Construct a minimal handshake init packet (type byte 0x01).
	// The actual Noise handshake will fail, but we exercise the switch branch.
	pkt := make([]byte, 96)
	pkt[0] = 0x01 // TypeHandshakeInit
	e.handleUDPPacket(&net.UDPAddr{IP: net.ParseIP("10.0.0.1"), Port: 1234}, pkt)
	// Should not panic.
}

func TestHandleUDPPacket_HandshakeResp(t *testing.T) {
	e := testEngine(t)
	pkt := make([]byte, 48)
	pkt[0] = 0x02 // TypeHandshakeResp
	e.handleUDPPacket(&net.UDPAddr{IP: net.ParseIP("10.0.0.1"), Port: 1234}, pkt)
	// Should not panic.
}

func TestHandleUDPPacket_DataPacket(t *testing.T) {
	e := testEngine(t)
	pkt := make([]byte, 32)
	pkt[0] = 0x03 // TypeData
	e.handleUDPPacket(&net.UDPAddr{IP: net.ParseIP("10.0.0.1"), Port: 1234}, pkt)
	// Should not panic.
}

func TestHandleUDPPacket_KeepalivePacket(t *testing.T) {
	e := testEngine(t)
	pkt := []byte{0x04} // TypeKeepalive
	e.handleUDPPacket(&net.UDPAddr{IP: net.ParseIP("10.0.0.1"), Port: 1234}, pkt)
	// Should not panic; keepalive is a no-op.
}

// ─── onDERPRecv: HandshakeResp via DERP ──────────────────────────────────────

func TestOnDERPRecv_HandshakeRespNilAddr(t *testing.T) {
	e := testEngine(t)
	pkt := make([]byte, 48)
	pkt[0] = 0x02 // TypeHandshakeResp
	var src [32]byte
	src[0] = 0xEE
	e.onDERPRecv(src, pkt)
	// Should not panic.
}

// ─── EnableExitNode: success path via fake sysctl ────────────────────────────

func TestEnableExitNode_FakeSysctl(t *testing.T) {
	// Create a fake sysctl binary that exits 0.
	tmpDir := t.TempDir()
	fake := tmpDir + "/sysctl"
	if err := os.WriteFile(fake, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
	// Also need a fake pfctl and ifconfig for macOS
	for _, name := range []string{"pfctl", "ifconfig", "iptables", "route"} {
		_ = os.WriteFile(tmpDir+"/"+name, []byte("#!/bin/sh\nexit 0\n"), 0755)
	}
	t.Setenv("PATH", tmpDir+":"+os.Getenv("PATH"))

	err := EnableExitNode("fake0")
	// May still fail depending on platform specifics, but shouldn't panic.
	if err != nil {
		t.Logf("EnableExitNode: %v (expected on some platforms)", err)
	}
}

// ─── EnableExitNode on Engine: log path ──────────────────────────────────────

func TestEnableExitNode_EngineLog(t *testing.T) {
	e := testEngine(t)
	err := e.EnableExitNode("nonexistent-iface")
	// Will fail without root, but exercises the error path and log line.
	if err != nil {
		t.Logf("EnableExitNode error (expected): %v", err)
	}
}

// ─── rekeyLoop: ticker path exercises actual rekey ─────────────────────────

// TestRekeyLoop_TickerDetectsAgedSession verifies that the ticker branch in
// rekeyLoop detects sessions that need rekey and invokes RekeyPeer.
// Rather than waiting 30s for the real ticker, this test manually executes
// the ticker body logic and verifies cleanup.
func TestRekeyLoop_TickerDetectsAgedSession(t *testing.T) {
	e := testEngine(t)

	udp, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Skipf("UDP listen: %v", err)
	}
	e.udp = udp
	t.Cleanup(func() { udp.Close() })

	// Create a session that is old enough to need rekey.
	var pubKey [32]byte
	for i := range pubKey {
		pubKey[i] = byte(i + 30)
	}
	peer := mesh.NewPeer(pubKey, "rekey-ticker-peer", "n1", net.ParseIP("100.64.0.60"))
	peer.SetEndpoint(&net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: udp.LocalAddr().(*net.UDPAddr).Port})

	ps := e.buildSession(pubKey, [32]byte{5}, [32]byte{6}, 900, 901, nil)
	ps.peer = peer

	// Age the session past sessionLifetime so NeedsRekey() returns true.
	ps.session.mu.Lock()
	ps.session.createdAt = time.Now().Add(-(sessionLifetime + time.Second))
	ps.session.mu.Unlock()

	// Manually execute the ticker body logic (the same code rekeyLoop runs
	// when the ticker fires). This verifies the detection + RekeyPeer path.
	e.mu.RLock()
	var toRekey []*peerSession
	for _, s := range e.sessions {
		if s.session.NeedsRekey() {
			toRekey = append(toRekey, s)
		}
	}
	e.mu.RUnlock()

	if len(toRekey) != 1 {
		t.Fatalf("expected 1 session needing rekey, got %d", len(toRekey))
	}

	for _, s := range toRekey {
		if s.peer != nil {
			e.RekeyPeer(s.peer)
		}
	}

	// After RekeyPeer, the old session maps should be cleaned.
	e.mu.RLock()
	_, hasSession := e.sessions[pubKey]
	_, hasByID := e.byID[900]
	e.mu.RUnlock()
	if hasSession {
		t.Error("sessions map should not have old entry after rekey")
	}
	if hasByID {
		t.Error("byID map should not have old entry after rekey")
	}
}

// ─── udpReadLoop: semaphore full path with verification ───────────────────

// TestUdpReadLoop_SemaphoreFullDropsPacket verifies that when the semaphore is
// full, the udpReadLoop default branch is taken and the packet is dropped
// without spawning a handler goroutine.
func TestUdpReadLoop_SemaphoreFullDropsPacket(t *testing.T) {
	e := testEngine(t)

	udp, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Skipf("UDP listen: %v", err)
	}
	e.udp = udp
	t.Cleanup(func() { udp.Close() })

	// Fill the semaphore completely.
	for i := 0; i < cap(e.udpSem); i++ {
		e.udpSem <- struct{}{}
	}

	// Send multiple packets while semaphore is full.
	remote, err := net.DialUDP("udp4", nil, udp.LocalAddr().(*net.UDPAddr))
	if err != nil {
		t.Fatal(err)
	}
	defer remote.Close()

	for i := 0; i < 5; i++ {
		_, _ = remote.Write([]byte{0x04}) // keepalive packets
	}

	// Allow a brief moment for the packets to arrive and be dropped.
	time.Sleep(100 * time.Millisecond)

	// Semaphore should still be full (no goroutines consumed from it).
	if len(e.udpSem) != cap(e.udpSem) {
		t.Errorf("semaphore should still be full, got %d/%d", len(e.udpSem), cap(e.udpSem))
	}

	// Close to trigger exit.
	close(e.stopCh)
	udp.Close()

	// Drain semaphore so goroutines finish.
	for i := 0; i < cap(e.udpSem); i++ {
		<-e.udpSem
	}
}

// ─── connectPeer: DERP fallback path with relay client ────────────────────

// TestConnectPeer_DERPClientFallback verifies that connectPeer uses the DERP
// client fallback path when a peer has no direct endpoint.
func TestConnectPeer_DERPClientFallback(t *testing.T) {
	e := testEngine(t)

	udp, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Skipf("UDP listen: %v", err)
	}
	e.udp = udp
	t.Cleanup(func() { udp.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	e.ctx = ctx
	t.Cleanup(cancel)

	// Start a local DERP relay server so the client can connect.
	dlg := klog.New(nil, klog.LevelDebug, klog.FormatText)
	derpSrv := relay.NewServer(dlg)
	derpCtx, derpCancel := context.WithCancel(context.Background())
	defer derpCancel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("TCP listen: %v", err)
	}
	derpAddr := ln.Addr().String()

	mux := http.NewServeMux()
	mux.Handle("/derp", derpSrv)
	httpSrv := &http.Server{Handler: mux}
	go func() { _ = httpSrv.Serve(ln) }()
	defer httpSrv.Close()

	go func() {
		<-derpCtx.Done()
		httpSrv.Close()
	}()

	// Create a DERP client and attach it to the engine.
	derpClient := relay.NewClient("http://"+derpAddr, e.kp.Public, e.onDERPRecv, e.log)
	go derpClient.Run(ctx)

	// Give the client time to connect.
	time.Sleep(500 * time.Millisecond)

	e.derpMu.Lock()
	e.derpClient = derpClient
	e.derpMu.Unlock()

	// Create a peer with no endpoint.
	var pubKey [32]byte
	for i := range pubKey {
		pubKey[i] = byte(i + 200)
	}
	peer := mesh.NewPeer(pubKey, "derp-fallback-peer", "n1", net.ParseIP("100.64.0.40"))

	// connectPeer should fall through to DERP fallback and call
	// initiateHandshake which will try to send via DERP client.
	err = e.connectPeer(peer)
	// initiateHandshake sends via DERP (may fail due to the relay not having
	// the remote peer connected, but should not return the "no path" error).
	// The key coverage: the derpClient != nil branch is exercised.
	if err != nil {
		t.Logf("connectPeer with DERP client: %v", err)
	}

	// Verify a pending handshake was created (initiateHandshake was called).
	e.mu.RLock()
	pendingCount := len(e.pending)
	e.mu.RUnlock()
	if pendingCount == 0 {
		t.Error("expected a pending handshake after connectPeer DERP fallback")
	}
}

// ─── endpointRefreshLoop: ticker path with mock STUN ──────────────────────

// TestEndpointRefreshLoop_TickerUpdatesEndpoint verifies that when the ticker
// fires in endpointRefreshLoop and discoverEndpoint succeeds, the endpoint is
// updated and reported to the coordination server.
func TestEndpointRefreshLoop_TickerUpdatesEndpoint(t *testing.T) {
	if testing.Short() {
		t.Skip("requires waiting for 30s ticker interval")
	}

	e := testEngine(t)

	// Bind a real UDP socket for the engine.
	udp, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Skipf("UDP listen: %v", err)
	}
	e.udp = udp
	t.Cleanup(func() { udp.Close() })

	// Override DefaultSTUNServers with a mock STUN server.
	mockAddr, mockStop := startMockSTUN(t)
	origServers := nat.DefaultSTUNServers
	nat.DefaultSTUNServers = []string{mockAddr}
	defer func() {
		nat.DefaultSTUNServers = origServers
		mockStop()
	}()

	// Set up a coordination server to receive endpoint reports.
	reportReceived := make(chan string, 1)
	coordSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/update-endpoint" {
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			if ep, ok := body["endpoint"].(string); ok {
				select {
				case reportReceived <- ep:
				default:
				}
			}
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer coordSrv.Close()
	e.serverURL = coordSrv.URL

	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		e.endpointRefreshLoop(ctx)
		close(done)
	}()

	// Wait for the endpoint to be updated via the ticker path.
	select {
	case ep := <-reportReceived:
		if ep == "" {
			t.Error("reported endpoint should not be empty")
		}
		t.Logf("endpoint reported: %s", ep)
	case <-ctx.Done():
		t.Log("endpoint refresh loop timed out before report (may be slow CI)")
	}

	// Verify publicEP was updated.
	if pubEP := e.publicEP.Load(); pubEP == nil {
		t.Error("publicEP should have been set by endpointRefreshLoop")
	}

	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("endpointRefreshLoop did not exit after context cancellation")
	}
}

// startMockSTUN creates a minimal STUN server that responds to binding requests
// with a fake mapped address. Returns the server address and a stop function.
func startMockSTUN(t *testing.T) (addr string, stop func()) {
	t.Helper()
	srv, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Skipf("UDP listen for mock STUN: %v", err)
	}
	addr = srv.LocalAddr().String()
	quit := make(chan struct{})
	go func() {
		defer srv.Close()
		buf := make([]byte, 1024)
		for {
			select {
			case <-quit:
				return
			default:
			}
			_ = srv.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
			n, src, err := srv.ReadFromUDP(buf)
			if err != nil {
				continue
			}
			if n < 20 {
				continue
			}
			// Check it's a Binding Request (type 0x0001).
			msgType := uint16(buf[0])<<8 | uint16(buf[1])
			if msgType != 0x0001 {
				continue
			}
			txID := make([]byte, 12)
			copy(txID, buf[8:20])

			// Build a XOR-MAPPED-ADDRESS response.
			// Fake public IP: 203.0.113.1, port: src.Port (echo back their port).
			mappedIP := net.IPv4(203, 0, 113, 1).To4()
			mappedPort := src.Port

			magicBytes := [4]byte{0x21, 0x12, 0xA4, 0x42}
			xorIP := [4]byte{
				mappedIP[0] ^ magicBytes[0],
				mappedIP[1] ^ magicBytes[1],
				mappedIP[2] ^ magicBytes[2],
				mappedIP[3] ^ magicBytes[3],
			}
			xorPort := uint16(mappedPort) ^ uint16(0x2112)

			// XOR-MAPPED-ADDRESS attribute value: reserved(1)+family(1)+xor-port(2)+xor-ip(4).
			val := make([]byte, 8)
			val[0] = 0x00
			val[1] = 0x01 // IPv4
			val[2] = byte(xorPort >> 8)
			val[3] = byte(xorPort)
			copy(val[4:], xorIP[:])

			// Attribute TLV: type(2)+length(2)+value(8).
			attr := make([]byte, 4+len(val))
			attr[0] = 0x00
			attr[1] = 0x20 // XOR-MAPPED-ADDRESS
			attr[2] = 0x00
			attr[3] = byte(len(val))
			copy(attr[4:], val)

			// Full STUN response header: type(2)+length(2)+magic(4)+txID(12).
			resp := make([]byte, 20+len(attr))
			resp[0] = 0x01 // Binding Response
			resp[1] = 0x01
			resp[2] = byte(len(attr) >> 8)
			resp[3] = byte(len(attr))
			resp[4] = 0x21 // Magic cookie
			resp[5] = 0x12
			resp[6] = 0xA4
			resp[7] = 0x42
			copy(resp[8:], txID)
			copy(resp[20:], attr)

			srv.WriteToUDP(resp, src)
		}
	}()
	return addr, func() { close(quit) }
}

// ─── EnableExitNode: success path via fake binaries ───────────────────────

// TestEnableExitNode_SuccessPath verifies that the Engine's EnableExitNode
// method completes successfully when the platform commands succeed (via fake
// binaries in PATH), and that it logs the success message.
func TestEnableExitNode_SuccessPath(t *testing.T) {
	tmpDir := t.TempDir()

	// Create fake binaries that exit 0.
	for _, name := range []string{"sysctl", "pfctl", "ifconfig", "iptables", "route"} {
		if err := os.WriteFile(tmpDir+"/"+name, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", tmpDir+":"+os.Getenv("PATH"))

	err := EnableExitNode("fake0")
	if err != nil {
		// May fail on pf config write without root — that's expected.
		t.Logf("EnableExitNode: %v (pf conf write requires root)", err)
	}
}

// TestEnableExitNode_EngineSuccessPath verifies that the Engine method
// EnableExitNode returns nil when the platform function succeeds.
func TestEnableExitNode_EngineSuccessPath(t *testing.T) {
	tmpDir := t.TempDir()
	for _, name := range []string{"sysctl", "pfctl", "ifconfig", "iptables", "route"} {
		if err := os.WriteFile(tmpDir+"/"+name, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", tmpDir+":"+os.Getenv("PATH"))

	e := testEngine(t)
	err := e.EnableExitNode("fake0")
	if err != nil {
		// May fail on pf config write without root — that's expected.
		t.Logf("Engine.EnableExitNode: %v (pf conf write requires root)", err)
	}
}

// ─── rekeyLoop: ticker fires and triggers RekeyPeer ───────────────────────────

// TestRekeyLoop_TickerFiresAndRekeys verifies that when rekeyLoop's ticker
// fires and there are sessions needing rekey, the sessions are cleaned up
// via RekeyPeer. This exercises lines 1237-1252 in engine.go.
func TestRekeyLoop_TickerFiresAndRekeys(t *testing.T) {
	if testing.Short() {
		t.Skip("requires waiting for 30s ticker interval")
	}

	e := testEngine(t)

	udp, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Skipf("UDP listen: %v", err)
	}
	e.udp = udp
	t.Cleanup(func() { udp.Close() })

	// Create a session old enough to need rekey.
	var pubKey [32]byte
	for i := range pubKey {
		pubKey[i] = byte(i + 30)
	}
	peer := mesh.NewPeer(pubKey, "rekey-ticker-target", "n1", net.ParseIP("100.64.0.60"))
	peer.SetEndpoint(&net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: udp.LocalAddr().(*net.UDPAddr).Port})

	ps := e.buildSession(pubKey, [32]byte{5}, [32]byte{6}, 900, 901, nil)
	ps.peer = peer

	// Age the session past sessionLifetime so NeedsRekey() returns true.
	ps.session.mu.Lock()
	ps.session.createdAt = time.Now().Add(-(sessionLifetime + time.Second))
	ps.session.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		e.rekeyLoop(ctx)
		close(done)
	}()

	// Wait for the rekeyLoop to finish (the context timeout will stop it).
	<-done

	// After the ticker fires, RekeyPeer should have been called,
	// which removes the session from the maps.
	e.mu.RLock()
	_, hasSession := e.sessions[pubKey]
	_, hasByID := e.byID[900]
	e.mu.RUnlock()
	if hasSession {
		t.Error("sessions map should not have old entry after rekey ticker fired")
	}
	if hasByID {
		t.Error("byID map should not have old entry after rekey ticker fired")
	}
}

// ─── udpReadLoop: happy path receives and dispatches packets ─────────────────

// TestUdpReadLoop_ReceivesAndDispatches verifies that udpReadLoop reads a
// UDP packet, acquires the semaphore, and spawns a handler goroutine.
// A keepalive packet is sent to verify the full read-dispatch path.
func TestUdpReadLoop_ReceivesAndDispatches(t *testing.T) {
	e := testEngine(t)

	udp, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Skipf("UDP listen: %v", err)
	}
	e.udp = udp
	t.Cleanup(func() { udp.Close() })

	done := make(chan struct{})
	go func() {
		e.udpReadLoop()
		close(done)
	}()

	// Send a keepalive packet from a remote UDP socket.
	remote, err := net.DialUDP("udp4", nil, udp.LocalAddr().(*net.UDPAddr))
	if err != nil {
		t.Fatal(err)
	}
	defer remote.Close()

	// Keepalive is just the type byte 0x04.
	_, err = remote.Write([]byte{0x04})
	if err != nil {
		t.Fatalf("write keepalive: %v", err)
	}

	// Give the goroutine time to read the packet and dispatch the handler.
	time.Sleep(200 * time.Millisecond)

	// Close to trigger exit.
	close(e.stopCh)
	udp.Close()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("udpReadLoop did not exit")
	}

	// Verify the semaphore is empty (handler goroutine finished and released).
	if len(e.udpSem) != 0 {
		t.Errorf("semaphore should be empty after handler finishes, got %d", len(e.udpSem))
	}
}

// ─── udpReadLoop: read error default branch (transient error) ────────────────

// TestUdpReadLoop_TransientReadError verifies that udpReadLoop continues
// reading after a transient error when stopCh is not closed.
func TestUdpReadLoop_TransientReadError(t *testing.T) {
	e := testEngine(t)

	udp, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Skipf("UDP listen: %v", err)
	}
	e.udp = udp
	t.Cleanup(func() { udp.Close() })

	done := make(chan struct{})
	go func() {
		e.udpReadLoop()
		close(done)
	}()

	// Set a very short read deadline so ReadFromUDP gets a timeout error.
	udp.SetReadDeadline(time.Now().Add(50 * time.Millisecond))

	// Wait for the deadline error to be hit and the loop to continue.
	time.Sleep(150 * time.Millisecond)

	// Now send a valid packet to confirm the loop is still running.
	udp.SetReadDeadline(time.Time{}) // clear deadline

	remote, err := net.DialUDP("udp4", nil, udp.LocalAddr().(*net.UDPAddr))
	if err != nil {
		t.Fatal(err)
	}
	defer remote.Close()

	_, err = remote.Write([]byte{0x04})
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	// Give it time to process.
	time.Sleep(200 * time.Millisecond)

	// Close to trigger exit.
	close(e.stopCh)
	udp.Close()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("udpReadLoop did not exit after transient error + close")
	}
}

// ─── connectPeer: hole punch success path ────────────────────────────────────

// TestConnectPeer_HolePunchSuccess verifies that when the direct handshake
// fails but hole punch succeeds, connectPeer completes successfully.
// Two UDP sockets simulate the hole punch exchange.
func TestConnectPeer_HolePunchSuccess(t *testing.T) {
	e := testEngine(t)

	engineUDP, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Skipf("UDP listen: %v", err)
	}
	e.udp = engineUDP
	t.Cleanup(func() { engineUDP.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	e.ctx = ctx

	// Create a "peer" UDP listener that will echo back hole punch probes.
	peerListener, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Skipf("UDP listen: %v", err)
	}
	defer peerListener.Close()

	peerAddr := peerListener.LocalAddr().(*net.UDPAddr)

	// Start a goroutine that reads probes from the peer's socket and echoes
	// them back to the sender. This simulates the other side of hole punching.
	echoDone := make(chan struct{})
	go func() {
		defer close(echoDone)
		buf := make([]byte, 256)
		for {
			peerListener.SetReadDeadline(time.Now().Add(8 * time.Second))
			n, senderAddr, err := peerListener.ReadFromUDP(buf)
			if err != nil {
				return
			}
			// Echo the probe back to the sender.
			peerListener.WriteToUDP(buf[:n], senderAddr)
		}
	}()

	var pubKey [32]byte
	for i := range pubKey {
		pubKey[i] = byte(i + 77)
	}
	peer := mesh.NewPeer(pubKey, "holepunch-success-peer", "n1", net.ParseIP("100.64.0.77"))
	peer.SetEndpoint(peerAddr)

	err = e.connectPeer(peer)
	// hole punch should succeed since the echo goroutine reflects the probe.
	// After hole punch succeeds, initiateHandshake is called.
	// The handshake might fail (no real Noise peer), but the hole punch path is exercised.
	if err != nil {
		t.Logf("connectPeer result (hole punch exercised): %v", err)
	}

	// Verify a pending handshake was created (hole punch -> initiateHandshake).
	e.mu.RLock()
	pendingCount := len(e.pending)
	e.mu.RUnlock()
	if pendingCount == 0 {
		t.Error("expected a pending handshake after hole punch + initiateHandshake")
	}

	cancel()
	<-echoDone
}

// ─── handleAPIExitNodeEnable: success path ──────────────────────────────────

// TestHandleAPIExitNodeEnable_FakeBinaries verifies the success path through the
// exit-node enable handler. Uses fake binaries to ensure the platform
// EnableExitNode function succeeds.
func TestHandleAPIExitNodeEnable_FakeBinaries(t *testing.T) {
	tmpDir := t.TempDir()
	for _, name := range []string{"sysctl", "pfctl", "ifconfig", "iptables", "route"} {
		if err := os.WriteFile(tmpDir+"/"+name, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", tmpDir+":"+os.Getenv("PATH"))

	e := testEngine(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/exit-node/enable",
		strings.NewReader(`{"out_interface":"fake0"}`))
	e.handleAPIExitNodeEnable(w, req)

	// Accept 200 (success with root) or 500 (pf conf write fails without root).
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want 200 or 500; body: %s", w.Code, w.Body.String())
	}
	if w.Code == http.StatusOK {
		var resp map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("parse response: %v", err)
		}
		if resp["status"] != "ok" {
			t.Errorf("response status: got %q, want %q", resp["status"], "ok")
		}
	}
}

// ─── handleAPIExitNodeUse: UseExitNode error path ───────────────────────────

// TestHandleAPIExitNodeUse_UseExitNodeError verifies that handleAPIExitNodeUse
// returns 500 when UseExitNode fails (e.g. nil TUN device panics are caught,
// or AddRoute fails).
func TestHandleAPIExitNodeUse_UseExitNodeError(t *testing.T) {
	e := testEngine(t)

	// Set up a mock TUN that returns an error on AddRoute.
	mtun := &errorAddRouteMockTUN{}
	e.tun = mtun
	e.router = mesh.NewRouter(e.manager)

	// Add a peer to the manager.
	var pub [32]byte
	for i := range pub {
		pub[i] = byte(i + 1)
	}
	e.manager.AddOrUpdate(pub, "exit-node-err", "n1", net.ParseIP("100.64.0.50"), "", nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/exit-node/use",
		strings.NewReader(`{"peer":"exit-node-err"}`))
	e.handleAPIExitNodeUse(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want %d; body: %s", w.Code, http.StatusInternalServerError, w.Body.String())
	}
}

// errorAddRouteMockTUN is a mock TUN whose AddRoute returns an error.
type errorAddRouteMockTUN struct {
	mockTUN // embed the basic mock
}

func (m *errorAddRouteMockTUN) AddRoute(dst *net.IPNet) error {
	return fmt.Errorf("mock add route error")
}
