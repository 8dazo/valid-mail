package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/SalvioniDigitalSolutions/gopdf"
)

const maxPDFUpload = 12 << 20
const maxPDFRequest = 28 << 20
const maxPDFPages = 120

type pdfPageInfo struct {
	Number int     `json:"number"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}
type pdfRunInfo struct {
	Text string  `json:"text"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	Size float64 `json:"size"`
	Font string  `json:"font"`
}
type pdfFormInfo struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

func pdfError(w http.ResponseWriter, code int, message string) {
	writeJSON(w, code, map[string]any{"success": false, "error": message})
}

func pdfUpload(r *http.Request, field string, optional bool) ([]byte, error) {
	f, _, err := r.FormFile(field)
	if err != nil {
		if optional && errors.Is(err, http.ErrMissingFile) {
			return nil, nil
		}
		return nil, fmt.Errorf("upload a PDF document for %s", field)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxPDFUpload+1))
	if err != nil {
		return nil, errors.New("could not read the PDF upload")
	}
	if len(data) > maxPDFUpload {
		return nil, errors.New("each PDF must be under 12 MB")
	}
	if len(data) < 8 || !bytes.HasPrefix(data, []byte("%PDF-")) {
		return nil, errors.New("the uploaded file is not a valid PDF header")
	}
	return data, nil
}

func pdfNumber(r *http.Request, key string, fallback float64) (float64, error) {
	s := strings.TrimSpace(r.FormValue(key))
	if s == "" {
		return fallback, nil
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil || n != n || n < -10000 || n > 10000 {
		return 0, fmt.Errorf("invalid %s value", key)
	}
	return n, nil
}

func pdfPageSelection(s string, count int) ([]int, error) {
	if count < 1 || count > maxPDFPages {
		return nil, errors.New("PDF has an unsupported page count")
	}
	if strings.TrimSpace(s) == "" {
		pages := make([]int, count)
		for i := range pages {
			pages[i] = i
		}
		return pages, nil
	}
	pages := []int{}
	for _, token := range strings.Split(s, ",") {
		token = strings.TrimSpace(token)
		if token == "" {
			return nil, errors.New("empty page selection")
		}
		bounds := strings.Split(token, "-")
		if len(bounds) > 2 {
			return nil, errors.New("invalid page range")
		}
		from, err := strconv.Atoi(strings.TrimSpace(bounds[0]))
		if err != nil {
			return nil, errors.New("page selection must contain numbers and ranges")
		}
		to := from
		if len(bounds) == 2 {
			to, err = strconv.Atoi(strings.TrimSpace(bounds[1]))
			if err != nil {
				return nil, errors.New("invalid page range")
			}
		}
		if from < 1 || from > count || to < from || to > count {
			return nil, errors.New("selected page is outside the PDF")
		}
		for n := from; n <= to; n++ {
			pages = append(pages, n-1)
			if len(pages) > maxPDFPages {
				return nil, errors.New("too many selected pages")
			}
		}
	}
	return pages, nil
}

func pdfColor(hex string) (gopdf.Color, error) {
	hex = strings.TrimPrefix(strings.TrimSpace(hex), "#")
	if hex == "" {
		return gopdf.RGB(48, 45, 70), nil
	}
	if len(hex) != 6 {
		return gopdf.Color{}, errors.New("color must be a six-digit hex code")
	}
	var red, green, blue uint8
	if _, err := fmt.Sscanf(hex, "%02x%02x%02x", &red, &green, &blue); err != nil {
		return gopdf.Color{}, errors.New("invalid color value")
	}
	return gopdf.RGB(red, green, blue), nil
}

func pdfFont(style string) *gopdf.Font {
	switch style {
	case "bold":
		return gopdf.HelveticaBold
	case "italic":
		return gopdf.HelveticaOblique
	case "bolditalic":
		return gopdf.HelveticaBoldOblique
	case "serif":
		return gopdf.TimesRoman
	default:
		return gopdf.Helvetica
	}
}

func pdfFormFile(r *http.Request, key string, limit int64) ([]byte, error) {
	f, _, err := r.FormFile(key)
	if err != nil {
		return nil, errors.New("draw or upload your signature image")
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil || int64(len(b)) > limit {
		return nil, errors.New("signature image exceeds 2 MB")
	}
	if len(b) == 0 {
		return nil, errors.New("signature image is empty")
	}
	return b, nil
}

// handlePDF performs isolated, bounded, multipart PDF operations. No source PDFs
// or signatures are stored; all output files are removed after the response.
func (app *application) handlePDF(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		pdfError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxPDFRequest)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		pdfError(w, http.StatusBadRequest, "Invalid or oversized PDF upload (28 MB maximum request)")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	data, err := pdfUpload(r, "pdf", false)
	if err != nil {
		pdfError(w, 400, err.Error())
		return
	}
	src, err := gopdf.NewReaderPassword(data, r.FormValue("password"))
	if err != nil {
		pdfError(w, 422, "Could not open the PDF. Check the file and document password.")
		return
	}
	count := src.NumPages()
	if count < 1 || count > maxPDFPages {
		pdfError(w, 422, "PDF must contain between 1 and 120 pages")
		return
	}
	switch r.URL.Path {
	case "/api/pdf/inspect":
		page, _ := strconv.Atoi(r.FormValue("page"))
		if page < 1 || page > count {
			page = 1
		}
		pages := make([]pdfPageInfo, 0, count)
		for i := 0; i < count; i++ {
			size, err := src.PageSize(i)
			if err != nil {
				pdfError(w, 422, "Could not read PDF page sizes")
				return
			}
			pages = append(pages, pdfPageInfo{i + 1, size.W, size.H})
		}
		doc := gopdf.New()
		edit, err := doc.EditPage(src, page-1)
		if err != nil {
			pdfError(w, 422, "Could not inspect PDF text on this page")
			return
		}
		runs := make([]pdfRunInfo, 0)
		for _, run := range edit.Runs() {
			t := strings.TrimSpace(run.Text)
			if t == "" {
				continue
			}
			if len(t) > 300 {
				t = t[:300]
			}
			runs = append(runs, pdfRunInfo{t, run.X, run.Y, run.FontSize, run.FontName})
			if len(runs) >= 250 {
				break
			}
		}
		fields := []pdfFormInfo{}
		for _, f := range src.FormFields() {
			fields = append(fields, pdfFormInfo{f.Name, string(f.Type), f.Value})
			if len(fields) >= 150 {
				break
			}
		}
		writeJSON(w, 200, map[string]any{
			"success": true, "page_count": count, "pages": pages, "runs": runs, "fields": fields,
			"encrypted": src.IsEncrypted(), "has_signatures": src.HasSignatures(),
		})
		return
	case "/api/pdf/preview":
		page, _ := strconv.Atoi(r.FormValue("page"))
		if page < 1 || page > count {
			pdfError(w, 400, "Select a valid page")
			return
		}
		size, err := src.PageSize(page - 1)
		if err != nil || size.W <= 0 || size.H <= 0 || size.W > 1600 || size.H > 1600 {
			pdfError(w, 422, "Page is too large to preview")
			return
		}
		img, err := src.RenderPage(page-1, gopdf.RenderOpts{
			DPI: 90, IncludeText: true, IncludeRasterImages: true, IncludeVector: true,
			IncludeAnnotations: true, SubstituteFont: gopdf.SystemFonts(),
		})
		if err != nil {
			pdfError(w, 422, "Preview unavailable for this PDF; the original can still be viewed in your browser")
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "no-store")
		if err := png.Encode(w, img); err != nil {
			return
		}
		return
	case "/api/pdf/edit":
		// continue below
	default:
		pdfError(w, 404, "Unknown PDF operation")
		return
	}
	op := r.FormValue("operation")
	selectedPage, err := strconv.Atoi(r.FormValue("page"))
	if err != nil || selectedPage < 1 || selectedPage > count {
		pdfError(w, 400, "Choose a valid page")
		return
	}
	doc := gopdf.New()
	success := 0
	switch op {
	case "merge":
		other, err := pdfUpload(r, "pdf2", false)
		if err != nil {
			pdfError(w, 400, err.Error())
			return
		}
		next, err := gopdf.NewReader(other)
		if err != nil || count+next.NumPages() > maxPDFPages {
			pdfError(w, 422, "Second PDF is invalid or combined document is too large")
			return
		}
		err = doc.AppendPDF(src)
		if err == nil {
			err = doc.AppendPDF(next)
		}
		if err != nil {
			pdfError(w, 422, "Could not merge these PDFs")
			return
		}
	case "extract", "reorder":
		pages, err := pdfPageSelection(r.FormValue("pages"), count)
		if err != nil {
			pdfError(w, 400, err.Error())
			return
		}
		for _, i := range pages {
			if _, err := doc.ImportPage(src, i); err != nil {
				pdfError(w, 422, "Could not copy the selected PDF pages")
				return
			}
		}
	case "fill":
		values := map[string]string{}
		if err := json.Unmarshal([]byte(r.FormValue("fields")), &values); err != nil || len(values) == 0 || len(values) > 100 {
			pdfError(w, 400, "Provide the PDF fields and their values")
			return
		}
		n, err := doc.FillForm(src, values)
		if err != nil || n == 0 {
			pdfError(w, 422, "Could not fill the requested form fields")
			return
		}
		success = n
	case "redact":
		literal := strings.TrimSpace(r.FormValue("find"))
		if literal == "" || len(literal) > 200 {
			pdfError(w, 400, "Enter a search term to permanently remove")
			return
		}
		rd := gopdf.Redact(src)
		rd.Text(literal)
		rd.StripMetadata(true)
		rd.SetOverlay(true)
		marks, err := rd.Marks()
		if err != nil || len(marks) == 0 {
			pdfError(w, 422, "No selectable text matched. Scanned documents require OCR.")
			return
		}
		temp, err := os.CreateTemp("", "weme-redacted-*.pdf")
		if err != nil {
			pdfError(w, 500, "Could not create output")
			return
		}
		path := temp.Name()
		temp.Close()
		defer os.Remove(path)
		if err := rd.Save(path); err != nil {
			pdfError(w, 422, "Could not safely redact this PDF")
			return
		}
		pdfSendFile(w, path, "weme-redacted.pdf")
		return
	case "replace", "restyle", "text", "watermark", "sign", "highlight", "rectangle", "rotate":
		newText := strings.TrimSpace(r.FormValue("text"))
		find := strings.TrimSpace(r.FormValue("find"))
		size, err := pdfNumber(r, "size", 18)
		if err != nil || size < 6 || size > 120 {
			pdfError(w, 400, "Font size must be between 6 and 120")
			return
		}
		color, err := pdfColor(r.FormValue("color"))
		if err != nil {
			pdfError(w, 400, err.Error())
			return
		}
		x, err := pdfNumber(r, "x", 72)
		if err != nil {
			pdfError(w, 400, err.Error())
			return
		}
		y, err := pdfNumber(r, "y", 100)
		if err != nil {
			pdfError(w, 400, err.Error())
			return
		}
		width, err := pdfNumber(r, "width", 180)
		if err != nil || width < 1 || width > 2000 {
			pdfError(w, 400, "Invalid selection width")
			return
		}
		height, err := pdfNumber(r, "height", 38)
		if err != nil || height < 1 || height > 2000 {
			pdfError(w, 400, "Invalid selection height")
			return
		}
		allPages := r.FormValue("all_pages") == "true"
		if op == "replace" || op == "restyle" {
			if find == "" || len(find) > 220 {
				pdfError(w, 400, "Enter text to find")
				return
			}
			if op == "replace" && len(newText) > 350 {
				pdfError(w, 400, "Replacement is too long")
				return
			}
		}
		if (op == "text" || op == "watermark" || op == "sign") && (newText == "" || len(newText) > 180) && !(op == "sign" && r.FormValue("signature_mode") == "draw") {
			pdfError(w, 400, "Enter the text you want to add")
			return
		}
		var signature *gopdf.Image
		if op == "sign" && r.FormValue("signature_mode") == "draw" {
			bytes, err := pdfFormFile(r, "signature", 2<<20)
			if err != nil {
				pdfError(w, 400, err.Error())
				return
			}
			signature, err = doc.AddImageReader(strings.NewReader(string(bytes)))
			if err != nil {
				pdfError(w, 422, "Upload a valid PNG or JPEG signature")
				return
			}
		}
		for i := 0; i < count; i++ {
			page, err := doc.EditPage(src, i)
			if err != nil {
				pdfError(w, 422, "Could not edit this PDF page")
				return
			}
			if !allPages && i != selectedPage-1 {
				continue
			}
			switch op {
			case "replace":
				n, err := page.ReplaceText(find, newText)
				if err != nil {
					pdfError(w, 422, "Text replacement failed: "+err.Error())
					return
				}
				success += n
			case "restyle":
				style := gopdf.TextStyle{Size: size, Color: &color, Font: pdfFont(r.FormValue("style"))}
				for _, run := range page.Runs() {
					if strings.Contains(run.Text, find) {
						if err := run.Restyle(style); err != nil {
							pdfError(w, 422, "Formatting could not be applied: "+err.Error())
							return
						}
						success++
					}
				}
			case "text", "sign":
				if op == "sign" && signature != nil {
					page.DrawImage(signature, x, y, width, height)
				} else {
					page.SetFont(pdfFont(r.FormValue("style")), size)
					page.SetFillColor(color)
					page.Text(x, y, newText)
				}
				success++
			case "watermark":
				page.Push()
				page.SetAlpha(0.19, 0.19)
				page.SetFillColor(color)
				page.SetFont(pdfFont("bold"), size)
				page.RotateAt(-35, page.Width()/2, page.Height()/2)
				page.TextAligned(0, page.Height()/2, page.Width(), gopdf.AlignCenter, newText)
				page.Pop()
				success++
			case "highlight":
				page.AddHighlight(x, y, width, height, newText, gopdf.NoteOptions{})
				success++
			case "rectangle":
				page.AddSquareAnnotation(x, y, width, height, newText, gopdf.NoteOptions{})
				success++
			case "rotate":
				deg, err := strconv.Atoi(r.FormValue("degrees"))
				if err != nil || (deg != 0 && deg != 90 && deg != 180 && deg != 270) {
					pdfError(w, 400, "Rotation must be 0, 90, 180 or 270 degrees")
					return
				}
				page.SetRotate(deg)
				success++
			}
		}
		if (op == "replace" || op == "restyle") && success == 0 {
			pdfError(w, 422, "No selectable text matched. Try another phrase or a non-scanned PDF.")
			return
		}
	default:
		pdfError(w, 400, "This PDF operation is not supported")
		return
	}
	temp, err := os.CreateTemp("", "weme-output-*.pdf")
	if err != nil {
		pdfError(w, 500, "Could not create output")
		return
	}
	path := temp.Name()
	temp.Close()
	defer os.Remove(path)
	if err := doc.Save(path); err != nil {
		pdfError(w, 422, "PDF save failed: "+err.Error())
		return
	}
	pdfSendFile(w, path, "weme-"+op+".pdf")
}

func pdfSendFile(w http.ResponseWriter, path, name string) {
	f, err := os.Open(path)
	if err != nil {
		pdfError(w, 500, "Could not read output")
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name))
	io.Copy(w, f)
}

// Compile-time assertion that our multipart parsing returns the standard file type.
var _ multipart.File
