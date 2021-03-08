package seloger

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"
)

const (
	testUser = "frederic.bidon@yahoo.com"
	testPwd  = "B6cAjvaC65TG28V"
)

func TestConnect(t *testing.T) {

	resp, err := Connect(testUser, testPwd)
	require.NoError(t, err)

	t.Logf("Credentials: %v", resp)
}

func TestGetListing(t *testing.T) {
	creds, err := Connect(testUser, testPwd)
	require.NoError(t, err)

	listings, err := GetListings(creds)
	require.NoError(t, err)

	spew.Dump(listings)
}
