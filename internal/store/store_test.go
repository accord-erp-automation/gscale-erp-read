package store

import "testing"

func TestRankItemsMatchesXotLanchPhrase(t *testing.T) {
	items := []Item{
		{ItemCode: "xot lanch sochnaya kuritsa 90gr", ItemName: "xot lanch sochnaya kuritsa 90gr"},
		{ItemCode: "xot lanch sochnaya kuritsa ostriy 90gr", ItemName: "xot lanch sochnaya kuritsa ostriy 90gr"},
		{ItemCode: "Asl Sifat Hot Dog", ItemName: "Asl Sifat Hot Dog"},
	}

	got := rankItems(items, searchTerms("xot lanch"))
	if len(got) < 2 {
		t.Fatalf("expected at least 2 items, got %d", len(got))
	}
	if got[0].ItemCode != "xot lanch sochnaya kuritsa 90gr" {
		t.Fatalf("first item = %q", got[0].ItemCode)
	}
	if got[1].ItemCode != "xot lanch sochnaya kuritsa ostriy 90gr" {
		t.Fatalf("second item = %q", got[1].ItemCode)
	}
}

func TestRankItemsAvoidsNoisyShortMatches(t *testing.T) {
	items := []Item{
		{ItemCode: "Asl Sifat Hot Dog", ItemName: "Asl Sifat Hot Dog"},
		{ItemCode: "Asl sfat hot dog sosiski kuriniy", ItemName: "Asl sfat hot dog sosiski kuriniy"},
		{ItemCode: "elitex svitshot mujskoy zip paket", ItemName: "elitex svitshot mujskoy zip paket"},
	}

	got := rankItems(items, searchTerms("hot"))
	if len(got) != 2 {
		t.Fatalf("expected 2 items, got %d: %+v", len(got), got)
	}
	for _, item := range got {
		if item.ItemCode == "elitex svitshot mujskoy zip paket" {
			t.Fatalf("unexpected noisy match: %+v", item)
		}
	}
}
