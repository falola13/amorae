// Package photos holds the pictures attached to memories.
//
// The browser uploads straight to Cloudinary rather than through this API.
// A 10MB photo travelling through a free container host — to be forwarded
// somewhere else — is bandwidth and memory spent to no purpose, on the tier
// least able to spare either. So the API's whole job here is to say, in a
// form Cloudinary will believe, "this person may put one file, at exactly
// this name, for the next hour".
//
// The name is not negotiable and never comes from the request. The server
// derives it from the couple and the memory, so a signed ticket can only
// ever write to that memory's own photo — there is no field for a client to
// point somewhere else.
package photos

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	cldapi "github.com/cloudinary/cloudinary-go/v2/api"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/cloudinary/cloudinary-go/v2/asset"
	"github.com/cloudinary/cloudinary-go/v2/config"
	"github.com/google/uuid"
)

// ErrNotConfigured is what every call answers when there are no credentials.
// Photos are the one feature that needs an account somewhere else, so the
// app runs without them rather than refusing to start.
var ErrNotConfigured = errors.New("photos: Cloudinary is not configured")

// Ticket is everything the browser needs to upload one file, and nothing more.
//
// Fields is the form exactly as Cloudinary must receive it, so the browser
// appends the file and sends what it was given. It is deliberately not a set
// of named values for the client to reassemble: the signature covers a
// precise list, and a client rebuilding that list is a client that can get it
// subtly wrong — for which Cloudinary's only answer is "Invalid Signature".
type Ticket struct {
	UploadURL string            `json:"upload_url"`
	Fields    map[string]string `json:"fields"`
}

type Store struct {
	cloudName string
	apiKey    string
	apiSecret string
}

// New returns nil when Cloudinary is not configured, and every caller checks.
func New(cloudName, apiKey, apiSecret string) *Store {
	if cloudName == "" || apiKey == "" || apiSecret == "" {
		return nil
	}
	return &Store{cloudName: cloudName, apiKey: apiKey, apiSecret: apiSecret}
}

// PublicID is where a memory's photo lives, derived and never supplied.
//
// Scoped by couple so one couple's id can never address another's, and by
// memory so re-uploading replaces the picture rather than accumulating
// pictures nobody can see.
func PublicID(coupleID, memoryID uuid.UUID) string {
	return fmt.Sprintf("amorae/%s/memories/%s", coupleID, memoryID)
}

// Ticket signs permission to upload one file to one name.
//
// Cloudinary's rule: sign every parameter that will be sent except file,
// cloud_name, resource_type and api_key. The SDK does the sorting, encoding
// and hashing, because that is exactly the sort of detail that is wrong in a
// way nothing fails loudly about.
func (s *Store) Ticket(publicID string, at time.Time) (Ticket, error) {
	if s == nil {
		return Ticket{}, ErrNotConfigured
	}
	// Signed: everything that will be sent except file, cloud_name,
	// resource_type and api_key, which Cloudinary never signs.
	signed := map[string]string{
		"public_id": publicID,
		"timestamp": strconv.FormatInt(at.Unix(), 10),
		// Not readable from a guessed URL; delivery needs the signature that
		// URL below produces.
		"type": "authenticated",
		// Replace rather than add: a second upload to the same memory is a
		// correction, not a collection.
		"overwrite":       "true",
		"invalidate":      "true",
		"unique_filename": "false",
	}

	params := url.Values{}
	for k, v := range signed {
		params.Set(k, v)
	}
	// SignParameters is the SDK's own default: SHA-1, sorted, the secret
	// appended. Their algorithm rather than a reading of their documentation.
	sig, err := cldapi.SignParameters(params, s.apiSecret)
	if err != nil {
		return Ticket{}, fmt.Errorf("signing upload: %w", err)
	}

	fields := make(map[string]string, len(signed)+2)
	for k, v := range signed {
		fields[k] = v
	}
	fields["api_key"] = s.apiKey
	fields["signature"] = sig

	return Ticket{
		UploadURL: fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/image/upload", s.cloudName),
		Fields:    fields,
	}, nil
}

// Destroy deletes a picture from Cloudinary, for good.
//
// Removing a photo has to mean removing the photo. Clearing the pointer in
// our own database and leaving the file sitting in an account somewhere is a
// deletion that is true on the screen and false everywhere else, which is
// not a distinction to make on somebody's behalf about their own pictures.
//
// A file that has already gone is not a failure. Cloudinary answers "not
// found", the world is in the state that was asked for, and a removal
// interrupted halfway can be finished by asking again.
func (s *Store) Destroy(ctx context.Context, publicID string) error {
	if s == nil {
		return ErrNotConfigured
	}
	cld, err := cloudinary.NewFromParams(s.cloudName, s.apiKey, s.apiSecret)
	if err != nil {
		return fmt.Errorf("configuring deletion: %w", err)
	}
	res, err := cld.Upload.Destroy(ctx, uploader.DestroyParams{
		PublicID: publicID,
		// Uploaded as authenticated, so deleted as authenticated: the
		// delivery type is part of which asset this names.
		Type: "authenticated",
		// Drop the CDN's copies too, or the picture keeps being served from
		// the edge after it stops existing at the origin.
		Invalidate: cldapi.Bool(true),
	})
	if err != nil {
		return fmt.Errorf("deleting photo: %w", err)
	}
	// The SDK reports a refusal in the body rather than as an error, so a
	// deletion that did not happen looks like success unless this is read.
	if res.Error.Message != "" {
		return fmt.Errorf("deleting photo: %s", res.Error.Message)
	}
	if res.Result != "ok" && res.Result != "not found" {
		return fmt.Errorf("deleting photo: %s", res.Result)
	}
	return nil
}

// URL is a delivery address for an authenticated asset.
//
// Authenticated assets cannot be fetched by guessing a URL: the path carries
// a signature over the transformation and the name. This app hands one out
// only to a signed-in member of the couple the memory belongs to.
//
// Q-06 asked for short-lived signed URLs. These are signed but do not expire
// — expiring tokens are not part of Cloudinary's free plan — so the honest
// description is an unguessable address given only to the two people who may
// see it. Worth revisiting if this ever leaves the two of them.
//
// f_auto and q_auto re-encode on delivery, which also drops whatever EXIF and
// GPS the camera wrote into the original (Q-06 again).
func (s *Store) URL(publicID string) (string, error) {
	if s == nil {
		return "", ErrNotConfigured
	}
	conf, err := config.NewFromParams(s.cloudName, s.apiKey, s.apiSecret)
	if err != nil {
		return "", fmt.Errorf("configuring delivery: %w", err)
	}
	conf.URL.SignURL = true

	img, err := asset.Image(publicID, conf)
	if err != nil {
		return "", fmt.Errorf("building photo url: %w", err)
	}
	img.DeliveryType = cldapi.Authenticated
	img.Transformation = "f_auto,q_auto"

	out, err := img.String()
	if err != nil {
		return "", fmt.Errorf("building photo url: %w", err)
	}
	return out, nil
}
