//
//  qrget: simple download tool using QR code
//  (c) 2018 michal vyskocil mail-starts-with-g com
//  licensed under 3 Clause BSD License
//

package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"time"

	freeport "github.com/phayes/freeport"
	nucular "github.com/aarzilli/nucular"
	nstyle "github.com/aarzilli/nucular/style"
	qrcode "github.com/skip2/go-qrcode"
)

type errQqget string

func (e errQqget) Error() string {
	return string(e)
}

// UI model
type qrgetModel struct {
	Img *image.RGBA
}

func newQrgetModel(qr []byte) (qm *qrgetModel) {
	qm = &qrgetModel{}
	r := bytes.NewReader(qr)
	img, _ := png.Decode(r)
	qm.Img = image.NewRGBA(img.Bounds())
	draw.Draw(qm.Img, img.Bounds(), img, image.Point{}, draw.Src)

	return qm
}

func (qm *qrgetModel) updatefn(w *nucular.Window) {
	w.RowScaled(256).StaticScaled(256)
	w.Image(qm.Img)
}

//  find address of wireless interface or error
//  on !Linux
//  if no wlan was found
//  if more than one wlan was found
//  otherwise
func findWirelessIP() (string, net.IP, error) {
	// TODO: use build time way - https://stackoverflow.com/questions/19847594/how-to-reliably-detect-os-platform-in-go
	if runtime.GOOS != "linux" {
		return "", nil, errQqget("Support of other OSes than Linux is not (yet) supported")
	}

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

func main() {
	// Argument parsing
	var verbose bool
	var timeout time.Duration
	flag.BoolVar(&verbose, "verbose", false, "Increase verbosity")
	flag.BoolVar(&verbose, "v", false, "Increase verbosity")
	flag.DurationVar(&timeout, "timeout", 300*time.Second, "Timeout for HTTP serve, 0 is infinite")

	flag.Parse()

	var dirMode bool
	var name string // file or directory name
	var err error
	switch len(flag.Args()) {
	case 0:
		dirMode = true
		name, err = os.Getwd()
		if err != nil {
			panic(err)
		}
	case 1:
		name = flag.Args()[0]
	default:
		log.Fatal("Serving more than one file is not yet supported")
	}

	if verbose {
		if dirMode {
			log.Printf("I: serving directory '%s'", name)
		} else {
			log.Printf("I: serving file '%s'", name)
		}
	}

	// detect local wlan interface
	wlanName, ip, err := findWirelessIP()
	if err != nil {
		panic(err)
	}

	// generate url
	port, err := freeport.GetFreePort()
	if err != nil {
		panic(err)
	}
	url := fmt.Sprintf("http://%s:%d/", ip, port)
	if verbose {
		log.Printf("I: wlan=\"%s\", ip=%s, url=%s\n", wlanName, ip, url)
	}

	// TODO: test too long QR codes
	// generate QR code
	//png, err := qrcode.Encode(url, qrcode.Medium, 256)
	qr, err := qrcode.Encode(url, qrcode.Medium, 256)
	if err != nil {
		panic(err)
	}

	// channel used by several coroutines to notify that we are about end
	endChan := make(chan bool, 1)

	// 1. HTTP Server goroutine
	// run HTTP server to serve the file
	// TODO: gracefull shutdow nwhen file was downloaded
	// TODO: look here - https://stackoverflow.com/questions/39320025/how-to-stop-http-listenandserve
	go func() {
		ports := fmt.Sprintf(":%d", port)
		if dirMode {
			http.Handle("/", http.FileServer(http.Dir(name)))
		} else {
			http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				http.ServeFile(w, r, name)
			})
		}
		log.Fatal(http.ListenAndServe(ports, nil))
	}()

	qm := newQrgetModel(qr)

	wnd := nucular.NewMasterWindowSize(0, url, image.Point{276, 280}, qm.updatefn)
	wnd.SetStyle(nstyle.FromTheme(nstyle.DefaultTheme, 1.0))

	// 2. GUI goroutine
	go func(ec chan<- bool) {
		wnd.Main()
		for {
			if wnd.Closed() {
				ec <- true
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
	}(endChan)

	// 3. Timeout goroutine
	go func(ec chan<- bool) {
		if timeout > 0 {
			if verbose {
				log.Printf("I: serving %s for %s\n", name, timeout)
			}
			time.Sleep(timeout)
		} else {
			if verbose {
				log.Printf("I: serving %s for indifinitelly\n", name)
			}
			for {
				time.Sleep(24 * time.Hour)
			}
		}
		ec <- true
	}(endChan)

	// wait until end
	<-endChan

	wnd.Close()
	if verbose {
		log.Printf("I: finished")
	}
}
