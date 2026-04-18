package monitoring

import (
	"bytes"
	"fmt"
	"net"
)

// SendMagicPacket sends a WOL Magic Packet to a specific MAC address
func SendMagicPacket(macAddr string) error {
	hw, err := net.ParseMAC(macAddr)
	if err != nil {
		return fmt.Errorf("invalid MAC address: %w", err)
	}

	// Build Magic Packet: 6 bytes 0xFF + 16 * MAC
	packet := bytes.NewBuffer(make([]byte, 0, 102))
	for i := 0; i < 6; i++ {
		packet.WriteByte(0xff)
	}
	for i := 0; i < 16; i++ {
		packet.Write(hw)
	}

	// Send to broadcast address
	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{
		IP:   net.IPv4bcast,
		Port: 9,
	})
	if err != nil {
		return fmt.Errorf("dial UDP: %w", err)
	}
	defer conn.Close()

	_, err = conn.Write(packet.Bytes())
	return err
}
