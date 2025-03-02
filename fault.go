package soldr

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"
)

type FaultType uint32

const (
	NA FaultType = 1 << iota
	FieldLevel
	RequestLevel
	CustomLevel
	ConfigLevel
)

func ConfigFault(err error) *Fault {
	return &Fault{
		Type: ConfigLevel,
		Err:  err,
	}
}

func FieldFault(field string, err error) *Fault {
	return &Fault{
		Type:  FieldLevel,
		Err:   err,
		Field: field,
	}
}

func CustomFault(err error) *Fault {
	return &Fault{
		Type: CustomLevel,
		Err:  err,
	}
}

func RequestFault(err error) *Fault {
	return &Fault{
		Type: RequestLevel,
		Err:  err,
	}
}

type Fault struct {
	Type  FaultType
	Field string
	Err   error
}

type ByType []*Fault

func (u ByType) Len() int {
	return len(u)
}

func (u ByType) Less(i, j int) bool {
	return u[i].Type < u[j].Type
}

func (u ByType) Swap(i, j int) {
	u[i], u[j] = u[j], u[i]
}

type FaultHandler interface {
	ToError(faults []*Fault) error
}

var _ FaultHandler = (*defaultFaultHandler)(nil)

type defaultFaultHandler struct {
	format Format
}

type defaultFaultHandlerOption func(*defaultFaultHandler)

func withJSONFormt() defaultFaultHandlerOption {
	return func(d *defaultFaultHandler) {
		d.format = JSON
	}
}

func newDefaultFaultHandler(options ...defaultFaultHandlerOption) *defaultFaultHandler {
	h := &defaultFaultHandler{}
	if len(options) > 0 {
		for _, o := range options {
			o(h)
		}
	}
	return h
}

// Process implements ErrResultHandler.
func (*defaultFaultHandler) ToError(faults []*Fault) error {
	if len(faults) == 0 {
		return nil
	}
	var (
		buffer         bytes.Buffer
		sectionWritten string
	)
	// sort the faults by type so that we can process the sections easier
	sort.Sort(ByType(faults))
	for _, v := range faults {
		if sectionWritten == "" || sectionWritten != v.Type.String() {
			// if we are starting a new sction, end the current one
			if sectionWritten != "" {
				buffer.WriteString("]\n")
			}
			buffer.WriteString(fmt.Sprintf("%s.issues: [\n", strings.ToLower(v.Type.String())))
			sectionWritten = v.Type.String()
		}
		// if on the type because errors at the request level won't be scoped to a field
		var lineitem string
		if v.Type == RequestLevel {
			lineitem = fmt.Sprintf("%s\\n\n", v.Err.Error())
		} else {
			lineitem = fmt.Sprintf("%s: %s\\n\n", v.Field, v.Err.Error())
		}
		buffer.WriteString(lineitem)
	}
	buffer.WriteString("]\n")
	return errors.New(buffer.String())
}

// parseFieldNameFromPath parses the target field's name from a "." delimited path.
// returns the parent path and the field's name respectively.
func parseFieldNameFromPath(path string) (string, string) {
	sp := strings.Split(path, ".")
	var parsedName, parentPath string
	if len(sp) > 1 {
		parsedName = sp[len(sp)-1]
		parentPath = strings.Join(sp[:len(sp)-1], ".")
	} else {
		parsedName = sp[0]
	}
	return parentPath, parsedName
}
