package cardigann

import "strings"

// These are derived from the host query, not configurable template inputs.
// In particular an explicitly present empty string ID differs from a nil ID.
func applyDerivedQueryFields(variables Variables, mode string) {
	variables[".Query.IMDBIDShort"] = TemplateValue{Text: strings.TrimLeft(variables[".Query.IMDBID"].Text, "t")}
	present := func(name string) bool { _, ok := variables[".Query."+name]; return ok }
	nonblank := func(name string) bool { return strings.TrimSpace(variables[".Query."+name].Text) != "" }
	identifier := nonblank("Ep") || nonblank("Season") || present("Year")
	for _, name := range []string{"IMDBID", "TVDBID", "TVRageID", "TMDBID", "TVMazeID", "TraktID", "DoubanID"} {
		identifier = identifier || present(name)
	}
	for _, name := range []string{"Album", "Artist", "Label", "Genre", "Track", "Author", "Title", "Publisher"} {
		identifier = identifier || nonblank(name)
	}
	flags := map[string]bool{
		"IsSearch": mode == "search", "IsMovieSearch": mode == "movie",
		"IsTVSearch": mode == "tvsearch", "IsMusicSearch": mode == "music", "IsBookSearch": mode == "book",
		"IsIdSearch": identifier, "IsRssSearch": !nonblank("Q") && !identifier,
		"IsImdbQuery": present("IMDBID"), "IsTvdbQuery": present("TVDBID"),
		"IsTVRageQuery": present("TVRageID"), "IsTmdbQuery": present("TMDBID"),
		"IsTvmazeQuery": present("TVMazeID"), "IsTraktQuery": present("TraktID"),
		"IsDoubanQuery": present("DoubanID"), "IsGenreQuery": present("Genre"),
	}
	for name, enabled := range flags {
		value := TemplateValue{}
		if enabled {
			value.Text = "True"
		}
		variables[".Query."+name] = value
	}
}
