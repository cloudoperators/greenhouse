// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package changemanagement

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client", func() {
	It("sends the payload as JSON with basic auth", func(ctx SpecContext) {
		var username, password, contentType string
		var body []byte
		endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username, password, _ = r.BasicAuth()
			contentType = r.Header.Get("Content-Type")
			var err error
			if body, err = io.ReadAll(r.Body); err != nil {
				w.WriteHeader(http.StatusBadRequest)
			}
		}))
		defer endpoint.Close()

		Expect(New(endpoint.URL, Auth{Username: "greenhouse", Password: "secret"}).Send(ctx, []byte(`{"digest": "sha256:1"}`))).To(Succeed())
		Expect(username).To(Equal("greenhouse"))
		Expect(password).To(Equal("secret"))
		Expect(contentType).To(Equal("application/json"))
		Expect(body).To(MatchJSON(`{"digest": "sha256:1"}`))
	})

	DescribeTable("trades the credentials for an OAuth2 token at the token URL",
		func(ctx SpecContext, tokenHeader, header, value string) {
			var clientID, clientSecret, grantType, received string
			mux := http.NewServeMux()
			mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
				defer GinkgoRecover()
				clientID, clientSecret, _ = r.BasicAuth()
				grantType = r.FormValue("grant_type")
				w.Header().Set("Content-Type", "application/json")
				_, err := io.WriteString(w, `{"access_token": "token", "token_type": "Bearer"}`)
				Expect(err).ToNot(HaveOccurred())
			})
			mux.HandleFunc("/events", func(_ http.ResponseWriter, r *http.Request) {
				received = r.Header.Get(header)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			auth := Auth{Username: "greenhouse", Password: "secret", TokenURL: server.URL + "/token", TokenHeader: tokenHeader}
			Expect(New(server.URL+"/events", auth).Send(ctx, []byte(`{}`))).To(Succeed())
			Expect(clientID).To(Equal("greenhouse"))
			Expect(clientSecret).To(Equal("secret"))
			Expect(grantType).To(Equal("client_credentials"))
			Expect(received).To(Equal(value))
		},
		Entry("in the configured header", "X-Auth-Token", "X-Auth-Token", "token"),
		Entry("as a bearer token by default", "", "Authorization", "Bearer token"),
	)

	It("fails when the endpoint does not accept the change", func(ctx SpecContext) {
		endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer endpoint.Close()

		Expect(New(endpoint.URL, Auth{}).Send(ctx, []byte(`{}`))).To(MatchError(ContainSubstring("503")))
	})

	It("returns transport errors as url errors", func(ctx SpecContext) {
		endpoint := httptest.NewServer(http.NotFoundHandler())
		endpoint.Close()

		err := New(endpoint.URL, Auth{}).Send(ctx, []byte(`{}`))
		var urlErr *url.Error
		Expect(errors.As(err, &urlErr)).To(BeTrue(), "the plugin controller keeps the condition message stable for url errors")
	})
})

var _ = Describe("Render", func() {
	data := map[string]any{
		"Organization": "demo",
		"Release":      map[string]any{"Digest": "sha256:1", "Status": "failed"},
	}

	It("renders the template with sprig functions", func() {
		payload, err := Render(`{"organization": {{ .Organization | upper | toJson }}, "digest": {{ .Release.Digest | toJson }}}`, data)
		Expect(err).ToNot(HaveOccurred())
		Expect(payload).To(MatchJSON(`{"organization": "DEMO", "digest": "sha256:1"}`))
	})

	It("renders nothing when the template skips the change", func() {
		payload, err := Render(`{{ if eq .Release.Status "deployed" }}{"digest": {{ .Release.Digest | toJson }}}{{ end }}`+"\n", data)
		Expect(err).ToNot(HaveOccurred())
		Expect(payload).To(BeNil())
	})

	It("does not expose the controller environment", func() {
		_, err := Render(`{"home": {{ env "HOME" | toJson }}}`, data)
		Expect(err).To(MatchError(ContainSubstring(`function "env" not defined`)))
	})

	It("rejects a payload that is not JSON", func() {
		_, err := Render(`{"digest": {{ .Release.Digest }}}`, data)
		Expect(err).To(MatchError(ContainSubstring("valid JSON")))
	})
})
