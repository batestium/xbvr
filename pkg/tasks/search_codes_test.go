package tasks

import (
	"fmt"
	"testing"

	"github.com/blevesearch/bleve/v2"
	"github.com/xbapps/xbvr/pkg/common"
)

func TestTitleCodeNumbersAreSearchable(t *testing.T) {
	oldDir := common.IndexDirV2
	common.IndexDirV2 = t.TempDir()
	defer func() { common.IndexDirV2 = oldDir }()
	idx, err := NewIndex("codes")
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Bleve.Close()
	// More than one page of same-prefix titles must not hide the right number.
	for n := 100; n < 140; n++ {
		id := fmt.Sprintf("stash-unrelated-%d", n)
		if err := idx.Bleve.Index(id, SceneIndexed{Id: id, Title: fmt.Sprintf("ABCD-%d", n)}); err != nil {
			t.Fatal(err)
		}
	}
	for id, title := range map[string]string{
		"stash-aaaaaaaa": "ABCD-494", "stash-bbbbbbbb": "ABCD-495",
		"stash-cccccccc": "ABCD-503",
	} {
		if err := idx.Bleve.Index(id, SceneIndexed{Id: id, Title: title}); err != nil {
			t.Fatal(err)
		}
	}
	for _, text := range []string{"+title:abcd +title:494", "+abcd +494"} {
		result, err := idx.Bleve.Search(bleve.NewSearchRequest(bleve.NewQueryStringQuery(text)))
		if err != nil {
			t.Fatal(err)
		}
		if result.Total != 1 || result.Hits[0].ID != "stash-aaaaaaaa" {
			t.Fatalf("query %q did not distinguish numeric codes: %+v", text, result)
		}
	}
}

func TestTitleAnalyzerPreservesPunctuationSearch(t *testing.T) {
	oldDir := common.IndexDirV2
	common.IndexDirV2 = t.TempDir()
	defer func() { common.IndexDirV2 = oldDir }()
	idx, err := NewIndex("punctuation")
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Bleve.Close()
	if err := idx.Bleve.Index("stash-punctuation", SceneIndexed{Title: "Sister's Visit 2"}); err != nil {
		t.Fatal(err)
	}
	// The API normalizes apostrophes to spaces; retain that behavior (#2229).
	result, err := idx.Bleve.Search(bleve.NewSearchRequest(bleve.NewQueryStringQuery(`+title:sister +title:s +title:visit +title:2`)))
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 1 {
		t.Fatalf("apostrophe-separated title tokens were lost: %+v", result)
	}
}
