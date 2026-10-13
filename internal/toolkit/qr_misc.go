package toolkit

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	qrcode "github.com/skip2/go-qrcode"
)

func generateQR(values map[string]string) (Result, error) {
	input := values["input"]
	if len([]rune(input)) > 2500 {
		return Result{}, errors.New("For a readable QR code, keep your content under 2,500 characters.")
	}
	level := qrcode.Medium
	switch values["correction"] {
	case "L":
		level = qrcode.Low
	case "Q":
		level = qrcode.High
	case "H":
		level = qrcode.Highest
	case "M", "":
	default:
		return Result{}, errors.New("Choose L, M, Q or H error correction.")
	}
	code, err := qrcode.New(input, level)
	if err != nil {
		return Result{}, fmt.Errorf("This content cannot fit the selected QR settings: %w", err)
	}
	bitmap := code.Bitmap()
	size := len(bitmap)
	var path strings.Builder
	for y, row := range bitmap {
		for x := 0; x < len(row); {
			if !row[x] {
				x++
				continue
			}
			start := x
			for x < len(row) && row[x] {
				x++
			}
			fmt.Fprintf(&path, "M%d %dh%dv1H%dz", start, y, x-start, start)
		}
	}
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512" viewBox="0 0 %d %d" shape-rendering="crispEdges"><rect width="100%%" height="100%%" fill="white"/><path d="%s" fill="#075b53"/></svg>`, size, size, path.String())
	return Result{Text: svg, Image: "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(svg)), ImageAlt: "Generated QR code", Filename: "weme-qr.svg", Mime: "image/svg+xml", Language: "xml", Title: "Ready to scan", Facts: []Fact{{Label: "Error correction", Value: values["correction"]}, {Label: "Matrix", Value: fmt.Sprintf("%d × %d", size, size)}}}, nil
}
func handleGzip(values map[string]string) (Result, error) {
	if values["mode"] == "decompress" {
		compressed, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(values["input"]), ""))
		if err != nil {
			return Result{}, errors.New("Use Base64-encoded gzip data.")
		}
		reader, err := gzip.NewReader(bytes.NewReader(compressed))
		if err != nil {
			return Result{}, errors.New("This input is not a valid gzip stream.")
		}
		defer reader.Close()
		output, err := io.ReadAll(io.LimitReader(reader, (1<<20)+1))
		if err != nil {
			return Result{}, errors.New("This gzip stream is corrupt or incomplete.")
		}
		if len(output) > 1<<20 {
			return Result{}, errors.New("Decompressed output exceeds the 1 MB budget.")
		}
		if !utf8.Valid(output) {
			return Result{}, errors.New("The decompressed content is binary, not UTF-8 text.")
		}
		return Result{Text: string(output), Title: "Text, unpacked", Filename: "decompressed.txt", Mime: "text/plain"}, nil
	}
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	if _, err := writer.Write([]byte(values["input"])); err != nil {
		return Result{}, err
	}
	if err := writer.Close(); err != nil {
		return Result{}, err
	}
	encoded := base64.StdEncoding.EncodeToString(buffer.Bytes())
	return Result{Text: encoded, Title: "Compressed text", Filename: "compressed-base64.txt", Mime: "text/plain", Binary: &Binary{Base64: encoded, Mime: "application/gzip", Filename: "weme.txt.gz"}, Facts: []Fact{{Label: "Original", Value: fmt.Sprintf("%d bytes", len(values["input"]))}, {Label: "Gzip", Value: fmt.Sprintf("%d bytes", buffer.Len())}}, Notice: "The result is Base64 for easy copying; Download saves a binary .gz file. Decompression is limited to 1 MB."}, nil
}
func chmodPermissions(values map[string]string) (Result, error) {
	input := strings.TrimSpace(values["input"])
	if len(input) < 3 || len(input) > 4 {
		return Result{}, errors.New("Use three or four octal digits, such as 755 or 0644.")
	}
	value, err := strconv.ParseUint(input, 8, 12)
	if err != nil {
		return Result{}, errors.New("Unix permissions use octal digits 0–7.")
	}
	permissions := fmt.Sprintf("%03o", value)
	symbolic := ""
	rows := [][]string{}
	for index, name := range []string{"Owner", "Group", "Others"} {
		mask := (value >> uint((2-index)*3)) & 7
		read, write, execute := "No", "No", "No"
		segment := ""
		if mask&4 != 0 {
			read = "Yes"
			segment += "r"
		} else {
			segment += "-"
		}
		if mask&2 != 0 {
			write = "Yes"
			segment += "w"
		} else {
			segment += "-"
		}
		if mask&1 != 0 {
			execute = "Yes"
			segment += "x"
		} else {
			segment += "-"
		}
		symbolic += segment
		rows = append(rows, []string{name, read, write, execute})
	}
	special := []rune(symbolic)
	if value&04000 != 0 {
		special[2] = 'S'
		if value&0100 != 0 {
			special[2] = 's'
		}
	}
	if value&02000 != 0 {
		special[5] = 'S'
		if value&0010 != 0 {
			special[5] = 's'
		}
	}
	if value&01000 != 0 {
		special[8] = 'T'
		if value&0001 != 0 {
			special[8] = 't'
		}
	}
	symbolic = string(special)
	result, err := jsonResult(map[string]any{"octal": permissions, "symbolic": symbolic, "command": "chmod " + permissions + " <path>"}, "Permissions, explained", "unix-permissions.json")
	result.Table = &Table{Columns: []string{"Role", "Read", "Write", "Execute"}, Rows: rows}
	result.Facts = []Fact{{Label: "Octal", Value: permissions}, {Label: "Symbolic", Value: symbolic}}
	result.Notice = "Inspection only; no filesystem permissions are changed. Execute on a directory controls traversal."
	return result, err
}
