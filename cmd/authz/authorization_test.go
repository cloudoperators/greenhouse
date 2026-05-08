// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	authv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/test"
)

var (
	testTeam = test.NewTeam(test.Ctx, "demo", "my-org",
		test.WithTeamLabel(greenhouseapis.LabelKeySupportGroup, "true"))
)

var _ = Describe("extractServiceAccountName", func() {
	DescribeTable("extracting service account name from username",
		func(username string, namespace string, expected string) {
			result := extractServiceAccountName(username, namespace)
			Expect(result).To(Equal(expected), "should correctly extract SA name from username")
		},
		Entry("valid SA returns name",
			"system:serviceaccount:my-org:demo-sa", "my-org", "demo-sa"),
		Entry("SA with any name returns the name",
			"system:serviceaccount:my-org:demo", "my-org", "demo"),
		Entry("SA in a different namespace returns empty string",
			"system:serviceaccount:other-ns:demo-sa", "my-org", ""),
		Entry("regular user (not a service account) returns empty string",
			"demo-user", "my-org", ""),
		Entry("SA with hyphenated name returns full name",
			"system:serviceaccount:my-org:my-team-name-sa", "my-org", "my-team-name-sa"),
		Entry("empty username returns empty string",
			"", "my-org", ""),
	)
})

var _ = Describe("handleAuthorize", func() {
	Context("HTTP method validation", func() {
		It("should reject non-POST methods", func() {
			h := makeHandler(nil)
			req := httptest.NewRequest(http.MethodGet, "/authorize", http.NoBody)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			Expect(rec.Code).To(Equal(http.StatusMethodNotAllowed), "GET requests should be rejected with 405 status")
		})
	})

	Context("request validation", func() {
		It("should deny requests with missing resource attributes", func() {
			h := makeHandler(nil)
			review := authv1.SubjectAccessReview{
				Spec: authv1.SubjectAccessReviewSpec{
					User:   "demo-user",
					Groups: []string{"support-group:demo"},
					// ResourceAttributes intentionally nil
				},
			}
			resp := postReview(h, review)
			Expect(resp.Status.Allowed).To(BeFalse(), "requests without resource attributes should be denied")
			Expect(resp.Status.Reason).To(ContainSubstring("missing resource attributes"), "denial reason should mention missing attributes")
		})
	})

	Context("user authorization with support groups", func() {
		It("should allow user with matching support-group", func() {
			plugin := test.NewPlugin(test.Ctx, "plugin-demo", "my-org",
				test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, "demo"))
			h := makeHandler(plugin, testTeam)

			review := authv1.SubjectAccessReview{
				Spec: authv1.SubjectAccessReviewSpec{
					User:               "demo-user",
					Groups:             []string{"support-group:demo"},
					ResourceAttributes: pluginAttrs("plugin-demo"),
				},
			}
			resp := postReview(h, review)
			Expect(resp.Status.Allowed).To(BeTrue(), "user with matching support-group should be allowed")
			Expect(resp.Status.Reason).To(ContainSubstring("demo"), "approval reason should mention the team name")
		})

		It("should deny user with non-matching support-group", func() {
			plugin := test.NewPlugin(test.Ctx, "plugin-demo", "my-org",
				test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, "demo"))
			h := makeHandler(plugin, testTeam)

			review := authv1.SubjectAccessReview{
				Spec: authv1.SubjectAccessReviewSpec{
					User:               "other-user",
					Groups:             []string{"support-group:other-team"},
					ResourceAttributes: pluginAttrs("plugin-demo"),
				},
			}
			resp := postReview(h, review)
			Expect(resp.Status.Allowed).To(BeFalse(), "user without matching support-group should be denied")
		})

		It("should deny user with no support-group claims", func() {
			plugin := test.NewPlugin(test.Ctx, "plugin-demo", "my-org",
				test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, "demo"))
			h := makeHandler(plugin)

			review := authv1.SubjectAccessReview{
				Spec: authv1.SubjectAccessReviewSpec{
					User:               "regular-user",
					Groups:             []string{"some-other-group"},
					ResourceAttributes: pluginAttrs("plugin-demo"),
				},
			}
			resp := postReview(h, review)
			Expect(resp.Status.Allowed).To(BeFalse(), "user with no support-group claims should be denied")
			Expect(resp.Status.Reason).To(ContainSubstring("no support-group claims"), "denial reason should mention missing claims")
		})

		It("should allow user with multiple support-groups when one matches", func() {
			plugin := test.NewPlugin(test.Ctx, "plugin-demo", "my-org",
				test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, "demo"))
			h := makeHandler(plugin, testTeam)

			review := authv1.SubjectAccessReview{
				Spec: authv1.SubjectAccessReviewSpec{
					User:               "multi-group-user",
					Groups:             []string{"support-group:other-team", "support-group:demo", "support-group:third-team"},
					ResourceAttributes: pluginAttrs("plugin-demo"),
				},
			}
			resp := postReview(h, review)
			Expect(resp.Status.Allowed).To(BeTrue(), "user with multiple support-groups should be allowed if one matches")
			Expect(resp.Status.Reason).To(ContainSubstring("demo"), "approval reason should mention the matching team")
		})

		It("should deny user with multiple support-groups when none match", func() {
			plugin := test.NewPlugin(test.Ctx, "plugin-demo", "my-org",
				test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, "demo"))
			h := makeHandler(plugin, testTeam)

			review := authv1.SubjectAccessReview{
				Spec: authv1.SubjectAccessReviewSpec{
					User:               "multi-group-user",
					Groups:             []string{"support-group:other-team", "support-group:third-team", "support-group:fourth-team"},
					ResourceAttributes: pluginAttrs("plugin-demo"),
				},
			}
			resp := postReview(h, review)
			Expect(resp.Status.Allowed).To(BeFalse(), "user with multiple non-matching support-groups should be denied")
		})
	})

	Context("service account authorization", func() {
		It("should allow service account that owns the resource", func() {
			plugin := test.NewPlugin(test.Ctx, "plugin-demo", "my-org",
				test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, "demo"))
			sa := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "demo-sa",
					Namespace: "my-org",
					Labels:    map[string]string{greenhouseapis.LabelKeyOwnedBy: "demo"},
				},
			}
			h := makeHandler(plugin, sa, testTeam)

			review := authv1.SubjectAccessReview{
				Spec: authv1.SubjectAccessReviewSpec{
					User:               "system:serviceaccount:my-org:demo-sa",
					ResourceAttributes: pluginAttrs("plugin-demo"),
				},
			}
			resp := postReview(h, review)
			Expect(resp.Status.Allowed).To(BeTrue(), "SA for the owning team should be allowed")
			Expect(resp.Status.Reason).To(ContainSubstring("demo"), "approval reason should mention the team name")
		})

		It("should deny service account from different team", func() {
			plugin := test.NewPlugin(test.Ctx, "plugin-demo", "my-org",
				test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, "demo"))
			sa := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "other-team-sa",
					Namespace: "my-org",
					Labels:    map[string]string{greenhouseapis.LabelKeyOwnedBy: "other-team"},
				},
			}
			h := makeHandler(plugin, sa, testTeam)

			review := authv1.SubjectAccessReview{
				Spec: authv1.SubjectAccessReviewSpec{
					User:               "system:serviceaccount:my-org:other-team-sa",
					ResourceAttributes: pluginAttrs("plugin-demo"),
				},
			}
			resp := postReview(h, review)
			Expect(resp.Status.Allowed).To(BeFalse(), "SA for a different team should be denied")
			Expect(resp.Status.Reason).To(ContainSubstring("does not match"), "denial reason should mention mismatch")
		})

		It("should deny service account from non-matching namespace", func() {
			plugin := test.NewPlugin(test.Ctx, "plugin-demo", "my-org",
				test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, "demo"))
			h := makeHandler(plugin)

			// The SA is in a different namespace from the resource, so extractTeamFromServiceAccount
			// won't match the prefix and will return "".  The request should then fall through
			// to the support-group path and be denied for having no claims.
			review := authv1.SubjectAccessReview{
				Spec: authv1.SubjectAccessReviewSpec{
					User:               "system:serviceaccount:other-ns:demo-sa",
					ResourceAttributes: pluginAttrs("plugin-demo"),
				},
			}
			resp := postReview(h, review)
			Expect(resp.Status.Allowed).To(BeFalse(), "SA in wrong namespace should be denied")
			Expect(resp.Status.Reason).To(ContainSubstring("no support-group claims"), "denial reason should indicate missing claims")
		})
	})

	Context("resource label validation", func() {
		It("should deny access to resource without owned-by label", func() {
			// Plugin without owned-by label
			plugin := &greenhousev1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "unlabeled-plugin",
					Namespace: "my-org",
				},
			}
			h := makeHandler(plugin)

			review := authv1.SubjectAccessReview{
				Spec: authv1.SubjectAccessReviewSpec{
					User:               "demo-user",
					Groups:             []string{"support-group:demo"},
					ResourceAttributes: pluginAttrs("unlabeled-plugin"),
				},
			}
			resp := postReview(h, review)
			Expect(resp.Status.Allowed).To(BeFalse(), "resource without owned-by label should be denied")
			Expect(resp.Status.Reason).To(ContainSubstring("no owned-by label"), "denial reason should mention missing label")
		})
	})
})

var _ = Describe("authorizeServiceAccount", func() {
	var (
		c      client.Client
		mapper meta.RESTMapper
	)

	BeforeEach(func() {
		mapper = buildRESTMapper()
	})

	Context("when ServiceAccount and resource owner match", func() {
		It("should allow access", func() {
			plugin := test.NewPlugin(test.Ctx, "plugin-demo", "my-org",
				test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, "demo"))
			sa := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "demo-sa",
					Namespace: "my-org",
					Labels:    map[string]string{greenhouseapis.LabelKeyOwnedBy: "demo"},
				},
			}
			c = buildFakeClient(plugin, sa, testTeam)

			attrs := pluginAttrs("plugin-demo")
			supportGroups, reasonCode, err := getSupportGroups(test.Ctx, c, "system:serviceaccount:my-org:demo-sa", nil, "my-org")
			Expect(err).ToNot(HaveOccurred())
			Expect(reasonCode).To(BeEmpty(), "reason should be empty on success")
			Expect(supportGroups).To(Equal([]string{"demo"}))

			allowed, reasonCode, ownedBy, _ := authorizeAccess(test.Ctx, c, mapper, attrs, supportGroups)
			Expect(allowed).To(BeTrue(), "service account with matching team should be allowed")
			Expect(reasonCode).To(BeEmpty(), "reason should be empty on success")
			Expect(ownedBy).To(Equal("demo"), "ownedBy should be returned")
		})
	})

	Context("when ServiceAccount does not exist", func() {
		It("should deny access", func() {
			plugin := test.NewPlugin(test.Ctx, "plugin-demo", "my-org",
				test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, "demo"))
			c = buildFakeClient(plugin) // no SA pre-seeded

			supportGroups, reasonCode, err := getSupportGroups(test.Ctx, c, "system:serviceaccount:my-org:demo-sa", nil, "my-org")
			Expect(err).To(HaveOccurred(), "getSupportGroups should fail when SA doesn't exist")
			Expect(reasonCode).To(Equal(reasonServiceAccountNotFound), "should return service_account_not_found constant")
			Expect(supportGroups).To(BeNil())
		})
	})

	Context("when ServiceAccount missing owned-by label", func() {
		It("should deny access", func() {
			plugin := test.NewPlugin(test.Ctx, "plugin-demo", "my-org",
				test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, "demo"))
			sa := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "demo-sa",
					Namespace: "my-org",
					// No owned-by label
				},
			}
			c = buildFakeClient(plugin, sa)

			_, reasonCode, err := getSupportGroups(test.Ctx, c, "system:serviceaccount:my-org:demo-sa", nil, "my-org")
			Expect(err).To(HaveOccurred(), "getSupportGroups should fail when SA missing owned-by label")
			Expect(reasonCode).To(Equal(reasonNoOwnedByLabel), "should return no_owned_by_label constant")
		})
	})

	Context("when resource owner does not match service account team", func() {
		It("should deny access", func() {
			plugin := test.NewPlugin(test.Ctx, "plugin-demo", "my-org",
				test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, "demo"))
			sa := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "other-team-sa",
					Namespace: "my-org",
					Labels:    map[string]string{greenhouseapis.LabelKeyOwnedBy: "other-team"},
				},
			}
			c = buildFakeClient(plugin, sa, testTeam)

			attrs := pluginAttrs("plugin-demo")
			supportGroups, _, err := getSupportGroups(test.Ctx, c, "system:serviceaccount:my-org:other-team-sa", nil, "my-org")
			Expect(err).ToNot(HaveOccurred())
			Expect(supportGroups).To(Equal([]string{"other-team"}))

			allowed, reasonCode, _, _ := authorizeAccess(test.Ctx, c, mapper, attrs, supportGroups)
			Expect(allowed).To(BeFalse(), "service account with non-matching resource owner should be denied")
			Expect(reasonCode).To(Equal(reasonSupportGroupMismatch), "should return support_group_mismatch constant")
		})
	})

	Context("when resource does not exist", func() {
		It("should deny access", func() {
			sa := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "demo-sa",
					Namespace: "my-org",
					Labels:    map[string]string{greenhouseapis.LabelKeyOwnedBy: "demo"},
				},
			}
			c = buildFakeClient(sa) // SA exists but no plugin pre-seeded

			attrs := pluginAttrs("nonexistent-plugin")
			supportGroups, _, err := getSupportGroups(test.Ctx, c, "system:serviceaccount:my-org:demo-sa", nil, "my-org")
			Expect(err).ToNot(HaveOccurred())

			allowed, reasonCode, _, _ := authorizeAccess(test.Ctx, c, mapper, attrs, supportGroups)
			Expect(allowed).To(BeFalse(), "access to non-existent resource should be denied")
			Expect(reasonCode).To(Equal(reasonObjectNotFound), "should return object_not_found constant")
		})
	})
})

// buildRESTMapper creates a simple REST mapper that maps Plugin and Team GVR -> GVK.
func buildRESTMapper() meta.RESTMapper {
	mapper := meta.NewDefaultRESTMapper(nil)
	mapper.Add(schema.GroupVersionKind{
		Group:   greenhousev1alpha1.GroupVersion.Group,
		Version: greenhousev1alpha1.GroupVersion.Version,
		Kind:    "Plugin",
	}, meta.RESTScopeNamespace)
	mapper.Add(schema.GroupVersionKind{
		Group:   greenhousev1alpha1.GroupVersion.Group,
		Version: greenhousev1alpha1.GroupVersion.Version,
		Kind:    "Team",
	}, meta.RESTScopeNamespace)
	return mapper
}

// buildFakeClient creates a fake client with the given objects pre-seeded.
func buildFakeClient(objects ...client.Object) client.Client {
	scheme := runtime.NewScheme()
	Expect(corev1.AddToScheme(scheme)).To(Succeed())
	Expect(greenhousev1alpha1.SchemeBuilder.AddToScheme(scheme)).To(Succeed())

	builder := fake.NewClientBuilder().WithScheme(scheme)
	// Filter out nil objects
	var validObjects []client.Object
	for _, obj := range objects {
		if obj != nil {
			validObjects = append(validObjects, obj)
		}
	}
	if len(validObjects) > 0 {
		builder = builder.WithObjects(validObjects...)
	}
	return builder.Build()
}

// postReview posts a SubjectAccessReview to the handler and returns the decoded response.
func postReview(h http.Handler, review authv1.SubjectAccessReview) authv1.SubjectAccessReview {
	body, err := json.Marshal(review)
	Expect(err).ToNot(HaveOccurred(), "marshaling SubjectAccessReview should succeed")

	req := httptest.NewRequest(http.MethodPost, "/authorize", bytes.NewReader(body))
	req = req.WithContext(context.Background())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	Expect(rec.Code).To(Equal(http.StatusOK), "handler should return 200 OK for valid POST requests")

	var resp authv1.SubjectAccessReview
	Expect(json.NewDecoder(rec.Body).Decode(&resp)).To(Succeed(), "response should be valid JSON")
	return resp
}

func makeHandler(objects ...client.Object) http.Handler {
	c := buildFakeClient(objects...)
	mapper := buildRESTMapper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleAuthorize(w, r, c, mapper)
	})
}

func pluginAttrs(name string) *authv1.ResourceAttributes {
	return &authv1.ResourceAttributes{
		Namespace: "my-org",
		Verb:      "get",
		Group:     greenhousev1alpha1.GroupVersion.Group,
		Version:   greenhousev1alpha1.GroupVersion.Version,
		Resource:  "plugins",
		Name:      name,
	}
}
