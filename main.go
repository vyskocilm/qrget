// 
//  qrget: simple download tool using QR code
//  (c) 2018 michal vyskocil mail-starts-with-g com
//  licensed under 3 Clause BSD License
//

package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"time"

	qrcode "github.com/skip2/go-qrcode"
	/*
	   nucular "github.com/aarzilli/nucular"
	   style "github.com/aarzilli/nucular/style"
	*/)

type ErrGoget string

func (self ErrGoget) Error() string {
	return string(self)
}

//  find address of wireless interface or error
//  on !Linux
//  if no wlan was found
//  if more than one wlan was found
//  otherwise
func findWirelessIP() (string, net.IP, error) {
	// TODO: use build time way - https://stackoverflow.com/questions/19847594/how-to-reliably-detect-os-platform-in-go
	if runtime.GOOS != "linux" {
		return "", nil, ErrGoget("Support of other OSes than Linux is not (yet) supported")
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return "", nil, err
	}

	var ip net.IP
	var wlan string
	wlan_found := false
	for _, iface := range ifaces {
		fi, err := os.Lstat(fmt.Sprintf("/sys/class/net/%s/wireless/", iface.Name))
		if err != nil {
			continue
		}
		if fi.Mode().IsDir() {
			if wlan_found {
				return "", nil, ErrGoget("Support for more than one wlan interface is not implemented")
			}
			wlan_found = true
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
	if !wlan_found {
		return "", nil, ErrGoget("Support for non wlan interface is not implemented")
	}

	return wlan, ip, nil
}

func main() {

	// detect local wlan interface
	name, ip, err := findWirelessIP()
	if err != nil {
		panic(err)
	}
	// TODO: command line parsing, export the file (or dir?)
	file := "main.go"
	port := 8042
	url := fmt.Sprintf("http://%s:%d/%s", ip, port, file)
	log.Printf("I: name=\"%s\", ip=%s, url=%s\n", name, ip, url)

	// TODO: test too long QR codes
	// generate QR code
	//png, err := qrcode.Encode(url, qrcode.Medium, 256)
	err = qrcode.WriteFile(url, qrcode.Medium, 256, "download.png")
	if err != nil {
		panic(err)
	}
	log.Println("I: 'download.png' saved, please open it via different app")

	// run HTTP server to serve the file
	// TODO: gracefull shutdow nwhen file was downloaded
	// TODO: look here - https://stackoverflow.com/questions/39320025/how-to-stop-http-listenandserve
	go func() {
		http.HandleFunc(fmt.Sprintf("/%s", file), func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, file)
		})

		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
	}()

    log.Printf("I: serving %s for 60 seconds\n", file)
	time.Sleep(60 * time.Second)
    log.Printf("I: finished")
	/*
	   // TODO: show qrcode
	   wnd := nucular.NewMasterWindow(0, "Title")
	   wnd.SetStyle(style.FromTheme(nucular.DarkTheme, 1.0))
	   wnd.Main()
	*/
}
