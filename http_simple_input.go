package plugins

import (
	"code.google.com/p/go-uuid/uuid"
	"fmt"
	"github.com/mozilla-services/heka/message"
	"github.com/mozilla-services/heka/pipeline"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
)

type HttpSimpleInput struct {
	Address string

	listener net.Listener
	packs    chan *pipeline.PipelinePack
	input    chan *pipeline.PipelinePack
	stop     chan bool
	errch    chan error
}

func (hsi *HttpSimpleInput) Stop() {
	if hsi.stop != nil {
		hsi.stop <- true
	}
}

func (hsi *HttpSimpleInput) listen() {
	var err error
	if hsi.listener, err = net.Listen("tcp", hsi.Address); err != nil {
		hsi.errch <- err
	}
	s := &http.Server{Addr: hsi.Address, Handler: http.HandlerFunc(hsi.handler)}
	if err = s.Serve(hsi.listener); err != nil {
		hsi.errch <- err
	}
}

func (hsi *HttpSimpleInput) Run(ir pipeline.InputRunner, h pipeline.PluginHelper) (err error) {
	hsi.stop = make(chan bool)
	hsi.input = make(chan *pipeline.PipelinePack)
	hsi.errch = make(chan error, 1)
	hsi.packs = ir.InChan()

	go hsi.listen()
	var pack *pipeline.PipelinePack
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
				break
			}
		}
	}
	select {
	case err = <-hsi.errch:
		return
	default:
		close(hsi.errch)
	}
	return nil
}

func (hsi *HttpSimpleInput) handler(w http.ResponseWriter, r *http.Request) {
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
	q := r.URL.Query()
	msg := new(message.Message)

	msg.Uuid = []byte(q.Get("uuid"))
	var (
		i   int64
		err error
	)
	s := q.Get("timestamp")
	if s != "" {
		i := strings.Index(s, ".")
		if i > 0 {
			s = s[:i]
		}
		if *msg.Timestamp, err = strconv.ParseInt(s, 10, 64); err != nil {
			parsErr(fmt.Errorf("error parsing timestamp %s: %s", s, err))
			return
		}
	}
	s = q.Get("type")
	if s != "" {
		t := s
		msg.Type = &t
	}
	s = q.Get("logger")
	if s != "" {
		t := s
		msg.Logger = &t
	}
	s = q.Get("severity")
	if s != "" {
		if i, err = strconv.ParseInt(s, 10, 32); err != nil {
			parsErr(fmt.Errorf("error parsing severity %s: %s", s, err))
			return
		}
		j := int32(i)
		msg.Severity = &j
	}
	s = q.Get("envversion")
	if s != "" {
		t := s
		msg.EnvVersion = &t
	}
	s = q.Get("hostname")
	if s != "" {
		t := s
		msg.Hostname = &t
	}
	s = q.Get("pid")
	if s != "" {
		if i, err = strconv.ParseInt(s, 10, 32); err != nil {
			parsErr(fmt.Errorf("error parsing pid %s: %s", s, err))
			return
		}
		j := int32(i)
		msg.Pid = &j
	}
	s = q.Get("payload")
	if s != "" {
		t := s
		msg.Payload = &t
	} else {
		var buf []byte
		if buf, err = ioutil.ReadAll(r.Body); err != nil {
			parsErr(fmt.Errorf("error reading body: %s", err))
			return
		}
        t := string(buf)
		msg.Payload = &t
	}
	if msg.Uuid == nil || len(msg.Uuid) == 0 {
		msg.Uuid = []byte(uuid.NewRandom())
	}

	pack := <-hsi.packs
	w.WriteHeader(201)

	pack.Message = msg
	pack.Decoded = true
	w.Write([]byte{})
	hsi.input <- pack
}

type HttpSimpleInputConfig struct {
	Address string `toml:"address"`
}

func (hsi *HttpSimpleInput) ConfigStruct() interface{} {
	return new(HttpSimpleInputConfig)
}

// Extract address value from config and store it on the plugin instance.
func (hsi *HttpSimpleInput) Init(config interface{}) error {
	conf := config.(*HttpSimpleInputConfig)
	hsi.Address = conf.Address
	return nil
}

func init() {
	pipeline.RegisterPlugin("HttpSimpleInput", func() interface{} {
		return new(HttpSimpleInput)
	})
}
