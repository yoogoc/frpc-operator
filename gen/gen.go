package gen

import (
	"bytes"
	_ "embed"
	"text/template"

	frpcv1 "github.com/YoogoC/frpc-operator/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// type ClientCommon config.ClientCommonConf
// type TCPProxy config.TCPProxyConf

type FrpcConfig struct {
	Common     ClientCommon
	TCPProxies []TCPProxy
}

type ClientCommon struct {
	ServerAddress string
	ServerPort    int
	Token         string
	AdminAddress  string
	AdminPort     int
	AdminUsername string
	AdminPassword string
}

type TCPProxy struct {
	Name       string
	LocalAddr  string
	LocalPort  string
	RemotePort string
}

//go:embed frpc.ini.tmpl
var frpcIniTmpl string

func NewConfig(k8sClient client.Client, clientObj *frpcv1.Client, proxies []frpcv1.Proxy) (*FrpcConfig, error) {
	var tcpProxies []TCPProxy
	for _, proxy := range proxies {
		tcpProxies = append(tcpProxies, TCPProxy{
			Name:       proxy.Name,
			LocalAddr:  proxy.Spec.LocalAddr,
			LocalPort:  proxy.Spec.LocalPort,
			RemotePort: proxy.Spec.TCPProxy.RemotePort,
		})
	}
	frpcConfig := &FrpcConfig{
		Common: ClientCommon{
			ServerAddress: clientObj.Spec.Common.ServerAddr,
			ServerPort:    clientObj.Spec.Common.ServerPort,
			Token:         clientObj.Spec.Common.Token.Value, // TODO
			AdminAddress:  "0.0.0.0",                         // TODO
			AdminPort:     7400,                              // TODO
			AdminUsername: "frpc-admin",                      // TODO
			AdminPassword: "frpc-password",                   // TODO
		},
		TCPProxies: tcpProxies,
	}
	return frpcConfig, nil
}

func (config *FrpcConfig) Gen() (string, error) {
	tmpl, err := template.New("frpc").Parse(frpcIniTmpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return "", err
	}
	return string(buf.Bytes()), nil
}

func Gen(k8sClient client.Client, clientObj *frpcv1.Client, proxies []frpcv1.Proxy) (string, error) {
	config, err := NewConfig(k8sClient, clientObj, proxies)
	if err != nil {
		return "", err
	}
	return config.Gen()
}
