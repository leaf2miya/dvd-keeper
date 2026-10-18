package avwiki

import (
	"testing"
)

const searchResultHTML = `<!DOCTYPE html>
<html lang="ja">
<head><title>検索結果</title></head>
<body>
<main>
<div class="col-main clearfix">
<div class="archive-title"><h1><span>検索ワード：NHDTB-096<i class="fa fa-angle-right"></i>1件</span></h1></div>
<div class="column-flex clearfix">
<article class="archive-list">
<header class="archive-header">
<h2 class="archive-header-title"><a href="#">テスト作品タイトル</a></h2>
<ul class="post-meta clearfix">
<li class="actress-name"><i class="fa fa-venus"></i><a href="#" rel="tag">女優A</a></li>
<li><i class="fa fa-clone"></i><a href="#">メーカー - NHDTB</a></li>
<li><i class="fa fa-circle-o"></i>NHDTB-096</li>
</ul>
</header>
<ul class="link-wrapper">
<li class="link-btn"><a href="https://www.mgstage.com/product/product_detail/116NHDTB-096/?aff=TEST">MGS</a></li>
<li class="link-btn"><a href="https://al.fanza.co.jp/?lurl=https%3A%2F%2Fvideo.dmm.co.jp%2Fav%2Fcontent%2F%3Fid%3D1nhdtb00096%2F&af_id=test">FANZA</a></li>
<li class="link-btn"><a href="https://www.sokmil.com/av/_item/item282537.htm?affi=test">SOKMIL</a></li>
</ul>
</article>
</div>
</div>
</main>
</body>
</html>`

const noResultHTML = `<!DOCTYPE html>
<html lang="ja">
<head><title>検索結果</title></head>
<body>
<main>
<div class="col-main clearfix">
<div class="archive-title"><h1><span>検索ワード：hogehoge<i class="fa fa-angle-right"></i>0件</span></h1></div>
<p>検索ワードに一致する記事は見つかりませんでした。</p>
</div>
</main>
</body>
</html>`

const partialLinkHTML = `<!DOCTYPE html>
<html lang="ja">
<head><title>検索結果</title></head>
<body>
<main>
<div class="col-main clearfix">
<div class="archive-title"><h1><span>検索ワード：TEST-001<i class="fa fa-angle-right"></i>1件</span></h1></div>
<div class="column-flex clearfix">
<article class="archive-list">
<header class="archive-header">
<ul class="post-meta clearfix">
<li><i class="fa fa-circle-o"></i>TEST-001</li>
</ul>
</header>
<ul class="link-wrapper">
<li class="link-btn"><a href="https://al.fanza.co.jp/?lurl=https%3A%2F%2Fvideo.dmm.co.jp%2Fav%2Fcontent%2F%3Fid%3Dtest00001%2F&af_id=test">FANZA</a></li>
</ul>
</article>
</div>
</div>
</main>
</body>
</html>`

func TestParseSearchResult(t *testing.T) {
	tests := []struct {
		name             string
		html             string
		wantStatus       string
		wantError        string
		wantManufacturer string
		wantFanza        string
		wantMgs          string
		wantSokmil       string
	}{
		{
			name:             "正常系_全コード取得",
			html:             searchResultHTML,
			wantStatus:       "succeeded",
			wantManufacturer: "NHDTB-096",
			wantFanza:        "1nhdtb00096",
			wantMgs:          "116NHDTB-096",
			wantSokmil:       "item282537",
		},
		{
			name:       "異常系_検索結果0件",
			html:       noResultHTML,
			wantStatus: "failed",
			wantError:  "no search results",
		},
		{
			name:             "正常系_一部リンクのみ",
			html:             partialLinkHTML,
			wantStatus:       "succeeded",
			wantManufacturer: "TEST-001",
			wantFanza:        "test00001",
			wantMgs:          "",
			wantSokmil:       "",
		},
		{
			name:       "異常系_article要素なし",
			html:       `<!DOCTYPE html><html><body><main><p>空ページ</p></main></body></html>`,
			wantStatus: "failed",
			wantError:  "no article found in search results",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseSearchResult(tt.html)
			if err != nil {
				t.Fatalf("ParseSearchResult returned error: %v", err)
			}
			if result.Status != tt.wantStatus {
				t.Errorf("Status = %q, want %q", result.Status, tt.wantStatus)
			}
			if result.ErrorMessage != tt.wantError {
				t.Errorf("ErrorMessage = %q, want %q", result.ErrorMessage, tt.wantError)
			}
			if result.ManufacturerCode != tt.wantManufacturer {
				t.Errorf("ManufacturerCode = %q, want %q", result.ManufacturerCode, tt.wantManufacturer)
			}
			if result.FanzaCode != tt.wantFanza {
				t.Errorf("FanzaCode = %q, want %q", result.FanzaCode, tt.wantFanza)
			}
			if result.MgsCode != tt.wantMgs {
				t.Errorf("MgsCode = %q, want %q", result.MgsCode, tt.wantMgs)
			}
			if result.SokmilCode != tt.wantSokmil {
				t.Errorf("SokmilCode = %q, want %q", result.SokmilCode, tt.wantSokmil)
			}
		})
	}
}
