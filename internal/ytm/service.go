package ytm

import (
	"context"
	"log"
	"time"

	"chordly/internal/state"
)

// Service provides a synchronized bridge between InnerTube and the UI reactive state.
type Service struct {
	client *Client
}

var DefaultService = NewService()

func NewService() *Service {
	return &Service{
		client: NewClient(),
	}
}

// Client returns the underlying InnerTube client.
func (s *Service) Client() *Client {
	return s.client
}

// SyncHomeData asynchronously loads live YouTube Music content into reactive signals.
func (s *Service) SyncHomeData(ctx context.Context) {
	go func() {
		feedCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		feed, err := s.client.GetHomeFeed(feedCtx)
		if err != nil {
			log.Printf("[YTM] Home feed fetch notice (using cache/defaults): %v", err)
			return
		}

		// Update Mix cards if returned
		if len(feed.Mixes) > 0 {
			max := len(feed.Mixes)
			if max > 5 {
				max = 5
			}
			state.MadeForYouMixes.Set(feed.Mixes[:max])
		}

		// Update New Releases & Queue if tracks returned
		if len(feed.Tracks) > 0 {
			maxReleases := len(feed.Tracks)
			if maxReleases > 4 {
				maxReleases = 4
			}
			state.NewReleases.Set(feed.Tracks[:maxReleases])

			// Populate queue with live tracks
			state.Queue.Set(feed.Tracks)

			// Update Hero song
			if feed.HeroTrack != nil {
				state.CurrentTrack.Set(*feed.HeroTrack)
				state.Playhead.Set(0)
			}
			log.Printf("[YTM] Successfully synchronized %d live tracks from YouTube Music", len(feed.Tracks))
		}
	}()
}

// LiveSearch queries YouTube Music and updates the queue with real search hits.
func (s *Service) LiveSearch(ctx context.Context, query string) {
	go func() {
		searchCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
		defer cancel()

		tracks, err := s.client.Search(searchCtx, query)
		if err != nil {
			log.Printf("[YTM] Search error for %q: %v", query, err)
			return
		}

		if len(tracks) > 0 {
			state.Queue.Set(tracks)
			state.QueueIndex.Set(0)
			state.CurrentTrack.Set(tracks[0])
			state.Playhead.Set(0)
			log.Printf("[YTM] Search for %q loaded %d live tracks", query, len(tracks))
		}
	}()
}
