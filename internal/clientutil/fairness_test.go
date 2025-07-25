// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package clientutil_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	flowcontrolapi "k8s.io/api/flowcontrol/v1beta2"
	"k8s.io/client-go/rest"

	"github.com/cloudoperators/greenhouse/internal/clientutil"
)

var _ = Describe("isPriorityAndFairnessEnabled", func() {
	var ctx context.Context
	var cancel context.CancelFunc

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		DeferCleanup(cancel)
	})

	DescribeTable("should correctly detect Priority and Fairness support",
		func(tc struct {
			handler       func(req *http.Request) *http.Response
			expectEnabled bool
			expectErr     bool
		}) {
			// Setup configuration based on test case
			var cfg *rest.Config
			if tc.handler != nil {
				ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					resp := tc.handler(r)
					// copy headers
					for k, vals := range resp.Header {
						w.Header().Del(k)
						for _, v := range vals {
							w.Header().Add(k, v)
						}
					}
					w.WriteHeader(resp.StatusCode)
					// copy body
					_, err := io.Copy(w, resp.Body)
					Expect(err).ToNot(HaveOccurred())
				}))
				DeferCleanup(ts.Close)
				cfg = &rest.Config{Host: ts.URL}
			} else {
				// invalid host scenario
				cfg = &rest.Config{Host: "://bad-host"}
			}

			enabled, err := clientutil.ExportedIsPriorityAndFairnessEnabled(ctx, cfg)
			if tc.expectErr {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).ToNot(HaveOccurred())
				Expect(enabled).To(Equal(tc.expectEnabled))
			}
		},

		Entry("header found",
			struct {
				handler       func(req *http.Request) *http.Response
				expectEnabled bool
				expectErr     bool
			}{
				handler: func(req *http.Request) *http.Response {
					headers := http.Header{}
					headers.Add(flowcontrolapi.ResponseHeaderMatchedFlowSchemaUID, "uuid-1234")
					return &http.Response{
						StatusCode: http.StatusOK,
						Header:     headers,
						Body:       io.NopCloser(bytes.NewReader(nil)),
					}
				},
				expectEnabled: true,
				expectErr:     false,
			},
		),

		Entry("header not found",
			struct {
				handler       func(req *http.Request) *http.Response
				expectEnabled bool
				expectErr     bool
			}{
				handler: func(req *http.Request) *http.Response {
					return &http.Response{
						StatusCode: http.StatusOK,
						Header:     http.Header{},
						Body:       io.NopCloser(bytes.NewReader(nil)),
					}
				},
				expectEnabled: false,
				expectErr:     false,
			},
		),

		Entry("invalid host",
			struct {
				handler       func(req *http.Request) *http.Response
				expectEnabled bool
				expectErr     bool
			}{
				handler:       nil,
				expectEnabled: false,
				expectErr:     true,
			},
		),
	)
})
