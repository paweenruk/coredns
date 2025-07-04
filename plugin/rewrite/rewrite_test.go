package rewrite

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/metadata"
	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/plugin/test"
	"github.com/coredns/coredns/request"

	"github.com/miekg/dns"
)

func msgPrinter(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	if len(r.Answer) == 0 {
		r.Answer = []dns.RR{
			test.A(fmt.Sprintf("%s  5   IN  A  10.0.0.1", r.Question[0].Name)),
		}
	}
	w.WriteMsg(r)
	return 0, nil
}

func TestNewRule(t *testing.T) {
	tests := []struct {
		args        []string
		shouldError bool
		expType     reflect.Type
	}{
		{[]string{}, true, nil},
		{[]string{"foo"}, true, nil},
		{[]string{"name"}, true, nil},
		{[]string{"name", "a.com"}, true, nil},
		{[]string{"name", "a.com", "b.com", "c.com"}, true, nil},
		{[]string{"name", "a.com", "b.com"}, false, reflect.TypeOf(&exactNameRule{})},
		{[]string{"name", "exact", "a.com", "b.com"}, false, reflect.TypeOf(&exactNameRule{})},
		{[]string{"name", "prefix", "a.com", "b.com"}, false, reflect.TypeOf(&prefixNameRule{})},
		{[]string{"name", "suffix", "a.com", "b.com"}, false, reflect.TypeOf(&suffixNameRule{})},
		{[]string{"name", "substring", "a.com", "b.com"}, false, reflect.TypeOf(&substringNameRule{})},
		{[]string{"name", "regex", "([a])\\.com", "new-{1}.com"}, false, reflect.TypeOf(&regexNameRule{})},
		{[]string{"name", "regex", "([a]\\.com", "new-{1}.com"}, true, nil},
		{[]string{"name", "regex", "(dns)\\.(core)\\.(rocks)", "{2}.{1}.{3}", "answer", "name", "(core)\\.(dns)\\.(rocks)", "{2}.{1}.{3}"}, false, reflect.TypeOf(&regexNameRule{})},
		{[]string{"name", "regex", "(adns)\\.(core)\\.(rocks)", "{2}.{1}.{3}", "answer", "name", "(core)\\.(adns)\\.(rocks)", "{2}.{1}.{3}", "too.long", "way.too.long"}, true, nil},
		{[]string{"name", "regex", "(bdns)\\.(core)\\.(rocks)", "{2}.{1}.{3}", "NoAnswer", "name", "(core)\\.(bdns)\\.(rocks)", "{2}.{1}.{3}"}, true, nil},
		{[]string{"name", "regex", "(cdns)\\.(core)\\.(rocks)", "{2}.{1}.{3}", "answer", "ttl", "(core)\\.(cdns)\\.(rocks)", "{2}.{1}.{3}"}, true, nil},
		{[]string{"name", "regex", "(ddns)\\.(core)\\.(rocks)", "{2}.{1}.{3}", "answer", "name", "\xecore\\.(ddns)\\.(rocks)", "{2}.{1}.{3}"}, true, nil},
		{[]string{"name", "regex", "\xedns\\.(core)\\.(rocks)", "{2}.{1}.{3}", "answer", "name", "(core)\\.(edns)\\.(rocks)", "{2}.{1}.{3}"}, true, nil},
		{[]string{"name", "substring", "fcore.dns.rocks", "dns.fcore.rocks", "answer", "name", "(fcore)\\.(dns)\\.(rocks)", "{2}.{1}.{3}"}, false, reflect.TypeOf(&substringNameRule{})},
		{[]string{"name", "substring", "a.com", "b.com", "c.com"}, true, nil},
		{[]string{"type"}, true, nil},
		{[]string{"type", "a"}, true, nil},
		{[]string{"type", "any", "a", "a"}, true, nil},
		{[]string{"type", "any", "a"}, false, reflect.TypeOf(&typeRule{})},
		{[]string{"type", "XY", "WV"}, true, nil},
		{[]string{"type", "ANY", "WV"}, true, nil},
		{[]string{"class"}, true, nil},
		{[]string{"class", "IN"}, true, nil},
		{[]string{"class", "ch", "in", "in"}, true, nil},
		{[]string{"class", "ch", "in"}, false, reflect.TypeOf(&classRule{})},
		{[]string{"class", "XY", "WV"}, true, nil},
		{[]string{"class", "IN", "WV"}, true, nil},
		{[]string{"edns0"}, true, nil},
		{[]string{"edns0", "unknown-rule-type", "set"}, true, nil},
		{[]string{"edns0", "unknown-rule-type", "unset"}, true, nil},
		{[]string{"edns0", "local"}, true, nil},
		{[]string{"edns0", "local", "set"}, true, nil},
		{[]string{"edns0", "local", "set", "0xffee"}, true, nil},
		{[]string{"edns0", "local", "set", "invalid-uint", "abcdefg"}, true, nil},
		{[]string{"edns0", "local", "set", "65518", "abcdefg"}, false, reflect.TypeOf(&edns0LocalRule{})},
		{[]string{"edns0", "local", "set", "0xffee", "abcdefg"}, false, reflect.TypeOf(&edns0LocalRule{})},
		{[]string{"edns0", "local", "set", "0xffee", "abcdefg", "revert"}, false, reflect.TypeOf(&edns0LocalRule{})},
		{[]string{"edns0", "local", "append", "0xffee", "abcdefg"}, false, reflect.TypeOf(&edns0LocalRule{})},
		{[]string{"edns0", "local", "append", "0xffee", "abcdefg", "revert"}, false, reflect.TypeOf(&edns0LocalRule{})},
		{[]string{"edns0", "local", "replace", "0xffee", "abcdefg"}, false, reflect.TypeOf(&edns0LocalRule{})},
		{[]string{"edns0", "local", "replace", "0xffee", "abcdefg", "revert"}, false, reflect.TypeOf(&edns0LocalRule{})},
		{[]string{"edns0", "local", "unset", "0xffee"}, false, reflect.TypeOf(&edns0LocalRule{})},
		{[]string{"edns0", "local", "unset", "0xffee", "abcdefg"}, true, reflect.TypeOf(&edns0LocalRule{})},
		{[]string{"edns0", "local", "unset", "0xffee", "revert"}, true, reflect.TypeOf(&edns0LocalRule{})},
		{[]string{"edns0", "local", "foo", "0xffee", "abcdefg"}, true, nil},
		{[]string{"edns0", "local", "set", "0xffee", "0xabcdefg"}, true, nil},
		{[]string{"edns0", "nsid", "set", "junk"}, true, nil},
		{[]string{"edns0", "nsid", "set"}, false, reflect.TypeOf(&edns0NsidRule{})},
		{[]string{"edns0", "nsid", "set", "revert"}, false, reflect.TypeOf(&edns0NsidRule{})},
		{[]string{"edns0", "nsid", "append"}, false, reflect.TypeOf(&edns0NsidRule{})},
		{[]string{"edns0", "nsid", "append", "revert"}, false, reflect.TypeOf(&edns0NsidRule{})},
		{[]string{"edns0", "nsid", "replace"}, false, reflect.TypeOf(&edns0NsidRule{})},
		{[]string{"edns0", "nsid", "replace", "revert"}, false, reflect.TypeOf(&edns0NsidRule{})},
		{[]string{"edns0", "nsid", "unset"}, false, reflect.TypeOf(&edns0NsidRule{})},
		{[]string{"edns0", "nsid", "unset", "revert"}, true, reflect.TypeOf(&edns0NsidRule{})},
		{[]string{"edns0", "nsid", "foo"}, true, nil},
		{[]string{"edns0", "local", "set", "invalid-uint", "{qname}"}, true, nil},
		{[]string{"edns0", "local", "set", "0xffee", "{dummy}"}, true, nil},
		{[]string{"edns0", "local", "set", "0xffee", "{qname}"}, false, reflect.TypeOf(&edns0VariableRule{})},
		{[]string{"edns0", "local", "set", "0xffee", "{qtype}"}, false, reflect.TypeOf(&edns0VariableRule{})},
		{[]string{"edns0", "local", "set", "0xffee", "{client_ip}"}, false, reflect.TypeOf(&edns0VariableRule{})},
		{[]string{"edns0", "local", "set", "0xffee", "{client_port}"}, false, reflect.TypeOf(&edns0VariableRule{})},
		{[]string{"edns0", "local", "set", "0xffee", "{protocol}"}, false, reflect.TypeOf(&edns0VariableRule{})},
		{[]string{"edns0", "local", "set", "0xffee", "{server_ip}"}, false, reflect.TypeOf(&edns0VariableRule{})},
		{[]string{"edns0", "local", "set", "0xffee", "{server_port}"}, false, reflect.TypeOf(&edns0VariableRule{})},
		{[]string{"edns0", "local", "set", "0xffee", "{server_port}", "revert"}, false, reflect.TypeOf(&edns0VariableRule{})},
		{[]string{"edns0", "local", "append", "0xffee", "{dummy}"}, true, nil},
		{[]string{"edns0", "local", "append", "0xffee", "{qname}"}, false, reflect.TypeOf(&edns0VariableRule{})},
		{[]string{"edns0", "local", "append", "0xffee", "{qtype}"}, false, reflect.TypeOf(&edns0VariableRule{})},
		{[]string{"edns0", "local", "append", "0xffee", "{client_ip}"}, false, reflect.TypeOf(&edns0VariableRule{})},
		{[]string{"edns0", "local", "append", "0xffee", "{client_port}"}, false, reflect.TypeOf(&edns0VariableRule{})},
		{[]string{"edns0", "local", "append", "0xffee", "{protocol}"}, false, reflect.TypeOf(&edns0VariableRule{})},
		{[]string{"edns0", "local", "append", "0xffee", "{server_ip}"}, false, reflect.TypeOf(&edns0VariableRule{})},
		{[]string{"edns0", "local", "append", "0xffee", "{server_port}"}, false, reflect.TypeOf(&edns0VariableRule{})},
		{[]string{"edns0", "local", "append", "0xffee", "{server_port}", "revert"}, false, reflect.TypeOf(&edns0VariableRule{})},
		{[]string{"edns0", "local", "replace", "0xffee", "{dummy}"}, true, nil},
		{[]string{"edns0", "local", "replace", "0xffee", "{qname}"}, false, reflect.TypeOf(&edns0VariableRule{})},
		{[]string{"edns0", "local", "replace", "0xffee", "{qtype}"}, false, reflect.TypeOf(&edns0VariableRule{})},
		{[]string{"edns0", "local", "replace", "0xffee", "{client_ip}"}, false, reflect.TypeOf(&edns0VariableRule{})},
		{[]string{"edns0", "local", "replace", "0xffee", "{client_port}"}, false, reflect.TypeOf(&edns0VariableRule{})},
		{[]string{"edns0", "local", "replace", "0xffee", "{protocol}"}, false, reflect.TypeOf(&edns0VariableRule{})},
		{[]string{"edns0", "local", "replace", "0xffee", "{server_ip}"}, false, reflect.TypeOf(&edns0VariableRule{})},
		{[]string{"edns0", "local", "replace", "0xffee", "{server_port}"}, false, reflect.TypeOf(&edns0VariableRule{})},
		{[]string{"edns0", "local", "replace", "0xffee", "{server_port}", "revert"}, false, reflect.TypeOf(&edns0VariableRule{})},
		{[]string{"edns0", "subnet", "set", "-1", "56"}, true, nil},
		{[]string{"edns0", "subnet", "set", "24", "-56"}, true, nil},
		{[]string{"edns0", "subnet", "set", "33", "56"}, true, nil},
		{[]string{"edns0", "subnet", "set", "24", "129"}, true, nil},
		{[]string{"edns0", "subnet", "set", "24", "56"}, false, reflect.TypeOf(&edns0SubnetRule{})},
		{[]string{"edns0", "subnet", "set", "24", "56", "revert"}, false, reflect.TypeOf(&edns0SubnetRule{})},
		{[]string{"edns0", "subnet", "append", "24", "56"}, false, reflect.TypeOf(&edns0SubnetRule{})},
		{[]string{"edns0", "subnet", "append", "24", "56", "72"}, true, nil},
		{[]string{"edns0", "subnet", "append", "24", "56", "revert"}, false, reflect.TypeOf(&edns0SubnetRule{})},
		{[]string{"edns0", "subnet", "replace", "24", "56"}, false, reflect.TypeOf(&edns0SubnetRule{})},
		{[]string{"edns0", "subnet", "replace", "24", "56", "revert"}, false, reflect.TypeOf(&edns0SubnetRule{})},
		{[]string{"edns0", "subnet", "unset"}, false, reflect.TypeOf(&edns0SubnetRule{})},
		{[]string{"edns0", "subnet", "unset", "24", "56"}, true, reflect.TypeOf(&edns0SubnetRule{})},
		{[]string{"edns0", "subnet", "unset", "revert"}, true, reflect.TypeOf(&edns0SubnetRule{})},
		{[]string{"unknown-action", "name", "a.com", "b.com"}, true, nil},
		{[]string{"stop", "name", "a.com", "b.com"}, false, reflect.TypeOf(&exactNameRule{})},
		{[]string{"continue", "name", "a.com", "b.com"}, false, reflect.TypeOf(&exactNameRule{})},
		{[]string{"unknown-action", "type", "any", "a"}, true, nil},
		{[]string{"stop", "type", "any", "a"}, false, reflect.TypeOf(&typeRule{})},
		{[]string{"continue", "type", "any", "a"}, false, reflect.TypeOf(&typeRule{})},
		{[]string{"unknown-action", "class", "ch", "in"}, true, nil},
		{[]string{"stop", "class", "ch", "in"}, false, reflect.TypeOf(&classRule{})},
		{[]string{"continue", "class", "ch", "in"}, false, reflect.TypeOf(&classRule{})},
		{[]string{"unknown-action", "edns0", "local", "set", "0xffee", "abcedef"}, true, nil},
		{[]string{"stop", "edns0", "local", "set", "0xffee", "abcdefg"}, false, reflect.TypeOf(&edns0LocalRule{})},
		{[]string{"continue", "edns0", "local", "set", "0xffee", "abcdefg"}, false, reflect.TypeOf(&edns0LocalRule{})},
		{[]string{"unknown-action", "edns0", "nsid", "set"}, true, nil},
		{[]string{"stop", "edns0", "nsid", "set"}, false, reflect.TypeOf(&edns0NsidRule{})},
		{[]string{"continue", "edns0", "nsid", "set"}, false, reflect.TypeOf(&edns0NsidRule{})},
		{[]string{"unknown-action", "edns0", "local", "set", "0xffee", "{qname}"}, true, nil},
		{[]string{"stop", "edns0", "local", "set", "0xffee", "{qname}"}, false, reflect.TypeOf(&edns0VariableRule{})},
		{[]string{"stop", "edns0", "local", "set", "0xffee", "{qtype}"}, false, reflect.TypeOf(&edns0VariableRule{})},
		{[]string{"stop", "edns0", "local", "set", "0xffee", "{client_ip}"}, false, reflect.TypeOf(&edns0VariableRule{})},
		{[]string{"stop", "edns0", "local", "set", "0xffee", "{client_port}"}, false, reflect.TypeOf(&edns0VariableRule{})},
		{[]string{"stop", "edns0", "local", "set", "0xffee", "{protocol}"}, false, reflect.TypeOf(&edns0VariableRule{})},
		{[]string{"stop", "edns0", "local", "set", "0xffee", "{server_ip}"}, false, reflect.TypeOf(&edns0VariableRule{})},
		{[]string{"stop", "edns0", "local", "set", "0xffee", "{server_port}"}, false, reflect.TypeOf(&edns0VariableRule{})},
		{[]string{"continue", "edns0", "local", "set", "0xffee", "{qname}"}, false, reflect.TypeOf(&edns0VariableRule{})},
		{[]string{"continue", "edns0", "local", "set", "0xffee", "{qtype}"}, false, reflect.TypeOf(&edns0VariableRule{})},
		{[]string{"continue", "edns0", "local", "set", "0xffee", "{client_ip}"}, false, reflect.TypeOf(&edns0VariableRule{})},
		{[]string{"continue", "edns0", "local", "set", "0xffee", "{client_port}"}, false, reflect.TypeOf(&edns0VariableRule{})},
		{[]string{"continue", "edns0", "local", "set", "0xffee", "{protocol}"}, false, reflect.TypeOf(&edns0VariableRule{})},
		{[]string{"continue", "edns0", "local", "set", "0xffee", "{server_ip}"}, false, reflect.TypeOf(&edns0VariableRule{})},
		{[]string{"continue", "edns0", "local", "set", "0xffee", "{server_port}"}, false, reflect.TypeOf(&edns0VariableRule{})},
		{[]string{"unknown-action", "edns0", "subnet", "set", "24", "64"}, true, nil},
		{[]string{"stop", "edns0", "subnet", "set", "24", "56"}, false, reflect.TypeOf(&edns0SubnetRule{})},
		{[]string{"stop", "edns0", "subnet", "append", "24", "56"}, false, reflect.TypeOf(&edns0SubnetRule{})},
		{[]string{"stop", "edns0", "subnet", "replace", "24", "56"}, false, reflect.TypeOf(&edns0SubnetRule{})},
		{[]string{"stop", "edns0", "subnet", "unset"}, false, reflect.TypeOf(&edns0SubnetRule{})},
		{[]string{"continue", "edns0", "subnet", "set", "24", "56"}, false, reflect.TypeOf(&edns0SubnetRule{})},
		{[]string{"continue", "edns0", "subnet", "append", "24", "56"}, false, reflect.TypeOf(&edns0SubnetRule{})},
		{[]string{"continue", "edns0", "subnet", "replace", "24", "56"}, false, reflect.TypeOf(&edns0SubnetRule{})},
		{[]string{"continue", "edns0", "subnet", "unset"}, false, reflect.TypeOf(&edns0SubnetRule{})},
	}

	for i, tc := range tests {
		r, err := newRule(tc.args...)
		if err == nil && tc.shouldError {
			t.Errorf("Test %d: expected error but got success", i)
		} else if err != nil && !tc.shouldError {
			t.Errorf("Test %d: expected success but got error: %s", i, err)
		}

		if !tc.shouldError && reflect.TypeOf(r) != tc.expType {
			t.Errorf("Test %d: expected %q but got %q", i, tc.expType, r)
		}
	}
}

func TestRewriteDefaultRevertPolicy(t *testing.T) {
	rules := []Rule{}

	r, _ := newNameRule("stop", "prefix", "prefix", "to")
	rules = append(rules, r)
	r, _ = newNameRule("stop", "suffix", ".suffix.", ".nl.")
	rules = append(rules, r)
	r, _ = newNameRule("stop", "substring", "from.substring", "to")
	rules = append(rules, r)
	r, _ = newNameRule("stop", "regex", "(f.*m)\\.regex\\.(nl)", "to.{2}")
	rules = append(rules, r)

	rw := Rewrite{
		Next:  plugin.HandlerFunc(msgPrinter),
		Rules: rules,
		// use production (default) RevertPolicy
	}

	tests := []struct {
		from  string
		fromT uint16
		fromC uint16
		to    string
		toT   uint16
		toC   uint16
	}{
		{"prefix.nl.", dns.TypeA, dns.ClassINET, "to.nl.", dns.TypeA, dns.ClassINET},
		{"to.suffix.", dns.TypeA, dns.ClassINET, "to.nl.", dns.TypeA, dns.ClassINET},
		{"from.substring.nl.", dns.TypeA, dns.ClassINET, "to.nl.", dns.TypeA, dns.ClassINET},
		{"from.regex.nl.", dns.TypeA, dns.ClassINET, "to.nl.", dns.TypeA, dns.ClassINET},
	}

	ctx := context.TODO()
	for i, tc := range tests {
		m := new(dns.Msg)
		m.SetQuestion(tc.from, tc.fromT)
		m.Question[0].Qclass = tc.fromC

		rec := dnstest.NewRecorder(&test.ResponseWriter{})
		rw.ServeDNS(ctx, rec, m)

		resp := rec.Msg

		if resp.Question[0].Name != tc.from {
			t.Errorf("Test %d: Expected Name in Question to be %q but was %q", i, tc.from, resp.Question[0].Name)
		}

		if resp.Answer[0].Header().Name != tc.to {
			t.Errorf("Test %d: Expected Name in Answer to be %q but was %q", i, tc.to, resp.Answer[0].Header().Name)
		}
	}
}

func TestRewrite(t *testing.T) {
	rules := []Rule{}
	r, _ := newNameRule("stop", "from.nl.", "to.nl.")
	rules = append(rules, r)
	r, _ = newNameRule("stop", "regex", "(core)\\.(dns)\\.(rocks)\\.(nl)", "{2}.{1}.{3}.{4}", "answer", "name", "(dns)\\.(core)\\.(rocks)\\.(nl)", "{2}.{1}.{3}.{4}")
	rules = append(rules, r)
	r, _ = newNameRule("stop", "exact", "from.exact.nl.", "to.nl.")
	rules = append(rules, r)
	r, _ = newNameRule("stop", "prefix", "prefix", "to")
	rules = append(rules, r)
	r, _ = newNameRule("stop", "suffix", ".suffix.", ".nl.")
	rules = append(rules, r)
	r, _ = newNameRule("stop", "substring", "from.substring", "to")
	rules = append(rules, r)
	r, _ = newNameRule("stop", "regex", "(f.*m)\\.regex\\.(nl)", "to.{2}")
	rules = append(rules, r)
	r, _ = newNameRule("continue", "regex", "consul\\.(rocks)", "core.dns.{1}")
	rules = append(rules, r)
	r, _ = newNameRule("stop", "core.dns.rocks", "to.nl.")
	rules = append(rules, r)
	r, _ = newClassRule("continue", "HS", "CH")
	rules = append(rules, r)
	r, _ = newClassRule("stop", "CH", "IN")
	rules = append(rules, r)
	r, _ = newTypeRule("stop", "ANY", "HINFO")
	rules = append(rules, r)

	rw := Rewrite{
		Next:         plugin.HandlerFunc(msgPrinter),
		Rules:        rules,
		RevertPolicy: NoRevertPolicy(),
	}

	tests := []struct {
		from  string
		fromT uint16
		fromC uint16
		to    string
		toT   uint16
		toC   uint16
	}{
		{"from.nl.", dns.TypeA, dns.ClassINET, "to.nl.", dns.TypeA, dns.ClassINET},
		{"a.nl.", dns.TypeA, dns.ClassINET, "a.nl.", dns.TypeA, dns.ClassINET},
		{"a.nl.", dns.TypeA, dns.ClassCHAOS, "a.nl.", dns.TypeA, dns.ClassINET},
		{"a.nl.", dns.TypeANY, dns.ClassINET, "a.nl.", dns.TypeHINFO, dns.ClassINET},
		// name is rewritten, type is not.
		{"from.nl.", dns.TypeANY, dns.ClassINET, "to.nl.", dns.TypeANY, dns.ClassINET},
		{"from.exact.nl.", dns.TypeA, dns.ClassINET, "to.nl.", dns.TypeA, dns.ClassINET},
		{"prefix.nl.", dns.TypeA, dns.ClassINET, "to.nl.", dns.TypeA, dns.ClassINET},
		{"to.suffix.", dns.TypeA, dns.ClassINET, "to.nl.", dns.TypeA, dns.ClassINET},
		{"from.substring.nl.", dns.TypeA, dns.ClassINET, "to.nl.", dns.TypeA, dns.ClassINET},
		{"from.regex.nl.", dns.TypeA, dns.ClassINET, "to.nl.", dns.TypeA, dns.ClassINET},
		{"consul.rocks.", dns.TypeA, dns.ClassINET, "to.nl.", dns.TypeA, dns.ClassINET},
		// name is not, type is, but class is, because class is the 2nd rule.
		{"a.nl.", dns.TypeANY, dns.ClassCHAOS, "a.nl.", dns.TypeANY, dns.ClassINET},
		// class gets rewritten twice because of continue/stop logic: HS to CH, CH to IN
		{"a.nl.", dns.TypeANY, 4, "a.nl.", dns.TypeANY, dns.ClassINET},
		{"core.dns.rocks.nl.", dns.TypeA, dns.ClassINET, "dns.core.rocks.nl.", dns.TypeA, dns.ClassINET},
	}

	ctx := context.TODO()
	for i, tc := range tests {
		m := new(dns.Msg)
		m.SetQuestion(tc.from, tc.fromT)
		m.Question[0].Qclass = tc.fromC

		rec := dnstest.NewRecorder(&test.ResponseWriter{})
		rw.ServeDNS(ctx, rec, m)

		resp := rec.Msg
		if resp.Question[0].Name != tc.to {
			t.Errorf("Test %d: Expected Name to be %q but was %q", i, tc.to, resp.Question[0].Name)
		}
		if resp.Question[0].Qtype != tc.toT {
			t.Errorf("Test %d: Expected Type to be '%d' but was '%d'", i, tc.toT, resp.Question[0].Qtype)
		}
		if resp.Question[0].Qclass != tc.toC {
			t.Errorf("Test %d: Expected Class to be '%d' but was '%d'", i, tc.toC, resp.Question[0].Qclass)
		}
		if tc.fromT == dns.TypeA && tc.toT == dns.TypeA {
			if len(resp.Answer) > 0 {
				if resp.Answer[0].(*dns.A).Hdr.Name != tc.to {
					t.Errorf("Test %d: Expected Answer Name to be %q but was %q", i, tc.to, resp.Answer[0].(*dns.A).Hdr.Name)
				}
			}
		}
	}
}

func TestRewriteEDNS0Local(t *testing.T) {
	rw := Rewrite{
		Next:         plugin.HandlerFunc(msgPrinter),
		RevertPolicy: NoRevertPolicy(),
	}

	tests := []struct {
		fromOpts []dns.EDNS0
		args     []string
		toOpts   []dns.EDNS0
		doBool   bool
	}{
		{
			[]dns.EDNS0{},
			[]string{"local", "set", "0xffee", "0xabcdef"},
			[]dns.EDNS0{&dns.EDNS0_LOCAL{Code: 0xffee, Data: []byte{0xab, 0xcd, 0xef}}},
			false,
		},
		{
			[]dns.EDNS0{},
			[]string{"local", "append", "0xffee", "abcdefghijklmnop"},
			[]dns.EDNS0{&dns.EDNS0_LOCAL{Code: 0xffee, Data: []byte("abcdefghijklmnop")}},
			false,
		},
		{
			[]dns.EDNS0{},
			[]string{"local", "replace", "0xffee", "abcdefghijklmnop"},
			[]dns.EDNS0{},
			true,
		},
		{
			[]dns.EDNS0{},
			[]string{"nsid", "set"},
			[]dns.EDNS0{&dns.EDNS0_NSID{Code: dns.EDNS0NSID, Nsid: ""}},
			false,
		},
		{
			[]dns.EDNS0{},
			[]string{"nsid", "append"},
			[]dns.EDNS0{&dns.EDNS0_NSID{Code: dns.EDNS0NSID, Nsid: ""}},
			true,
		},
		{
			[]dns.EDNS0{},
			[]string{"nsid", "replace"},
			[]dns.EDNS0{},
			true,
		},
	}

	ctx := context.TODO()
	for i, tc := range tests {
		m := new(dns.Msg)
		m.SetQuestion("example.com.", dns.TypeA)
		m.Question[0].Qclass = dns.ClassINET

		r, err := newEdns0Rule("stop", tc.args...)
		if err != nil {
			t.Errorf("Error creating test rule: %s", err)
			continue
		}
		rw.Rules = []Rule{r}

		rec := dnstest.NewRecorder(&test.ResponseWriter{})
		rw.ServeDNS(ctx, rec, m)

		resp := rec.Msg
		o := resp.IsEdns0()
		o.SetDo(tc.doBool)
		if o == nil {
			t.Errorf("Test %d: EDNS0 options not set", i)
			continue
		}
		if o.Do() != tc.doBool {
			t.Errorf("Test %d: Expected %v but got %v", i, tc.doBool, o.Do())
		}
		if !optsEqual(o.Option, tc.toOpts) {
			t.Errorf("Test %d: Expected %v but got %v", i, tc.toOpts, o)
		}
	}
}

func TestEdns0MultiRule(t *testing.T) {
	tests := []struct {
		rules        [][]string
		fromOpts     []dns.EDNS0
		toOpts       []dns.EDNS0
		revertPolicy RevertPolicy
	}{
		// Local.
		{
			[][]string{
				{"stop", "local", "replace", "0xffee", "abcdef"},
				{"stop", "local", "set", "0xffee", "fedcba"},
			},
			nil,
			[]dns.EDNS0{&dns.EDNS0_LOCAL{Code: 0xffee, Data: []byte("fedcba")}},
			NoRevertPolicy(),
		},
		{
			[][]string{
				{"stop", "local", "replace", "0xffee", "abcdef"},
				{"stop", "local", "set", "0xffee", "fedcba"},
			},
			[]dns.EDNS0{&dns.EDNS0_LOCAL{Code: 0xffee, Data: []byte("foobar")}},
			[]dns.EDNS0{&dns.EDNS0_LOCAL{Code: 0xffee, Data: []byte("abcdef")}},
			NoRevertPolicy(),
		},
		// Local with "revert".
		{
			[][]string{
				{"stop", "local", "replace", "0xffee", "abcdef", "revert"},
				{"stop", "local", "set", "0xffee", "fedcba", "revert"},
			},
			nil,
			[]dns.EDNS0{},
			NewRevertPolicy(false, false),
		},
		{
			[][]string{
				{"stop", "local", "replace", "0xffee", "abcdef", "revert"},
				{"stop", "local", "set", "0xffee", "fedcba", "revert"},
			},
			[]dns.EDNS0{&dns.EDNS0_LOCAL{Code: 0xffee, Data: []byte("foobar")}},
			[]dns.EDNS0{&dns.EDNS0_LOCAL{Code: 0xffee, Data: []byte("foobar")}},
			NewRevertPolicy(false, false),
		},
		// Local variable.
		{
			[][]string{
				{"stop", "local", "replace", "0xffee", "{qname}"},
				{"stop", "local", "set", "0xffee", "{qtype}"},
			},
			nil,
			[]dns.EDNS0{&dns.EDNS0_LOCAL{Code: 0xffee, Data: []byte{0x00, 0x01}}},
			NoRevertPolicy(),
		},
		{
			[][]string{
				{"stop", "local", "replace", "0xffee", "{qname}"},
				{"stop", "local", "set", "0xffee", "{qtype}"},
			},
			[]dns.EDNS0{&dns.EDNS0_LOCAL{Code: 0xffee, Data: []byte("foobar")}},
			[]dns.EDNS0{&dns.EDNS0_LOCAL{Code: 0xffee, Data: []byte("example.com.")}},
			NoRevertPolicy(),
		},
		// Local variable with "revert".
		{
			[][]string{
				{"stop", "local", "replace", "0xffee", "{qname}", "revert"},
				{"stop", "local", "set", "0xffee", "{qtype}", "revert"},
			},
			nil,
			[]dns.EDNS0{},
			NewRevertPolicy(false, false),
		},
		{
			[][]string{
				{"stop", "local", "replace", "0xffee", "{qname}", "revert"},
				{"stop", "local", "set", "0xffee", "{qtype}", "revert"},
			},
			[]dns.EDNS0{&dns.EDNS0_LOCAL{Code: 0xffee, Data: []byte("foobar")}},
			[]dns.EDNS0{&dns.EDNS0_LOCAL{Code: 0xffee, Data: []byte("foobar")}},
			NewRevertPolicy(false, false),
		},
		// Nsid.
		{
			[][]string{
				{"stop", "nsid", "replace"},
				{"stop", "nsid", "set"},
			},
			nil,
			[]dns.EDNS0{&dns.EDNS0_NSID{Code: dns.EDNS0NSID, Nsid: ""}},
			NoRevertPolicy(),
		},
		{
			[][]string{
				{"stop", "nsid", "replace"},
				{"stop", "nsid", "set"},
			},
			[]dns.EDNS0{&dns.EDNS0_LOCAL{Code: 0xffee, Data: []byte("foobar")}},
			[]dns.EDNS0{&dns.EDNS0_LOCAL{Code: 0xffee, Data: []byte("foobar")}, &dns.EDNS0_NSID{Code: dns.EDNS0NSID, Nsid: ""}},
			NoRevertPolicy(),
		},
		{
			[][]string{
				{"stop", "nsid", "replace"},
				{"stop", "nsid", "set"},
			},
			[]dns.EDNS0{&dns.EDNS0_NSID{Code: dns.EDNS0NSID, Nsid: ""}},
			[]dns.EDNS0{&dns.EDNS0_NSID{Code: dns.EDNS0NSID, Nsid: ""}},
			NoRevertPolicy(),
		},
		// Nsid with "revert".
		{
			[][]string{
				{"stop", "nsid", "replace", "revert"},
				{"stop", "nsid", "set", "revert"},
			},
			nil,
			[]dns.EDNS0{},
			NewRevertPolicy(false, false),
		},
		{
			[][]string{
				{"stop", "nsid", "replace", "revert"},
				{"stop", "nsid", "set", "revert"},
			},
			[]dns.EDNS0{&dns.EDNS0_LOCAL{Code: 0xffee, Data: []byte("foobar")}},
			[]dns.EDNS0{&dns.EDNS0_LOCAL{Code: 0xffee, Data: []byte("foobar")}},
			NewRevertPolicy(false, false),
		},
		{
			[][]string{
				{"stop", "nsid", "replace", "revert"},
				{"stop", "nsid", "set", "revert"},
			},
			[]dns.EDNS0{&dns.EDNS0_NSID{Code: dns.EDNS0NSID, Nsid: ""}},
			[]dns.EDNS0{&dns.EDNS0_NSID{Code: dns.EDNS0NSID, Nsid: ""}},
			NewRevertPolicy(false, false),
		},
		// Subnet.
		{
			[][]string{
				{"stop", "subnet", "replace", "32", "56"},
				{"stop", "subnet", "set", "0", "56"},
			},
			nil,
			[]dns.EDNS0{&dns.EDNS0_SUBNET{Code: 0x8,
				Family:        0x1,
				SourceNetmask: 0x0,
				SourceScope:   0x0,
				Address:       []byte{0x00, 0x00, 0x00, 0x00},
			}},
			NoRevertPolicy(),
		},
		{
			[][]string{
				{"stop", "subnet", "replace", "32", "56"},
				{"stop", "subnet", "set", "0", "56"},
			},
			[]dns.EDNS0{&dns.EDNS0_SUBNET{Code: 0x8,
				Family:        0x1,
				SourceNetmask: 0x0,
				SourceScope:   0x0,
				Address:       []byte{0x00, 0x00, 0x00, 0x00},
			}},
			[]dns.EDNS0{&dns.EDNS0_SUBNET{Code: 0x8,
				Family:        0x1,
				SourceNetmask: 0x20,
				SourceScope:   0x0,
				Address:       []byte{0x0A, 0xF0, 0x00, 0x01},
			}},
			NoRevertPolicy(),
		},
		// Subnet with "revert".
		{
			[][]string{
				{"stop", "subnet", "replace", "32", "56", "revert"},
				{"stop", "subnet", "set", "0", "56", "revert"},
			},
			nil,
			[]dns.EDNS0{},
			NewRevertPolicy(false, false),
		},
		{
			[][]string{
				{"stop", "subnet", "replace", "32", "56", "revert"},
				{"stop", "subnet", "set", "0", "56", "revert"},
			},
			[]dns.EDNS0{&dns.EDNS0_SUBNET{Code: 0x8,
				Family:        0x1,
				SourceNetmask: 0x0,
				SourceScope:   0x0,
				Address:       []byte{0x00, 0x00, 0x00, 0x00},
			}},
			[]dns.EDNS0{&dns.EDNS0_SUBNET{Code: 0x8,
				Family:        0x1,
				SourceNetmask: 0x0,
				SourceScope:   0x0,
				Address:       []byte{0x00, 0x00, 0x00, 0x00},
			}},
			NewRevertPolicy(false, false),
		},
	}

	ctx := context.TODO()
	for i, tc := range tests {
		m := new(dns.Msg)
		m.SetQuestion("example.com.", dns.TypeA)
		m.Question[0].Qclass = dns.ClassINET
		if tc.fromOpts != nil {
			o := m.IsEdns0()
			if o == nil {
				m.SetEdns0(4096, true)
				o = m.IsEdns0()
			}
			o.Option = append(o.Option, tc.fromOpts...)
		}
		rec := dnstest.NewRecorder(&test.ResponseWriter{})

		rules := make([]Rule, 0, len(tc.rules))
		for _, rule := range tc.rules {
			r, _ := newEdns0Rule(rule[0], rule[1:]...)
			rules = append(rules, r)
		}

		rw := Rewrite{
			Next:         plugin.HandlerFunc(msgPrinter),
			Rules:        rules,
			RevertPolicy: tc.revertPolicy,
		}

		rw.ServeDNS(ctx, rec, m)

		resp := rec.Msg
		o := resp.IsEdns0()
		if o == nil {
			t.Errorf("Test %d: EDNS0 options not set", i)
			continue
		}
		if !optsEqual(o.Option, tc.toOpts) {
			t.Errorf("Test %d: Expected %v but got %v", i, tc.toOpts, o)
		}
	}
}

func optsEqual(a, b []dns.EDNS0) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		switch aa := a[i].(type) {
		case *dns.EDNS0_LOCAL:
			if bb, ok := b[i].(*dns.EDNS0_LOCAL); ok {
				if aa.Code != bb.Code {
					return false
				}
				if !bytes.Equal(aa.Data, bb.Data) {
					return false
				}
			} else {
				return false
			}
		case *dns.EDNS0_NSID:
			if bb, ok := b[i].(*dns.EDNS0_NSID); ok {
				if aa.Nsid != bb.Nsid {
					return false
				}
			} else {
				return false
			}
		case *dns.EDNS0_SUBNET:
			if bb, ok := b[i].(*dns.EDNS0_SUBNET); ok {
				if aa.Code != bb.Code {
					return false
				}
				if aa.Family != bb.Family {
					return false
				}
				if aa.SourceNetmask != bb.SourceNetmask {
					return false
				}
				if aa.SourceScope != bb.SourceScope {
					return false
				}
				if !aa.Address.Equal(bb.Address) {
					return false
				}
			} else {
				return false
			}

		default:
			return false
		}
	}
	return true
}

type testProvider map[string]metadata.Func

func (tp testProvider) Metadata(ctx context.Context, state request.Request) context.Context {
	for k, v := range tp {
		metadata.SetValueFunc(ctx, k, v)
	}
	return ctx
}

func TestRewriteEDNS0LocalVariable(t *testing.T) {
	rw := Rewrite{
		Next:         plugin.HandlerFunc(msgPrinter),
		RevertPolicy: NoRevertPolicy(),
	}

	expectedMetadata := []metadata.Provider{
		testProvider{"test/label": func() string { return "my-value" }},
		testProvider{"test/empty": func() string { return "" }},
	}

	meta := metadata.Metadata{
		Zones:     []string{"."},
		Providers: expectedMetadata,
		Next:      &rw,
	}

	// test.ResponseWriter has the following values:
	// 		The remote will always be 10.240.0.1 and port 40212.
	// 		The local address is always 127.0.0.1 and port 53.

	tests := []struct {
		fromOpts []dns.EDNS0
		args     []string
		toOpts   []dns.EDNS0
		doBool   bool
	}{
		{
			[]dns.EDNS0{},
			[]string{"local", "set", "0xffee", "{qname}"},
			[]dns.EDNS0{&dns.EDNS0_LOCAL{Code: 0xffee, Data: []byte("example.com.")}},
			true,
		},
		{
			[]dns.EDNS0{},
			[]string{"local", "set", "0xffee", "{qtype}"},
			[]dns.EDNS0{&dns.EDNS0_LOCAL{Code: 0xffee, Data: []byte{0x00, 0x01}}},
			false,
		},
		{
			[]dns.EDNS0{},
			[]string{"local", "set", "0xffee", "{client_ip}"},
			[]dns.EDNS0{&dns.EDNS0_LOCAL{Code: 0xffee, Data: []byte{0x0A, 0xF0, 0x00, 0x01}}},
			false,
		},
		{
			[]dns.EDNS0{},
			[]string{"local", "set", "0xffee", "{client_port}"},
			[]dns.EDNS0{&dns.EDNS0_LOCAL{Code: 0xffee, Data: []byte{0x9D, 0x14}}},
			true,
		},
		{
			[]dns.EDNS0{},
			[]string{"local", "set", "0xffee", "{protocol}"},
			[]dns.EDNS0{&dns.EDNS0_LOCAL{Code: 0xffee, Data: []byte("udp")}},
			false,
		},
		{
			[]dns.EDNS0{},
			[]string{"local", "set", "0xffee", "{server_port}"},
			[]dns.EDNS0{&dns.EDNS0_LOCAL{Code: 0xffee, Data: []byte{0x00, 0x35}}},
			true,
		},
		{
			[]dns.EDNS0{},
			[]string{"local", "set", "0xffee", "{server_ip}"},
			[]dns.EDNS0{&dns.EDNS0_LOCAL{Code: 0xffee, Data: []byte{0x7F, 0x00, 0x00, 0x01}}},
			true,
		},
		{
			[]dns.EDNS0{},
			[]string{"local", "set", "0xffee", "{test/label}"},
			[]dns.EDNS0{&dns.EDNS0_LOCAL{Code: 0xffee, Data: []byte("my-value")}},
			true,
		},
		{
			[]dns.EDNS0{},
			[]string{"local", "set", "0xffee", "{test/empty}"},
			nil,
			false,
		},
		{
			[]dns.EDNS0{},
			[]string{"local", "set", "0xffee", "{test/does-not-exist}"},
			nil,
			false,
		},
	}

	for i, tc := range tests {
		m := new(dns.Msg)
		m.SetQuestion("example.com.", dns.TypeA)

		r, err := newEdns0Rule("stop", tc.args...)
		if err != nil {
			t.Errorf("Error creating test rule: %s", err)
			continue
		}
		rw.Rules = []Rule{r}

		rec := dnstest.NewRecorder(&test.ResponseWriter{})
		ctx := meta.Collect(context.TODO(), request.Request{W: rec, Req: m})
		meta.ServeDNS(ctx, rec, m)

		resp := rec.Msg
		o := resp.IsEdns0()
		if o == nil {
			if tc.toOpts != nil {
				t.Errorf("Test %d: EDNS0 options not set", i)
			}
			continue
		}
		o.SetDo(tc.doBool)
		if o.Do() != tc.doBool {
			t.Errorf("Test %d: Expected %v but got %v", i, tc.doBool, o.Do())
		}
		if !optsEqual(o.Option, tc.toOpts) {
			t.Errorf("Test %d: Expected %v but got %v", i, tc.toOpts, o)
		}
	}
}

func TestRewriteEDNS0Subnet(t *testing.T) {
	rw := Rewrite{
		Next:         plugin.HandlerFunc(msgPrinter),
		RevertPolicy: NoRevertPolicy(),
	}

	tests := []struct {
		writer   dns.ResponseWriter
		fromOpts []dns.EDNS0
		args     []string
		toOpts   []dns.EDNS0
		doBool   bool
	}{
		{
			&test.ResponseWriter{},
			[]dns.EDNS0{},
			[]string{"subnet", "set", "24", "56"},
			[]dns.EDNS0{&dns.EDNS0_SUBNET{Code: 0x8,
				Family:        0x1,
				SourceNetmask: 0x18,
				SourceScope:   0x0,
				Address:       []byte{0x0A, 0xF0, 0x00, 0x00},
			}},
			true,
		},
		{
			&test.ResponseWriter{},
			[]dns.EDNS0{},
			[]string{"subnet", "set", "32", "56"},
			[]dns.EDNS0{&dns.EDNS0_SUBNET{Code: 0x8,
				Family:        0x1,
				SourceNetmask: 0x20,
				SourceScope:   0x0,
				Address:       []byte{0x0A, 0xF0, 0x00, 0x01},
			}},
			false,
		},
		{
			&test.ResponseWriter{},
			[]dns.EDNS0{},
			[]string{"subnet", "set", "0", "56"},
			[]dns.EDNS0{&dns.EDNS0_SUBNET{Code: 0x8,
				Family:        0x1,
				SourceNetmask: 0x0,
				SourceScope:   0x0,
				Address:       []byte{0x00, 0x00, 0x00, 0x00},
			}},
			false,
		},
		{
			&test.ResponseWriter6{},
			[]dns.EDNS0{},
			[]string{"subnet", "set", "24", "56"},
			[]dns.EDNS0{&dns.EDNS0_SUBNET{Code: 0x8,
				Family:        0x2,
				SourceNetmask: 0x38,
				SourceScope:   0x0,
				Address: []byte{0xfe, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			}},
			true,
		},
		{
			&test.ResponseWriter6{},
			[]dns.EDNS0{},
			[]string{"subnet", "set", "24", "128"},
			[]dns.EDNS0{&dns.EDNS0_SUBNET{Code: 0x8,
				Family:        0x2,
				SourceNetmask: 0x80,
				SourceScope:   0x0,
				Address: []byte{0xfe, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x42, 0x00, 0xff, 0xfe, 0xca, 0x4c, 0x65},
			}},
			false,
		},
		{
			&test.ResponseWriter6{},
			[]dns.EDNS0{},
			[]string{"subnet", "set", "24", "0"},
			[]dns.EDNS0{&dns.EDNS0_SUBNET{Code: 0x8,
				Family:        0x2,
				SourceNetmask: 0x0,
				SourceScope:   0x0,
				Address: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			}},
			true,
		},
	}

	ctx := context.TODO()
	for i, tc := range tests {
		m := new(dns.Msg)
		m.SetQuestion("example.com.", dns.TypeA)

		r, err := newEdns0Rule("stop", tc.args...)
		if err != nil {
			t.Errorf("Error creating test rule: %s", err)
			continue
		}
		rw.Rules = []Rule{r}
		rec := dnstest.NewRecorder(tc.writer)
		rw.ServeDNS(ctx, rec, m)

		resp := rec.Msg
		o := resp.IsEdns0()
		o.SetDo(tc.doBool)
		if o == nil {
			t.Errorf("Test %d: EDNS0 options not set", i)
			continue
		}
		if o.Do() != tc.doBool {
			t.Errorf("Test %d: Expected %v but got %v", i, tc.doBool, o.Do())
		}
		if !optsEqual(o.Option, tc.toOpts) {
			t.Errorf("Test %d: Expected %v but got %v", i, tc.toOpts, o)
		}
	}
}

func TestRewriteEDNS0Revert(t *testing.T) {
	rw := Rewrite{
		Next:         plugin.HandlerFunc(msgPrinter),
		RevertPolicy: NewRevertPolicy(false, false),
	}

	tests := []struct {
		fromOpts []dns.EDNS0
		args     []string
		toOpts   []dns.EDNS0
		doBool   bool
	}{
		{
			[]dns.EDNS0{},
			[]string{"local", "set", "0xffee", "0xabcdef", "revert"},
			[]dns.EDNS0{},
			false,
		},
		{
			[]dns.EDNS0{},
			[]string{"local", "append", "0xffee", "abcdefghijklmnop", "revert"},
			[]dns.EDNS0{},
			false,
		},
		{
			[]dns.EDNS0{},
			[]string{"local", "replace", "0xffee", "abcdefghijklmnop", "revert"},
			[]dns.EDNS0{},
			true,
		},
		{
			[]dns.EDNS0{},
			[]string{"nsid", "set", "revert"},
			[]dns.EDNS0{},
			false,
		},
		{
			[]dns.EDNS0{},
			[]string{"nsid", "append", "revert"},
			[]dns.EDNS0{},
			true,
		},
		{
			[]dns.EDNS0{},
			[]string{"nsid", "replace"},
			[]dns.EDNS0{},
			true,
		},

		{
			[]dns.EDNS0{&dns.EDNS0_LOCAL{Code: 0xffed, Data: []byte{0xab, 0xcd, 0xef}}},
			[]string{"local", "set", "0xffee", "0xabcd", "revert"},
			[]dns.EDNS0{&dns.EDNS0_LOCAL{Code: 0xffed, Data: []byte{0xab, 0xcd, 0xef}}},
			false,
		},
		{
			[]dns.EDNS0{&dns.EDNS0_LOCAL{Code: 0xffef, Data: []byte{0xab, 0xcd, 0xef}}},
			[]string{"local", "replace", "0xffee", "abcdefghijklmnop"},
			[]dns.EDNS0{&dns.EDNS0_LOCAL{Code: 0xffef, Data: []byte{0xab, 0xcd, 0xef}}},
			true,
		},
	}

	ctx := context.TODO()
	for i, tc := range tests {
		m := new(dns.Msg)
		m.SetQuestion("example.com.", dns.TypeA)
		m.Question[0].Qclass = dns.ClassINET

		r, err := newEdns0Rule("stop", tc.args...)
		if err != nil {
			t.Errorf("Error creating test rule: %s", err)
			continue
		}
		rw.Rules = []Rule{r}

		rec := dnstest.NewRecorder(&test.ResponseWriter{})
		rw.ServeDNS(ctx, rec, m)

		resp := rec.Msg
		o := resp.IsEdns0()
		o.SetDo(tc.doBool)
		if tc.fromOpts != nil {
			o.Option = append(o.Option, tc.fromOpts...)
		}
		if o == nil {
			t.Errorf("Test %d: EDNS0 options not set", i)
			continue
		}
		if o.Do() != tc.doBool {
			t.Errorf("Test %d: Expected %v but got %v", i, tc.doBool, o.Do())
		}
		if !optsEqual(o.Option, tc.toOpts) {
			t.Errorf("Test %d: Expected %v but got %v", i, tc.toOpts, o)
		}
	}
}

func TestRewriteEDNS0Unset(t *testing.T) {
	rw := Rewrite{
		Next:         plugin.HandlerFunc(msgPrinter),
		RevertPolicy: NewRevertPolicy(false, false),
	}

	tests := []struct {
		fromOpts []dns.EDNS0
		args     []string
		toOpts   []dns.EDNS0
	}{
		{
			[]dns.EDNS0{},
			[]string{"local", "unset", "0xffee"},
			[]dns.EDNS0{},
		},
		{
			[]dns.EDNS0{&dns.EDNS0_LOCAL{Code: 0xffee, Data: []byte{0xab, 0xcd, 0xef}}},
			[]string{"local", "unset", "0xffee"},
			[]dns.EDNS0{},
		},
		{
			[]dns.EDNS0{&dns.EDNS0_LOCAL{Code: 0xffee, Data: []byte{0xab, 0xcd, 0xef}}},
			[]string{"local", "unset", "0xffed"},
			[]dns.EDNS0{&dns.EDNS0_LOCAL{Code: 0xffee, Data: []byte{0xab, 0xcd, 0xef}}},
		},
		{
			[]dns.EDNS0{&dns.EDNS0_NSID{Code: dns.EDNS0NSID, Nsid: ""}},
			[]string{"nsid", "unset"},
			[]dns.EDNS0{},
		},
		{
			[]dns.EDNS0{},
			[]string{"nsid", "unset"},
			[]dns.EDNS0{},
		},
		{
			[]dns.EDNS0{&dns.EDNS0_SUBNET{Code: 0x8,
				Family:        0x1,
				SourceNetmask: 0x0,
				SourceScope:   0x0,
				Address:       []byte{0x00, 0x00, 0x00, 0x00},
			}},
			[]string{"subnet", "unset"},
			[]dns.EDNS0{},
		},
		{
			[]dns.EDNS0{},
			[]string{"subnet", "unset"},
			[]dns.EDNS0{},
		},
		{
			[]dns.EDNS0{&dns.EDNS0_LOCAL{Code: 0xffee, Data: []byte("foobar")}, &dns.EDNS0_NSID{Code: dns.EDNS0NSID, Nsid: ""}},
			[]string{"nsid", "unset"},
			[]dns.EDNS0{&dns.EDNS0_LOCAL{Code: 0xffee, Data: []byte("foobar")}},
		},
		{
			[]dns.EDNS0{&dns.EDNS0_LOCAL{Code: 0xffee, Data: []byte("foobar")}, &dns.EDNS0_NSID{Code: dns.EDNS0NSID, Nsid: ""}},
			[]string{"local", "unset", "0xffee"},
			[]dns.EDNS0{&dns.EDNS0_NSID{Code: dns.EDNS0NSID, Nsid: ""}},
		},
		{
			[]dns.EDNS0{&dns.EDNS0_LOCAL{Code: 0xffee, Data: []byte("foobar")}, &dns.EDNS0_NSID{Code: dns.EDNS0NSID, Nsid: ""}},
			[]string{"subnet", "unset"},
			[]dns.EDNS0{&dns.EDNS0_LOCAL{Code: 0xffee, Data: []byte("foobar")}, &dns.EDNS0_NSID{Code: dns.EDNS0NSID, Nsid: ""}},
		},
	}

	ctx := context.TODO()
	for i, tc := range tests {
		m := new(dns.Msg)
		m.SetQuestion("example.com.", dns.TypeA)
		m.Question[0].Qclass = dns.ClassINET
		o := m.IsEdns0()
		if tc.fromOpts != nil {
			if o == nil {
				m.SetEdns0(4096, true)
				o = m.IsEdns0()
			}
			o.Option = append(o.Option, tc.fromOpts...)
		}

		r, err := newEdns0Rule("stop", tc.args...)
		if err != nil {
			t.Errorf("Error creating test rule: %s", err)
			continue
		}
		rw.Rules = []Rule{r}

		rec := dnstest.NewRecorder(&test.ResponseWriter{})
		rw.ServeDNS(ctx, rec, m)

		if !optsEqual(o.Option, tc.toOpts) {
			t.Errorf("Test %d: Expected %v but got %v", i, tc.toOpts, o)
		}
	}
}
