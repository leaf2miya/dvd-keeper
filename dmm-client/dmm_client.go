package dmmclient

import (
	"fmt"

	"github.com/dmmlabo/dmm-go-sdk/api"
)

// Client は DMM 商品情報 API クライアントです。
type Client struct {
	affiliateID string
	apiID       string
	site        string
}

// SearchItemListParams は SearchItemList の任意検索パラメータです。
type SearchItemListParams struct {
	Keyword   string
	Sort      string
	Length    int64
	Offset    int64
	Service   string
	Floor     string
	Article   string
	ArticleID string
	GteDate   string
	LteDate   string
	Stock     string
}

// NewClient は affiliateID, apiID, site を受け取り Client を返します。
// site には "DMM.com" または "DMM.R18" を指定してください。
func NewClient(affiliateID, apiID, site string) (*Client, error) {
	if !api.ValidateAffiliateID(affiliateID) {
		return nil, fmt.Errorf("invalid affiliateID: %s", affiliateID)
	}
	if apiID == "" {
		return nil, fmt.Errorf("apiID must not be empty")
	}
	if !api.ValidateSite(site) {
		return nil, fmt.Errorf("invalid site: %s (must be %q or %q)", site, api.SiteGeneral, api.SiteAdult)
	}
	return &Client{affiliateID: affiliateID, apiID: apiID, site: site}, nil
}

// SearchItemList は params を元に DMM 商品一覧を検索し、結果を返します。
func (c *Client) SearchItemList(params SearchItemListParams) (*api.ProductResponse, error) {
	svc := api.NewProductService(c.affiliateID, c.apiID)
	svc.Site = c.site
	svc.Keyword = params.Keyword
	svc.Sort = params.Sort
	svc.Length = params.Length
	svc.Offset = params.Offset
	svc.Service = params.Service
	svc.Floor = params.Floor
	svc.Article = params.Article
	svc.ArticleID = params.ArticleID
	svc.GteDate = params.GteDate
	svc.LteDate = params.LteDate
	svc.Stock = params.Stock
	return svc.Execute()
}
