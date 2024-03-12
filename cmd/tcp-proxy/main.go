// Copyright 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*******************************************************************************
* MIT License
*
* Copyright (c) 2020 dev@jpillora.com
*
* Permission is hereby granted, free of charge, to any person obtaining a copy
* of this software and associated documentation files (the "Software"), to deal
* in the Software without restriction, including without limitation the rights
* to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
* copies of the Software, and to permit persons to whom the Software is
* furnished to do so, subject to the following conditions:
*
* The above copyright notice and this permission notice shall be included in all
* copies or substantial portions of the Software.
*
* THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
* IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
* FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
* AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
* LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
* OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
* SOFTWARE.
*******************************************************************************/

package main

import (
	goflag "flag"
	"fmt"
	"net"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	flag "github.com/spf13/pflag"
	"k8s.io/klog/v2"

	"github.com/cloudoperators/greenhouse/pkg/tcp-proxy/metrics"
	"github.com/cloudoperators/greenhouse/pkg/tcp-proxy/proxy"
	"github.com/cloudoperators/greenhouse/pkg/version"
)

var (
	localAddr     = flag.String("l", ":443", "local address")
	remoteAddr    = flag.String("r", fmt.Sprintf("%s:%s", "kubernetes.default.svc.cluster.local", os.Getenv("KUBERNETES_SERVICE_PORT_HTTPS")), "remote address")
	metricAddress = flag.String("metrics", "127.0.0.1:3002", "IP address and port number to expose prometheus metrics on")
	hex           = flag.Bool("h", false, "output hex")
	unwrapTLS     = flag.Bool("unwrap-tls", false, "remote connection with TLS exposed unencrypted locally")
)

func init() {
	prometheus.MustRegister(metrics.InboundConnCounter)
	prometheus.MustRegister(metrics.OutboundConnCounter)
	prometheus.MustRegister(metrics.InboundBytesCounter)
	prometheus.MustRegister(metrics.OutboundBytesCounter)
	prometheus.MustRegister(metrics.ActiveInboundConnGauge)
	prometheus.MustRegister(metrics.ActiveOutboundConnGauge)
}

func main() {
	goFlagSet := goflag.CommandLine
	flag.CommandLine.AddGoFlagSet(goFlagSet)
	flag.Parse()
	version.ShowVersionAndExit("tcp-proxy")

	klog.Infof("tcp-proxy proxing from %v to %v ", *localAddr, *remoteAddr)

	laddr, err := net.ResolveTCPAddr("tcp", *localAddr)
	if err != nil {
		klog.Errorf("Failed to resolve local address: %s", err)
		os.Exit(1)
	}

	_, err = net.LookupIP(*remoteAddr)
	if err != nil {
		klog.Errorf("Failed to resolve remote address: %s", err)
		klog.Infof("Fallback to KUBERNETES_SERVICE_HOST env variable")
		*remoteAddr = fmt.Sprintf("%s:%s", os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT_HTTPS"))
	}

	raddr, err := net.ResolveTCPAddr("tcp", *remoteAddr)
	if err != nil {
		klog.Errorf("Failed to resolve remote address: %s", err)
		os.Exit(1)
	}
	listener, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		klog.Errorf("Failed to open local port to listen: %s", err)
		os.Exit(1)
	}

	prom := metrics.Helper{}
	go prom.StartMetricsServer(*metricAddress)

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			klog.Errorf("Failed to accept connection '%s'", err)
			continue
		}

		var p *proxy.Proxy
		if *unwrapTLS {
			klog.Info("Unwrapping TLS")
			p = proxy.NewTLSUnwrapped(conn, laddr, raddr, *remoteAddr)
		} else {
			p = proxy.New(conn, laddr, raddr)
		}

		p.OutputHex = *hex

		metrics.IncrementActiveInboundGauge()
		go p.Start()
	}
}
