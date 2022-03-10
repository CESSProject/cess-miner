package rpc

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"unicode"

	"github.com/golang/protobuf/proto"
)

const methodSuffix = "Action"

type serviceRouter struct {
	mu       sync.Mutex
	services map[string]service
}

func newServiceRouter() *serviceRouter {
	return &serviceRouter{
		services: make(map[string]service),
	}
}

// service represents a registered object.
type service struct {
	name         string
	handlers     map[string]handleWrapper
}

func (r *serviceRouter) registerName(name string, svc interface{}) error {
	recvVal := reflect.ValueOf(svc)
	if name == "" {
		return fmt.Errorf("no service name for type %s", recvVal.Type().String())
	}
	handlers := suitableHandlers(recvVal)
	if len(handlers) == 0 {
		return fmt.Errorf("service %T doesn't have any suitable public method", recvVal)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.services == nil {
		r.services = make(map[string]service)
	}

	s, ok := r.services[name]
	if !ok {
		s = service{
			name:          name,
			handlers:     make(map[string]handleWrapper),
		}
		r.services[name] = s
	}
	for name, h := range handlers {
		s.handlers[name] = h
	}
	return nil
}

func (r *serviceRouter) lookup(srvName, method string) handleWrapper {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.services[srvName].handlers[method]
}

func suitableHandlers(receiver reflect.Value) map[string]handleWrapper {
	typ := receiver.Type()
	handlers := make(map[string]handleWrapper)
	for m := 0; m < typ.NumMethod(); m++ {
		method := typ.Method(m)
		if method.PkgPath != "" {
			continue // private method
		}
		if strings.HasSuffix(method.Name, methodSuffix) {
			fn := method.Func
			name := formatName(method.Name)
			handlers[name] = func(id uint64, body []byte) *RespMsg {
				results := fn.Call([]reflect.Value{receiver, reflect.ValueOf(body)})
				var (
					rel proto.Message
					err error
				)
				if !results[0].IsNil() {
					rel = results[0].Interface().(proto.Message)
				}

				if !results[1].IsNil() {
					err = results[1].Interface().(error)
					resp := errorMessage(err)
					resp.Id = uint64(id)
					return resp
				}

				resp := &RespMsg{
					Id: uint64(id),
				}
				bs, _ := proto.Marshal(rel)
				resp.Body = bs
				return resp
			}
		}
	}

	return handlers
}

func formatName(name string) string {
	name = strings.TrimSuffix(name, methodSuffix)
	ret := []rune(name)
	if len(ret) > 0 {
		ret[0] = unicode.ToLower(ret[0])
	}
	return string(ret)
}
