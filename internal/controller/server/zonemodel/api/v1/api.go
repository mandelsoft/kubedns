package v1

import (
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
)

type Error struct {
	Error string `json:"error"`
}

type Answer struct {
	Zone    Zone     `json:"zone"`
	Names   []string `json:"names"`
	Records Records  `json:"records"`
}

type Zone struct {
	NameServers []string `json:"nameservers,omitempty"`
	Names       []string `json:"names"`
	EMail       string   `json:"email"`
	MinimumTTL  int      `json:"minimumTTL"`
	Expire      int      `json:"expire"`
	Refresh     int      `json:"refresh"`
}

type Records struct {
	// +optional
	A []string `json:"A,omitempty"`
	// +optional
	AAAA []string `json:"AAAA,omitempty"`
	// +optional
	TXT []string `json:"TXT,omitempty"`
	// +optional
	SRV []corednsv1alpha1.ServiceSpec `json:"SRV,omitempty"`
	// +optional
	CNAME string `json:"CNAME,omitempty"`
	// +optional
	NS []string `json:"NS,omitempty"`
}
