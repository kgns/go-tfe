//go:build integration
// +build integration

package tfe

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditTrailRead(t *testing.T) {
	skipIfFreeOnly(t)
	skipIfNoOrgToken(t)

	client := testClient(t)
	ctx := context.Background()

	orgToken := os.Getenv("TFE_ORGANIZATION_TOKEN")

	t.Run("with no specified timeframe", func(t *testing.T) {
		atl, err := client.AuditTrails.Read(ctx, orgToken, nil)
		require.NoError(t, err)

		assert.Greater(t, len(atl.Items), 0)

		log := atl.Items[0]
		assert.NotEmpty(t, log.ID)
		assert.NotEmpty(t, log.Timestamp)
		assert.NotEmpty(t, log.Type)
		assert.NotEmpty(t, log.Version)
		assert.NotNil(t, log.Resource)
		assert.NotNil(t, log.Auth)
		assert.NotNil(t, log.Request)

		t.Run("with resource deserialized correctly", func(t *testing.T) {
			assert.NotEmpty(t, log.Resource.ID)
			assert.NotEmpty(t, log.Resource.Type)
			assert.NotEmpty(t, log.Resource.Action)

			// we don't test against log.Resource.Meta since we don't know the nature
			// of the audit trail log we're testing against as it can be nil or contain a k-v map
		})

		t.Run("with auth deserialized correctly", func(t *testing.T) {
			assert.NotEmpty(t, log.Auth.AccessorID)
			assert.NotEmpty(t, log.Auth.Description)
			assert.NotEmpty(t, log.Auth.Type)
			assert.NotEmpty(t, log.Auth.OrganizationID)
		})

		t.Run("with request deserialized correctly", func(t *testing.T) {
			assert.NotEmpty(t, log.Request.ID)
		})
	})

	t.Run("using since query param", func(t *testing.T) {
		orgName := os.Getenv("TFE_ORGANIZATION")
		if orgName == "" {
			t.Skip("Skipping subtest that requires TFE_ORGANIZATION to have a value.")
		}

		since := time.Now()

		// Let's create an event that is sent to the audit log
		org := &Organization{
			Name: orgName,
		}
		_, wsCleanup := createWorkspace(t, client, org)
		t.Cleanup(wsCleanup)

		// wait for the event to be logged
		time.Sleep(1 * time.Second)

		atl, err := client.AuditTrails.Read(ctx, orgToken, &AuditTrailReadOptions{
			Since: since,
			ListOptions: &ListOptions{
				PageNumber: 1,
				PageSize:   20,
			},
		})
		require.NoError(t, err)

		assert.LessOrEqual(t, len(atl.Items), 20)

		for _, log := range atl.Items {
			assert.True(t, log.Timestamp.After(since))
		}
	})
}
