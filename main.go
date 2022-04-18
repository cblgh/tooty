package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"encoding/json"
	"tooty/util"

	"github.com/McKael/madon"
)

type Credentials struct {
	Key	string `json:"key"`
	Secret string `json:"secret"`
	Token string `json:"token"`
}

func prepareClient (credpath string) madon.Client {
	b, err := os.ReadFile(credpath)
	util.Check(err, "read credentials file")

	var credentials Credentials
	err = json.Unmarshal(b, &credentials)
	util.Check(err, "unmarshal credentials")

	token := madon.UserToken{AccessToken: credentials.Token}
	client := madon.Client{Name: "tooty", ID: credentials.Key, Secret: credentials.Secret, APIBase: "https://merveilles.town/api/v1/", InstanceURL: "https://merveilles.town", UserToken: &token}
	return client
}

type Post struct {
	text []string
	visibility string
	media []Media
	mediaID []int64
	replyID int64
}

type Media struct {
	path string
	description string
}

func (p Post) IsReplying() bool {
	return p.replyID > 0
}

func (p Post) HasMedia() bool {
	return len(p.media) > 0
}

func (p *Post) DefaultVisibility() {
	if (p.visibility == "") {
		p.visibility = "public"
	}
}

var headerPattern = regexp.MustCompile(`\+\s?(\w+):\s?(.*)`)
func (p *Post) HandleHeader (line string) {
	match := headerPattern.FindStringSubmatch(line)
	if len(match) < 3 {
		return
	}
	header := match[1]
	value := match[2]
	switch (header) {
	case "reply":
		v, err := strconv.ParseInt(value, 10, 64)
		util.Check(err, "convert replyid to int64")
		p.replyID = v
	case "mode":
		p.visibility = value
	case "media":
		parts := strings.Split(value, ";")
		p.media = append(p.media, Media{path: parts[0], description: parts[1]})
	}
}

func (p *Post) UploadMedia (client *madon.Client) {
	if len(p.media) == 0 {
		return
	}
	for _, media := range p.media {
		attachment, err := client.UploadMedia(media.path, media.description, "0,0")
		util.Check(err, "upload media (%s)", media.path)
		p.mediaID = append(p.mediaID, attachment.ID)
	}
}

func (p Post) Send (client *madon.Client) {
	p.DefaultVisibility()
	p.UploadMedia(client)
	contents := strings.TrimSpace(strings.Join(p.text, "\n"))
	status, err := client.PostStatus(contents, p.replyID, p.mediaID, false, "", p.visibility)
	util.Check(err, "post status to mastodon via madon")
	fmt.Printf("posted (%d)\n", status.ID)
	if p.IsReplying() {
		fmt.Printf("in reply to %d\n", p.replyID)
	}
	fmt.Println(contents)
	if p.HasMedia() {
		fmt.Println("attachments:", p.media)
	}
}

func parsePosts (postpath string) []Post {
	var result []Post
	b, err := os.ReadFile(postpath)
	util.Check(err, "read posts file")

	contents := strings.TrimSpace(string(b))
	if len(contents) == 0 {
		return result
	}
	posts := strings.Split(contents, "---")
	for _, post := range posts {
		p := Post{}
		var headersFinished bool
		for _, line := range strings.Split(post, "\n") {
			if strings.HasPrefix(line, "+") && !headersFinished {
				p.HandleHeader(line)
			} else {
				if (!headersFinished) {
					headersFinished = true
				}
				p.text = append(p.text, line)
			}
		}
		result = append(result, p)
	}

	return result
}

func clearPosts(postpath string) {
	err := os.WriteFile(postpath, []byte(""), 0666)
	util.Check(err, "clear posts file")
}

func main () {
	postpath := "./example.txt"
	fmt.Println(postpath)
	client := prepareClient("./creds.json")
	posts := parsePosts(postpath)
	for _, post := range posts {
		post.Send(&client)
	}
	clearPosts(postpath)
}


