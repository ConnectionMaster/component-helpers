/*
Copyright 2023 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"fmt"
	"net"
	"reflect"
	"strings"
	"testing"

	netutils "k8s.io/utils/net"
)

func TestParseNodeIPArgument(t *testing.T) {
	testCases := []struct {
		desc  string
		in    string
		out   []net.IP
		err   string
		ssErr string
	}{
		{
			desc: "empty --node-ip",
			in:   "",
			out:  nil,
		},
		{
			desc: "just whitespace (ignored)",
			in:   " ",
			out:  nil,
		},
		{
			desc: "garbage (ignored)",
			in:   "blah",
			out:  nil,
		},
		{
			desc: "single IPv4",
			in:   "1.2.3.4",
			out: []net.IP{
				netutils.ParseIPSloppy("1.2.3.4"),
			},
		},
		{
			desc: "single IPv4 with whitespace",
			in:   " 1.2.3.4   ",
			out: []net.IP{
				netutils.ParseIPSloppy("1.2.3.4"),
			},
		},
		{
			desc: "single IPv4 non-canonical",
			in:   "01.2.3.004",
			out: []net.IP{
				netutils.ParseIPSloppy("1.2.3.4"),
			},
		},
		{
			desc: "single IPv4 invalid (ignored)",
			in:   "1.2.3",
			out:  nil,
		},
		{
			desc: "single IPv4 CIDR (ignored)",
			in:   "1.2.3.0/24",
			out:  nil,
		},
		{
			desc: "single IPv4 unspecified",
			in:   "0.0.0.0",
			out: []net.IP{
				net.IPv4zero,
			},
		},
		{
			desc: "single IPv4 plus ignored garbage",
			in:   "1.2.3.4,not-an-IPv6-address",
			out: []net.IP{
				netutils.ParseIPSloppy("1.2.3.4"),
			},
		},
		{
			desc: "single IPv6",
			in:   "abcd::ef01",
			out: []net.IP{
				netutils.ParseIPSloppy("abcd::ef01"),
			},
		},
		{
			desc: "single IPv6 non-canonical",
			in:   "abcd:0abc:00ab:0000:0000::1",
			out: []net.IP{
				netutils.ParseIPSloppy("abcd:abc:ab::1"),
			},
		},
		{
			desc: "simple dual-stack",
			in:   "1.2.3.4,abcd::ef01",
			out: []net.IP{
				netutils.ParseIPSloppy("1.2.3.4"),
				netutils.ParseIPSloppy("abcd::ef01"),
			},
			ssErr: "not supported in this configuration",
		},
		{
			desc: "dual-stack with whitespace",
			in:   "abcd::ef01 , 1.2.3.4",
			out: []net.IP{
				netutils.ParseIPSloppy("abcd::ef01"),
				netutils.ParseIPSloppy("1.2.3.4"),
			},
			ssErr: "not supported in this configuration",
		},
		{
			desc: "double IPv4",
			in:   "1.2.3.4,5.6.7.8",
			err:  "either a single IP or a dual-stack pair of IPs",
		},
		{
			desc: "double IPv6",
			in:   "abcd::1,abcd::2",
			err:  "either a single IP or a dual-stack pair of IPs",
		},
		{
			desc:  "dual-stack with unspecified",
			in:    "1.2.3.4,::",
			err:   "cannot include '0.0.0.0' or '::'",
			ssErr: "not supported in this configuration",
		},
		{
			desc:  "dual-stack with unspecified",
			in:    "0.0.0.0,abcd::1",
			err:   "cannot include '0.0.0.0' or '::'",
			ssErr: "not supported in this configuration",
		},
		{
			desc: "dual-stack plus ignored garbage",
			in:   "abcd::ef01 , 1.2.3.4, something else",
			out: []net.IP{
				netutils.ParseIPSloppy("abcd::ef01"),
				netutils.ParseIPSloppy("1.2.3.4"),
			},
			ssErr: "not supported in this configuration",
		},
		{
			desc: "triple stack!",
			in:   "1.2.3.4,abcd::1,5.6.7.8",
			err:  "either a single IP or a dual-stack pair of IPs",
		},
	}

	configurations := []struct {
		cloudProvider       string
		allowCloudDualStack bool
		dualStackSupported  bool
	}{
		{cloudProviderNone, false, true},
		{cloudProviderNone, true, true},
		{cloudProviderExternal, false, false},
		{cloudProviderExternal, true, true},
		{"gce", false, false},
		{"gce", true, false},
	}

	for _, tc := range testCases {
		for _, conf := range configurations {
			desc := fmt.Sprintf("%s, cloudProvider=%q, allowCloudDualStack=%v", tc.desc, conf.cloudProvider, conf.allowCloudDualStack)
			t.Run(desc, func(t *testing.T) {
				parsed, err := ParseNodeIPArgument(tc.in, conf.cloudProvider, conf.allowCloudDualStack)

				expectedOut := tc.out
				expectedErr := tc.err

				if !conf.dualStackSupported {
					if len(tc.out) == 2 {
						expectedOut = nil
					}
					if tc.ssErr != "" {
						expectedErr = tc.ssErr
					}
				}

				if !reflect.DeepEqual(parsed, expectedOut) {
					t.Errorf("expected %#v, got %#v", expectedOut, parsed)
				}
				if err != nil {
					if expectedErr == "" {
						t.Errorf("unexpected error %v", err)
					} else if !strings.Contains(err.Error(), expectedErr) {
						t.Errorf("expected error with %q, got %v", expectedErr, err)
					}
				} else if expectedErr != "" {
					t.Errorf("expected error with %q, got no error", expectedErr)
				}
			})
		}
	}
}

func TestParseNodeIPAnnotation(t *testing.T) {
	testCases := []struct {
		desc string
		in   string
		out  net.IP
		err  string
	}{
		{
			desc: "empty --node-ip",
			in:   "",
			err:  "could not parse",
		},
		{
			desc: "just whitespace",
			in:   " ",
			err:  "could not parse",
		},
		{
			desc: "garbage",
			in:   "blah",
			err:  "could not parse",
		},
		{
			desc: "single IPv4",
			in:   "1.2.3.4",
			out:  netutils.ParseIPSloppy("1.2.3.4"),
		},
		{
			desc: "single IPv4 with whitespace",
			in:   " 1.2.3.4   ",
			err:  "could not parse",
		},
		{
			desc: "single IPv4 non-canonical",
			in:   "01.2.3.004",
			out:  netutils.ParseIPSloppy("1.2.3.4"),
		},
		{
			desc: "single IPv4 invalid",
			in:   "1.2.3",
			err:  "could not parse",
		},
		{
			desc: "single IPv4 CIDR",
			in:   "1.2.3.0/24",
			err:  "could not parse",
		},
		{
			desc: "single IPv4 unspecified",
			in:   "0.0.0.0",
			out:  net.IPv4zero,
		},
		{
			desc: "single IPv4 plus garbage",
			in:   "1.2.3.4,not-an-IPv6-address",
			err:  "could not parse",
		},
		{
			desc: "single IPv6",
			in:   "abcd::ef01",
			out:  netutils.ParseIPSloppy("abcd::ef01"),
		},
		{
			desc: "single IPv6 non-canonical",
			in:   "abcd:0abc:00ab:0000:0000::1",
			out:  netutils.ParseIPSloppy("abcd:abc:ab::1"),
		},
		{
			desc: "simple dual-stack",
			in:   "1.2.3.4,abcd::ef01",
			err:  "not supported in this configuration",
		},
		{
			desc: "dual-stack with whitespace",
			in:   "abcd::ef01 , 1.2.3.4",
			err:  "could not parse",
		},
		{
			desc: "double IPv4",
			in:   "1.2.3.4,5.6.7.8",
			err:  "either a single IP or a dual-stack pair of IPs",
		},
		{
			desc: "double IPv6",
			in:   "abcd::1,abcd::2",
			err:  "either a single IP or a dual-stack pair of IPs",
		},
		{
			desc: "dual-stack with unspecified",
			in:   "1.2.3.4,::",
			err:  "not supported in this configuration",
		},
		{
			desc: "dual-stack with unspecified",
			in:   "0.0.0.0,abcd::1",
			err:  "not supported in this configuration",
		},
		{
			desc: "dual-stack plus garbage",
			in:   "abcd::ef01 , 1.2.3.4, something else",
			err:  "could not parse",
		},
		{
			desc: "triple stack!",
			in:   "1.2.3.4,abcd::1,5.6.7.8",
			err:  "either a single IP or a dual-stack pair of IPs",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			parsed, err := ParseNodeIPAnnotation(tc.in)

			if !reflect.DeepEqual(parsed, tc.out) {
				t.Errorf("expected %#v, got %#v", tc.out, parsed)
			}
			if err != nil {
				if tc.err == "" {
					t.Errorf("unexpected error %v", err)
				} else if !strings.Contains(err.Error(), tc.err) {
					t.Errorf("expected error with %q, got %v", tc.err, err)
				}
			} else if tc.err != "" {
				t.Errorf("expected error with %q, got no error", tc.err)
			}
		})
	}
}