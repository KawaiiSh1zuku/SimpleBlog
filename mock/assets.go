package main

import (
	"image"
	"image/color"
	"image/png"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

func (app *mockServer) mockAsset(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "asset")
	width, height := 1200, 675
	if strings.HasPrefix(name, "avatar-") {
		width, height = 160, 160
	} else if strings.HasPrefix(name, "favicon-") {
		width, height = 64, 64
	}

	seed := uint32(2166136261)
	for _, char := range name {
		seed ^= uint32(char)
		seed *= 16777619
	}
	base := color.RGBA{R: uint8(48 + seed%144), G: uint8(48 + (seed>>8)%144), B: uint8(48 + (seed>>16)%144), A: 255}
	accent := color.RGBA{R: 240 - base.R/3, G: 240 - base.G/3, B: 240 - base.B/3, A: 255}
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	stripeWidth := max(width/8, 1)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pixel := base
			if ((x+y/2)/stripeWidth)%3 == 0 {
				pixel = accent
			}
			canvas.SetRGBA(x, y, pixel)
		}
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_ = png.Encode(w, canvas)
}
