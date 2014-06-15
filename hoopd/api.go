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
	TCPHoops map[int]*hoop.Hoop
}

func newAPIServer() *apiServer {
	s := &http.Server{
		Addr:           ":8080",
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	api := &apiServer{s, make(map[int]*hoop.Hoop)}
	api.setUp()
	return api
}

func (api *apiServer) setUp() {
	m := http.NewServeMux()
	m.Handle("/ports/tcp/", http.StripPrefix("/ports/tcp/", http.HandlerFunc(api.handleTCPPort)))
	m.Handle("/ports", http.StripPrefix("/ports", http.HandlerFunc(api.handlePorts)))
	api.Server.Handler = m
}

func (api *apiServer) start() {
	log.Println("start server")
	log.Fatal(api.ListenAndServe())
}

func (api *apiServer) handlePorts(w http.ResponseWriter, r *http.Request) {
	b := bytes.NewBuffer(nil)
	for lport, h := range api.TCPHoops {
		b.WriteString(fmt.Sprintf("%d -> %s\n", lport, h.Remote))
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

func (api *apiServer) handleTCPPort(w http.ResponseWriter, r *http.Request) {
	lport, err := api.parseLocalPort(r)
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte("invalid listen port"))
		return
	}

	if r.Method == "DELETE" {
		h, ok := api.TCPHoops[lport]
		if ok {
			h.Stop()
		}
		delete(api.TCPHoops, lport)
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

	h, ok := api.TCPHoops[lport]
	if !ok {
		h = hoop.NewHoop(lport, string(remote))
		api.TCPHoops[lport] = h
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
