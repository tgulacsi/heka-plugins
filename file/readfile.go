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

package file

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"text/template"

	"github.com/mozilla-services/heka/message"
	"github.com/mozilla-services/heka/pipeline"
)

// FileReader plugin for expanding messages from external files based on template
// in the template you can use the Message's fields
type FileReader struct {
	tmpl    *template.Template
	newType string
}

// Extract hosts value from config and store it on the plugin instance.
func (fr FileReader) Init(config interface{}) (err error) {
	var (
		value interface{}
		str   string
		ok    bool
	)
	conf := config.(pipeline.PluginConfig)
	if value, ok = conf["pattern"]; !ok {
		return errors.New("FileReader: No 'pattern' setting specified.")
	}
	if str, ok = value.(string); !ok {
		return errors.New("FileReader: 'pattern' setting not a string value.")
	}
	fr.tmpl, err = template.New("filereader").Parse(str)

	if value, ok = conf["newType"]; !ok {
		return errors.New("FileReader: No 'newType' setting specified.")
	}
	if fr.newType, ok = value.(string); !ok {
		return errors.New("FileReader: 'newType' setting not a string value.")
	}

	return err
}

// Run runs the FileReader filter, which inspects each message, and appends
// the content of the file named as the executed template to the existing payload.
// The resulting message will be injected back, and have newType type.
func (fr FileReader) Run(r pipeline.FilterRunner, h pipeline.PluginHelper) (err error) {
	var (
		fh           *os.File
		npack, opack *pipeline.PipelinePack
	)
	out := bytes.NewBuffer(make([]byte, 0, 4096))
	for opack = range r.InChan() {
		if opack.Decoded {
			out.Reset()
			if err = fr.tmpl.Execute(out, opack.Message); err != nil {
				opack.Recycle()
				return fmt.Errorf("FileReader: error executing template %v with message %v: %v",
					fr.tmpl, opack.Message, err)
			}
			if fh, err = os.Open(out.String()); err != nil {
				opack.Recycle()
				return fmt.Errorf("FileReader: cannot read %q: %v", out, err)
			}
			out.Reset()
			//if _, err = io.Copy(out, io.LimitedReader{R: fh, N: 65000}); err != nil && err != io.EOF {
			if _, err = io.Copy(out, fh); err != nil && err != io.EOF {
				opack.Recycle()
				fh.Close()
				return fmt.Errorf("FileReader: error reading %q: %v", fh.Name(), err)
			}
			fh.Close()

			npack = h.PipelinePack(opack.MsgLoopCount)
			if npack == nil {
				opack.Recycle()
				return errors.New("FileReader: no output pack - infinite loop?")
			}
			npack.Decoded = true
			npack.Message = message.CopyMessage(opack.Message)
			npack.Message.SetType(fr.newType)
			npack.Message.SetPayload(npack.Message.GetPayload() + "\n" + out.String())
			if !r.Inject(npack) {
				log.Printf("FileReader: cannot inject new pack %v", npack)
			}
		}
		opack.Recycle()
	}
	return nil
}

func init() {
	pipeline.RegisterPlugin("FileReaderFilter", func() interface{} { return new(FileReader) })
}
