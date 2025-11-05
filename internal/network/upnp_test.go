package network

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUPnPDiscovery(t *testing.T) {
	// Test UPnP gateway discovery (may not work in all environments)
	upnp, err := NewUPnPClient()
	
	if err != nil {
		// UPnP not available in test environment - this is expected
		t.Skipf("UPnP not available: %v", err)
		return
	}
	
	assert.NotNil(t, upnp)
	assert.NotEmpty(t, upnp.gatewayURL)
	assert.NotEmpty(t, upnp.localIP)
}

func TestUPnPPortMapping(t *testing.T) {
	upnp, err := NewUPnPClient()
	if err != nil {
		t.Skipf("UPnP not available: %v", err)
		return
	}
	
	// Test port mapping (use high port to avoid conflicts)
	testPort := 19999
	
	// Add port mapping
	err = upnp.AddPortMapping(testPort, testPort, "TCP")
	if err != nil {
		t.Skipf("UPnP port mapping failed (router may not support it): %v", err)
		return
	}
	
	// Wait a moment for the mapping to take effect
	time.Sleep(100 * time.Millisecond)
	
	// Remove port mapping
	err = upnp.RemovePortMapping(testPort, "TCP")
	assert.NoError(t, err)
}

func TestGetLocalIP(t *testing.T) {
	ip, err := getLocalIP()
	assert.NoError(t, err)
	assert.NotEmpty(t, ip)
	assert.NotEqual(t, "127.0.0.1", ip, "Should not return localhost")
}

func TestDiscoverGateway(t *testing.T) {
	// This test may fail in environments without UPnP routers
	gatewayURL, err := discoverGateway()
	
	if err != nil {
		t.Skipf("No UPnP gateway found: %v", err)
		return
	}
	
	assert.NotEmpty(t, gatewayURL)
	assert.Contains(t, gatewayURL, "http")
}
