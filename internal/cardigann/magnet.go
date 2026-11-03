package cardigann

import "strings"

// Match the pinned Jackett host defaults used by the executable comparison.
// Source attribution and the upstream revision are recorded in NOTICE.md.
var publicMagnetTrackers = [...]string{
	"udp://tracker.opentrackr.org:1337/announce",
	"udp://open.demonii.com:1337/announce",
	"udp://open.stealth.si:80/announce",
	"udp://tracker.torrent.eu.org:451/announce",
	"udp://vito-tracker.space:6969/announce",
	"udp://vito-tracker.duckdns.org:6969/announce",
	"udp://tracker.theoks.net:6969/announce",
	"udp://tracker.srv00.com:6969/announce",
	"udp://tracker.qu.ax:6969/announce",
	"udp://tracker.corpscorp.online:80/announce",
}

// Generated magnet parameters always use UTF-8 form encoding, independently of
// a site's response/query encoding. Keep all generation paths consistent and
// prevent a result hash or title from injecting additional query parameters.
func generatedMagnetURI(hash, title string) string {
	var result strings.Builder
	result.WriteString("magnet:?xt=urn:btih:")
	result.WriteString(encodeFormBytes(hash))
	result.WriteString("&dn=")
	result.WriteString(encodeFormBytes(title))
	for _, tracker := range publicMagnetTrackers {
		result.WriteString("&tr=")
		result.WriteString(encodeFormBytes(tracker))
	}
	return result.String()
}
