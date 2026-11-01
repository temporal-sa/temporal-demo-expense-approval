package main

import (
	"crypto/tls"
	"testing"

	"go.temporal.io/sdk/client"
)

func TestTemporalUIURL(t *testing.T) {
	cloud := client.Options{Namespace: "ns.acct", ConnectionOptions: client.ConnectionOptions{TLS: &tls.Config{}}}
	local := client.Options{Namespace: "default"}
	for _, c := range []struct {
		name, uiURL, uiNamespace string
		opts                     client.Options
		want                     string
	}{
		{"platform base and namespace", "https://cloud.temporal.io", "tmprl-dem-cld-fk-expense-approval.sdvdw", cloud, "https://cloud.temporal.io/namespaces/tmprl-dem-cld-fk-expense-approval.sdvdw"},
		{"workspace base with slash", "https://expense-approvals-86e3.test.tmprl-demo.cloud/_temporal/", "dev", local, "https://expense-approvals-86e3.test.tmprl-demo.cloud/_temporal/namespaces/dev"},
		{"full namespace URL, no namespace env", "http://localhost:8233/namespaces/default/", "", local, "http://localhost:8233/namespaces/default"},
		{"namespace without base is ignored", "", "other", cloud, "https://cloud.temporal.io/namespaces/ns.acct"},
		{"nothing set, cloud", "", "", cloud, "https://cloud.temporal.io/namespaces/ns.acct"},
		{"nothing set, local", "", "", local, "http://localhost:8233/namespaces/default"},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("TEMPORAL_UI_URL", c.uiURL)
			t.Setenv("TEMPORAL_UI_NAMESPACE", c.uiNamespace)
			if got := temporalUIURL(c.opts); got != c.want {
				t.Errorf("temporalUIURL() = %q, want %q", got, c.want)
			}
		})
	}
}
