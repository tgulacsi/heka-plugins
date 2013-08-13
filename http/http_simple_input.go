/***** BEGIN LICENSE BLOCK *****
# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this file,
# You can obtain one at http://mozilla.org/MPL/2.0/.
#
# The Initial Developer of the Original Code is Tamás Gulácsi.
# Portions created by the Initial Developer are Copyright (C) 2013
# the Initial Developer. All Rights Reserved.
#
# ***** END LICENSE BLOCK *****/

package http

import (
	"code.google.com/p/go-uuid/uuid"
	"github.com/mozilla-services/heka/message"
	"github.com/mozilla-services/heka/pipeline"

	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// HTTPSimpleInput holds the address where we listen to POST/PUT HTTP requests
//Can be reached with `curl -XPOST 'http://localhost:5566/?payload=abbraka&severity=3'`
type HTTPSimpleInput struct {
	Address string

	listener net.Listener
	packs    chan *pipeline.PipelinePack
	input    chan *pipeline.PipelinePack
	ds       pipeline.DecoderSet
	stop     chan bool
	errch    chan error
}

// Stop is called when the main hekad wants to stop
func (hsi *HTTPSimpleInput) Stop() {
	if hsi.stop != nil {
		hsi.stop <- true
	}
}

func (hsi *HTTPSimpleInput) listen() {
	var err error
	if hsi.listener, err = net.Listen("tcp", hsi.Address); err != nil {
		hsi.errch <- err
		return
	}
	s := &http.Server{Addr: hsi.Address, Handler: http.HandlerFunc(hsi.handler)}
	if err = s.Serve(hsi.listener); err != nil && hsi != nil && hsi.errch != nil {
		hsi.errch <- err
	}
	if hsi.errch != nil {
		close(hsi.errch)
	}
}

// Run is the main loop which listens for incoming requests and injects the
// messages read into the heka machinery
func (hsi *HTTPSimpleInput) Run(ir pipeline.InputRunner, h pipeline.PluginHelper) (err error) {
	hsi.stop = make(chan bool)
	hsi.input = make(chan *pipeline.PipelinePack)
	hsi.errch = make(chan error, 1)
	hsi.packs = ir.InChan()
	hsi.ds = h.DecoderSet()

	go hsi.listen()
	var pack *pipeline.PipelinePack
INPUT:
	for {
		select {
		case err = <-hsi.errch:
			if err != nil {
				return
			}
		case pack = <-hsi.input:
			ir.Inject(pack)
		case _ = <-hsi.stop:
			if hsi.listener != nil {
				hsi.listener.Close()
				hsi.packs = nil
			}
			break INPUT
		}
	}
	select {
	case err = <-hsi.errch:
		return
	default:
		close(hsi.errch)
		hsi.errch = nil
	}
	return nil
}

func (hsi *HTTPSimpleInput) handler(w http.ResponseWriter, r *http.Request) {
	if r.Body != nil {
		defer r.Body.Close()
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	parsErr := func(err error) {
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		w.Write([]byte{'\n'})
	}
	if r.Method != "POST" && r.Method != "PUT" {
		parsErr(fmt.Errorf("POST needed!"))
		return
	}
	var err error
	pack := <-hsi.packs
	defer func() {
		if err != nil && pack != nil {
			pack.Recycle()
		}
	}()

	ct := r.Header.Get("Content-Type")
	if ct != "" && strings.HasPrefix(ct, "application/") &&
		(ct == "application/json" || ct == "application/x-protobuf") {
		k := "JSON"
		if ct == "application/x-protobuf" {
			k = "PROTOCOL_BUFFER"
		}
		dr, ok := hsi.ds.ByName(k)
		if !ok {
			parsErr(fmt.Errorf("cannot get decoder for %s", k))
			return
		}
		if pack.MsgBytes, err = ioutil.ReadAll(r.Body); err != nil {
			parsErr(fmt.Errorf("error reading request body: %s", err))
			return
		}
		w.WriteHeader(201)
		dr.InChan() <- pack
		w.Write([]byte{})
		return
	}
	q := r.URL.Query()
	var (
		i int64
		s string
        f *message.Field
	)
	start := time.Now().UnixNano() - 1000000

	for k, vs := range q {
		k = strings.ToLower(k)
		switch k {
		case "uuid":
			pack.Message.Uuid = []byte(vs[0])
		case "timestamp":
			s = vs[0]
			i := strings.Index(s, ".")
			if i > 0 {
				s = s[:i]
			}
			ts, e := strconv.ParseInt(s, 10, 64)
			if e != nil {
				parsErr(fmt.Errorf("error parsing timestamp %s: %s", s, e))
				return
			}
			pack.Message.Timestamp = &ts
		case "type":
			if vs[0] != "" {
				t := vs[0]
				pack.Message.Type = &t
			}
		case "logger":
			if vs[0] != "" {
				t := vs[0]
				pack.Message.Logger = &t
			}
		case "severity":
			if i, err = strconv.ParseInt(vs[0], 10, 32); err != nil {
				parsErr(fmt.Errorf("error parsing severity %s: %s", vs[0], err))
				return
			}
			j := int32(i)
			pack.Message.Severity = &j
		case "envversion":
			if vs[0] != "" {
				t := vs[0]
				pack.Message.EnvVersion = &t

			}
		case "hostname":
			if vs[0] != "" {
				t := vs[0]
				pack.Message.Hostname = &t
			}
		case "pid":
			if vs[0] != "" {
				if i, err = strconv.ParseInt(vs[0], 10, 32); err != nil {
					parsErr(fmt.Errorf("error parsing pid %s: %s", vs[0], err))
					return
				}
				j := int32(i)
				pack.Message.Pid = &j

			}
		case "payload":
			s = strings.Join(vs, " ")
			if s != "" {
				t := s
				pack.Message.Payload = &t
			}
		default:
            if f, err = message.NewField(k, vs[0], vs[0]); err != nil {
                parsErr(fmt.Errorf("cannot create field for %q=%q: %s", k, vs[0], err))
            }
            if f != nil && f.ValueType != nil {
                pack.Message.AddField(f)
            }
		}
	}
	if pack.Message.Payload == nil || *pack.Message.Payload == "" {
		var buf []byte
		if buf, err = ioutil.ReadAll(r.Body); err != nil {
			parsErr(fmt.Errorf("error reading body: %s", err))
			return
		}
		t := string(buf)
		pack.Message.Payload = &t
	}
	if pack.Message.Hostname == nil {
		pack.Message.SetHostname(r.Host)
	}
	if pack.Message.Uuid == nil || len(pack.Message.Uuid) == 0 {
		pack.Message.Uuid = []byte(uuid.NewRandom())
	}
	if pack.Message.Type == nil {
		pack.Message.SetType("heka.httpdata-simple")
	}
	if pack.Message.Timestamp == nil || *pack.Message.Timestamp < start {
		//fmt.Printf("setting timestamp to %s", time.Now().UnixNano())
		pack.Message.SetTimestamp(time.Now().UnixNano())
	}
	w.WriteHeader(201)

	pack.Decoded = true
	w.Write([]byte{})
	hsi.input <- pack
}

// HTTPSimpleInputConfig holds the user-configurable values:
//the HTTP address we should listen on
type HTTPSimpleInputConfig struct {
	Address string `toml:"address"`
}

// ConfigStruct returns a new config struct to be used to read the config file
func (hsi *HTTPSimpleInput) ConfigStruct() interface{} {
	return new(HTTPSimpleInputConfig)
}

// Init initializes the Input instance by extracting the address value
//from the config and store it on the plugin instance.
func (hsi *HTTPSimpleInput) Init(config interface{}) error {
	conf := config.(*HTTPSimpleInputConfig)
	hsi.Address = conf.Address
	return nil
}

func init() {
	i := func() interface{} {
		return new(HTTPSimpleInput)
	}
	pipeline.RegisterPlugin("HttpSimpleInput", i)
	pipeline.RegisterPlugin("HTTPSimpleInput", i)
}
