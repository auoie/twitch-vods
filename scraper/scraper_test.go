package scraper

import (
	"testing"
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
