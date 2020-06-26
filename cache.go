package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"

	"github.com/apenwarr/notionapi"
)

const cacheFile = "cache.json"

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

func (c *Cache) Update() (changed bool) {
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
			log.Printf("GetActivityLog: %s\n", err)
		}

		mergeRecordMap(&c.RecordMap, acts.RecordMap)

		for _, aid := range acts.ActivityIDs {
			if aid == lookingFor {
				// caught up to old cached values
				nids = append(nids, oids...)
				break retrieve
			} else {
				nids = append(nids, aid)
			}
			n++
		}

		next = acts.NextID
		if next == "" {
			break
		}
		limit = 20
	}

	c.ActivityIDs = nids
	return n > 0
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
