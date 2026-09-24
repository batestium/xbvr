package api

import (
	"testing"

	"github.com/xbapps/xbvr/pkg/models"
)

func TestSceneCodeQuery(t *testing.T) {
	for input, want := range map[string]string{
		"abcd 503": "ABCD-503", "ABCD-494": "ABCD-494",
		" abcd_0494 ": "ABCD-0494", "abcd  503": "ABCD-503",
		"+id:abcd +id:494": "", "abcd 503 duration:81": "",
		"abcd 503 extra": "", "abcd": "", "": "",
	} {
		if got := sceneCodeQuery(input); got != want {
			t.Errorf("sceneCodeQuery(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestExactSceneOutranksDurationAndSurvivesCandidateLimit(t *testing.T) {
	// Regression: unrelated 82-minute scenes outranked ABCD-503 at 81 minutes.
	candidates := []models.Scene{
		{ID: 1, Title: "ABCD-373", Score: 1.1190843766507619},
		{ID: 2, Title: "ABCD-375", Score: 1.117652117787986},
		{ID: 3, Title: "ABCD-503", Score: 0.9539321499840989},
	}
	exact := []models.Scene{{ID: 3, Title: "ABCD-503"}}
	got := prioritizeExactScenes(candidates, exact)
	if len(got) != 3 || got[0].ID != 3 || got[0].Score <= got[1].Score {
		t.Fatalf("exact match must be first without a duplicate: %+v", got)
	}
	// The correct scene must also appear when absent from the 25 text hits.
	candidates = make([]models.Scene, 25)
	for i := range candidates {
		candidates[i] = models.Scene{ID: uint(i + 10), Score: 1.2}
	}
	got = prioritizeExactScenes(candidates, exact)
	if len(got) != 26 || got[0].ID != 3 || got[0].Score <= got[1].Score {
		t.Fatalf("exact match outside the text limit was lost: %+v", got)
	}
}
