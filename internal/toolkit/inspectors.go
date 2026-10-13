package toolkit

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/golang-jwt/jwt/v5"
	"github.com/nyaruka/phonenumbers"
)

func inspectPhone(values map[string]string) (Result, error) {
	region := values["country"]
	if region == "" {
		region = "ZZ"
	}
	number, err := phonenumbers.Parse(values["input"], region)
	if err != nil {
		return Result{}, errors.New("That number could not be parsed. Include a country code, such as +44.")
	}
	valid := phonenumbers.IsValidNumber(number)
	country := phonenumbers.GetRegionCodeForNumber(number)
	if country == "" {
		country = "Non-geographic"
	}
	international := phonenumbers.Format(number, phonenumbers.INTERNATIONAL)
	e164 := phonenumbers.Format(number, phonenumbers.E164)
	data := map[string]any{"valid": valid, "possible": phonenumbers.IsPossibleNumber(number), "country": country, "international": international, "national": phonenumbers.Format(number, phonenumbers.NATIONAL), "e164": e164, "type": phoneTypeName(phonenumbers.GetNumberType(number)), "extension": number.GetExtension()}
	title, tone := "Invalid phone format", "bad"
	if valid {
		title, tone = "Valid phone format", "good"
	}
	result, err := jsonResult(data, title, "phone-report.json")
	result.Facts = []Fact{{Label: "Validation", Value: fmt.Sprint(valid), Tone: tone}, {Label: "Country", Value: country}, {Label: "International", Value: international}, {Label: "E.164", Value: e164}}
	result.Table = &Table{Columns: []string{"Format", "Number"}, Rows: [][]string{{"International", international}, {"National", phonenumbers.Format(number, phonenumbers.NATIONAL)}, {"E.164", e164}}}
	result.Notice = "Format validation does not prove this number is active or belongs to a person."
	return result, err
}
func inspectURL(values map[string]string) (Result, error) {
	input := strings.TrimSpace(values["input"])
	parsed, err := url.Parse(input)
	if err != nil || parsed.User != nil || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || strings.ContainsFunc(input, unicode.IsSpace) {
		return Result{}, errors.New("Enter a valid HTTP or HTTPS URL without embedded credentials.")
	}
	port := parsed.Port()
	if port == "" {
		port = "80"
		if parsed.Scheme == "https" {
			port = "443"
		}
	}
	path := parsed.EscapedPath()
	if path == "" {
		path = "/"
	}
	query := []map[string]string{}
	tableRows := [][]string{}
	for _, field := range strings.Split(parsed.RawQuery, "&") {
		if field == "" {
			continue
		}
		key, value, _ := strings.Cut(field, "=")
		key, err = url.QueryUnescape(key)
		if err != nil {
			return Result{}, errors.New("Invalid percent encoding in the URL query.")
		}
		value, err = url.QueryUnescape(value)
		if err != nil {
			return Result{}, errors.New("Invalid percent encoding in the URL query.")
		}
		if !utf8.ValidString(key) || !utf8.ValidString(value) {
			return Result{}, errors.New("The URL query contains invalid UTF-8 percent encoding.")
		}
		query = append(query, map[string]string{"key": key, "value": value})
		tableRows = append(tableRows, []string{key, value})
	}
	fragment := ""
	if parsed.Fragment != "" {
		fragment = "#" + parsed.Fragment
	}
	result, err := jsonResult(map[string]any{"protocol": parsed.Scheme + ":", "hostname": parsed.Hostname(), "port": port, "path": path, "query": query, "fragment": fragment, "origin": parsed.Scheme + "://" + parsed.Host}, "A closer look at your URL", "url-report.json")
	result.Facts = []Fact{{Label: "Protocol", Value: parsed.Scheme}, {Label: "Hostname", Value: parsed.Hostname()}, {Label: "Path", Value: path}, {Label: "Query parameters", Value: fmt.Sprint(len(query))}}
	result.Table = &Table{Columns: []string{"Query key", "Value"}, Rows: tableRows}
	result.Notice = "Inspected locally. No request was sent to this destination."
	return result, err
}
func inspectIP(values map[string]string) (Result, error) {
	input := strings.TrimSpace(values["input"])
	var address netip.Addr
	var prefix *netip.Prefix
	if strings.Contains(input, "/") {
		parsed, err := netip.ParsePrefix(input)
		if err != nil {
			return Result{}, errors.New("Use a valid IPv4/IPv6 address and prefix length.")
		}
		address = parsed.Addr()
		prefix = &parsed
	} else {
		parsed, err := netip.ParseAddr(input)
		if err != nil {
			return Result{}, errors.New("Use a valid IPv4 or IPv6 address.")
		}
		address = parsed
	}
	version := "ipv6"
	bits := 128
	if address.Is4() {
		version = "ipv4"
		bits = 32
	}
	scope := "unicast"
	switch {
	case address.IsPrivate():
		scope = "private"
	case address.IsLoopback():
		scope = "loopback"
	case address.IsLinkLocalUnicast():
		scope = "linkLocal"
	case address.IsMulticast():
		scope = "multicast"
	case address.IsUnspecified():
		scope = "unspecified"
	}
	data := map[string]any{"version": version, "normalized": address.String(), "range": scope, "prefix": nil, "network": nil, "subnetMask": nil}
	facts := []Fact{{Label: "Version", Value: version}, {Label: "Scope", Value: scope}, {Label: "Normalized", Value: address.String()}}
	if prefix != nil {
		data["prefix"] = prefix.Bits()
		data["network"] = prefix.Masked().Addr().String()
		data["subnetMask"] = net.IP(net.CIDRMask(prefix.Bits(), bits)).String()
		facts = append(facts, Fact{Label: "Network", Value: prefix.Masked().Addr().String()})
	}
	result, err := jsonResult(data, "Address and network", "ip-report.json")
	result.Facts = facts
	return result, err
}
func inspectJWT(values map[string]string) (Result, error) {
	token := strings.TrimSpace(values["input"])
	if len(strings.Split(token, ".")) != 3 {
		return Result{}, errors.New("A signed JWT must contain three dot-separated parts.")
	}
	parser := jwt.NewParser(jwt.WithJSONNumber())
	decoded, _, err := parser.ParseUnverified(token, jwt.MapClaims{})
	if err != nil {
		return Result{}, errors.New("The JWT header or payload could not be decoded.")
	}
	claims := decoded.Claims.(jwt.MapClaims)
	var expiry any
	expiryStatus := "Not supplied"
	if value, ok := claims["exp"]; ok {
		timestamp, err := numericClaim(value)
		if err != nil {
			return Result{}, errors.New("The JWT exp claim must be a finite Unix timestamp in seconds.")
		}
		expiry = timestamp.UTC().Format(time.RFC3339Nano)
		expiryStatus = "Not expired"
		if time.Now().After(timestamp) {
			expiryStatus = "Expired"
		}
	}
	rows := [][]string{}
	keys := make([]string, 0, len(claims))
	for key := range claims {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := claims[key]
		text := fmt.Sprint(value)
		if result, err := jsonResult(value, "", ""); err == nil {
			if str, ok := value.(string); ok {
				text = str
			} else {
				text = result.Text
			}
		}
		rows = append(rows, []string{key, text})
	}
	result, err := jsonResult(map[string]any{"header": decoded.Header, "payload": claims, "expiresAt": expiry}, "Token contents", "jwt-report.json")
	expiryText := "Not supplied"
	if expiry != nil {
		expiryText = fmt.Sprint(expiry)
	}
	result.Facts = []Fact{{Label: "Signature", Value: "Not verified", Tone: "warn"}, {Label: "Expires", Value: expiryText}, {Label: "Expiry status", Value: expiryStatus, Tone: "warn"}}
	result.Table = &Table{Columns: []string{"Claim", "Value"}, Rows: rows}
	result.Notice = "Decoded only — the signature and authenticity have NOT been verified."
	return result, err
}

func phoneTypeName(kind phonenumbers.PhoneNumberType) string {
	names := map[phonenumbers.PhoneNumberType]string{phonenumbers.FIXED_LINE: "Fixed line", phonenumbers.MOBILE: "Mobile", phonenumbers.FIXED_LINE_OR_MOBILE: "Fixed line or mobile", phonenumbers.TOLL_FREE: "Toll free", phonenumbers.PREMIUM_RATE: "Premium rate", phonenumbers.SHARED_COST: "Shared cost", phonenumbers.VOIP: "VoIP", phonenumbers.PERSONAL_NUMBER: "Personal number", phonenumbers.PAGER: "Pager", phonenumbers.UAN: "Universal access", phonenumbers.VOICEMAIL: "Voicemail"}
	if name, ok := names[kind]; ok {
		return name
	}
	return "Unknown"
}
