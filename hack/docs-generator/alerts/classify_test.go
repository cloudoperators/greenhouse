// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package main

import "testing"

func TestClassify(t *testing.T) {
	cases := []struct {
		name         string
		supportGroup string
		hasLabel     bool
		want         Bucket
	}{
		{"literal greenhouse-admin", "greenhouse-admin", true, AdminBucket},
		{"templated namespace-admin (control-plane namespace) is platform admin", "{{ $labels.namespace }}-admin", true, AdminBucket},
		{"templated organization-admin is tenant org admin", "{{ $labels.organization }}-admin", true, OrgAdminBucket},
		{"templated organization-admin without leading $ is tenant org admin", "{{ .Labels.organization }}-admin", true, OrgAdminBucket},
		{"templated owned_by", "{{ $labels.owned_by }}", true, SupportGroupsBucket},
		{"some other literal", "some-team", true, SupportGroupsBucket},
		{"non-template literal -admin defaults to platform admin", "platform-admin", true, AdminBucket},
		{"missing label defaults to admin", "", false, AdminBucket},
		{"empty label value defaults to admin", "", true, AdminBucket},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			labels := map[string]string{}
			if tc.hasLabel {
				labels["support_group"] = tc.supportGroup
			}
			got := classify(Rule{Labels: labels})
			if got != tc.want {
				t.Errorf("classify(support_group=%q hasLabel=%v) = %v, want %v",
					tc.supportGroup, tc.hasLabel, got, tc.want)
			}
		})
	}
}
