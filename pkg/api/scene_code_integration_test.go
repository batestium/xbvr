package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/simple"
	"github.com/emicklei/go-restful/v3"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
	"github.com/xbapps/xbvr/pkg/tasks"
)

// Use a subprocess so models' package initialization opens only a disposable DB.
func TestSceneCodeSearchIntegration(t *testing.T) {
	if os.Getenv("XBVR_CODE_SEARCH_TEST") != "1" {
		cmd := exec.Command(os.Args[0], "-test.run=^TestSceneCodeSearchIntegration$", "-test.v")
		appDir := t.TempDir()
		cmd.Env = append(os.Environ(), "XBVR_CODE_SEARCH_TEST=1", "XBVR_APPDIR="+appDir, "DATABASE_URL=sqlite:"+filepath.Join(appDir, "main.db"), "XBVR_SEARCHDIR="+filepath.Join(appDir, "search-v2"))
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("isolated API test: %v\n%s", err, out)
		} else {
			t.Log(string(out))
		}
		return
	}
	db, err := models.GetDB()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.AutoMigrate(&models.Scene{}, &models.Tag{}, &models.Actor{}, &models.File{}, &models.History{}, &models.SceneCuepoint{}).Error; err != nil {
		t.Fatal(err)
	}
	file := models.File{Filename: "abcd-503.mp4", VideoDuration: 82 * 60}
	if err := db.Create(&file).Error; err != nil {
		t.Fatal(err)
	}
	exact := models.Scene{SceneID: "stash-exact", Title: " abcd-503 ", Duration: 81}
	leadingZero := models.Scene{SceneID: "stash-zero", Title: "ABCD-0503"}
	byID := models.Scene{SceneID: " abcd-494 ", Title: "A descriptive title"}
	for _, scene := range []*models.Scene{&exact, &leadingZero, &byID} {
		if err := db.Create(scene).Error; err != nil {
			t.Fatal(err)
		}
	}
	// A legacy simple index has no numeric title tokens. Omit the exact scene
	// entirely to prove retrieval works before an index rebuild and beyond 25 hits.
	mapping := bleve.NewIndexMapping()
	doc := bleve.NewDocumentMapping()
	title := bleve.NewTextFieldMapping()
	title.Analyzer = simple.Name
	doc.AddFieldMappingsAt("title", title)
	mapping.AddDocumentMapping("_default", doc)
	idx, err := bleve.New(filepath.Join(common.IndexDirV2, "scenes"), mapping)
	if err != nil {
		t.Fatal(err)
	}
	for n := 100; n < 140; n++ {
		scene := models.Scene{SceneID: fmt.Sprintf("stash-%d", n), Title: fmt.Sprintf("ABCD-%d", n), Duration: 82}
		if err := db.Create(&scene).Error; err != nil {
			t.Fatal(err)
		}
		if err := idx.Index(scene.SceneID, tasks.SceneIndexed{Id: scene.SceneID, Title: scene.Title}); err != nil {
			t.Fatal(err)
		}
	}
	if err := idx.Close(); err != nil {
		t.Fatal(err)
	}
	container := restful.NewContainer()
	container.Add(SceneResource{}.WebService())
	search := func(t *testing.T, q string, matching bool) ResponseGetScenes {
		t.Helper()
		path := "/api/scene/search?q=" + url.QueryEscape(q)
		if matching {
			path += fmt.Sprintf("&fileId=%d", file.ID)
		}
		req := httptest.NewRequest(http.MethodGet, path, nil)
		recorder := httptest.NewRecorder()
		container.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusOK {
			t.Fatalf("status %d: %s", recorder.Code, recorder.Body.String())
		}
		var result ResponseGetScenes
		if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
			t.Fatalf("decode: %v; body: %s", err, recorder.Body.String())
		}
		return result
	}
	for _, tc := range []struct {
		q  string
		id uint
	}{{"abcd 503", exact.ID}, {"ABCD_0503", leadingZero.ID}, {"abcd-494", byID.ID}} {
		t.Run(tc.q, func(t *testing.T) {
			result := search(t, tc.q, true)
			if len(result.Scenes) == 0 || result.Scenes[0].ID != tc.id {
				t.Fatalf("exact scene %d not first: %+v", tc.id, result)
			}
			if len(result.Scenes) > 1 && result.Scenes[0].Score <= result.Scenes[1].Score {
				t.Fatal("duration boost outranked exact code")
			}
			seen := map[uint]bool{}
			for _, scene := range result.Scenes {
				if seen[scene.ID] {
					t.Fatalf("duplicate scene %d", scene.ID)
				}
				seen[scene.ID] = true
			}
			if tc.id == exact.ID && len(result.Scenes) != 26 {
				t.Fatalf("expected 25 text hits plus exact match, got %d", len(result.Scenes))
			}
			if tc.id == exact.ID && seen[leadingZero.ID] {
				t.Fatal("leading zero treated as insignificant")
			}
		})
	}
	for _, tc := range []struct {
		q        string
		matching bool
	}{{"abcd 503", false}, {"+title:abcd +title:503", true}, {"abcd 503 extra", true}} {
		result := search(t, tc.q, tc.matching)
		for _, scene := range result.Scenes {
			if scene.ID == exact.ID {
				t.Fatalf("exact match injected for q=%q matching=%v", tc.q, tc.matching)
			}
		}
	}
	// Once indexed as well, the exact hit must still appear only once.
	reopened, err := tasks.NewIndex("scenes")
	if err != nil {
		t.Fatal(err)
	}
	if err := reopened.Bleve.Index(exact.SceneID, tasks.SceneIndexed{Id: exact.SceneID, Title: exact.Title}); err != nil {
		t.Fatal(err)
	}
	if err := reopened.Bleve.Close(); err != nil {
		t.Fatal(err)
	}
	result := search(t, "abcd 503", true)
	count := 0
	for _, scene := range result.Scenes {
		if scene.ID == exact.ID {
			count++
		}
	}
	if count != 1 || result.Scenes[0].ID != exact.ID {
		t.Fatalf("expected exactly one first-place exact hit, got %d", count)
	}
}
