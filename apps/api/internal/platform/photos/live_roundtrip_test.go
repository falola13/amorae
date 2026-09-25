package photos

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/google/uuid"
)

const onePixelPNG = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="

// Signed upload, authenticated delivery, deletion, gone — against the real
// Cloudinary.
func TestLiveRoundTrip(t *testing.T) {
	if os.Getenv("AMORAE_LIVE_PHOTOS") == "" {
		t.Skip("AMORAE_LIVE_PHOTOS not set; skipping the live Cloudinary round trip")
	}
	store := New(
		os.Getenv("CLOUDINARY_CLOUD_NAME"),
		os.Getenv("CLOUDINARY_API_KEY"),
		os.Getenv("CLOUDINARY_API_SECRET"),
	)
	if store == nil {
		t.Skip("no credentials")
	}
	ctx := context.Background()
	publicID := PublicID(uuid.New(), uuid.New())

	ticket, err := store.Ticket(publicID, time.Now())
	if err != nil {
		t.Fatalf("ticket: %v", err)
	}
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	for k, v := range ticket.Fields {
		_ = form.WriteField(k, v)
	}
	part, _ := form.CreateFormFile("file", "pixel.png")
	raw, _ := base64.StdEncoding.DecodeString(onePixelPNG)
	_, _ = part.Write(raw)
	_ = form.Close()

	resp, err := http.Post(ticket.UploadURL, form.FormDataContentType(), &body)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	out, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("upload refused: HTTP %d\n%s", resp.StatusCode, out)
	}
	t.Log("upload: ok")

	url, err := store.URL(publicID, time.Now().Unix())
	if err != nil {
		t.Fatalf("building url: %v", err)
	}
	got, err := http.Get(url)
	if err != nil {
		t.Fatalf("fetching: %v", err)
	}
	got.Body.Close()
	if got.StatusCode != http.StatusOK {
		t.Fatalf("the delivery url does not serve the photo: HTTP %d", got.StatusCode)
	}
	t.Log("authenticated delivery: ok")

	if err := store.Destroy(ctx, publicID); err != nil {
		t.Fatalf("destroy: %v", err)
	}
	t.Log("destroy: ok")

	// Not the URL: CDN copies outlive the original. Asked directly because
	// Destroy treats "ok" and "not found" alike.
	cld, err := cloudinary.NewFromParams(store.cloudName, store.apiKey, store.apiSecret)
	if err != nil {
		t.Fatalf("configuring check: %v", err)
	}
	res, err := cld.Upload.Destroy(ctx, uploader.DestroyParams{PublicID: publicID, Type: "authenticated"})
	if err != nil {
		t.Fatalf("checking: %v", err)
	}
	if res.Result != "not found" {
		t.Errorf("Cloudinary still has the photo: destroy answered %q, want \"not found\"", res.Result)
	}
	t.Logf("origin after deletion: %s", res.Result)

	if gone, err := http.Get(url); err == nil {
		gone.Body.Close()
		t.Logf("the delivery url serves HTTP %d (a CDN copy may outlive the original)", gone.StatusCode)
	}

	if err := store.Destroy(ctx, publicID); err != nil {
		t.Errorf("destroying an already-deleted photo failed: %v", err)
	}
	t.Log("destroy is repeatable: ok")
}
