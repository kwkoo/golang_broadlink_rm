package broadlinkrm

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

const defaultTimeout = 5 // seconds

// Broadlink keeps a track of all the devices and sockets.
type Broadlink struct {
	timeout int // in seconds
	devices []device
}

// WithTimeout sets the timeout for all subsequent read operations.
func (b *Broadlink) WithTimeout(t int) *Broadlink {
	b.timeout = t
	return b
}

// Discover will populate the Broadlink struct with a slice of Devices.
func (b *Broadlink) Discover() error {
	addresses, err := hostAddresses()
	if err != nil {
		return fmt.Errorf("error retrieving list of host addresses: %v", err)
	}

	for _, ip := range addresses {
		conn, err := net.ListenPacket("udp4", ip.String()+":0")
		if err != nil {
			log.Printf("could not bind UDP listener to %v: %v", ip.String(), err)
			continue
		}
		log.Printf("Listening to address %v", conn.LocalAddr().String())
		err = sendBroadcastPacket(conn)
		if err != nil {
			log.Printf("Error sending broadcast packet: %v", err)
			conn.Close()
			continue
		}
		b.readPacket(conn)
		conn.Close()
	}

	return nil
}

// Learn sends an enterLearning command to the first device.
func (b *Broadlink) Learn() error {
	if len(b.devices) == 0 {
		return fmt.Errorf("no devices")
	}
	d := b.devices[0]
	resp, err := d.learn()
	if err != nil {
		return fmt.Errorf("error while calling learn: %v", err)
	}
	log.Printf("Received response of payload type %v", resp.Type)
	dumpPacket(resp.Data)
	return nil
}

// Send sends the argument to the first device.
func (b *Broadlink) Send(s string) error {
	if len(b.devices) == 0 {
		return fmt.Errorf("no devices")
	}
	d := b.devices[0]
	return d.sendString(s)
}

func hostAddresses() ([]net.IP, error) {
	var filtered []net.IP
	addresses, err := net.InterfaceAddrs()
	if err != nil {
		return filtered, fmt.Errorf("could not retrieve host addresses: %v", err)
	}
	for _, a := range addresses {
		s := a.String()
		if index := strings.Index(s, "/"); index != -1 {
			s = s[:index]
		}
		ip := net.ParseIP(s)
		if ip == nil {
			continue
		}
		ip4 := ip.To4()
		if ip4 == nil || ip4.IsLoopback() {
			continue
		}
		filtered = append(filtered, ip4)
	}
	return filtered, nil
}

func (b *Broadlink) readPacket(conn net.PacketConn) {
	var buf [1024]byte
	if b.timeout <= 0 {
		b.timeout = defaultTimeout
	}
	for {
		conn.SetReadDeadline(time.Now().Add(time.Duration(b.timeout) * time.Second))
		plen, remote, err := conn.ReadFrom(buf[:])
		if err != nil {
			e, ok := err.(net.Error)
			if ok && e.Timeout() {
				break
			}
			log.Printf("Error reading UDP packet: %v", err)
		}
		log.Printf("Received packet of length %v bytes from %v", plen, remote.String())
		if plen < 0x40 {
			log.Print("Ignoring packet because it is too short")
			return
		}
		var mac net.HardwareAddr
		mac = append(mac, buf[0x3f])
		mac = append(mac, buf[0x3e])
		mac = append(mac, buf[0x3d])
		mac = append(mac, buf[0x3c])
		mac = append(mac, buf[0x3b])
		mac = append(mac, buf[0x3a])

		deviceType := (int)(buf[0x34]) | ((int)(buf[0x35]) << 8)

		b.addDevice(conn.LocalAddr().String(), remote, mac, deviceType)
	}
}

func (b *Broadlink) addDevice(localAddr string, remoteAddr net.Addr, mac net.HardwareAddr, deviceType int) {
	known, name, supported := isKnownDevice(deviceType)
	if !known {
		log.Printf("Unknown device at address %v, MAC %v", remoteAddr.String(), mac.String())
		return
	}
	if !supported {
		log.Printf("Unsupported %v found at address %v, MAC %v", name, remoteAddr.String(), mac.String())
	}
	if strings.Contains(localAddr, ":") {
		index := strings.Index(localAddr, ":")
		localAddr = localAddr[:index]
	}
	log.Printf("Found a supported %v at address %v, MAC %v from local address %v", name, remoteAddr.String(), mac.String(), localAddr)
	dev, err := newDevice(localAddr, remoteAddr.String(), mac, b.timeout)
	if err != nil {
		log.Printf("Error creating new device: %v", err)
	}
	b.devices = append(b.devices, dev)
}

func sendBroadcastPacket(conn net.PacketConn) error {
	ip, port, err := parseIPAndPort(conn.LocalAddr().String())
	if err != nil {
		return err
	}

	var packet [0x30]byte

	t := currentTime()
	copy(packet[0x08:], t[:])
	copy(packet[0x18:], ip[:])
	copy(packet[0x1c:], port[:])
	packet[0x26] = 6
	checksum := calculateChecksum(packet[:])
	copy(packet[0x20:], checksum[:])

	return sendPacket(packet[:], conn, "255.255.255.255:80")
}

func sendPacket(p []byte, conn net.PacketConn, dest string) error {
	destAddr, err := net.ResolveUDPAddr("udp", "255.255.255.255:80")
	if err != nil {
		return fmt.Errorf("could not resolve broadcast address: %v", err)
	}

	_, err = conn.WriteTo(p, destAddr)
	if err != nil {
		return fmt.Errorf("error while writing broadcast message: %v", err)
	}

	return nil
}

func parseIPAndPort(address string) ([4]byte, [2]byte, error) {
	var ip [4]byte
	var port [2]byte

	if !strings.Contains(address, ":") {
		return ip, port, fmt.Errorf("%v is not of the form XXX.XXX.XXX.XXX:XXX", address)
	}

	index := strings.Index(address, ":")
	p, err := strconv.Atoi(address[index+1:])
	if err != nil {
		return [4]byte{}, [2]byte{},

			fmt.Errorf("could not parse port number %v", address[index+1:])
	}
	port[0] = (byte)(p & 0xff)
	port[1] = (byte)(p >> 8)

	components := strings.Split(address[:index], ".")
	if len(components) != 4 {
		return ip, port, fmt.Errorf("%v is not of the form XXX.XXX.XXX.XXX", address[:index])
	}

	for i := 0; i < 4; i++ {
		tmp, err := strconv.Atoi(components[i])
		if err != nil || tmp < 0 || tmp > 255 {
			return ip, port, fmt.Errorf("%v is not a valid IP address", address[:index])
		}
		ip[i] = (byte)(tmp)
	}

	return ip, port, nil
}

func currentTime() [12]byte {
	var b [12]byte

	now := time.Now()
	_, offset := now.Local().Zone()
	offset = offset / 3600

	if offset < 0 {
		b[0] = (byte)(0xff + offset - 1)
		b[1] = 0xff
		b[2] = 0xff
		b[3] = 0xff
	} else {
		b[0] = (byte)(offset)
		b[1] = 0
		b[2] = 0
		b[3] = 0
	}

	year := now.Year()
	b[4] = (byte)(year & 0xff)
	b[5] = (byte)(year >> 8)
	b[6] = (byte)(now.Minute())
	b[7] = (byte)(now.Hour())
	b[8] = (byte)(year % 100)
	b[9] = (byte)(now.Weekday())
	b[10] = (byte)(now.Day())
	b[11] = (byte)(now.Month())

	return b
}

func calculateChecksum(p []byte) [2]byte {
	checksum := 0xbeaf

	for _, v := range p {
		checksum += (int)(v)
	}

	checksum = checksum & 0xffff

	return [2]byte{(byte)(checksum & 0xff), (byte)(checksum >> 8)}
}
