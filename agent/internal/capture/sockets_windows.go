//go:build windows

package capture

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync"
	"unsafe"

	"github.com/Goldpip3/albion-info-tracker/agent/internal/photonports"
	"golang.org/x/sys/windows"
)

// Packet is one captured UDP datagram already stripped of IP/UDP headers.
type Packet struct {
	SrcPort uint16
	DstPort uint16
	Payload []byte
}

// Sink receives captured packets. Implementations must not retain Payload —
// the buffer is reused across calls.
type Sink interface {
	OnPacket(Packet)
}

// SinkFunc adapts a function to the Sink interface.
type SinkFunc func(Packet)

func (f SinkFunc) OnPacket(p Packet) { f(p) }

// Run binds a raw socket to every non-loopback IPv4 unicast address, enables
// SIO_RCVALL promiscuous mode, parses each frame's IP+UDP headers, and
// forwards Photon-looking UDP payloads to sink. Blocks until ctx is cancelled.
//
// Requires administrator privileges. Returns first bind/ioctl error if no
// socket could be opened; otherwise returns ctx.Err() when ctx cancels.
func Run(ctx context.Context, sink Sink) error {
	addrs, err := localUnicastV4()
	if err != nil {
		return fmt.Errorf("enumerate addresses: %w", err)
	}
	if len(addrs) == 0 {
		return errors.New("no non-loopback ipv4 addresses found")
	}

	var (
		wg        sync.WaitGroup
		fds       []windows.Handle
		anyOpened bool
		firstErr  error
	)
	for _, ip := range addrs {
		fd, err := openRawSocket(ip)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("open raw socket on %s: %w", ip, err)
			}
			continue
		}
		anyOpened = true
		fds = append(fds, fd)
		wg.Add(1)
		go func(fd windows.Handle) {
			defer wg.Done()
			receiveLoop(ctx, fd, sink)
		}(fd)
	}

	if !anyOpened {
		return firstErr
	}

	// Close every socket when ctx cancels. receiveLoop blocks in
	// Recvfrom, which only returns once its socket is closed — without
	// this, wg.Wait would deadlock on a silent socket and the capture
	// supervisor could never reopen after a stall.
	go func() {
		<-ctx.Done()
		for _, fd := range fds {
			windows.Closesocket(fd)
		}
	}()

	wg.Wait()
	return ctx.Err()
}

func localUnicastV4() ([]net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var out []net.IP
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			ipNet, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			ip4 := ipNet.IP.To4()
			if ip4 == nil {
				continue
			}
			out = append(out, ip4)
		}
	}
	return out, nil
}

func openRawSocket(ip net.IP) (windows.Handle, error) {
	fd, err := windows.WSASocket(
		windows.AF_INET,
		windows.SOCK_RAW,
		windows.IPPROTO_IP,
		nil, 0, windows.WSA_FLAG_OVERLAPPED,
	)
	if err != nil {
		return 0, fmt.Errorf("wsasocket: %w", err)
	}

	sa := &windows.SockaddrInet4{Port: 0}
	copy(sa.Addr[:], ip.To4())
	if err := windows.Bind(fd, sa); err != nil {
		windows.Closesocket(fd)
		return 0, fmt.Errorf("bind: %w", err)
	}

	if err := windows.SetsockoptInt(fd, windows.IPPROTO_IP, windows.IP_HDRINCL, 1); err != nil {
		windows.Closesocket(fd)
		return 0, fmt.Errorf("setsockopt IP_HDRINCL: %w", err)
	}

	// SIO_RCVALL = 0x98000001 (capture all IPv4 packets through this interface).
	// Requires administrator privileges.
	const SIO_RCVALL = 0x98000001
	var (
		in       uint32 = 1
		returned uint32
	)
	if err := windows.WSAIoctl(
		fd,
		SIO_RCVALL,
		(*byte)(unsafe.Pointer(&in)),
		4,
		nil, 0,
		&returned,
		nil, 0,
	); err != nil {
		windows.Closesocket(fd)
		return 0, fmt.Errorf("WSAIoctl SIO_RCVALL (administrator required?): %w", err)
	}

	return fd, nil
}

func receiveLoop(ctx context.Context, fd windows.Handle, sink Sink) {
	buf := make([]byte, 65535)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		n, _, err := windows.Recvfrom(fd, buf, 0)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			// Transient errors are common on raw sockets (e.g. WSAEMSGSIZE
			// for jumbograms). Just keep going.
			continue
		}
		if n <= 0 {
			continue
		}
		processIPv4(buf[:n], sink)
	}
}

func processIPv4(frame []byte, sink Sink) {
	if len(frame) < 20 {
		return
	}
	verIHL := frame[0]
	if verIHL>>4 != 4 {
		return
	}
	ihl := int(verIHL&0x0F) * 4
	if ihl < 20 || len(frame) < ihl {
		return
	}

	flagsFrag := binary.BigEndian.Uint16(frame[6:8])
	hasMoreFragments := flagsFrag&0x2000 != 0
	fragOffset := int(flagsFrag&0x1FFF) * 8
	if hasMoreFragments || fragOffset != 0 {
		return
	}

	if frame[9] != 17 {
		return
	}
	if len(frame) < ihl+8 {
		return
	}

	udp := frame[ihl:]
	srcPort := binary.BigEndian.Uint16(udp[0:2])
	dstPort := binary.BigEndian.Uint16(udp[2:4])
	udpLen := binary.BigEndian.Uint16(udp[4:6])

	payloadStart := ihl + 8
	if payloadStart >= len(frame) {
		return
	}
	maxPayload := len(frame) - payloadStart
	payloadLen := int(udpLen) - 8
	if payloadLen <= 0 {
		return
	}
	if payloadLen > maxPayload {
		payloadLen = maxPayload
	}
	payload := frame[payloadStart : payloadStart+payloadLen]

	if !photonports.IsPhoton(srcPort) && !photonports.IsPhoton(dstPort) {
		if !photonports.LooksLikePhotonPayload(payload) {
			return
		}
	}

	sink.OnPacket(Packet{SrcPort: srcPort, DstPort: dstPort, Payload: payload})
}
