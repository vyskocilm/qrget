package main

import (
	"fmt"
	"net"
	"os"
)

//  find address of wireless interface or error
//  on !Linux
//  if no wlan was found
//  if more than one wlan was found
//  otherwise
func findWirelessIP() (string, net.IP, error) {

	ifaces, err := net.Interfaces()
	if err != nil {
		return "", nil, err
	}

	var ip net.IP
	var wlan string
	wlanFound := false
	for _, iface := range ifaces {
		fi, err := os.Lstat(fmt.Sprintf("/sys/class/net/%s/wireless/", iface.Name))
		if err != nil {
			continue
		}
		if fi.Mode().IsDir() {
			if wlanFound {
				return "", nil, errQqget("Support for more than one wlan interface is not implemented")
			}
			wlanFound = true
			wlan = iface.Name

			// find ip address
			addrs, err := iface.Addrs()
			if err != nil {
				return "", nil, err
			}

			for _, addr := range addrs {
				// https://stackoverflow.com/questions/23558425/how-do-i-get-the-local-ip-address-in-go
				switch v := addr.(type) {
				case *net.IPNet:
					ip = v.IP
				case *net.IPAddr:
					ip = v.IP
				}
				// TODO: here we assume first ip is great, but if not?
				break
			}

		}
	}
	if !wlanFound {
		return "", nil, errQqget("Support for non wlan interface is not implemented")
	}

	return wlan, ip, nil
}
