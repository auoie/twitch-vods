package scraper

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/auoie/goVods/vods"
	"github.com/klauspost/compress/zstd"
)

func assertEqual[T comparable](t testing.TB, got, want T) {
	t.Helper()
	if got != want {
		t.Fatalf(`got %v want %v`, got, want)
	}
}

func TestSetProfileImageWidth(t *testing.T) {
	profileImageUrl := "https://static-cdn.jtvnw.net/jtv_user_pictures/5b609411-9eb4-4996-91da-bff6ce94bd55-profile_image-300x300.png"
	want := "https://static-cdn.jtvnw.net/jtv_user_pictures/5b609411-9eb4-4996-91da-bff6ce94bd55-profile_image-50x50.png"
	result := SetProfileImageWidth(profileImageUrl, 50)
	assertEqual(t, result, want)
}

func TestSetBoxArtWidthHeight(t *testing.T) {
	boxArtUrl := "https://static-cdn.jtvnw.net/ttv-boxart/32399_IGDB-{width}x{height}.jpg"
	want := "https://static-cdn.jtvnw.net/ttv-boxart/32399_IGDB-40x56.jpg"
	result := SetBoxArtWidthHeight(boxArtUrl, 40, 56)
	assertEqual(t, result, want)
}

func TestDwpFromThumbnailURL(t *testing.T) {
	videoData := &vods.VideoData{StreamerName: "sodapoppin", VideoId: "321790810458", Time: time.Unix(1791655055, 0).UTC()}
	thumbnailUrl := "https://static-cdn.jtvnw.net/cf_vods/d1m7jfoe9zdc1j/36c663f90afb445621c4_sodapoppin_321790810458_1791655055//thumb/thumb0-%{width}x%{height}.jpg"
	dwp := dwpFromThumbnailURL(thumbnailUrl, videoData)
	if dwp == nil {
		t.Fatalf("got nil dwp for a valid thumbnail url")
	}
	assertEqual(t, dwp.Domain, "https://d1m7jfoe9zdc1j.cloudfront.net/")
	assertEqual(t, dwp.Path.UrlPath, "36c663f90afb445621c4_sodapoppin_321790810458_1791655055")
}

func TestDwpFromThumbnailURLWrongStreamID(t *testing.T) {
	videoData := &vods.VideoData{StreamerName: "sodapoppin", VideoId: "111111111111", Time: time.Unix(1791655055, 0).UTC()}
	thumbnailUrl := "https://static-cdn.jtvnw.net/cf_vods/d1m7jfoe9zdc1j/36c663f90afb445621c4_sodapoppin_321790810458_1791655055//thumb/thumb0-%{width}x%{height}.jpg"
	if dwpFromThumbnailURL(thumbnailUrl, videoData) != nil {
		t.Fatalf("got dwp for a thumbnail of a different stream")
	}
}

func TestDwpFromThumbnailURLProcessingPlaceholder(t *testing.T) {
	if dwpFromThumbnailURL("https://vod-secure.twitch.tv/_404/404_processing_640x360.png", nil) != nil {
		t.Fatalf("got dwp for the processing placeholder")
	}
}

func TestDwpFromThumbnailURLUnderscoredLogin(t *testing.T) {
	thumbnailUrl := "https://static-cdn.jtvnw.net/cf_vods/d2nvs31859zcd8/0123456789abcdef0123_some_one_123_987654321012_1791655055//thumb/thumb0-320x180.jpg"
	dwp := dwpFromThumbnailURL(thumbnailUrl, nil)
	if dwp == nil {
		t.Fatalf("got nil dwp for an underscored login")
	}
	assertEqual(t, dwp.Path.UrlPath, "0123456789abcdef0123_some_one_123_987654321012_1791655055")
}

func TestHarvestScanConcurrent(t *testing.T) {
	// hlsWorkerFetchCompressSend runs in several goroutines; harvesting (writes) and
	// scan-domain reads must be safe to run concurrently.
	done := make(chan struct{})
	for i := 0; i < 8; i++ {
		go func(n int) {
			for j := 0; j < 300; j++ {
				harvestDomain(fmt.Sprintf("d%02dtest12345", n%10))
				_ = getScanDomains()
			}
			done <- struct{}{}
		}(i)
	}
	for i := 0; i < 8; i++ {
		<-done
	}
}

func compressForTest(t testing.TB, data string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w, err := zstd.NewWriter(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(data)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestDistrosFromPlaylists(t *testing.T) {
	decoder, err := zstd.NewReader(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer decoder.Close()
	valid := compressForTest(t, "#EXTM3U\nhttps://d1m7jfoe9zdc1j.cloudfront.net/36c663f90afb445621c4_sodapoppin_321790810458_1791655055/chunked/0.ts\nhttps://d1m7jfoe9zdc1j.cloudfront.net/36c663f90afb445621c4_sodapoppin_321790810458_1791655055/chunked/1.ts\n")
	facade := compressForTest(t, "#EXTM3U\nhttps://vod-metro.twitch.tv/f275dfbc25f017b6a751_northernlion_319512439639_1791561385/chunked/0.ts\n")
	mixed := compressForTest(t, "https://d3stzm2eumvgb4.cloudfront.net/842e6414cd3b1d8e9602_caedrel_317938870626_1791448125/chunked/0.ts\n")
	garbage := []byte{0x28, 0xb5, 0x2f, 0xfd, 0x00, 0x00, 'n', 'o', 't', ' ', 'z', 's', 't', 'd'}

	distros := distrosFromPlaylists(decoder, [][]byte{valid, facade, mixed, garbage, nil})
	if !distros["d1m7jfoe9zdc1j"] || !distros["d3stzm2eumvgb4"] {
		t.Fatalf("missing expected distros, got %v", distros)
	}
	if len(distros) != 2 {
		t.Fatalf("facade hosts or garbage leaked into the pool: %v", distros)
	}
}

func TestDistrosFromPlaylistsNoPanicOnHostileInput(t *testing.T) {
	decoder, err := zstd.NewReader(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer decoder.Close()
	hostile := [][]byte{
		{},                                // empty
		bytes.Repeat([]byte{0xff}, 1<<20), // random junk claiming to be data
		compressForTest(t, strings.Repeat("https://evil.cloudfront.net/x\n", 10000)),
		compressForTest(t, "https://UPPER.case.CLOUDFRONT.net/x\nhttps://a.b.cloudfront.net/x\nhttps://x.cloudfront.net.evil.com/x\n"),
		compressForTest(t, "\x00\x01\x02binary\x00junk"),
	}
	for _, playlist := range hostile {
		distros := distrosFromPlaylists(decoder, [][]byte{playlist})
		for distro := range distros {
			// whatever survives must be a single lowercase-alnum label
			if !regexp.MustCompile(`^[a-z0-9]+$`).MatchString(distro) {
				t.Fatalf("hostile input produced non-distro key %q", distro)
			}
		}
	}
}
