// ローカルテスト用ファイル
// 以下のように実施する
// ```bash
// cd dmm-client
// DMM_AFFILIATE_ID=xxxx-990 DMM_API_ID=yyyy DMM_SITE="DMM.com" go run cmd/main.go
package main

import (
	"fmt"
	"os"

	dmmclient "github.com/leaf2miya/dvd-keeper/dmm-client"
)

func main() {
	affiliateID := os.Getenv("DMM_AFFILIATE_ID")
	apiID := os.Getenv("DMM_API_ID")
	site := os.Getenv("DMM_SITE")

	c, err := dmmclient.NewClient(affiliateID, apiID, site)
	if err != nil {
		fmt.Println("NewClient error:", err)
		return
	}

	resp, err := c.SearchItemList(dmmclient.SearchItemListParams{
		Keyword: "DVD",
		Length:  5,
	})
	if err != nil {
		fmt.Println("SearchItemList error:", err)
		return
	}

	fmt.Printf("総件数: %d\n", resp.TotalCount)
	for _, item := range resp.Items {
		fmt.Printf("- [%s] %s\n", item.ContentID, item.Title)
	}
}
