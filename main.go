package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/pelletier/go-toml/v2"
	"golang.org/x/tools/blog/atom"
)

type NotePayload struct {
	AccessToken       string       `json:"i"`
	Visibility        string       `json:"visibility"`
	VisibleUserIds    []string     `json:"visibleUserIds,omitempty"`
	Text              string       `json:"text"`
	CW                *string      `json:"cw,omitempty"`
	LocalOnly         bool         `json:"localOnly"`
	NoExtractMentions bool         `json:"noExtractMentions"`
	NoExtractHashtags bool         `json:"noExtractHashtags"`
	NoExtractEmojis   bool         `json:"noExtractEmojis"`
	Poll              *interface{} `json:"poll,omitempty"`
	FileIds           *[]string    `json:"fileIds,omitempty"`
	MediaIds          *[]string    `json:"mediaIds,omitempty"`
	RenoteId          *[]string    `json:"renoteId,omitempty"`
	ChannelId         *[]string    `json:"channelId,omitempty"`
}

type UsersNotesRequestPayload struct {
	UserId      string `json:"userId"`
	WithReplies bool   `json:"withReplies"`
	WithRenotes bool   `json:"withRenotes"`
	Limit       int    `json:"limit"`
}

type UsersNotesResponsePayload struct {
	Id        string `json:"id"`
	CreatedAt string `json:"createdAt"`
}

type MisskeyConfig struct {
	AccessToken string `toml:"token"`
	URL         string `toml:"url"`
	UserId      string `toml:"userId"`
}

type Config struct {
	Misskey MisskeyConfig `toml:"misskey"`
}

func main() {
	var (
		config_file_path = flag.String("config", "config.toml", "config file's path")
	)
	config_file, err := os.Open(*config_file_path)
	if err != nil {
		log.Fatalf("config error: %v\n", err)
	}
	config := new(Config)
	if err := toml.NewDecoder(config_file).Decode(config); err != nil {
		log.Fatalf("config error: %v\n", err)
	}
	server_url := new(url.URL)
	server_url.Host = config.Misskey.URL
	server_url.Scheme = "https"

	w := new(bytes.Buffer)
	userNotesReqPayload := &UsersNotesRequestPayload{
		UserId:      config.Misskey.UserId,
		WithReplies: false,
		WithRenotes: false,
		Limit:       1,
	}

	if err = json.NewEncoder(w).Encode(userNotesReqPayload); err != nil {
		log.Fatalf("time err: %v\n", err)
		return
	}
	server_url.Path = "/api/users/notes"
	client := new(http.Client)
	resp, err := client.Post(server_url.String(), "application/json", w)
	if err != nil {
		log.Fatalf("user data fetch error: %s\n", resp.Status)
		return
	}
	body := make([]UsersNotesResponsePayload, 1)
	if err = json.NewDecoder(resp.Body).Decode(&body); err != nil {
		log.Fatalf("decode err: %v\n", err)
		return
	}

	resp, err = client.Get("https://mkobayashime.github.io/twins-announcements/twins-announcements-atom1.xml")
	if err != nil {
		log.Fatalf("feed error: %s\n", resp.Status)
	}

	var feed atom.Feed
	if err = xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		log.Fatalf("feed read error: %v\n", err)
	}

	accountLastUpdate := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	if len(body) != 0 {
		accountLastUpdate, err = time.Parse(time.RFC3339, body[0].CreatedAt)
		if err != nil {
			log.Fatalf("time.parse() last_update err: %v\n", err)
		}
	}
	feed_updated, err := time.Parse(time.RFC3339Nano, string(feed.Updated))
	if err != nil {
		log.Fatalf("time.parse() feed_updated err: %v\n", err)
	}

	if !accountLastUpdate.Before(feed_updated) {
		log.Println("found no new article")
		return
	}

	var entryUpdated time.Time
	for _, entry := range feed.Entry {
		entryUpdated, err = time.Parse(time.RFC3339Nano, string(entry.Updated))
		if err == nil && !accountLastUpdate.Before(entryUpdated) {
			continue
		}
		payload := new(NotePayload)
		payload.Visibility = "public"
		payload.AccessToken = config.Misskey.AccessToken
		payload.Text = fmt.Sprintf("%s\n%s\n", entry.Title, entry.Link[0].Href)

		w := new(bytes.Buffer)
		if err != json.NewEncoder(w).Encode(payload) {
			log.Fatalf("time err: %v\n", err)
			os.Exit(1)
			return
		}
		server_url.Path = "/api/notes/create"
		resp, err := client.Post(server_url.String(), "application/json", w)
		if err != nil {
			log.Fatalf("misskey err: %v. id: %s\n", err, entry.ID)
			os.Exit(1)
			return
		}

		if resp.StatusCode != http.StatusOK {
			log.Fatalf("misskey err: %v. id: %s. status: %d\n", err, entry.ID, resp.StatusCode)
			os.Exit(1)
			return
		}
	}
}
