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
	"net/http"
	"os"
	"time"
	"net"

	nucular "github.com/aarzilli/nucular"
	nstyle "github.com/aarzilli/nucular/style"
	freeport "github.com/phayes/freeport"
	qrcode "github.com/skip2/go-qrcode"
)

type errQqget string

func (e errQqget) Error() string {
	return string(e)
}

// App model
type qrgetModel struct {
	port int
	dirMode bool
	name string
	verbose bool
	url string
	srv *http.Server
	wnd nucular.MasterWindow
	Img *image.RGBA
}

// save qr code to image
func (qm *qrgetModel) qr(wlanName string, ip net.IP) error {
	qm.url = fmt.Sprintf("http://%s:%d/", ip, qm.port)
	if qm.verbose {
		log.Printf("I: wlan=\"%s\", ip=%s, url=%s\n", wlanName, ip, qm.url)
	}
	qr, err := qrcode.Encode(qm.url, qrcode.Medium, 256)
	if err != nil {
		return err
	}
	r := bytes.NewReader(qr)
	img, _ := png.Decode(r)
	qm.Img = image.NewRGBA(img.Bounds())
	draw.Draw(qm.Img, img.Bounds(), img, image.Point{}, draw.Src)
	return nil
}

func (qm *qrgetModel) updatefn(w *nucular.Window) {
	w.RowScaled(256).StaticScaled(256)
	w.Image(qm.Img)
}

// with help of https://stackoverflow.com/questions/39320025/how-to-stop-http-listenandserve
func (qm *qrgetModel) startHttpServer() {
	if qm.port == 0 {
		panic("port == 0")
	}
	port := fmt.Sprintf(":%d", qm.port)
	qm.srv = &http.Server{Addr: port}

	if qm.dirMode {
		http.Handle("/", http.FileServer(http.Dir(qm.name)))
	} else {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, qm.name)
		})
	}
	go func() {
		if err := qm.srv.ListenAndServe(); err != nil {
			// cannot panic, because this probably is an intentional close
			if err.Error() != "http: Server closed" {
				log.Printf("Httpserver: ListenAndServe() error: %s", err)
			}
		}
	}()
}

func (qm *qrgetModel) masterWindow() nucular.MasterWindow {
	qm.wnd = nucular.NewMasterWindowSize(0, qm.url, image.Point{276, 280}, qm.updatefn)
	qm.wnd.SetStyle(nstyle.FromTheme(nstyle.DefaultTheme, 1.0))
	return qm.wnd
}

func (qm *qrgetModel) closeAll() {
	qm.wnd.Close()
	if err := qm.srv.Shutdown(nil); err != nil {
		panic(err) // failure/timeout shutting down the server gracefully
	}
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
	qm := &qrgetModel{
		dirMode: dirMode,
		name: name,
		verbose: verbose,
		port: port}

	// generate QR code
	err = qm.qr(wlanName, ip)
	if err != nil {
		panic(err)
	}

	// channel used by several coroutines to notify that we are about end
	endChan := make(chan bool, 1)

	// 1. HTTP Server goroutine
	// run HTTP server to serve the file
	qm.startHttpServer()

	wnd := qm.masterWindow()
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

	qm.closeAll()
	if verbose {
		log.Printf("I: finished")
	}
}
