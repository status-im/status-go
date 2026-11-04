package dnsdisc

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestResolver(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resolver1 := GetResolver(ctx, "")
	require.Equal(t, net.DefaultResolver, resolver1)

	resolver2 := GetResolver(ctx, "1.1.1.1")
	require.NotEqual(t, net.DefaultResolver, resolver2)
	require.True(t, resolver2.PreferGo)

	cname, err := resolver2.LookupCNAME(ctx, "status.im")
	require.NoError(t, err)
	require.Equal(t, "status.im.", cname)
}
