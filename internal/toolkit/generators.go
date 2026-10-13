package toolkit

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"strings"
	"time"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code128"
	"github.com/boombuler/barcode/code39"
	"github.com/boombuler/barcode/ean"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/oklog/ulid/v2"
	"golang.org/x/crypto/bcrypt"
)

func generateBarcode(values map[string]string) (Result, error) {
	input := values["input"]
	if input == "" || len(input) > 100 {
		return Result{}, errors.New("Use between 1 and 100 characters for a readable barcode.")
	}
	var encoded barcode.Barcode
	var err error
	switch values["mode"] {
	case "code39":
		encoded, err = code39.Encode(input, false, false)
	case "ean":
		encoded, err = ean.Encode(input)
	case "code128", "":
		encoded, err = code128.Encode(input)
	default:
		return Result{}, errors.New("Choose Code 128, Code 39 or EAN.")
	}
	if err != nil {
		return Result{}, fmt.Errorf("This value cannot be encoded with the selected format: %w", err)
	}
	width, err := number(values, "width", 600, 200, 1600)
	if err != nil {
		return Result{}, err
	}
	height, err := number(values, "height", 160, 60, 400)
	if err != nil {
		return Result{}, err
	}
	factor := width / (encoded.Bounds().Dx() + 20)
	if factor < 1 {
		return Result{}, errors.New("The barcode is too dense at this width. Increase the width or shorten the content.")
	}
	bodyWidth := encoded.Bounds().Dx() * factor
	scaled, err := barcode.Scale(encoded, bodyWidth, height-24)
	if err != nil {
		return Result{}, errors.New("The barcode is too dense at this width. Increase the width or shorten the content.")
	}
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	left := (width - bodyWidth) / 2
	draw.Draw(canvas, image.Rect(left, 12, left+bodyWidth, height-12), scaled, image.Point{}, draw.Src)
	var output bytes.Buffer
	if err := png.Encode(&output, canvas); err != nil {
		return Result{}, err
	}
	result, err := jsonResult(map[string]any{"content": encoded.Content(), "format": encoded.Metadata().CodeKind, "width": width, "height": height}, "Ready to label", "weme-barcode.png")
	result.Mime = "image/png"
	result.Image = "data:image/png;base64," + base64.StdEncoding.EncodeToString(output.Bytes())
	result.ImageAlt = "Generated barcode"
	result.Facts = []Fact{{Label: "Format", Value: encoded.Metadata().CodeKind}, {Label: "Content", Value: encoded.Content()}}
	return result, err
}
func convertColor(values map[string]string) (Result, error) {
	color, err := colorful.Hex(strings.TrimSpace(values["input"]))
	if err != nil {
		return Result{}, errors.New("Use a hexadecimal color such as #075b53 or #0a8.")
	}
	r, g, b := color.RGB255()
	h, s, l := color.Hsl()
	vh, vs, vv := color.Hsv()
	hex := color.Hex()
	rgb := fmt.Sprintf("rgb(%d, %d, %d)", r, g, b)
	hsl := fmt.Sprintf("hsl(%.1f, %.1f%%, %.1f%%)", h, s*100, l*100)
	hsv := fmt.Sprintf("hsv(%.1f, %.1f%%, %.1f%%)", vh, vs*100, vv*100)
	result, err := jsonResult(map[string]any{"hex": hex, "rgb": rgb, "hsl": hsl, "hsv": hsv}, "One color, every format", "color-formats.json")
	result.Table = &Table{Columns: []string{"Format", "Value"}, Rows: [][]string{{"HEX", hex}, {"RGB", rgb}, {"HSL", hsl}, {"HSV", hsv}}}
	result.Swatches = []Swatch{{Name: "Original", Color: hex}, {Name: "20% lighter", Color: color.BlendLab(colorful.Color{R: 1, G: 1, B: 1}, .2).Clamped().Hex()}, {Name: "20% darker", Color: color.BlendLab(colorful.Color{}, .2).Clamped().Hex()}}
	result.Notice = "Conversions use sRGB. The lighter and darker swatches use Lab blending; alpha transparency is not supported."
	return result, err
}
func handleULID(values map[string]string) (Result, error) {
	if values["mode"] == "decode" {
		id, err := ulid.ParseStrict(strings.TrimSpace(values["input"]))
		if err != nil {
			return Result{}, errors.New("Use a valid 26-character ULID without overflow or ambiguous characters.")
		}
		instant := time.UnixMilli(int64(id.Time())).UTC()
		result, err := jsonResult(map[string]any{"ulid": id.String(), "timestampMilliseconds": id.Time(), "createdAtUTC": instant.Format(time.RFC3339Nano)}, "Timestamp inside the ID", "ulid-report.json")
		result.Facts = []Fact{{Label: "Created at (UTC)", Value: instant.Format(time.RFC3339Nano)}}
		return result, err
	}
	count, err := number(values, "count", 5, 1, 100)
	if err != nil {
		return Result{}, err
	}
	ids := make([]string, 0, count)
	for i := 0; i < count; i++ {
		id, err := ulid.New(ulid.Timestamp(time.Now()), rand.Reader)
		if err != nil {
			return Result{}, errors.New("Cryptographic randomness is unavailable.")
		}
		ids = append(ids, id.String())
	}
	return Result{Text: strings.Join(ids, "\n"), Title: "Fresh ULIDs", Filename: "ulids.txt", Mime: "text/plain", Notice: "ULIDs reveal their creation timestamp. IDs generated within the same millisecond have random ordering."}, nil
}
func handleBcrypt(values map[string]string) (Result, error) {
	password := []byte(values["input"])
	if len(password) == 0 || len(password) > 72 {
		return Result{}, errors.New("Bcrypt accepts passwords of 1–72 UTF-8 bytes. Longer passwords are rejected rather than truncated.")
	}
	if values["mode"] == "verify" {
		hash := values["hash"]
		cost, err := bcrypt.Cost([]byte(hash))
		if err != nil {
			return Result{}, errors.New("Use a valid bcrypt hash.")
		}
		if cost > 12 {
			return Result{}, errors.New("This hash uses a cost above 12, beyond the browser's verification budget.")
		}
		err = bcrypt.CompareHashAndPassword([]byte(hash), password)
		matched := err == nil
		if err != nil && !errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return Result{}, errors.New("This bcrypt hash could not be verified.")
		}
		result, err := jsonResult(map[string]any{"matched": matched, "cost": cost}, "Password comparison", "bcrypt-verification.json")
		tone := "bad"
		label := "Does not match"
		if matched {
			tone = "good"
			label = "Matches"
		}
		result.Facts = []Fact{{Label: "Password", Value: label, Tone: tone}, {Label: "Cost", Value: fmt.Sprint(cost)}}
		result.Notice = "Comparison runs locally; the password and hash are not stored or uploaded."
		return result, err
	}
	cost, err := number(values, "cost", 10, 8, 12)
	if err != nil {
		return Result{}, err
	}
	hash, err := bcrypt.GenerateFromPassword(password, cost)
	if err != nil {
		return Result{}, errors.New("The bcrypt hash could not be generated.")
	}
	return Result{Text: string(hash), Title: "Salted bcrypt hash", Filename: "bcrypt-hash.txt", Mime: "text/plain", Notice: "Generated locally with a fresh random salt. Bcrypt is for password storage, not reversible encryption."}, nil
}
