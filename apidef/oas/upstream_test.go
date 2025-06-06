package oas

import (
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/TykTechnologies/tyk/apidef"
	"github.com/TykTechnologies/tyk/internal/time"
)

func TestCacheOptions(t *testing.T) {
	t.Parallel()

	emptyCache := &ServiceDiscoveryCache{}
	enabledCache := &ServiceDiscoveryCache{
		Enabled: true,
		Timeout: 123,
	}

	var disabled bool
	enabled := !disabled

	testcases := []struct {
		title   string
		obj     *ServiceDiscovery
		timeout int64
		enabled bool
	}{
		{
			"new",
			&ServiceDiscovery{
				Enabled: true,
				Cache:   enabledCache,
			},
			123,
			enabled,
		},
		{
			"new and old",
			&ServiceDiscovery{
				Enabled:      true,
				CacheTimeout: 10,
				Cache:        enabledCache,
			},
			123,
			enabled,
		},
		{
			// This test case is particular to the behaviour of
			// timeouts; if the new cache config value is set but
			// is empty, we use that for the config. In practice,
			// removing cache options on encoding with omitempty
			// ensures that we rarely hit this case.
			"new disabled and old",
			&ServiceDiscovery{
				Enabled:      true,
				CacheTimeout: 10,
				Cache:        emptyCache,
			},
			0,
			disabled,
		},
		{
			"new nil and old",
			&ServiceDiscovery{
				Enabled:      true,
				CacheTimeout: 10,
			},
			10,
			enabled,
		},
		{
			"empty",
			&ServiceDiscovery{
				Enabled: true,
			},
			0,
			disabled,
		},
		{
			"nothing",
			&ServiceDiscovery{},
			0,
			disabled,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.title, func(t *testing.T) {
			timeout, ok := tc.obj.CacheOptions()
			assert.Equal(t, tc.timeout, timeout)
			assert.Equal(t, tc.enabled, ok)
		})
	}
}

func TestUpstream(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		var emptyUpstream Upstream

		var convertedAPI apidef.APIDefinition
		convertedAPI.SetDisabledFlags()
		emptyUpstream.ExtractTo(&convertedAPI)

		var resultUpstream Upstream
		resultUpstream.Fill(convertedAPI)

		assert.Equal(t, emptyUpstream, resultUpstream)
	})

	t.Run("rate limit", func(t *testing.T) {
		t.Run("valid duration", func(t *testing.T) {
			rateLimitUpstream := Upstream{
				RateLimit: &RateLimit{
					Enabled: true,
					Rate:    10,
					Per:     ReadableDuration(time.Hour + 20*time.Minute + 10*time.Second),
				},
			}

			var convertedAPI apidef.APIDefinition
			convertedAPI.SetDisabledFlags()
			rateLimitUpstream.ExtractTo(&convertedAPI)

			assert.Equal(t, float64(4810), convertedAPI.GlobalRateLimit.Per)

			var resultUpstream Upstream
			resultUpstream.Fill(convertedAPI)

			assert.Equal(t, rateLimitUpstream, resultUpstream)
		})

	})
}

func TestServiceDiscovery(t *testing.T) {
	var emptyServiceDiscovery ServiceDiscovery

	var convertedServiceDiscovery apidef.ServiceDiscoveryConfiguration
	emptyServiceDiscovery.ExtractTo(&convertedServiceDiscovery)

	var resultServiceDiscovery ServiceDiscovery
	resultServiceDiscovery.Fill(convertedServiceDiscovery)

	assert.Equal(t, emptyServiceDiscovery, resultServiceDiscovery)
}

func TestUptimeTests(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		var emptyTest UptimeTests

		var convertedTest apidef.UptimeTests
		emptyTest.ExtractTo(&convertedTest)

		var resultTest UptimeTests
		resultTest.Fill(convertedTest)

		assert.Equal(t, emptyTest, resultTest)
	})

	t.Run("filled & check timeout", func(t *testing.T) {
		var uptimeTests = UptimeTests{
			Enabled:          true,
			ServiceDiscovery: nil,
			Tests: []UptimeTest{
				{
					CheckURL: "http://test.com",
					Timeout:  ReadableDuration(time.Millisecond * 50),
					Method:   "POST",
				},
			},
		}

		var convertedTest apidef.UptimeTests

		uptimeTests.ExtractTo(&convertedTest)

		assert.Equal(t, time.Millisecond*50, convertedTest.CheckList[0].Timeout)
		assert.Equal(t, uptimeTests.Tests[0].CheckURL, convertedTest.CheckList[0].CheckURL)
		assert.Equal(t, uptimeTests.Tests[0].Method, convertedTest.CheckList[0].Method)
	})
}

func TestUpstreamMutualTLS(t *testing.T) {
	t.Parallel()
	t.Run("extractTo api definition", func(t *testing.T) {
		t.Parallel()
		testcases := []struct {
			title       string
			input       MutualTLS
			expectValue apidef.APIDefinition
		}{
			{
				"enabled=false, domain to certs nil",
				MutualTLS{Enabled: false, DomainToCertificates: nil},
				apidef.APIDefinition{
					UpstreamCertificatesDisabled: true,
				},
			},
			{
				"enabled=false, valid domain to cert mapping",
				MutualTLS{Enabled: false, DomainToCertificates: []DomainToCertificate{
					{Domain: "example.org", Certificate: "cert-1"},
					{Domain: "example.com", Certificate: "cert-2"},
				}},
				apidef.APIDefinition{
					UpstreamCertificatesDisabled: true,
					UpstreamCertificates: map[string]string{
						"example.org": "cert-1",
						"example.com": "cert-2",
					},
				},
			},
			{
				"enabled=true, valid domain to cert mapping",
				MutualTLS{Enabled: true, DomainToCertificates: []DomainToCertificate{
					{Domain: "example.org", Certificate: "cert-1"},
					{Domain: "example.com", Certificate: "cert-2"},
				}},
				apidef.APIDefinition{
					UpstreamCertificatesDisabled: false,
					UpstreamCertificates: map[string]string{
						"example.org": "cert-1",
						"example.com": "cert-2",
					},
				},
			},
			{
				"enabled=true, empty domain to cert mapping",
				MutualTLS{Enabled: true, DomainToCertificates: nil},
				apidef.APIDefinition{},
			},
		}

		for _, tc := range testcases {
			t.Run(tc.title, func(t *testing.T) {
				var apiDef apidef.APIDefinition

				tc.input.ExtractTo(&apiDef)

				assert.Equal(t, tc.expectValue, apiDef)
			})
		}
	})
	t.Run("fillFrom api definition", func(t *testing.T) {
		t.Parallel()
		testcases := []struct {
			title         string
			input         apidef.APIDefinition
			expectedValue MutualTLS
		}{
			{
				"disabled=false, empty domain to cert mapping",
				apidef.APIDefinition{UpstreamCertificatesDisabled: false},
				MutualTLS{
					Enabled: true,
				},
			},
			{
				"disabled=true, empty domain to cert mapping",
				apidef.APIDefinition{UpstreamCertificatesDisabled: true},
				MutualTLS{
					Enabled: false,
				},
			},
			{
				"disabled=false, valid domain to cert mapping",
				apidef.APIDefinition{UpstreamCertificatesDisabled: false,
					UpstreamCertificates: map[string]string{
						"example.org": "cert-1",
						"example.com": "cert-2",
					}},
				MutualTLS{
					Enabled: true,
					DomainToCertificates: []DomainToCertificate{
						{
							Domain:      "example.org",
							Certificate: "cert-1",
						},
						{
							Domain:      "example.com",
							Certificate: "cert-2",
						},
					},
				},
			},
			{
				"disabled=true, valid domain to cert mapping",
				apidef.APIDefinition{UpstreamCertificatesDisabled: true,
					UpstreamCertificates: map[string]string{
						"example.org": "cert-1",
						"example.com": "cert-2",
					}},
				MutualTLS{
					Enabled: false,
					DomainToCertificates: []DomainToCertificate{
						{
							Domain:      "example.org",
							Certificate: "cert-1",
						},
						{
							Domain:      "example.com",
							Certificate: "cert-2",
						},
					},
				},
			},
		}

		for _, tc := range testcases {
			t.Run(tc.title, func(t *testing.T) {
				var mutualTLS MutualTLS

				mutualTLS.Fill(tc.input)

				assert.Equal(t, tc.expectedValue.Enabled, mutualTLS.Enabled)
				assert.ElementsMatch(t, tc.expectedValue.DomainToCertificates, mutualTLS.DomainToCertificates)
			})
		}
	})
}

func TestPinnedPublicKeys(t *testing.T) {
	t.Parallel()

	var pinnedPublicKeys PinnedPublicKeys
	Fill(t, &pinnedPublicKeys, 0)

	convertedPinnedPublicKeys := make(map[string]string)
	pinnedPublicKeys.ExtractTo(convertedPinnedPublicKeys)

	resultPinnedPublicKeys := make(PinnedPublicKeys, len(pinnedPublicKeys))
	resultPinnedPublicKeys.Fill(convertedPinnedPublicKeys)

	assert.Equal(t, pinnedPublicKeys, resultPinnedPublicKeys)
}

func TestCertificatePinning(t *testing.T) {
	t.Run("extractTo api definition", func(t *testing.T) {
		testcases := []struct {
			title       string
			input       CertificatePinning
			expectValue apidef.APIDefinition
		}{
			{
				"enabled=false, domain to public keys nil",
				CertificatePinning{Enabled: false, DomainToPublicKeysMapping: nil},
				apidef.APIDefinition{
					CertificatePinningDisabled: true,
				},
			},
			{
				"enabled=false, valid domain to public keys mapping",
				CertificatePinning{Enabled: false, DomainToPublicKeysMapping: PinnedPublicKeys{
					{Domain: "example.org", PublicKeys: []string{"key-1", "key-2"}},
					{Domain: "example.com", PublicKeys: []string{"key-1", "key-2"}},
				}},
				apidef.APIDefinition{
					CertificatePinningDisabled: true,
					PinnedPublicKeys: map[string]string{
						"example.org": "key-1,key-2",
						"example.com": "key-1,key-2",
					},
				},
			},
			{
				"enabled=true, valid domain to public keys mapping",
				CertificatePinning{Enabled: true, DomainToPublicKeysMapping: PinnedPublicKeys{
					{Domain: "example.org", PublicKeys: []string{"key-1", "key-2"}},
					{Domain: "example.com", PublicKeys: []string{"key-1", "key-2"}},
				}},
				apidef.APIDefinition{
					CertificatePinningDisabled: false,
					PinnedPublicKeys: map[string]string{
						"example.org": "key-1,key-2",
						"example.com": "key-1,key-2",
					},
				},
			},
			{
				"enabled=true, empty domain to public keys mapping",
				CertificatePinning{Enabled: true, DomainToPublicKeysMapping: nil},
				apidef.APIDefinition{},
			},
		}

		for _, tc := range testcases {
			t.Run(tc.title, func(t *testing.T) {
				var apiDef apidef.APIDefinition

				tc.input.ExtractTo(&apiDef)

				assert.Equal(t, tc.expectValue, apiDef)
			})
		}
	})
	t.Run("fillFrom api definition", func(t *testing.T) {
		testcases := []struct {
			title         string
			input         apidef.APIDefinition
			expectedValue CertificatePinning
		}{
			{
				"disabled=false, empty domain to cert mapping",
				apidef.APIDefinition{CertificatePinningDisabled: false},
				CertificatePinning{
					Enabled: true,
				},
			},
			{
				"disabled=true, empty domain to cert mapping",
				apidef.APIDefinition{CertificatePinningDisabled: true},
				CertificatePinning{
					Enabled: false,
				},
			},
			{
				"disabled=false, valid domain to cert mapping",
				apidef.APIDefinition{CertificatePinningDisabled: false,
					PinnedPublicKeys: map[string]string{
						"example.org": "key-1,key-2",
						"example.com": "key-1,key-2",
					}},
				CertificatePinning{
					Enabled: true,
					DomainToPublicKeysMapping: PinnedPublicKeys{
						{
							Domain: "example.org",
							PublicKeys: []string{
								"key-1",
								"key-2",
							},
						},
						{
							Domain: "example.com",
							PublicKeys: []string{
								"key-1",
								"key-2",
							},
						},
					},
				},
			},
			{
				"disabled=true, valid domain to cert mapping",
				apidef.APIDefinition{CertificatePinningDisabled: true,
					PinnedPublicKeys: map[string]string{
						"example.org": "key-1,key-2",
						"example.com": "key-1,key-2",
					}},
				CertificatePinning{
					Enabled: false,
					DomainToPublicKeysMapping: PinnedPublicKeys{
						{
							Domain: "example.org",
							PublicKeys: []string{
								"key-1",
								"key-2",
							},
						},
						{
							Domain: "example.com",
							PublicKeys: []string{
								"key-1",
								"key-2",
							},
						},
					},
				},
			},
		}

		for _, tc := range testcases {
			t.Run(tc.title, func(t *testing.T) {
				var certificatePinning CertificatePinning

				certificatePinning.Fill(tc.input)

				assert.Equal(t, tc.expectedValue.Enabled, certificatePinning.Enabled)
				assert.ElementsMatch(t, tc.expectedValue.DomainToPublicKeysMapping, certificatePinning.DomainToPublicKeysMapping)
			})
		}
	})
	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		var emptyCertificatePinnning CertificatePinning

		var convertedAPI apidef.APIDefinition
		emptyCertificatePinnning.ExtractTo(&convertedAPI)

		var resultCertificatePinning CertificatePinning
		resultCertificatePinning.Fill(convertedAPI)

		assert.Equal(t, emptyCertificatePinnning, resultCertificatePinning)
	})
}

func TestUpstreamRequestSigning(t *testing.T) {
	t.Parallel()
	t.Run("fill", func(t *testing.T) {
		t.Parallel()
		testcases := []struct {
			title    string
			input    apidef.APIDefinition
			expected *UpstreamAuth
		}{
			{
				title: "request signing disabled and everything else is empty should omit",
				input: apidef.APIDefinition{
					RequestSigning: apidef.RequestSigningMeta{
						IsEnabled:       false,
						Secret:          "",
						KeyId:           "",
						Algorithm:       "",
						HeaderList:      nil,
						CertificateId:   "",
						SignatureHeader: "",
					},
				},
				expected: nil,
			},
			{
				title: "request signing enabled and values are set",
				input: apidef.APIDefinition{
					RequestSigning: apidef.RequestSigningMeta{
						IsEnabled:       true,
						Secret:          "secret",
						KeyId:           "key-1",
						Algorithm:       "hmac-sha256",
						HeaderList:      []string{"header1", "header2"},
						CertificateId:   "cert-1",
						SignatureHeader: "Signature",
					},
				},
				expected: &UpstreamAuth{
					RequestSigning: &UpstreamRequestSigning{
						Enabled:         true,
						SignatureHeader: "Signature",
						Algorithm:       "hmac-sha256",
						KeyID:           "key-1",
						Headers:         []string{"header1", "header2"},
						Secret:          "secret",
						CertificateID:   "cert-1",
					},
				},
			},
		}

		for _, tc := range testcases {
			tc := tc
			t.Run(tc.title, func(t *testing.T) {
				t.Parallel()

				g := new(Upstream)
				g.Fill(tc.input)

				assert.Equal(t, tc.expected, g.Authentication)
			})
		}
	})

	t.Run("extractTo", func(t *testing.T) {
		t.Parallel()

		testcases := []struct {
			title                  string
			input                  *UpstreamRequestSigning
			expectedRequestSigning apidef.RequestSigningMeta
		}{
			{
				title: "request signing disabled and everything else is empty",
				input: &UpstreamRequestSigning{
					Enabled:         false,
					SignatureHeader: "",
					Algorithm:       "",
					KeyID:           "",
					Headers:         nil,
					Secret:          "",
					CertificateID:   "",
				},
				expectedRequestSigning: apidef.RequestSigningMeta{
					IsEnabled:       false,
					Secret:          "",
					KeyId:           "",
					Algorithm:       "",
					HeaderList:      nil,
					CertificateId:   "",
					SignatureHeader: "",
				},
			},
			{
				title: "request signing enabled and values are set",
				input: &UpstreamRequestSigning{
					Enabled:         true,
					SignatureHeader: "Signature",
					Algorithm:       "hmac-sha256",
					KeyID:           "key-1",
					Headers:         []string{"header1", "header2"},
					Secret:          "secret",
					CertificateID:   "cert-1",
				},
				expectedRequestSigning: apidef.RequestSigningMeta{
					IsEnabled:       true,
					Secret:          "secret",
					KeyId:           "key-1",
					Algorithm:       "hmac-sha256",
					HeaderList:      []string{"header1", "header2"},
					CertificateId:   "cert-1",
					SignatureHeader: "Signature",
				},
			},
		}

		for _, tc := range testcases {
			tc := tc // Creating a new 'tc' scoped to the loop
			t.Run(tc.title, func(t *testing.T) {
				t.Parallel()

				g := new(Upstream)
				g.Authentication = &UpstreamAuth{
					RequestSigning: tc.input,
				}

				var apiDef apidef.APIDefinition
				apiDef.RequestSigning.HeaderList = []string{"headerOld1", "headerOld2"}
				g.ExtractTo(&apiDef)

				assert.Equal(t, tc.expectedRequestSigning, apiDef.RequestSigning)
			})
		}
	})
}

func TestTLSTransportProxy(t *testing.T) {
	t.Run("with tls settings", func(t *testing.T) {
		transport := TLSTransport{
			InsecureSkipVerify:   true,
			MinVersion:           "1.2",
			MaxVersion:           "1.3",
			ForceCommonNameCheck: true,
		}

		var convertedAPI apidef.APIDefinition
		var resultTransport TLSTransport

		convertedAPI.SetDisabledFlags()
		transport.ExtractTo(&convertedAPI)

		assert.Equal(t, transport.InsecureSkipVerify, convertedAPI.Proxy.Transport.SSLInsecureSkipVerify)
		assert.Equal(t, uint16(tls.VersionTLS12), convertedAPI.Proxy.Transport.SSLMinVersion)
		assert.Equal(t, uint16(tls.VersionTLS13), convertedAPI.Proxy.Transport.SSLMaxVersion)
		assert.Equal(t, transport.ForceCommonNameCheck, convertedAPI.Proxy.Transport.SSLForceCommonNameCheck)

		resultTransport.Fill(convertedAPI)

		assert.Equal(t, transport, resultTransport)
	})

	t.Run("emmpty tls settings", func(t *testing.T) {
		var emptyTlsTransport TLSTransport
		var convertedAPI apidef.APIDefinition
		var resultTransport TLSTransport

		convertedAPI.SetDisabledFlags()
		emptyTlsTransport.ExtractTo(&convertedAPI)
		resultTransport.Fill(convertedAPI)

		assert.Equal(t, emptyTlsTransport, resultTransport)
	})

	t.Run("proxy settings", func(t *testing.T) {
		proxyTransport := Proxy{
			URL: "proxy-url",
		}
		var convertedAPI apidef.APIDefinition
		var resultProxy Proxy

		convertedAPI.SetDisabledFlags()
		proxyTransport.ExtractTo(&convertedAPI)

		assert.Equal(t, "proxy-url", convertedAPI.Proxy.Transport.ProxyURL)

		resultProxy.Fill(convertedAPI)
		assert.Equal(t, proxyTransport, resultProxy)
	})
}

func TestLoadBalancing(t *testing.T) {
	t.Parallel()
	t.Run("fill", func(t *testing.T) {
		t.Parallel()
		testcases := []struct {
			title    string
			input    apidef.APIDefinition
			expected *LoadBalancing
		}{
			{
				title: "disable load balancing when targets list is empty",
				input: apidef.APIDefinition{
					Proxy: apidef.ProxyConfig{
						EnableLoadBalancing: true,
						Targets:             []string{},
					},
				},
				expected: nil,
			},
			{
				title: "load balancing disabled with filled target list",
				input: apidef.APIDefinition{
					Proxy: apidef.ProxyConfig{
						EnableLoadBalancing: false,
						Targets: []string{
							"http://upstream-one",
							"http://upstream-one",
							"http://upstream-one",
							"http://upstream-one",
							"http://upstream-one",
							"http://upstream-three",
							"http://upstream-three",
						},
					},
				},
				expected: &LoadBalancing{
					Enabled: false,
					Targets: []LoadBalancingTarget{
						{
							URL:    "http://upstream-one",
							Weight: 5,
						},
						{
							URL:    "http://upstream-three",
							Weight: 2,
						},
					},
				},
			},
			{
				title: "load balancing enabled with filled target list",
				input: apidef.APIDefinition{
					Proxy: apidef.ProxyConfig{
						EnableLoadBalancing: true,
						Targets: []string{
							"http://upstream-one",
							"http://upstream-one",
							"http://upstream-one",
							"http://upstream-one",
							"http://upstream-one",
							"http://upstream-three",
							"http://upstream-three",
						},
					},
				},
				expected: &LoadBalancing{
					Enabled: true,
					Targets: []LoadBalancingTarget{
						{
							URL:    "http://upstream-one",
							Weight: 5,
						},
						{
							URL:    "http://upstream-three",
							Weight: 2,
						},
					},
				},
			},
		}

		for _, tc := range testcases {
			tc := tc
			t.Run(tc.title, func(t *testing.T) {
				t.Parallel()

				g := new(Upstream)
				g.Fill(tc.input)

				assert.Equal(t, tc.expected, g.LoadBalancing)
			})
		}
	})

	t.Run("extractTo", func(t *testing.T) {
		t.Parallel()

		testcases := []struct {
			title           string
			input           *LoadBalancing
			expectedEnabled bool
			expectedTargets []string
		}{
			{
				title: "disable load balancing when targets list is empty",
				input: &LoadBalancing{
					Enabled: false,
					Targets: nil,
				},
				expectedEnabled: false,
				expectedTargets: nil,
			},
			{
				title: "load balancing disabled with filled target list",
				input: &LoadBalancing{
					Enabled: false,
					Targets: []LoadBalancingTarget{
						{
							URL:    "http://upstream-one",
							Weight: 5,
						},
						{
							URL:    "http://upstream-two",
							Weight: 0,
						},
						{
							URL:    "http://upstream-three",
							Weight: 2,
						},
					},
				},
				expectedEnabled: false,
				expectedTargets: []string{
					"http://upstream-one",
					"http://upstream-one",
					"http://upstream-one",
					"http://upstream-one",
					"http://upstream-one",
					"http://upstream-three",
					"http://upstream-three",
				},
			},
			{
				title: "load balancing enabled with filled target list",
				input: &LoadBalancing{
					Enabled: true,
					Targets: []LoadBalancingTarget{
						{
							URL:    "http://upstream-one",
							Weight: 5,
						},
						{
							URL:    "http://upstream-two",
							Weight: 0,
						},
						{
							URL:    "http://upstream-three",
							Weight: 2,
						},
					},
				},
				expectedEnabled: true,
				expectedTargets: []string{
					"http://upstream-one",
					"http://upstream-one",
					"http://upstream-one",
					"http://upstream-one",
					"http://upstream-one",
					"http://upstream-three",
					"http://upstream-three",
				},
			},
		}

		for _, tc := range testcases {
			tc := tc // Creating a new 'tc' scoped to the loop
			t.Run(tc.title, func(t *testing.T) {
				t.Parallel()

				g := new(Upstream)
				g.LoadBalancing = tc.input

				var apiDef apidef.APIDefinition
				apiDef.Proxy.Targets = []string{
					"http://old1.upstream.test",
					"http://old2.upstream.test",
					"http://old3.upstream.test",
				}
				g.ExtractTo(&apiDef)

				assert.Equal(t, tc.expectedEnabled, apiDef.Proxy.EnableLoadBalancing)
				assert.Equal(t, tc.expectedTargets, apiDef.Proxy.Targets)
			})
		}
	})
}

func TestPreserveHostHeader(t *testing.T) {
	t.Run("fill", func(t *testing.T) {
		type testCase struct {
			title    string
			input    apidef.APIDefinition
			expected *PreserveHostHeader
		}
		testCases := []testCase{
			{
				title: "preserve host header disabled",
				input: apidef.APIDefinition{
					Proxy: apidef.ProxyConfig{
						PreserveHostHeader: false,
					},
				},
				expected: nil,
			},
			{
				title: "preserve host header enabled",
				input: apidef.APIDefinition{
					Proxy: apidef.ProxyConfig{
						PreserveHostHeader: true,
					},
				},
				expected: &PreserveHostHeader{
					Enabled: true,
				},
			},
		}
		for _, tc := range testCases {
			tc := tc
			t.Run(tc.title, func(t *testing.T) {
				t.Parallel()

				g := new(Upstream)
				g.Fill(tc.input)

				assert.Equal(t, tc.expected, g.PreserveHostHeader)
			})
		}
	})

	t.Run("extractTo", func(t *testing.T) {
		type testCase struct {
			title           string
			input           *PreserveHostHeader
			expectedEnabled bool
		}
		testcases := []testCase{
			{
				title: "preserve host header disabled",
				input: &PreserveHostHeader{
					Enabled: false,
				},
				expectedEnabled: false,
			},
			{
				title: "preserve host header enabled",
				input: &PreserveHostHeader{
					Enabled: true,
				},
				expectedEnabled: true,
			},
		}

		for _, tc := range testcases {
			tc := tc // Creating a new 'tc' scoped to the loop
			t.Run(tc.title, func(t *testing.T) {
				g := new(Upstream)
				g.PreserveHostHeader = tc.input

				var apiDef apidef.APIDefinition
				apiDef.Proxy.PreserveHostHeader = true
				g.ExtractTo(&apiDef)

				assert.Equal(t, tc.expectedEnabled, apiDef.Proxy.PreserveHostHeader)
			})
		}
	})
}
