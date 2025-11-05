# Internet-Wide Sync Design

## Problem
Current Fybrk only works on local networks (192.168.x.x). Users want:
1. QR code pairing (no manual file copying)
2. Internet-wide sync (not just local network)
3. No manual setup or server management

## Solution: Decentralized Bootstrap Network

Instead of a centralized server, use a **decentralized bootstrap network** with multiple approaches:

### 1. QR Code Pairing with Temporary Rendezvous

**How it works:**
```
Device A (has files)          Device B (wants to sync)
     |                              |
     v                              v
1. fybrk /path pair          2. Scan QR code
   - Generates QR code          - Gets rendezvous info
   - Creates temp ID            - Connects to rendezvous
   - Connects to rendezvous     - Exchanges keys
     |                              |
     v                              v
3. Direct P2P connection established
   - UPnP hole punching
   - Direct file transfer
   - No more rendezvous needed
```

**QR Code contains:**
- Temporary rendezvous ID (expires in 10 minutes)
- Device public key
- Sync folder encryption key
- Connection info

### 2. Multiple Bootstrap Methods

**A. Public Bootstrap Nodes (Optional)**
- `bootstrap1.fybrk.com`, `bootstrap2.fybrk.com`
- Only for initial device discovery
- No file data passes through them
- Users can opt out and use alternatives

**B. DHT (Distributed Hash Table)**
- Use existing DHT networks (like BitTorrent DHT)
- Completely decentralized
- No Fybrk infrastructure needed

**C. Local Network + Internet Fallback**
- Try local network first (current behavior)
- Fall back to internet methods if needed

### 3. Implementation Plan

**Phase 1: QR Code Pairing (Local Network)**
```bash
fybrk /path pair              # Shows QR code
fybrk pair-with <qr-data>     # Joins sync from QR code
```

**Phase 2: Internet Bootstrap**
- Add bootstrap node discovery
- Implement hole punching
- Add DHT support as backup

**Phase 3: Full Decentralization**
- Remove dependency on any Fybrk servers
- Pure P2P with multiple discovery methods

### 4. Security Model

**Rendezvous Security:**
- Temporary IDs (expire quickly)
- End-to-end encryption
- Bootstrap nodes can't decrypt data
- Public key verification

**No Trust Required:**
- Bootstrap nodes are just for discovery
- All file data is end-to-end encrypted
- Users can run their own bootstrap nodes

### 5. User Experience

**Simple Pairing:**
```bash
# Device A
fybrk ~/Documents scan        # Initialize
fybrk ~/Documents pair        # Show QR code

# Device B  
fybrk pair-with <qr-data>     # Scan QR, auto-sync
```

**No Manual Setup:**
- No server configuration
- No port forwarding
- No IP addresses to remember
- Works behind NAT/firewalls

### 6. Fallback Strategy

If internet methods fail, fall back to:
1. Local network discovery (current)
2. Manual IP connection
3. File export/import

### 7. Privacy Guarantees

- No file data on bootstrap servers
- Temporary rendezvous only
- End-to-end encryption always
- Optional: run your own bootstrap nodes

## Implementation Priority

1. **Immediate**: Add `pair` command with local network QR codes
2. **Next**: Add internet bootstrap for discovery
3. **Future**: Full DHT integration for complete decentralization

This gives users the QR code experience they want while maintaining Fybrk's zero-trust philosophy.
