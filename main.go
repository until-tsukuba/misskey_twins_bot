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
	AccessToken string `json:"i"`
	Visibility string `json:"visibility"`
	VisibleUserIds []string `json:"visibleUserIds,omitempty"`
	Text string `json:"text"`
	CW *string `json:"cw,omitempty"`
	LocalOnly bool `json:"localOnly"`
	NoExtractMentions bool `json:"noExtractMentions"`
	NoExtractHashtags bool `json:"noExtractHashtags"`
	NoExtractEmojis bool `json:"noExtractEmojis"`
	Poll *interface{} `json:"poll,omitempty"`
	FileIds *[]string `json:"fileIds,omitempty"`
	MediaIds *[]string `json:"mediaIds,omitempty"`
	RenoteId *[]string `json:"renoteId,omitempty"`
	ChannelId *[]string `json:"channelId,omitempty"`
}

type IPayload struct {
	Id string `json:"id"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"UpdatedAt"`
}

type MisskeyConfig struct {
	AccessToken string `toml:"token"`
	URL string `toml:"url"`
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
	if err = json.NewEncoder(w).Encode(
			&struct{I string `json:"i"`}{I: config.Misskey.AccessToken},
		); err != nil {
			log.Fatalf("time err: %v\n", err)
			os.Exit(1)
			return
	}
	server_url.Path = "/api/i"
	client := new(http.Client)
	resp, err := client.Post(server_url.String(), "application/json", w)
	if err != nil {
		log.Fatalf("user data fetch error: %s\n", resp.Status)
	}
	body := new(IPayload)
	if err = json.NewDecoder(resp.Body).Decode(body); err != nil {
		log.Fatalf("time err: %v\n", err)
		os.Exit(1)
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

	last_update, err := time.Parse(time.RFC3339Nano, body.UpdatedAt)
	if err != nil {
		log.Fatalf("time.parse() last_update err: %v\n", err)
	}
	feed_updated, _ := time.Parse(time.RFC3339Nano, string(feed.Updated))
	if err != nil {
		log.Fatalf("time.parse() feed_updated err: %v\n", err)
	}

	if !last_update.Before(feed_updated) {
		os.Exit(0)
		return
	}

	for _, entry := range feed.Entry {
		feed_updated, _ = time.Parse(time.RFC3339Nano, string(entry.Updated))
		if !last_update.Before(feed_updated) {
			os.Exit(0)
			return
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
