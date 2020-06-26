package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/kjk/notionapi"
)

var keyFile = "notion.key"
var spaceFile = "space.id"
var cacheFile = "cache.json"

type Cache struct {
	client  *notionapi.Client
	spaceID string

	ActivityIDs []string
	RecordMap   notionapi.RecordMap
}

func NewCache(client *notionapi.Client, spaceID string) *Cache {
	rm := notionapi.RecordMap{
		Activities:      make(map[string]*notionapi.Record),
		Blocks:          make(map[string]*notionapi.Record),
		Spaces:          make(map[string]*notionapi.Record),
		Users:           make(map[string]*notionapi.Record),
		Collections:     make(map[string]*notionapi.Record),
		CollectionViews: make(map[string]*notionapi.Record),
		Comments:        make(map[string]*notionapi.Record),
		Discussions:     make(map[string]*notionapi.Record),
	}
	return &Cache{
		client:  client,
		spaceID: spaceID,

		ActivityIDs: []string{},
		RecordMap:   rm,
	}
}

func (c *Cache) Update() {
	next := ""
	n := 0

	nids := []string{}
	oids := c.ActivityIDs
	lookingFor := ""
	if len(oids) > 0 {
		lookingFor = oids[0]
	}

	limit := 1
retrieve:
	for n < 1000 {
		log.Printf("Retrieving %-5d %q", limit, next)
		acts, err := c.client.GetActivityLog(c.spaceID, next, limit)
		if err != nil {
			logf("GetActivityLog: %s\n", err)
		}

		mergeRecordMap(&c.RecordMap, acts.RecordMap)

		for _, aid := range acts.ActivityIDs {
			n++
			if aid == lookingFor {
				// caught up to old cached values
				nids = append(nids, oids...)
				break retrieve
			} else {
				nids = append(nids, aid)
			}
		}

		next = acts.NextID
		if next == "" {
			break
		}
		limit = 20
	}

	c.ActivityIDs = nids
}

func (c *Cache) Load() {
	b, err := ioutil.ReadFile(cacheFile)
	if err != nil {
		log.Printf("load cache: %v (ignored)", err)
		return
	}

	err = json.Unmarshal(b, c)
	if err != nil {
		log.Fatalf("unmarshal cache: %v", err)
	}

	err = notionapi.ParseRecordMap(&c.RecordMap)
	if err != nil {
		log.Fatalf("ParseRecordMap: %v", err)
	}
}

func (c *Cache) Save() {
	b, err := json.Marshal(c)
	if err != nil {
		log.Fatalf("cache marshal: %v", err)
	}
	newFile := cacheFile + ".new"
	ioutil.WriteFile(newFile, b, 0o666)
	err = os.Rename(newFile, cacheFile)
	if err != nil {
		log.Fatalf("rename %q -> %q: %v", newFile, cacheFile, err)
	}
}

func mergeRecords(into, from map[string]*notionapi.Record) {
	for k, v := range from {
		into[k] = v
	}
}

func mergeRecordMap(into, from *notionapi.RecordMap) {
	mergeRecords(into.Activities, from.Activities)
	mergeRecords(into.Blocks, from.Blocks)
	mergeRecords(into.Spaces, from.Spaces)
	mergeRecords(into.Users, from.Users)
	mergeRecords(into.Collections, from.Collections)
	mergeRecords(into.CollectionViews, from.CollectionViews)
	mergeRecords(into.Comments, from.Comments)
	mergeRecords(into.Discussions, from.Discussions)
}

func logf(fmt string, args ...interface{}) {
	log.Printf(fmt, args...)
}

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
