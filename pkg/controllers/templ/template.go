package templ

import (
	"bytes"
	"k8s.io/klog/v2"
	text_template "text/template"
)

type Template struct {
	tmpl *text_template.Template
}

func NewTemplate(templateFileDir string) (*Template, error) {

	tmpl, err := text_template.ParseFiles(templateFileDir)
	if err != nil {
		return nil, err
	}
	return &Template{
		tmpl: tmpl,
	}, nil
}

func (t *Template) Render(data interface{}) ([]byte, error) {

	outPutBuf := bytes.NewBuffer(make([]byte, 0))
	err := t.tmpl.Execute(outPutBuf, data)
	if err != nil {
		klog.Errorf("write log template error: %v", err)
		return nil, err
	}
	return outPutBuf.Bytes(), nil
}
