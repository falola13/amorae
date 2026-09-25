package photos

import (
	"net/url"
	"strings"
	"testing"
	"time"

	cldapi "github.com/cloudinary/cloudinary-go/v2/api"
	"github.com/google/uuid"
)

func TestSignParameters_MatchesCloudinarysPublishedExample(t *testing.T) {
	// From Cloudinary's own documentation: secret "abcd", timestamp
	// 1315060510, string to sign "timestamp=1315060510abcd".
	//
	// This is here because the signature is the one part of an upload that
	// fails silently and identically for every possible mistake — a wrong
	// separator, the wrong hash, the secret in the wrong place all produce
	// "Invalid Signature" and nothing else. Pinning the library against their
	// published vector means a change in either is caught here rather than by
	// a photo that will not upload.
	got, err := cldapi.SignParameters(url.Values{"timestamp": {"1315060510"}}, "abcd")
	if err != nil {
		t.Fatalf("signing: %v", err)
	}
	if want := "a21ad0f63beb4de2e5575204b79ab90bffb02c10"; got != want {
		t.Errorf("signature = %q, want %q", got, want)
	}
}

func TestPublicID_ScopedSoOneCoupleCannotAddressAnother(t *testing.T) {
	couple, memory := uuid.New(), uuid.New()
	id := PublicID(couple, memory)

	if !strings.Contains(id, couple.String()) || !strings.Contains(id, memory.String()) {
		t.Errorf("public id = %q, want both ids in it", id)
	}
	// The same memory always lands in the same place, so a second upload
	// replaces the picture rather than adding one nobody can see.
	if again := PublicID(couple, memory); again != id {
		t.Errorf("not stable: %q then %q", id, again)
	}
	if other := PublicID(uuid.New(), memory); other == id {
		t.Error("two couples share a path for the same memory id")
	}
}

func TestStore_NilWithoutCredentials(t *testing.T) {
	// Photos are the one feature needing an account somewhere else. Without
	// it the app runs; it just cannot take pictures.
	for _, c := range [][3]string{{"", "k", "s"}, {"c", "", "s"}, {"c", "k", ""}, {"", "", ""}} {
		if New(c[0], c[1], c[2]) != nil {
			t.Errorf("New(%q, %q, %q) returned a store", c[0], c[1], c[2])
		}
	}
	if New("c", "k", "s") == nil {
		t.Error("a fully configured store was nil")
	}
}

func TestStore_NotConfiguredAnswersRatherThanPanics(t *testing.T) {
	var s *Store
	if _, err := s.Ticket("x", time.Now()); err != ErrNotConfigured {
		t.Errorf("Ticket err = %v", err)
	}
	if _, err := s.URL("x", 1); err != ErrNotConfigured {
		t.Errorf("URL err = %v", err)
	}
}

func TestTicket_SignsExactlyWhatTheBrowserSendsBack(t *testing.T) {
	s := New("demo", "1234", "abcd")
	ticket, err := s.Ticket("amorae/couple/memories/one", time.Unix(1315060510, 0))
	if err != nil {
		t.Fatalf("Ticket: %v", err)
	}

	// Re-sign the form the browser will actually send, minus the two
	// Cloudinary never signs. If this does not reproduce the signature the
	// upload is refused, and the only clue is "Invalid Signature".
	params := url.Values{}
	for k, v := range ticket.Fields {
		if k == "api_key" || k == "signature" {
			continue
		}
		params.Set(k, v)
	}
	want, err := cldapi.SignParameters(params, "abcd")
	if err != nil {
		t.Fatalf("re-signing: %v", err)
	}
	if ticket.Fields["signature"] != want {
		t.Errorf("the ticket signs something other than what it sends: got %q, want %q",
			ticket.Fields["signature"], want)
	}

	if ticket.Fields["type"] != "authenticated" {
		t.Errorf("type = %q: a photo readable from a guessed URL is not private", ticket.Fields["type"])
	}
	if ticket.Fields["public_id"] != "amorae/couple/memories/one" {
		t.Errorf("public_id = %q, want the name the server chose", ticket.Fields["public_id"])
	}
	if !strings.Contains(ticket.UploadURL, "demo") {
		t.Errorf("upload url = %q", ticket.UploadURL)
	}
	// The secret must never be among the things handed to a browser.
	for k, v := range ticket.Fields {
		if strings.Contains(v, "abcd") && k != "signature" {
			t.Errorf("field %q carries the API secret", k)
		}
	}
}

func TestURL_IsSignedAndStripsCameraMetadata(t *testing.T) {
	s := New("demo", "1234", "abcd")
	got, err := s.URL("amorae/couple/memories/one", 1790300717)
	if err != nil {
		t.Fatalf("URL: %v", err)
	}
	if !strings.Contains(got, "/authenticated/") {
		t.Errorf("url = %q, want an authenticated delivery type", got)
	}
	if !strings.Contains(got, "s--") {
		t.Errorf("url = %q, want a signature — without one it is guessable", got)
	}
	// f_auto/q_auto re-encode, which drops the EXIF and GPS the camera wrote
	// (Q-06).
	if !strings.Contains(got, "f_auto") {
		t.Errorf("url = %q, want the re-encoding that strips metadata", got)
	}
	if !strings.Contains(got, "/v1790300717/") {
		t.Errorf("url = %q, want the version that makes a new photo a new address", got)
	}
}

func TestURL_ChangesWhenThePhotoDoes(t *testing.T) {
	s := New("demo", "1234", "abcd")
	before, err := s.URL("amorae/couple/memories/one", 1790300717)
	if err != nil {
		t.Fatalf("URL: %v", err)
	}
	after, err := s.URL("amorae/couple/memories/one", 1790300999)
	if err != nil {
		t.Fatalf("URL: %v", err)
	}
	if before == after {
		t.Error("replacing a photo left the address alone, so the CDN would keep the old one")
	}
}
