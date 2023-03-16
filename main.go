package main

import (
	"encoding/json"
	"encoding/xml"
	"log"
	"io"
	"net/http"
	"os"
	"time"
	"bytes"
	"fmt"

	"golang.org/x/tools/blog/atom"
	"github.com/pelletier/go-toml/v2"
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

type MisskeyConfig struct {
	AccessToken string `toml:"token"`
	URL string `toml:"url"`
}

type Config struct {
	Misskey MisskeyConfig `toml:"misskey"`
}

func main() {
	config_file, err := os.OpenFile("./config.toml", os.O_RDONLY, 0444)
	if err != nil {
		log.Printf("config error: %v\n", err)
	}
	config := new(Config)
	if err := toml.NewDecoder(config_file).Decode(config); err != nil {
		log.Printf("config error: %v\n", err)
	}

	client := new(http.Client)
	resp, err := client.Get("https://mkobayashime.github.io/twins-announcements/twins-announcements-atom1.xml")
	if err != nil {
		log.Printf("feed error: %s\n", resp.Status)
	}

	var feed atom.Feed
	if err = xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		log.Printf("feed read error: %v\n", err)
	}

	f, err := os.OpenFile("./.cache/lastupdate", os.O_RDWR | os.O_CREATE, 0664)
	if err != nil {
		log.Printf("file err: %v\n", err)
	}
	defer f.Close()
	data_str, _ := io.ReadAll(f)
	data_str = data_str[:len(data_str)-1]

	last_update, err := time.Parse(time.RFC3339Nano, string(data_str))
	if err != nil {
		log.Printf("time.parse() last_update err: %v\n", err)
	}
	feed_updated, _ := time.Parse(time.RFC3339Nano, string(feed.Updated))
	if err != nil {
		log.Printf("time.parse() feed_updated err: %v\n", err)
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
			log.Printf("time err: %v\n", err)
			os.Exit(1)
			return
		}
		req, _ := http.NewRequest(http.MethodPost, "https://misskey.until.tsukuba.one/api/notes/create", w)
		req.Header.Add("Content-Type", "application/json")
		log.Println(w.String())
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("misskey err: %v. id: %s\n", err, entry.ID)
			os.Exit(1)
			return
		}

		if resp.StatusCode != http.StatusOK {
			log.Print("misskey err: %v. id: %s. status: %d\n", err, entry.ID, resp.StatusCode)
			os.Exit(1)
			return
		}
	}
	f.Truncate(0)
	f.WriteString(
		fmt.Sprintf("%s\n", time.Now().Format(time.RFC3339Nano)),
	)
}
