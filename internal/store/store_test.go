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

func TestSearchTermsExpandHotLanchVariants(t *testing.T) {
	terms := searchTerms("hotlunch")
	want := []string{"hotlunch", "hotlanch", "xotlunch", "xotlanch"}
	for _, expected := range want {
		found := false
		for _, term := range terms {
			if term == expected {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected variant %q in %#v", expected, terms)
		}
	}
}

func TestRankItemsMatchesHotLanchAliases(t *testing.T) {
	items := []Item{
		{ItemCode: "xot lanch sochnaya kuritsa 90gr", ItemName: "xot lanch sochnaya kuritsa 90gr"},
		{ItemCode: "xot lanch sochnaya kuritsa ostriy 90gr", ItemName: "xot lanch sochnaya kuritsa ostriy 90gr"},
		{ItemCode: "Asl Sifat Hot Dog", ItemName: "Asl Sifat Hot Dog"},
	}

	for _, query := range []string{"hot lanch", "hotlanch", "hotlunch", "xot lunch", "hot launch", "xotlanch"} {
		got := rankItems(items, searchTerms(query))
		if len(got) < 2 {
			t.Fatalf("query %q expected at least 2 items, got %d", query, len(got))
		}
		if got[0].ItemCode != "xot lanch sochnaya kuritsa 90gr" {
			t.Fatalf("query %q first item = %q", query, got[0].ItemCode)
		}
	}
}
