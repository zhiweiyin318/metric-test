package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/config"
	"github.com/spf13/cobra"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

type Options struct {
	Server string
	CA     string
	Token  string
}

func main() {
	opt := &Options{
		Server: "https://prometheus-k8s.openshift-monitoring.svc:9091",
		CA:     "/etc/metrics-test-ca/service-ca.crt",
		Token:  "/var/run/secrets/kubernetes.io/serviceaccount/token",
	}

	cmd := &cobra.Command{
		Short:         "test Prometheus retrieve",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return opt.Run()
		},
	}

	cmd.Flags().StringVar(&opt.Server, "server", opt.Server, "prometheus server url.")
	cmd.Flags().StringVar(&opt.CA, "ca", opt.CA, "ca to use to verify the server.")
	cmd.Flags().StringVar(&opt.Token, "token", opt.Token, "a bearer token to use to authenticate the server.")

	if err := cmd.Execute(); err != nil {
		fmt.Errorf("err %v", err)
		os.Exit(1)
	}
}

func (o *Options) Run() error {

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	pool, err := x509.SystemCertPool()
	if err != nil {
		fmt.Printf("failed to pool  %v", err)
		return err
	}
	data, err := ioutil.ReadFile(o.CA)
	if err != nil {
		fmt.Printf("failed to read ca %v", err)
		return err
	}
	if !pool.AppendCertsFromPEM(data) {
		return fmt.Errorf("no cert found in ca file")
	}

	transport.TLSClientConfig = &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}

	data, err = ioutil.ReadFile(o.Token)
	if err != nil {
		fmt.Printf("failed to read token %v", err)
		return err
	}
	o.Token = strings.TrimSpace(string(data))

	client, err := api.NewClient(api.Config{
		Address:      o.Server,
		RoundTripper: config.NewAuthorizationCredentialsRoundTripper("Bearer", config.Secret(o.Token), transport),
	})

	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		os.Exit(1)
	}

	v1api := v1.NewAPI(client)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// non-bare-metal OCP result is "" when retrieve machine_cpu_cores and machine_cpu_sockets
	result, warnings, err := v1api.Query(ctx, "machine_cpu_cores", time.Now())
	if err != nil {
		fmt.Printf("Error querying Prometheus: %v\n", err)
		os.Exit(1)
	}
	if len(warnings) > 0 {
		fmt.Printf("Warnings: %v\n", warnings)
	}

	fmt.Printf("Result:\n%v\n", result)
	fmt.Printf("Result.String: %v \n", result.String())


	for {
		time.Sleep(1 * time.Second)
	}
	return nil
}
