package network

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// UPnPClient handles automatic port forwarding via router UPnP
type UPnPClient struct {
	gatewayURL string
	localIP    string
}

// NewUPnPClient discovers UPnP gateway and creates client
func NewUPnPClient() (*UPnPClient, error) {
	// Discover UPnP gateway via SSDP multicast
	gatewayURL, err := discoverGateway()
	if err != nil {
		return nil, fmt.Errorf("UPnP gateway not found: %v", err)
	}

	localIP, err := getLocalIP()
	if err != nil {
		return nil, fmt.Errorf("failed to get local IP: %v", err)
	}

	return &UPnPClient{
		gatewayURL: gatewayURL,
		localIP:    localIP,
	}, nil
}

// AddPortMapping requests router to forward external port to local port
func (u *UPnPClient) AddPortMapping(externalPort, internalPort int, protocol string) error {
	// Simple UPnP port mapping request
	soapAction := `"urn:schemas-upnp-org:service:WANIPConnection:1#AddPortMapping"`

	body := fmt.Sprintf(`<?xml version="1.0"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
<s:Body>
<u:AddPortMapping xmlns:u="urn:schemas-upnp-org:service:WANIPConnection:1">
<NewRemoteHost></NewRemoteHost>
<NewExternalPort>%d</NewExternalPort>
<NewProtocol>%s</NewProtocol>
<NewInternalPort>%d</NewInternalPort>
<NewInternalClient>%s</NewInternalClient>
<NewEnabled>1</NewEnabled>
<NewPortMappingDescription>Fybrk P2P Sync</NewPortMappingDescription>
<NewLeaseDuration>0</NewLeaseDuration>
</u:AddPortMapping>
</s:Body>
</s:Envelope>`, externalPort, protocol, internalPort, u.localIP)

	req, err := http.NewRequest("POST", u.gatewayURL, strings.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", soapAction)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("UPnP request failed: %s", resp.Status)
	}

	return nil
}

// RemovePortMapping removes the port forwarding rule
func (u *UPnPClient) RemovePortMapping(externalPort int, protocol string) error {
	soapAction := `"urn:schemas-upnp-org:service:WANIPConnection:1#DeletePortMapping"`

	body := fmt.Sprintf(`<?xml version="1.0"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
<s:Body>
<u:DeletePortMapping xmlns:u="urn:schemas-upnp-org:service:WANIPConnection:1">
<NewRemoteHost></NewRemoteHost>
<NewExternalPort>%d</NewExternalPort>
<NewProtocol>%s</NewProtocol>
</u:DeletePortMapping>
</s:Body>
</s:Envelope>`, externalPort, protocol)

	req, err := http.NewRequest("POST", u.gatewayURL, strings.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", soapAction)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// discoverGateway finds UPnP gateway via SSDP multicast
func discoverGateway() (string, error) {
	// Send SSDP M-SEARCH request
	ssdpAddr, err := net.ResolveUDPAddr("udp4", "239.255.255.250:1900")
	if err != nil {
		return "", err
	}

	conn, err := net.DialUDP("udp4", nil, ssdpAddr)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	searchMsg := "M-SEARCH * HTTP/1.1\r\n" +
		"HOST: 239.255.255.250:1900\r\n" +
		"ST: urn:schemas-upnp-org:device:InternetGatewayDevice:1\r\n" +
		"MAN: \"ssdp:discover\"\r\n" +
		"MX: 3\r\n\r\n"

	_, err = conn.Write([]byte(searchMsg))
	if err != nil {
		return "", err
	}

	// Read response with timeout
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return "", err
	}

	response := string(buffer[:n])

	// Extract LOCATION header
	lines := strings.Split(response, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(strings.ToUpper(line), "LOCATION:") {
			location := strings.TrimSpace(line[9:])
			// Extract base URL for SOAP requests
			if idx := strings.Index(location[8:], "/"); idx != -1 {
				return location[:8+idx] + "/upnp/control/WANIPConn1", nil
			}
		}
	}

	return "", fmt.Errorf("no UPnP gateway found")
}

// getLocalIP gets the local IP address
func getLocalIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}
