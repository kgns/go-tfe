package tfe

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/google/go-querystring/query"
	"github.com/hashicorp/go-retryablehttp"
)

var _ AuditTrails = (*auditTrails)(nil)

type AuditTrails interface {
	Read(ctx context.Context, orgToken string, options *AuditTrailReadOptions) (*AuditTrailList, error)
}

type auditTrails struct {
	client *Client
}

type AuditTrailRequest struct {
	ID string `json:"id"`
}

type AuditTrailAuth struct {
	AccessorID     string  `json:"accessor_id"`
	Description    string  `json:"description"`
	Type           string  `json:"type"`
	ImpersonatorID *string `json:"impersonator_id"`
	OrganizationID string  `json:"organization_id"`
}

type AuditTrailResource struct {
	ID     string                  `json:"id"`
	Type   string                  `json:"type"`
	Action string                  `json:"create"`
	Meta   *map[string]interface{} `json:"meta"`
}

type AuditTrail struct {
	ID        string    `json:"id"`
	Version   string    `json:"version"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`

	Auth     *AuditTrailAuth     `json:"auth"`
	Request  *AuditTrailRequest  `json:"request"`
	Resource *AuditTrailResource `json:"resource"`
}

type AuditTrailList struct {
	*Pagination

	Items []*AuditTrail `json:"data"`
}

type AuditTrailReadOptions struct {
	Since time.Time `url:"since,omitempty"`
	*ListOptions
}

func (s *auditTrails) Read(ctx context.Context, orgToken string, options *AuditTrailReadOptions) (*AuditTrailList, error) {
	u, err := s.client.baseURL.Parse("/api/v2/organization/audit-trail")
	if err != nil {
		return nil, err
	}

	headers := make(http.Header)
	headers.Set("Authorization", "Bearer "+orgToken)
	headers.Set("Content-Type", "application/json")

	if options != nil {
		q, err := query.Values(options)
		if err != nil {
			return nil, err
		}

		u.RawQuery = encodeQueryParams(q)
	}

	req, err := retryablehttp.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	if err := s.client.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	resp, err := s.client.http.Do(req.WithContext(ctx))
	if err != nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			return nil, err
		}
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	atl := &AuditTrailList{}
	if err := json.Unmarshal(body, atl); err != nil {
		return nil, err
	}

	return atl, nil
}
