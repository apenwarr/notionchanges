package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/kjk/notionapi"
)

var keyFile = "notion.key"
var spaceFile = "space.id"

func lastEditor(cache *Cache, a *notionapi.Activity) (email string, lastEdit time.Time) {
	var last int64
	email = ""
	for _, e := range a.Edits {
		if e.Timestamp > last {
			last = e.Timestamp
			for _, auth := range e.Authors {
				u := cache.RecordMap.Users[auth.ID]
				if u != nil && u.User != nil {
					s := strings.Split(u.User.Email, "@")
					email = s[0]
				}
			}
		}
	}

	when := time.Unix(last/1000, (last%1000)*1000000)
	return email, when
}

func blockTitle(cache *Cache, id string) (string, error) {
	r := cache.RecordMap.Blocks[id]
	for r != nil {
		if r.Block != nil && r.Block.GetTitle() != nil {
			break
		}
		id = r.ID
		r = parentOf(cache, r)
	}
	if r == nil {
		return "", fmt.Errorf("no block object for %q", id)
	} else if r.Block == nil {
		return "", fmt.Errorf("no block sub-object for %q", id)
	} else if len(r.Block.GetTitle()) == 0 {
		return "", fmt.Errorf("title is empty for %q", id)
	} else {
		return notionapi.TextSpansToString(r.Block.GetTitle()), nil
	}
}

type Page struct {
	ID    string
	NavID string
	When  time.Time
	Who   string
	Event string

	// populated in second pass
	Permitted bool
	Title     string
	Path      []string
}

func (p *Page) URL() string {
	return fmt.Sprintf("https://notion.so/%s?__stamp=%d",
		strings.ReplaceAll(p.NavID, "-", ""),
		p.When.Unix())
}

func (p *Page) Date() string {
	return p.When.Format("2006-01-02")
}

func readString(filename string) string {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("%v", err)
	}
	return strings.TrimSpace(string(b))
}

func parentOf(cache *Cache, r *notionapi.Record) *notionapi.Record {
	pid := ""
	if r == nil {
		return nil
	} else if r.Block != nil {
		pid = r.Block.ParentID
	} else if r.Collection != nil {
		pid = r.Collection.ParentID
	} else if r.CollectionView != nil {
		pid = r.CollectionView.ParentID
	}
	if pid == "" {
		return nil
	}
	if x, ok := cache.RecordMap.Blocks[pid]; ok {
		return x
	}
	if x, ok := cache.RecordMap.Collections[pid]; ok {
		return x
	}
	if x, ok := cache.RecordMap.CollectionViews[pid]; ok {
		return x
	}
	if x, ok := cache.RecordMap.Spaces[pid]; ok {
		return x
	}
	return nil
}

func checkPermitted(cache *Cache, id string) bool {
	r := cache.RecordMap.Blocks[id]
	for r != nil {
		if r.Block != nil && !r.Block.Alive {
			return false
		} else if r.Collection != nil && !r.Collection.Alive {
			return false
		}
		if r.Block != nil && r.Block.Permissions != nil {
			for _, p := range *r.Block.Permissions {
				if p.Type == "space_permission" {
					// visible to everyone in workspace, so it's
					// ok to reveal in the log.
					return true
				}
			}
		}

		r = parentOf(cache, r)
	}

	return false
}

func collectHistory(cache *Cache) []Page {
	pages := []Page{}
	seen := map[string]struct{}{}

	for _, actid := range cache.ActivityIDs {
		act := cache.RecordMap.Activities[actid].Activity
		if act == nil {
			log.Printf("missing activity: %v", actid)
			continue
		}

		nav := ""
		if act.NavigableBlockID != "" {
			nav = act.NavigableBlockID
		} else if act.CollectionRowID != "" {
			nav = act.CollectionRowID
		} else if act.CollectionID != "" {
			nav = act.ParentID
		}

		if _, ok := seen[nav]; ok {
			// already seen, earlier entry wins
			continue
		}

		email, when := lastEditor(cache, act)
		p := Page{
			ID:    act.ParentID,
			NavID: nav,
			When:  when,
			Who:   email,
			Event: act.Type,
		}
		pages = append(pages, p)
		seen[nav] = struct{}{}
	}

	for i := range pages {
		p := &pages[i]

		title, err := blockTitle(cache, p.ID)
		if err != nil {
			p.Title = fmt.Sprintf("%s", err)
		} else {
			p.Title = title
		}

		p.Permitted = checkPermitted(cache, p.ID)
	}

	return pages
}

func main() {
	client := &notionapi.Client{
		AuthToken: readString(keyFile),
	}
	spaceID := readString(spaceFile)

	t, err := template.ParseFiles("main.html")
	if err != nil {
		log.Fatalf("main.html template: %v", err)
	}

	cache := NewCache(client, spaceID)
	cache.Load()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Refresh template so we don't have to recompile every time
		// while debugging.
		t, err = template.ParseFiles("main.html")
		if err != nil {
			log.Fatalf("main.html template: %v", err)
		}

		changed := cache.Update()
		if changed {
			cache.Save()
		}

		pages := collectHistory(cache)
		args := struct {
			Pages []Page
		}{
			Pages: pages,
		}

		t.Execute(w, args)
	})

	log.Fatal(http.ListenAndServe(":8187", nil))
}
