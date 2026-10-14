package main

import (
    "bytes"
    "encoding/json"
    "image"
    "image/color"
    "image/png"
    "io"
    "mime/multipart"
    "net/http"
    "net/http/httptest"
    "os"
    "path/filepath"
    "strings"
    "testing"

    "github.com/SalvioniDigitalSolutions/gopdf"
)

func fixturePDF(t *testing.T, text string) []byte {
    t.Helper()
    doc := gopdf.New()
    page := doc.AddPage()
    page.SetFont(gopdf.Helvetica, 18)
    page.Text(66, 120, text)
    filename := filepath.Join(t.TempDir(), "test.pdf")
    if err := doc.Save(filename); err != nil { t.Fatal(err) }
    b, err := os.ReadFile(filename)
    if err != nil { t.Fatal(err) }
    return b
}
func pdfRequest(t *testing.T, route string, data []byte, fields map[string]string, other []byte) *httptest.ResponseRecorder {
    t.Helper()
    var buffer bytes.Buffer
    mp := multipart.NewWriter(&buffer)
    file, err := mp.CreateFormFile("pdf", "sample.pdf")
    if err != nil { t.Fatal(err) }
    if _, err := file.Write(data); err != nil { t.Fatal(err) }
    if other != nil {
        second, err := mp.CreateFormFile("pdf2", "second.pdf")
        if err != nil { t.Fatal(err) }
        if _, err = second.Write(other); err != nil { t.Fatal(err) }
    }
    for key, value := range fields {
        if err := mp.WriteField(key, value); err != nil { t.Fatal(err) }
    }
    if err := mp.Close(); err != nil { t.Fatal(err) }
    request := httptest.NewRequest(http.MethodPost, route, &buffer)
    request.Header.Set("Content-Type", mp.FormDataContentType())
    recorder := httptest.NewRecorder()
    app := testApplication()
    app.routes().ServeHTTP(recorder, request)
    return recorder
}
func TestPDFInspectAndReplaceText(t *testing.T) {
    original := fixturePDF(t, "HELLO DRAFT 2026")
    inspected := pdfRequest(t, "/api/pdf/inspect", original, map[string]string{"page":"1"}, nil)
    if inspected.Code != http.StatusOK { t.Fatalf("inspect: %d %s", inspected.Code, inspected.Body.String()) }
    var metadata struct {
        PageCount int `json:"page_count"`
        Runs []pdfRunInfo `json:"runs"`
    }
    if err := json.Unmarshal(inspected.Body.Bytes(), &metadata); err != nil { t.Fatal(err) }
    if metadata.PageCount != 1 || len(metadata.Runs) == 0 {
        t.Fatalf("missing PDF text inspection: %+v", metadata)
    }
    response := pdfRequest(t, "/api/pdf/edit", original, map[string]string{
        "operation": "replace", "page": "1", "find": "DRAFT", "text": "FINAL",
    }, nil)
    if response.Code != 200 { t.Fatalf("replace: %d %s", response.Code, response.Body.String()) }
    if response.Header().Get("Content-Type") != "application/pdf" { t.Fatal("missing PDF download headers") }
    result, err := gopdf.NewReader(response.Body.Bytes())
    if err != nil { t.Fatal(err) }
    out := gopdf.New()
    page, err := out.EditPage(result, 0)
    if err != nil { t.Fatal(err) }
    var combined strings.Builder
    for _, run := range page.Runs() { combined.WriteString(run.Text) }
    if !strings.Contains(combined.String(), "FINAL") || strings.Contains(combined.String(), "DRAFT") {
        t.Fatalf("text was not truly replaced: %q", combined.String())
    }
}
func TestPDFMergeExtractAndRotate(t *testing.T) {
    original := fixturePDF(t, "ONE")
    second := fixturePDF(t, "TWO")
    merged := pdfRequest(t, "/api/pdf/edit", original, map[string]string{
        "operation":"merge", "page":"1",
    }, second)
    if merged.Code != 200 { t.Fatalf("merge: %d %s", merged.Code, merged.Body.String()) }
    read, err := gopdf.NewReader(merged.Body.Bytes())
    if err != nil || read.NumPages() != 2 { t.Fatalf("merge produced invalid PDF: %v", err) }
    extract := pdfRequest(t, "/api/pdf/edit", merged.Body.Bytes(), map[string]string{
        "operation":"extract", "page":"1", "pages":"2",
    }, nil)
    if extract.Code != 200 { t.Fatalf("extract: %d %s", extract.Code, extract.Body.String()) }
    pages, err := gopdf.NewReader(extract.Body.Bytes())
    if err != nil || pages.NumPages() != 1 { t.Fatalf("extract failed: %v", err) }
    rotated := pdfRequest(t, "/api/pdf/edit", original, map[string]string{
        "operation":"rotate", "page":"1", "degrees":"90",
    }, nil)
    if rotated.Code != 200 { t.Fatalf("rotate: %d %s", rotated.Code, rotated.Body.String()) }
    if _, err := gopdf.NewReader(rotated.Body.Bytes()); err != nil { t.Fatal(err) }
}
func TestPDFRedactionRemovesSelectableText(t *testing.T) {
    original := fixturePDF(t, "PRIVATE ALICE PUBLIC")
    response := pdfRequest(t, "/api/pdf/edit", original, map[string]string{
        "operation":"redact", "page":"1", "find":"ALICE",
    }, nil)
    if response.Code != 200 { t.Fatalf("redact: %d %s", response.Code, response.Body.String()) }
    out, err := gopdf.NewReader(response.Body.Bytes())
    if err != nil { t.Fatal(err) }
    doc := gopdf.New()
    page, err := doc.EditPage(out, 0)
    if err != nil { t.Fatal(err) }
    if strings.Contains(page.ExtractText(), "ALICE") { t.Fatal("redacted text still extracts") }
}
func TestPDFPreviewIsImage(t *testing.T) {
    source := fixturePDF(t, "PREVIEW")
    response := pdfRequest(t, "/api/pdf/preview", source, map[string]string{"page":"1"}, nil)
    if response.Code != 200 { t.Fatalf("preview: %d %s", response.Code, response.Body.String()) }
    if response.Header().Get("Content-Type") != "image/png" { t.Fatal("preview not PNG") }
    if _, err := png.Decode(bytes.NewReader(response.Body.Bytes())); err != nil { t.Fatal(err) }
}
func TestPDFSignatureUploadAndValidation(t *testing.T) {
    original := fixturePDF(t, "SIGN HERE")
    var pngData bytes.Buffer
    img := image.NewRGBA(image.Rect(0, 0, 160, 40))
    img.Set(12, 15, color.RGBA{R:35,G:24,B:66,A:255})
    if err := png.Encode(&pngData, img); err != nil { t.Fatal(err) }
    var body bytes.Buffer
    mp := multipart.NewWriter(&body)
    pdf, _ := mp.CreateFormFile("pdf", "original.pdf")
    pdf.Write(original)
    sign, _ := mp.CreateFormFile("signature", "signature.png")
    io.Copy(sign, &pngData)
    for k,v := range map[string]string{
        "page":"1", "operation":"sign", "signature_mode":"draw", "x":"90", "y":"150",
        "width":"150", "height":"42",
    } { mp.WriteField(k,v) }
    mp.Close()
    req := httptest.NewRequest("POST", "/api/pdf/edit", &body)
    req.Header.Set("Content-Type", mp.FormDataContentType())
    recorder := httptest.NewRecorder()
    testApplication().routes().ServeHTTP(recorder,req)
    if recorder.Code != 200 { t.Fatalf("sign: %d %s",recorder.Code,recorder.Body.String()) }
    if _, err := gopdf.NewReader(recorder.Body.Bytes()); err != nil { t.Fatal(err) }
}
func TestPDFMalformedAndPageBounds(t *testing.T) {
    response := pdfRequest(t, "/api/pdf/edit", []byte("not a PDF"), map[string]string{"operation":"rotate","page":"1"}, nil)
    if response.Code != 400 { t.Fatalf("malformed PDF: %d",response.Code) }
    source := fixturePDF(t, "DRAFT")
    response = pdfRequest(t, "/api/pdf/edit", source, map[string]string{"operation":"extract","page":"1","pages":"999"}, nil)
    if response.Code != 400 { t.Fatalf("invalid page selection: %d",response.Code) }
    response = pdfRequest(t, "/api/pdf/edit", source, map[string]string{"operation":"replace","page":"1","find":"ABSENT","text":"YES"}, nil)
    if response.Code != 422 { t.Fatalf("missing replacement should be inconclusive: %d",response.Code) }
}
