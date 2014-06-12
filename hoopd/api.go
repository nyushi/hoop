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
	Hoops map[int]*hoop.Hoop
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
	m.Handle("/ports/", http.StripPrefix("/ports/", http.HandlerFunc(api.handlePort)))
	m.Handle("/ports", http.StripPrefix("/ports", http.HandlerFunc(api.handlePorts)))
	api.Server.Handler = m
}

func (api *apiServer) start() {
	log.Println("start server")
	log.Fatal(api.ListenAndServe())
}

func (api *apiServer) handlePorts(w http.ResponseWriter, r *http.Request) {
	b := bytes.NewBuffer(nil)
	for lport, h := range api.Hoops {
		b.WriteString(fmt.Sprintf("%d -> %s\n", lport, h.Remote))
	}
	w.Write(b.Bytes())
}

func (api *apiServer) handlePort(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	lport64, err := strconv.ParseInt(r.URL.Path, 10, 64)
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte("invalid listen port"))
		return
	}
	lport := int(lport64)

	remote, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte("failed to read body"))
	}

	if r.Method == "DELETE" {
		h, ok := api.Hoops[lport]
		if ok {
			h.Stop()
		}
		delete(api.Hoops, lport)
		w.Write([]byte("OK"))
		log.Printf("delete %d -> %s", lport, string(remote))
		return
	}

	log.Printf("update %d -> %s", lport, string(remote))

	h, ok := api.Hoops[lport]
	if !ok {
		h = hoop.NewHoop(lport, string(remote))
		api.Hoops[lport] = h
		err = h.Start()
	} else {
		h.Remote = string(remote)
	}
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}
	w.Write([]byte("OK"))
}
