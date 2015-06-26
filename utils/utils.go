package utils

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
)

// GetMyIP returns my ip
func GetMyIP() (IP string) {
	ip, err := myExternalIP()
	if err != nil {
		log.Println("could not get my external adress!")
		HandleErr(err)
	} else {
		log.Printf("myExternalAdress = %s", ip)
	}
	IP = ip //external IP
	return
}

func myExternalIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("are you connected to the network?")
}

// HandleErr does not really handle errors
func HandleErr(err error) {
	if err != nil {
		log.Printf("No one in the chat yet, error = %s", err.Error())
	}
}

// ExitOnError outputs error and quits
func ExitOnError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}
