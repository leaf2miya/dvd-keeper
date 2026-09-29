package dmmclient

import (
	"testing"

	"github.com/dmmlabo/dmm-go-sdk/api"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		affiliateID string
		apiID       string
		site        string
		wantErr     bool
	}{
		{
			name:        "valid general site",
			affiliateID: "dummy-990",
			apiID:       "test-api-id",
			site:        api.SiteGeneral,
			wantErr:     false,
		},
		{
			name:        "valid adult site",
			affiliateID: "dummy-990",
			apiID:       "test-api-id",
			site:        api.SiteAdult,
			wantErr:     false,
		},
		{
			name:        "invalid affiliateID",
			affiliateID: "invalid",
			apiID:       "test-api-id",
			site:        api.SiteGeneral,
			wantErr:     true,
		},
		{
			name:        "empty apiID",
			affiliateID: "dummy-990",
			apiID:       "",
			site:        api.SiteGeneral,
			wantErr:     true,
		},
		{
			name:        "invalid site",
			affiliateID: "dummy-990",
			apiID:       "test-api-id",
			site:        "invalid-site",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewClient(tt.affiliateID, tt.apiID, tt.site)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if c.affiliateID != tt.affiliateID {
				t.Errorf("affiliateID = %s, want %s", c.affiliateID, tt.affiliateID)
			}
			if c.apiID != tt.apiID {
				t.Errorf("apiID = %s, want %s", c.apiID, tt.apiID)
			}
			if c.site != tt.site {
				t.Errorf("site = %s, want %s", c.site, tt.site)
			}
		})
	}
}

func TestSearchItemList_BuildsRequest(t *testing.T) {
	c, err := NewClient("dummy-990", "test-api-id", api.SiteGeneral)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	params := SearchItemListParams{
		Keyword: "test",
		Length:  10,
		Offset:  1,
	}

	// 実 API を呼ばずにリクエスト URL が正常に構築できることを検証する
	svc := api.NewProductService(c.affiliateID, c.apiID)
	svc.Site      = c.site
	svc.Keyword   = params.Keyword
	svc.Length    = params.Length
	svc.Offset    = params.Offset

	url, err := svc.BuildRequestURL()
	if err != nil {
		t.Errorf("BuildRequestURL failed: %v", err)
	}
	if url == "" {
		t.Errorf("expected non-empty URL")
	}
}
