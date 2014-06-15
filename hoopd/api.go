package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/nyushi/hoop"
)

type apiServer struct {
	*http.Server
	Hoops map[int]map[int]*hoop.Hoop
}

func newAPIServer() *apiServer {
	s := &http.Server{
		Addr:           ":8080",
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	m := make(map[int]map[int]*hoop.Hoop)
	m[hoop.TCP] = make(map[int]*hoop.Hoop)
	m[hoop.UDP] = make(map[int]*hoop.Hoop)
	api := &apiServer{s, m}
	api.setUp()
	return api
}

func (api *apiServer) setUp() {
	m := http.NewServeMux()
	m.Handle("/ports/tcp/", http.StripPrefix("/ports/tcp/", http.HandlerFunc(api.getProtoHandler(hoop.TCP))))
	m.Handle("/ports/udp/", http.StripPrefix("/ports/udp/", http.HandlerFunc(api.getProtoHandler(hoop.UDP))))
	m.Handle("/ports", http.StripPrefix("/ports", http.HandlerFunc(api.handlePorts)))
	api.Server.Handler = m
}

func (api *apiServer) start() {
	log.Println("start server")
	log.Fatal(api.ListenAndServe())
}

func (api *apiServer) handlePorts(w http.ResponseWriter, r *http.Request) {
	b := bytes.NewBuffer(nil)

	for p, m := range api.Hoops {
		proto := hoop.ProtoString(p)
		for lport, h := range m {
			b.WriteString(fmt.Sprintf("%s: %d -> %s\n", proto, lport, h.Remote))
		}
	}
	w.Write(b.Bytes())
}

func (api *apiServer) parseLocalPort(r *http.Request) (int, error) {
	lport64, err := strconv.ParseInt(r.URL.Path, 10, 64)
	if err != nil {
		return 0, err
	}
	return int(lport64), nil
}

func (api *apiServer) parseBody(r *http.Request) (remote string, err error) {
	defer r.Body.Close()
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}
	remote = string(b)
	return
}

func (api *apiServer) getProtoHandler(proto int) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		lport, err := api.parseLocalPort(r)
		if err != nil {
			w.WriteHeader(400)
			w.Write([]byte("invalid listen port"))
			return
		}

		if r.Method == "DELETE" {
			h, ok := api.Hoops[proto][lport]
			if ok {
				h.Stop()
			}
			delete(api.Hoops[proto], lport)
			w.Write([]byte("OK"))
			log.Printf("delete %d -> %s", lport, h.Remote)
			return
		}

		remote, err := api.parseBody(r)
		if err != nil {
			w.WriteHeader(400)
			w.Write([]byte("failed to read body"))
			return
		}

		log.Printf("update %d -> %s", lport, remote)

		h, ok := api.Hoops[proto][lport]
		if !ok {
			h = hoop.NewHoop(lport, proto, remote)
			api.Hoops[proto][lport] = h
			err = h.Start()
		} else {
			h.Remote = remote
		}
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(400)
			w.Write([]byte(err.Error()))
			return
		}
		w.Write([]byte("OK"))

	}
}
