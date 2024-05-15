// Package arping is a native go library to ping a host per arp datagram, or query a host mac address
//
// The currently supported platforms are: Linux and BSD.
//
// The library requires raw socket access. So it must run as root, or with appropriate capabilities under linux:
// `sudo setcap cap_net_raw+ep <BIN>`.
package arping

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"
)

var (
	// ErrTimeout error
	ErrTimeout = errors.New("timeout")

	verboseLog = log.New(io.Discard, "", 0)
	timeout    = time.Duration(500 * time.Millisecond)
)

type Result struct {
	HwAddr   net.HardwareAddr
	Duration time.Duration
}

// Ping sends an arp ping to 'dstIP'
func Ping(dstIP net.IP) ([]Result, error) {
	if err := validateIP(dstIP); err != nil {
		return nil, err
	}

	iface, err := findUsableInterfaceForNetwork(dstIP)
	if err != nil {
		return nil, err
	}
	return PingOverIface(dstIP, *iface)
}

// PingOverIfaceByName sends an arp ping over interface name 'ifaceName' to 'dstIP'
func PingOverIfaceByName(dstIP net.IP, ifaceName string) ([]Result, error) {
	if err := validateIP(dstIP); err != nil {
		return nil, err
	}

	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return nil, err
	}
	return PingOverIface(dstIP, *iface)
}

// PingOverIface sends an arp ping over interface 'iface' to 'dstIP'
func PingOverIface(dstIP net.IP, iface net.Interface) ([]Result, error) {
	if err := validateIP(dstIP); err != nil {
		return nil, err
	}

	srcMac := iface.HardwareAddr
	srcIP, err := findIPInNetworkFromIface(dstIP, iface)
	if err != nil {
		return nil, err
	}

	broadcastMac := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	request := newArpRequest(srcMac, srcIP, broadcastMac, dstIP)

	sock, err := initialize(iface)
	if err != nil {
		return nil, err
	}

	type PingResult struct {
		mac      net.HardwareAddr
		duration time.Duration
		err      error
	}
	pingResultChan := make(chan PingResult)
	running := true

	go func() {
		defer sock.deinitialize()
		// send arp request
		verboseLog.Printf("arping '%s' over interface: '%s' with address: '%s'\n", dstIP, iface.Name, srcIP)
		if sendTime, err := sock.send(request); err != nil {
			pingResultChan <- PingResult{nil, 0, err}
		} else {
			for running {
				// receive arp response
				response, receiveTime, err := sock.receive()

				if err != nil {
					pingResultChan <- PingResult{nil, 0, err}
					return
				}

				if response.IsResponseOf(request) {
					duration := receiveTime.Sub(sendTime)
					verboseLog.Printf("process received arp: srcIP: '%s', srcMac: '%s'\n",
						response.SenderIP(), response.SenderMac())
					pingResultChan <- PingResult{response.SenderMac(), duration, err}
				}

				verboseLog.Printf("ignore received arp: srcIP: '%s', srcMac: '%s'\n",
					response.SenderIP(), response.SenderMac())
			}
		}
	}()

	results := make([]Result, 0)

Break:
	for {
		select {
		case pingResult := <-pingResultChan:
			results = append(results, Result{HwAddr: pingResult.mac, Duration: pingResult.duration})
		case <-time.After(timeout):
			if len(results) == 0 {
				return nil, ErrTimeout
			}

			break Break
		}
	}

	running = false

	return results, nil
}

// GratuitousArp sends an gratuitous arp from 'srcIP'
func GratuitousArp(srcIP net.IP) error {
	if err := validateIP(srcIP); err != nil {
		return err
	}

	iface, err := findUsableInterfaceForNetwork(srcIP)
	if err != nil {
		return err
	}
	return GratuitousArpOverIface(srcIP, *iface)
}

// GratuitousArpOverIfaceByName sends an gratuitous arp over interface name 'ifaceName' from 'srcIP'
func GratuitousArpOverIfaceByName(srcIP net.IP, ifaceName string) error {
	if err := validateIP(srcIP); err != nil {
		return err
	}

	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return err
	}
	return GratuitousArpOverIface(srcIP, *iface)
}

// GratuitousArpOverIface sends an gratuitous arp over interface 'iface' from 'srcIP'
func GratuitousArpOverIface(srcIP net.IP, iface net.Interface) error {
	if err := validateIP(srcIP); err != nil {
		return err
	}

	srcMac := iface.HardwareAddr
	broadcastMac := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	request := newArpRequest(srcMac, srcIP, broadcastMac, srcIP)

	sock, err := initialize(iface)
	if err != nil {
		return err
	}
	defer sock.deinitialize()
	verboseLog.Printf("gratuitous arp over interface: '%s' with address: '%s'\n", iface.Name, srcIP)
	_, err = sock.send(request)
	return err
}

// EnableVerboseLog enables verbose logging on stdout
func EnableVerboseLog() {
	verboseLog = log.New(os.Stdout, "", 0)
}

// SetTimeout sets ping timeout
func SetTimeout(t time.Duration) {
	timeout = t
}

func validateIP(ip net.IP) error {
	// ip must be a valid V4 address
	if len(ip.To4()) != net.IPv4len {
		return fmt.Errorf("not a valid v4 Address: %s", ip)
	}
	return nil
}
