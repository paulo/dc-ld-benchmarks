package main

import (
	"os"
	"testing"
	"time"

	devcycle "github.com/devcyclehq/go-server-sdk/v2"
	"github.com/stretchr/testify/require"
	"gopkg.in/launchdarkly/go-sdk-common.v2/lduser"
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldvalue"
	ld "gopkg.in/launchdarkly/go-server-sdk.v5"
)

func BenchmarkDCvsLD(b *testing.B) {
	dcClient, dcUser := dcSetup(b)
	ldClient, ldUser := ldSetup(b)

	b.ResetTimer()
	b.Run("devcycle", func(b *testing.B) {
		b.ReportAllocs()
		var (
			val devcycle.Variable
			err error
		)

		for i := 0; i < b.N; i++ {
			val, err = dcClient.Variable(dcUser, "basic-boolean", false)
		}
		require.False(b, val.Value.(bool))
		require.NoError(b, err)
	})

	b.Run("devcycle-parallel", func(b *testing.B) {
		b.SetParallelism(1000)
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			var (
				val devcycle.Variable
				err error
				ran bool
			)
			for pb.Next() {
				ran = true
				val, err = dcClient.Variable(dcUser, "basic-boolean", false)
			}

			if ran {
				require.False(b, val.Value.(bool))
				require.NoError(b, err)
			}
		})
	})

	b.Run("current provider", func(b *testing.B) {
		b.ReportAllocs()
		var (
			val bool
			ran bool
			err error
		)

		for i := 0; i < b.N; i++ {
			ran = true
			val, err = ldClient.BoolVariation("cb-test-flag-2", ldUser, false)
			require.NoError(b, err)
		}

		if ran {
			require.False(b, val)
		}
	})

	b.Run("current-provider-parallel", func(b *testing.B) {
		b.SetParallelism(1000)
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			var (
				val bool
				err error
			)
			for pb.Next() {
				val, err = ldClient.BoolVariation("cb-test-flag-2", ldUser, false)
			}
			require.False(b, val)
			require.NoError(b, err)
		})
	})
}

func dcSetup(b *testing.B) (*devcycle.DVCClient, devcycle.DVCUser) {
	devOpts := &devcycle.DVCOptions{
		EnableEdgeDB:                 false,
		EnableCloudBucketing:         false,
		RequestTimeout:               time.Second * 2,
		DisableAutomaticEventLogging: false,
		DisableCustomEventLogging:    true,
		EventFlushIntervalMS:         time.Second * 30,
		ConfigPollingIntervalMS:      time.Second * 10,
		FlushEventQueueSize:          0,
		MaxEventQueueSize:            0,
	}

	dcKey := os.Getenv("DC_KEY")
	if dcKey == "" {
		b.Errorf("DC_KEY not set")
	}

	dc, err := devcycle.NewDVCClient(dcKey, devOpts)
	require.NoError(b, err)

	return dc, devcycle.DVCUser{UserId: "dontcare"}
}

func ldSetup(b *testing.B) (*ld.LDClient, lduser.User) {
	ldKey := os.Getenv("LD_KEY")
	if ldKey == "" {
		b.Errorf("LD_KEY not set")
	}

	c := ld.Config{Offline: false}

	ld, err := ld.MakeCustomClient(ldKey, c, 10*time.Second)
	if err != nil {
		b.Error(err)
	}

	defaultUserAttrs := map[string]string{
		"env":        "prod",
		"cdn_group":  "regular",
		"cdn_domain": "www.bitballoon.com",
		"cdn_dc":     "jfk",
		"cdn_host":   "cdn-reg-do-jfk-1",
		"node_name":  "cdn-reg-do-jfk-1",
	}

	attrs := ldvalue.ValueMapBuildWithCapacity(len(defaultUserAttrs))
	for k, v := range defaultUserAttrs {
		attrs.Set(k, ldvalue.String(v))
	}
	defaultAttrs := attrs.Build()

	ub := lduser.NewUserBuilder("dontcare")
	ub.CustomAll(defaultAttrs)

	return ld, ub.Build()
}
