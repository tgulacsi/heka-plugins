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

//  curl -v 'http://localhost:5566/?payload=abbraka&severity=3' -XPOST
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
	var (
		i   int64
		err error
		s   string
	)

	for k, vs := range q {
		k = strings.ToLower(k)
		switch k {
		case "uuid":
			msg.Uuid = []byte(vs[0])
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
			msg.Timestamp = &ts
		case "type":
			if vs[0] != "" {
				t := vs[0]
				msg.Type = &t
			}
		case "logger":
			if vs[0] != "" {
				t := vs[0]
				msg.Logger = &t
			}
		case "severity":
			if i, err = strconv.ParseInt(vs[0], 10, 32); err != nil {
				parsErr(fmt.Errorf("error parsing severity %s: %s", vs[0], err))
				return
			}
			j := int32(i)
			msg.Severity = &j
		case "envversion":
			if vs[0] != "" {
				t := vs[0]
				msg.EnvVersion = &t

			}
		case "hostname":
			if vs[0] != "" {
				t := vs[0]
				msg.Hostname = &t
			}
		case "pid":
			if vs[0] != "" {
				if i, err = strconv.ParseInt(vs[0], 10, 32); err != nil {
					parsErr(fmt.Errorf("error parsing pid %s: %s", vs[0], err))
					return
				}
				j := int32(i)
				msg.Pid = &j

			}
		case "payload":
			s = strings.Join(vs, " ")
			if s != "" {
				t := s
				msg.Payload = &t
			}
		default:
			if msg.Fields == nil {
				msg.Fields = make([]*message.Field, 0, 1)
			}
			t := k
			msg.Fields = append(msg.Fields, &message.Field{Name: &t, ValueString: []string{vs[0]}})
		}
	}
	if msg.Payload == nil || *msg.Payload == "" {
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
