package appui

import (
	"testing"

	"chordly/router"
)

func TestAlbumPageRenders(t *testing.T) {
	r := router.New(&router.Route{
		Segment: "album",
		Children: []*router.Route{
			{
				Segment: ":id",
				Page:    AlbumPage,
			},
		},
	})
	r.Navigate("/album/MPREb_test_album")

	ctx := router.RouteContext{
		Router: r,
		Match:  r.Current(),
	}

	w := AlbumPage(ctx)
	if w == nil {
		t.Fatal("expected non-nil widget from AlbumPage")
	}
}

func TestArtistPageRenders(t *testing.T) {
	r := router.New(&router.Route{
		Segment: "artist",
		Children: []*router.Route{
			{
				Segment: ":id",
				Page:    ArtistPage,
			},
		},
	})
	r.Navigate("/artist/UC_test_artist")

	ctx := router.RouteContext{
		Router: r,
		Match:  r.Current(),
	}

	w := ArtistPage(ctx)
	if w == nil {
		t.Fatal("expected non-nil widget from ArtistPage")
	}
}

func TestPlaylistAndMixPagesRender(t *testing.T) {
	r := router.New(
		&router.Route{
			Segment: "playlist",
			Children: []*router.Route{
				{
					Segment: ":id",
					Page:    PlaylistPage,
				},
			},
		},
		&router.Route{
			Segment: "mix",
			Children: []*router.Route{
				{
					Segment: ":id",
					Page:    MixPage,
				},
			},
		},
	)

	r.Navigate("/playlist/PL_test_playlist")
	ctx1 := router.RouteContext{Router: r, Match: r.Current()}
	w1 := PlaylistPage(ctx1)
	if w1 == nil {
		t.Fatal("expected non-nil widget from PlaylistPage")
	}

	r.Navigate("/mix/RD_test_mix")
	ctx2 := router.RouteContext{Router: r, Match: r.Current()}
	w2 := MixPage(ctx2)
	if w2 == nil {
		t.Fatal("expected non-nil widget from MixPage")
	}
}
