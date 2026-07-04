package content

import (
	"context"
	"encoding/xml"
	"html"
	"strings"
	"time"

	"github.com/rob121/cannon/internal/models"
)

type rssDoc struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel rssFeed  `xml:"channel"`
}

type rssFeed struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Language    string    `xml:"language,omitempty"`
	LastBuild   string    `xml:"lastBuildDate,omitempty"`
	Items       []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	GUID        string `xml:"guid"`
	PubDate     string `xml:"pubDate"`
	Author      string `xml:"author,omitempty"`
}

// BuildRSS renders an RSS 2.0 feed for the given items.
func BuildRSS(siteName, siteURL, feedTitle, feedLink, feedDesc string, items []models.Item) ([]byte, error) {
	if siteName == "" {
		siteName = "Cannon"
	}
	if feedTitle == "" {
		feedTitle = siteName
	}
	if feedLink == "" {
		feedLink = siteURL
	}
	if feedDesc == "" {
		feedDesc = "Latest content from " + siteName
	}
	base := strings.TrimRight(siteURL, "/")
	channel := rssFeed{
		Title:       feedTitle,
		Link:        feedLink,
		Description: feedDesc,
		Language:    "en-us",
		LastBuild:   time.Now().Format(time.RFC1123Z),
		Items:       make([]rssItem, 0, len(items)),
	}
	for _, item := range items {
		link := base + ItemURLForContext(WithLocale(context.Background(), item.Locale), item.Slug)
		desc := strings.TrimSpace(item.Intro)
		if desc == "" {
			desc = strings.TrimSpace(item.Body)
		}
		if len(desc) > 512 {
			desc = desc[:512] + "…"
		}
		author := ""
		if item.Author != nil {
			author = item.Author.Username
		}
		channel.Items = append(channel.Items, rssItem{
			Title:       item.Title,
			Link:        link,
			Description: html.EscapeString(desc),
			GUID:        link,
			PubDate:     item.CreatedAt.Format(time.RFC1123Z),
			Author:      author,
		})
	}
	doc := rssDoc{Version: "2.0", Channel: channel}
	raw, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), raw...), nil
}
