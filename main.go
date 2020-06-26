package main

import (
	"errors"
	"io/ioutil"
	"log"
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

func blockTitle(cache *notionapi.RecordMap, id string) (string, error) {
	b := cache.Blocks[id]
	for b != nil && b.Block != nil && b.Block.GetTitle() == nil {
		b = cache.Blocks[b.Block.ParentID]
	}
	if b != nil && b.Block != nil {
		return notionapi.TextSpansToString(b.Block.GetTitle()), nil
	} else {
		return "", errors.New("page not found")
	}
}

type Page struct {
	ID    string
	When  time.Time
	Who   string
	Event string
}

func readString(filename string) string {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("%v", err)
	}
	return strings.TrimSpace(string(b))
}

func main() {
	log.SetFlags(0)

	client := &notionapi.Client{
		AuthToken: readString(keyFile),
	}
	spaceID := readString(spaceFile)

	cache := NewCache(client, spaceID)
	cache.Load()
	cache.Update()
	cache.Save()

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
			ID:    nav,
			When:  when,
			Who:   email,
			Event: act.Type,
		}
		pages = append(pages, p)
		seen[nav] = struct{}{}
	}

	for _, p := range pages {
		nav := strings.ReplaceAll(p.ID, "-", "")
		title, err := blockTitle(&cache.RecordMap, p.ID)
		if err != nil {
			continue
		}
		log.Printf("%v %.10v %-10.10v %-17.17v %v", nav, p.When, p.Who, p.Event, title)
	}
}
