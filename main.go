//
//  qrget: simple download tool using QR code
//  (c) 2018 michal vyskocil mail-starts-with-g com
//  licensed under 3 Clause BSD License
//

package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"image/draw"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"time"

	qrcode "github.com/skip2/go-qrcode"
	nucular "github.com/aarzilli/nucular"
    nstyle "github.com/aarzilli/nucular/style"
)

type ErrGoget string

func (self ErrGoget) Error() string {
	return string(self)
}

// UI model
type qrgetModel struct {
	Img *image.RGBA
}

func newQrgetModel() (qm *qrgetModel) {
    qm = &qrgetModel{}
	fh, err := os.Open("download.png")      // fixme, pass []byte
	if err == nil {
		defer fh.Close()
		img, _ := png.Decode(fh)
		qm.Img = image.NewRGBA(img.Bounds())
		draw.Draw(qm.Img, img.Bounds(), img, image.Point{}, draw.Src)
	}

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

func drawqr(w *nucular.Window) {
    //keybindings(w)
    fh, err := os.Open("download.png")
    if err != nil {
        w.Row(25).Dynamic(1)
        w.Label("could not load qrcode image", "LC")
    } else {
        defer fh.Close()
        img, _ := png.Decode(fh)
        img_rgba := image.NewRGBA(img.Bounds())
        draw.Draw(img_rgba, img.Bounds(), img, image.Point{}, draw.Src)
		w.Image(img_rgba)
    }

}

func main() {
	// Argument parsing
	var verbose bool
    var timeout time.Duration
	flag.BoolVar(&verbose, "verbose", false, "Increase verbosity")
	flag.BoolVar(&verbose, "v", false, "Increase verbosity")
    flag.DurationVar(&timeout, "timeout", 300 * time.Second, "Timeout for HTTP serve, 0 is infinite")

	flag.Parse()

	var dir_mode bool
	var name string // file or directory name
	var err error
	switch len(flag.Args()) {
	case 0:
		dir_mode = true
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
		if dir_mode {
			log.Printf("I: serving files from directory '%s'", name)
		} else {
			log.Printf("I: serving file '%s'", name)
		}
	}

	if dir_mode {
		log.Fatal("Directory mode is not yet implemented")
	}

	// detect local wlan interface
	wlan_name, ip, err := findWirelessIP()
	if err != nil {
		panic(err)
	}

	// generate url
	port := 8042 // TODO: generate randomized ports
	url := fmt.Sprintf("http://%s:%d/%s", ip, port, name)
	if verbose {
		log.Printf("I: wlan=\"%s\", ip=%s, url=%s\n", wlan_name, ip, url)
	}

	// TODO: test too long QR codes
	// generate QR code
	//png, err := qrcode.Encode(url, qrcode.Medium, 256)
	err = qrcode.WriteFile(url, qrcode.Medium, 256, "download.png")
	if err != nil {
		panic(err)
	}
	go func() {
		if verbose {
			log.Println("I: 'download.png' saved, opening via xdg-open")
		}
        /*
		_, err := os.StartProcess("/usr/bin/xdg-open", []string{"xdg-open", "download.png"}, &os.ProcAttr{Dir: ".", Files: []*os.File{os.Stdin, os.Stdout, os.Stderr}})
		if err != nil {
			panic(err)
		}*/
	}()

	// run HTTP server to serve the file
	// TODO: gracefull shutdow nwhen file was downloaded
	// TODO: look here - https://stackoverflow.com/questions/39320025/how-to-stop-http-listenandserve
	go func() {
		http.HandleFunc(fmt.Sprintf("/%s", name), func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, name)
		})

		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
	}()

    qm := newQrgetModel()

    wnd := nucular.NewMasterWindowSize(0, url, image.Point{276, 280}, qm.updatefn)
    wnd.SetStyle(nstyle.FromTheme(nstyle.DefaultTheme, 1.0))
    go func() {
        wnd.Main()
    }()

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
	log.Printf("I: finished")
}
