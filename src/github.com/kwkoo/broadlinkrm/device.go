package broadlinkrm

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"
)

const learnTimeout = 20 // seconds

// ResponseType denotes the type of payload.
type ResponseType int

// Enumerations of PayloadType.
const (
	Unknown ResponseType = iota
	AuthOK
	Temperature
	CommandOK
	RawData
	RawRFData
	RawRFData2
)

// Response represents a decrypted payload from the device.
type Response struct {
	Type ResponseType
	Data []byte
}

type device struct {
	conn       *net.PacketConn
	remoteAddr string
	timeout    int
	deviceType int
	mac        net.HardwareAddr
	count      int
	key        []byte
	iv         []byte
	id         []byte
}

func newDevice(remoteAddr string, mac net.HardwareAddr, timeout, deviceType int) (*device, error) {
	rand.Seed(time.Now().Unix())
	d := &device{
		remoteAddr: remoteAddr,
		timeout:    timeout,
		deviceType: deviceType,
		mac:        mac,
		count:      rand.Intn(0xffff),
		key:        []byte{0x09, 0x76, 0x28, 0x34, 0x3f, 0xe9, 0x9e, 0x23, 0x76, 0x5c, 0x15, 0x13, 0xac, 0xcf, 0x8b, 0x02},
		iv:         []byte{0x56, 0x2e, 0x17, 0x99, 0x6d, 0x09, 0x3d, 0x28, 0xdd, 0xb3, 0xba, 0x69, 0x5a, 0x2e, 0x6f, 0x58},
		id:         []byte{0, 0, 0, 0},
	}

	response, err := d.serverRequest(authenticatePayload, d.timeout)
	d.close()
	if err != nil {
		return d, fmt.Errorf("error making server request: %v", err)
	}
	if response.Type != AuthOK {
		return d, fmt.Errorf("did not get an affirmative response to the authenticaton request - expected %v but got %v instead", AuthOK, response.Type)
	}

	return d, nil
}

// newManualDevice lets you create a device by specifying a key and id,
// skipping the authentication phase. All fields aside from the mac address are
// mandatory.
func newManualDevice(ip, mac, key, id string, timeout, deviceType int) (*device, error) {
	parsedip := net.ParseIP(ip)
	if parsedip.String() == "<nil>" {
		return nil, fmt.Errorf("%v is not a valid IP address", ip)
	}
	skipmac := false
	parsedmac, err := net.ParseMAC(mac)
	if err != nil {
		skipmac = true
	}
	keyhex, err := hex.DecodeString(key)
	if err != nil {
		return nil, fmt.Errorf("key %v is an invalid hex string: %v", key, err)
	}
	if len(keyhex) != 16 {
		return nil, fmt.Errorf("key has length of %v bytes - it should have a length of 16 bytes", len(keyhex))
	}
	idhex, err := hex.DecodeString(id)
	if err != nil {
		return nil, fmt.Errorf("id %v is an invalid hex string: %v", id, err)
	}
	if len(idhex) != 4 {
		return nil, fmt.Errorf("id has length of %v bytes - it should have a length of 4 bytes", len(idhex))
	}

	rand.Seed(time.Now().Unix())
	d := &device{
		remoteAddr: parsedip.String(),
		timeout:    timeout,
		count:      rand.Intn(0xffff),
		key:        keyhex,
		iv:         []byte{0x56, 0x2e, 0x17, 0x99, 0x6d, 0x09, 0x3d, 0x28, 0xdd, 0xb3, 0xba, 0x69, 0x5a, 0x2e, 0x6f, 0x58},
		id:         idhex,
	}
	if !skipmac {
		d.mac = parsedmac
	}

	return d, nil
}

func (d *device) serverRequest(fn func() (byte, []byte), timeout int) (Response, error) {
	respPayload := Response{}

	err := d.setupConnection()
	if err != nil {
		return respPayload, fmt.Errorf("could not setup UDP listener: %v", err)
	}
	defer d.close()

	command, reqPayload := fn()
	err = d.sendPacket(command, reqPayload)
	if err != nil {
		return respPayload, fmt.Errorf("could not send packet: %v", err)
	}

	return d.readPacket()
}

func (d *device) close() {
	if d.conn != nil {
		(*d.conn).Close()
		d.conn = nil
	}
}

func (d *device) setupConnection() error {
	if d.conn != nil {
		return nil
	}

	conn, err := net.ListenPacket("udp4", "")
	if err != nil {
		return err
	}

	/*
		file, _ := conn.(*net.UDPConn).File()
		fd := file.Fd()
		syscall.SetsockoptInt((int)(fd), syscall.SOL_SOCKET, syscall.SO_REUSEPORT, 0)
	*/

	d.conn = &conn
	return nil
}

func authenticatePayload() (byte, []byte) {
	payload := make([]byte, 0x50, 0x50)
	payload[0x04] = 0x31
	payload[0x05] = 0x31
	payload[0x06] = 0x31
	payload[0x07] = 0x31
	payload[0x08] = 0x31
	payload[0x09] = 0x31
	payload[0x0a] = 0x31
	payload[0x0b] = 0x31
	payload[0x0c] = 0x31
	payload[0x0d] = 0x31
	payload[0x0e] = 0x31
	payload[0x0f] = 0x31
	payload[0x10] = 0x31
	payload[0x11] = 0x31
	payload[0x12] = 0x31
	payload[0x1e] = 0x01
	payload[0x2d] = 0x01
	payload[0x30] = 'T'
	payload[0x31] = 'e'
	payload[0x32] = 's'
	payload[0x33] = 't'
	payload[0x34] = ' '
	payload[0x35] = ' '
	payload[0x36] = '1'

	return 0x65, payload
}

func (d *device) sendPacket(command byte, payload []byte) error {
	d.count = (d.count + 1) & 0xffff
	header := make([]byte, 0x38, 0x38)
	header[0x00] = 0x5a
	header[0x01] = 0xa5
	header[0x02] = 0xaa
	header[0x03] = 0x55
	header[0x04] = 0x5a
	header[0x05] = 0xa5
	header[0x06] = 0xaa
	header[0x07] = 0x55
	header[0x24] = 0x2a
	header[0x25] = 0x27
	header[0x26] = command
	header[0x28] = (byte)(d.count & 0xff)
	header[0x29] = (byte)(d.count >> 8)
	header[0x2a] = d.mac[5]
	header[0x2b] = d.mac[4]
	header[0x2c] = d.mac[3]
	header[0x2d] = d.mac[2]
	header[0x2e] = d.mac[1]
	header[0x2f] = d.mac[0]
	header[0x30] = d.id[0]
	header[0x31] = d.id[1]
	header[0x32] = d.id[2]
	header[0x33] = d.id[3]

	checksum := 0xbeaf
	for _, v := range payload {
		checksum += (int)(v)
		checksum = checksum & 0xffff
	}

	block, err := aes.NewCipher(d.key)
	if err != nil {
		return fmt.Errorf("unable to create new AES cipher: %v", err)
	}
	mode := cipher.NewCBCEncrypter(block, d.iv)
	encryptedPayload := make([]byte, len(payload))
	mode.CryptBlocks(encryptedPayload, payload)

	packet := make([]byte, len(header)+len(encryptedPayload))
	copy(packet, header)
	copy(packet[len(header):], encryptedPayload)

	packet[0x34] = (byte)(checksum & 0xff)
	packet[0x35] = (byte)(checksum >> 8)

	checksum = 0xbeaf
	for _, v := range packet {
		checksum += (int)(v)
		checksum = checksum & 0xffff
	}
	packet[0x20] = (byte)(checksum & 0xff)
	packet[0x21] = (byte)(checksum >> 8)

	destAddr, err := net.ResolveUDPAddr("udp", d.remoteAddr+":80")
	if err != nil {
		return fmt.Errorf("could not resolve device address %v: %v", d.remoteAddr, err)
	}

	err = d.setupConnection()
	if err != nil {
		return err
	}
	_, err = (*d.conn).WriteTo(packet, destAddr)
	if err != nil {
		return fmt.Errorf("could not send send packet: %v", err)
	}
	return nil
}

func (d *device) readPacket() (Response, error) {
	var buf [1024]byte
	processedPayload := Response{Type: Unknown}
	if d.conn == nil {
		return processedPayload, errors.New("a connection to the device does not exist")
	}
	(*d.conn).SetReadDeadline(time.Now().Add(time.Duration(d.timeout) * time.Second))
	plen, _, err := (*d.conn).ReadFrom(buf[:])
	if err != nil {
		return processedPayload, fmt.Errorf("error reading UDP packet: %v", err)
	}

	if plen < 0x38+16 {
		return processedPayload, fmt.Errorf("received a packet with a length of %v which is too short", plen)
	}
	encryptedPayload := make([]byte, plen-0x38, plen-0x38)
	copy(encryptedPayload, buf[0x38:plen])

	block, err := aes.NewCipher(d.key)
	if err != nil {
		return processedPayload, fmt.Errorf("error creating new decryption cipher: %v", err)
	}
	payload := make([]byte, len(encryptedPayload), len(encryptedPayload))
	mode := cipher.NewCBCDecrypter(block, d.iv)
	mode.CryptBlocks(payload, encryptedPayload)

	command := buf[0x26]
	if command == 0xe9 {
		copy(d.key, payload[0x04:0x14])
		copy(d.id, payload[:0x04])
		log.Printf("Device ready - updating to a new key %v and new id %v", hex.EncodeToString(d.key), hex.EncodeToString(d.id))
		processedPayload.Type = AuthOK
		return processedPayload, nil
	}

	if command == 0xee || command == 0xef {
		param := payload[0]
		errorCode := (int)(buf[0x22]) | ((int)(buf[0x23]) << 8)
		//log.Printf("*** command=0x%02x param=0x%02x errorCode=0x%04v", command, param, errorCode)
		if errorCode != 0 {
			// It's not a real error!
			return processedPayload, nil
		}
		switch param {
		case 1:
			processedPayload.Type = Temperature
			processedPayload.Data = []byte{(payload[0x4]*10 + payload[0x5]) / 10}
		case 2:
			processedPayload.Type = CommandOK
		case 4:
			processedPayload.Type = RawData
			processedPayload.Data = make([]byte, len(payload)-4, len(payload)-4)
			copy(processedPayload.Data, payload[4:])
		case 26:
			processedPayload.Data = make([]byte, len(payload)-4, len(payload)-4)
			copy(processedPayload.Data, payload[4:])
			if payload[0x4] == 1 {
				processedPayload.Type = RawRFData
			}
		case 27:
			processedPayload.Data = make([]byte, len(payload)-4, len(payload)-4)
			copy(processedPayload.Data, payload[4:])
			if payload[0x4] == 1 {
				processedPayload.Type = RawRFData2
			}
		}
		return processedPayload, nil
	}

	log.Printf("Unhandled command %v", command)
	return processedPayload, fmt.Errorf("unhandled command - %v", command)
}

func (d *device) checkData() (Response, error) {
	resp, err := d.serverRequest(checkDataPayload, d.timeout)
	if err != nil {
		return resp, fmt.Errorf("error making CheckData request: %v", err)
	}

	return resp, nil
}

func (d *device) sendString(s string) error {
	data, err := hex.DecodeString(s)
	if err != nil {
		return fmt.Errorf("error converting %v to hex: %v", s, err)
	}
	return d.sendData(data)
}

func (d *device) sendData(data []byte) error {
	var command byte
	command = 0x6a
	reqPayload := make([]byte, len(data)+4, len(data)+4)
	reqPayload[0] = 0x02
	reqPayload[1] = 0x00
	reqPayload[2] = 0x00
	reqPayload[3] = 0x00
	copy(reqPayload[4:], data)

	defer d.close()
	err := d.sendPacket(command, reqPayload)
	if err != nil {
		return fmt.Errorf("could not send packet: %v", err)
	}

	response, err := d.readPacket()
	if err != nil {
		return fmt.Errorf("error reading reseponse: %v", err)
	}
	if response.Type != CommandOK {
		return fmt.Errorf("did not get the expected response type from the device - expected %v but got %v instead", CommandOK, response.Type)
	}
	log.Print("Send successful")
	return nil
}

func (d *device) learn() (Response, error) {
	deadline := time.Now().Add(learnTimeout * time.Second)
	defer d.close()
	_, err := d.serverRequest(enterLearningPayload, d.timeout)
	if err != nil {
		return Response{}, fmt.Errorf("error making learning request: %v", err)
	}

	for {
		if time.Now().After(deadline) {
			return Response{}, errors.New("learning timeout")
		}
		resp, err := d.checkData()
		if err != nil {
			continue // we continue because it's probably just timed out waiting for a response from check data
		}
		if resp.Type == RawData {
			return resp, nil
		}
	}
}

func (d *device) checkTemperature() (Response, error) {
	defer d.close()
	resp, err := d.serverRequest(checkTemperaturePayload, d.timeout)
	if err != nil {
		return resp, fmt.Errorf("error making check temperature request: %v", err)
	}
	return resp, nil
}

func (d *device) cancelLearn() {
	command, payload := cancelLearnPayload()
	d.sendPacket(command, payload)
	d.close()
}

func checkDataPayload() (byte, []byte) {
	return 0x6a, basicRequestPayload(4)
}

func enterLearningPayload() (byte, []byte) {
	return 0x6a, basicRequestPayload(3)
}

func checkTemperaturePayload() (byte, []byte) {
	return 0x6a, basicRequestPayload(1)
}

func cancelLearnPayload() (byte, []byte) {
	return 0x6a, basicRequestPayload(0x1e)
}

func enterRFSweepPayload() (byte, []byte) {
	return 0x6a, basicRequestPayload(0x19)
}

func checkRFData() (byte, []byte) {
	return 0x6a, basicRequestPayload(0x1a)
}

func checkRFData2() (byte, []byte) {
	return 0x6a, basicRequestPayload(0x1b)
}

func basicRequestPayload(command byte) []byte {
	payload := make([]byte, 16, 16)
	payload[0] = command
	return payload
}
