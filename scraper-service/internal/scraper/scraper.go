package scraper

import (
	"github.com/mmcdole/gofeed"
)

type Article struct {
	Title       string
	Description string
	Link        string
	Published   string
}

func ScrapeFeed(url string) ([]Article, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(url)
	if err != nil {
		return nil, err
	}

	var articles []Article
	for _, item := range feed.Items {
		articles = append(articles, Article{
			Title:       item.Title,
			Description: item.Description,
			Link:        item.Link,
			Published:   item.Published,
		})
	}
	return articles, nil
}
