package g

import (
	"net/http"
	"time"

	"github.com/bincooo/ja3"
	"github.com/xllm-go/g/internal"
	"github.com/xllm-go/g/internal/v1"

	xtls "github.com/refraction-networking/utls"
)

var (
	cacheTransport = make(map[string]http.RoundTripper)
)

type interfaces struct {
	//
}

func Sdk() interface {
	Env() *v1.Environ
	Transport(proxies string) http.RoundTripper
	Support(...string) *builder
	OnInitialized(func())
	OnExited(func())
	OnPanic(func(interface{}))
} {
	return &interfaces{}
}

func (interfaces) Support(mod ...string) *builder {
	return (&builder{}).model(mod...)
}

func (interfaces) Transport(proxies string) http.RoundTripper {
	key := proxies
	if proxies == "" {
		key = "default"
	}

	if cacheTransport[key] != nil {
		return cacheTransport[key]
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.IdleConnTimeout = 120 * time.Second
	roundTripper := ja3.NewTransport(
		ja3.WithProxy(proxies),
		ja3.WithClientHelloID(xtls.HelloChrome_133),
		ja3.WithOriginalTransport(transport),
	)

	cacheTransport[key] = roundTripper
	return roundTripper
}

func (interfaces) Env() *v1.Environ {
	return v1.Env
}

func (interfaces) OnInitialized(f func()) {
	internal.AddInitialized(f)
}

func (interfaces) OnExited(f func()) {
	internal.AddExited(f)
}

func (interfaces) OnPanic(f func(interface{})) {
	internal.AddPanic(f)
}

func Execute() {
	internal.Execute()
}
