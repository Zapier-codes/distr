package api_test

import (
	"bytes"
	"image"
	"image/png"
	"strings"
	"testing"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/validation"
	"github.com/google/uuid"
)

func pngBytes(t *testing.T, width, height int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, width, height))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func validRequest() api.CreateProductRequestRequest {
	return api.CreateProductRequestRequest{
		ContactEmail:     "owner@example.com",
		AppName:          "Acme",
		ProductServiceID: uuid.New(),
		ThemeColor:       "#ff6600",
	}
}

func TestCreateProductRequestValidate(t *testing.T) {
	tests := map[string]struct {
		change  func(r *api.CreateProductRequestRequest)
		wantErr bool
	}{
		"valid without icon": {func(r *api.CreateProductRequestRequest) {}, false},
		"valid with icon": {func(r *api.CreateProductRequestRequest) {
			r.Icon = &api.ProductRequestIcon{ContentType: "image/png", Data: pngBytes(t, 192, 192)}
		}, false},
		"missing product": {func(r *api.CreateProductRequestRequest) { r.ProductServiceID = uuid.Nil }, true},
		"bad color":       {func(r *api.CreateProductRequestRequest) { r.ThemeColor = "orange" }, true},
		"icon not square": {func(r *api.CreateProductRequestRequest) {
			r.Icon = &api.ProductRequestIcon{ContentType: "image/png", Data: pngBytes(t, 192, 128)}
		}, true},
		"icon too small": {func(r *api.CreateProductRequestRequest) {
			r.Icon = &api.ProductRequestIcon{ContentType: "image/png", Data: pngBytes(t, 32, 32)}
		}, true},
		"icon too large in pixels": {func(r *api.CreateProductRequestRequest) {
			r.Icon = &api.ProductRequestIcon{ContentType: "image/png", Data: pngBytes(t, 2048, 2048)}
		}, true},
		"icon wrong declared type": {func(r *api.CreateProductRequestRequest) {
			r.Icon = &api.ProductRequestIcon{ContentType: "image/svg+xml", Data: pngBytes(t, 192, 192)}
		}, true},
		"icon declared png but not png": {func(r *api.CreateProductRequestRequest) {
			r.Icon = &api.ProductRequestIcon{ContentType: "image/png", Data: []byte("<svg onload=alert(1)>")}
		}, true},
		"icon over byte limit": {func(r *api.CreateProductRequestRequest) {
			r.Icon = &api.ProductRequestIcon{
				ContentType: "image/png",
				Data:        append(pngBytes(t, 192, 192), []byte(strings.Repeat("x", api.MaxProductRequestIconBytes))...),
			}
		}, true},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			r := validRequest()
			tt.change(&r)
			err := r.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !isValidationError(err) {
				t.Errorf("error %v is not a validation error", err)
			}
		})
	}
}

func isValidationError(err error) bool {
	return err != nil && strings.Contains(err.Error(), validation.ErrValidationFailed.Error())
}
