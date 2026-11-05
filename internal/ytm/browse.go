package ytm

import (
	"context"

	"chordly/internal/model"
)

// HomeFeedResult contains parsed live content from the YouTube Music Home page.
type HomeFeedResult struct {
	HeroTrack *model.Track
	Mixes     []model.MixCard
	Tracks    []model.Track
}

// GetHomeFeed fetches and parses live trending and recommendation tracks from YouTube Music.
func (c *Client) GetHomeFeed(ctx context.Context) (*HomeFeedResult, error) {
	payload := RequestPayload{
		BrowseID: "FEmusic_home",
	}

	var raw map[string]interface{}
	if err := c.Do(ctx, "browse", payload, &raw); err != nil {
		return nil, err
	}

	return parseHomeFeed(raw), nil
}

func parseHomeFeed(raw map[string]interface{}) *HomeFeedResult {
	res := &HomeFeedResult{}

	contents, ok := raw["contents"].(map[string]interface{})
	if !ok {
		return res
	}

	singleColumn, ok := contents["singleColumnBrowseResultsRenderer"].(map[string]interface{})
	if !ok {
		return res
	}

	tabs, ok := singleColumn["tabs"].([]interface{})
	if !ok || len(tabs) == 0 {
		return res
	}

	tab0, ok := tabs[0].(map[string]interface{})
	if !ok {
		return res
	}

	tabRenderer, ok := tab0["tabRenderer"].(map[string]interface{})
	if !ok {
		return res
	}

	content, ok := tabRenderer["content"].(map[string]interface{})
	if !ok {
		return res
	}

	sectionList, ok := content["sectionListRenderer"].(map[string]interface{})
	if !ok {
		return res
	}

	sections, ok := sectionList["contents"].([]interface{})
	if !ok {
		return res
	}

	for _, sec := range sections {
		secMap, ok := sec.(map[string]interface{})
		if !ok {
			continue
		}

		// Parse musicCarouselShelfRenderer
		if carousel, ok := secMap["musicCarouselShelfRenderer"].(map[string]interface{}); ok {
			items, ok := carousel["contents"].([]interface{})
			if !ok {
				continue
			}

			for _, it := range items {
				itMap, ok := it.(map[string]interface{})
				if !ok {
					continue
				}

				if respItem, ok := itMap["musicResponsiveListItemRenderer"].(map[string]interface{}); ok {
					if track := parseResponsiveItem(respItem); track != nil && track.ID != "" {
						if res.HeroTrack == nil {
							res.HeroTrack = track
						}
						res.Tracks = append(res.Tracks, *track)
					}
				}

				// Also check musicTwoRowItemRenderer (album / mix cards)
				if twoRow, ok := itMap["musicTwoRowItemRenderer"].(map[string]interface{}); ok {
					title := extractTextFromRuns(twoRow["title"])
					thumb := extractTwoRowThumbnail(twoRow)
					if title != "" {
						res.Mixes = append(res.Mixes, model.MixCard{
							ID:        title,
							Title:     title,
							Thumbnail: thumb,
							ColorHex:  pickColorForTitle(title),
						})
					}
				}
			}
		}
	}

	return res
}

func extractTwoRowThumbnail(item map[string]interface{}) string {
	if thumb, ok := item["thumbnailRenderer"].(map[string]interface{}); ok {
		if renderer, ok := thumb["musicThumbnailRenderer"].(map[string]interface{}); ok {
			if tmap, ok := renderer["thumbnail"].(map[string]interface{}); ok {
				if thumbs, ok := tmap["thumbnails"].([]interface{}); ok && len(thumbs) > 0 {
					if last, ok := thumbs[len(thumbs)-1].(map[string]interface{}); ok {
						if u, ok := last["url"].(string); ok {
							return u
						}
					}
				}
			}
		}
	}
	return ""
}

func pickColorForTitle(s string) uint32 {
	colors := []uint32{
		0xE2556E, // Rose
		0x2A7EB7, // Ocean Blue
		0x8C49E6, // Purple Glow
		0xD85D38, // Sunset Amber
		0x367F4F, // Emerald Green
	}
	var hash uint32
	for _, c := range s {
		hash = (hash * 31) + uint32(c)
	}
	return colors[hash%uint32(len(colors))]
}
