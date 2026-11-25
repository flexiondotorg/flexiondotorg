<p align="center">
  <a href="{{.Config.Profile.HeroURL}}">{{pictureTag .AssetBaseURL .HeroBase .Config.Profile.HeroTitle 840}}</a>
</p>
{{- if .NavPills}}
<p align="center">
{{- range .NavPills}}
  <a href="{{.URL}}">{{pictureTag $.AssetBaseURL .BaseName .Label 0}}</a>
{{- end}}
</p>
{{- end}}
{{- if .Config.Profile.Bio}}

{{.Config.Profile.Bio}}
{{- end}}
{{- if and .Discord .Discord.Enabled .Discord.BaseName}}

<p align="center">
  <a href="{{.Discord.URL}}">{{pictureTag .AssetBaseURL .Discord.BaseName .Discord.Name 840}}</a>
</p>
{{- end}}
{{- if .StatsBase}}

<p align="center">
  <a href="https://github.com/{{.Config.GitHub.Username}}">{{pictureTag .AssetBaseURL .StatsBase "GitHub statistics" 840}}</a>
</p>
{{- end}}
{{- if .Projects}}

<p align="center">
  {{pictureTag .AssetBaseURL (index .SectionBases "projects") "Projects" 840}}
</p>

<p align="center">
{{- range .Projects}}
  <a href="{{.URL}}">{{pictureTag $.AssetBaseURL .BaseName .Name 408}}</a>
{{- end}}
</p>
{{- end}}
{{- if and .Sponsors .Sponsors.Enabled .Sponsors.BaseName}}

<p align="center">
  <a href="{{.Sponsors.URL}}">{{pictureTag .AssetBaseURL .Sponsors.BaseName (or .Sponsors.Alt "Sponsor on GitHub") 840}}</a>
</p>
{{- else if and .GitHub .Config.GitHub.Username}}
{{- $sponsorsURL := .Config.GitHub.SponsorsURL}}
{{- if not $sponsorsURL}}{{$sponsorsURL = printf "https://github.com/sponsors/%s" .Config.GitHub.Username}}{{end}}

<p align="center"><sub>Enjoy my work? <a href="{{$sponsorsURL}}">Sponsor on GitHub</a></sub></p>
{{- end}}
{{- if and .Podcast .Podcast.Enabled .Podcast.Items}}

<p align="center">
  {{pictureTag .AssetBaseURL (index .SectionBases "podcast") "Podcast" 840}}
</p>
{{- if .Podcast.CoverURL}}

<a href="{{.Podcast.SiteURL}}"><img align="right" src="{{.Podcast.CoverURL}}" alt="{{.Podcast.Title}}" width="160" height="160"></a>
{{- end}}
{{range .Podcast.Items}}
- {{if .Episode}}**#{{.Episode}}** · {{end}}**[{{.Title}}]({{.URL}})**{{if .Duration}} · `{{.Duration}}`{{end}} ({{humanize .PublishedAt}})
{{- if .Subtitle}}
  - {{.Subtitle}}
{{- end}}
{{- end}}
<br clear="right" />
{{- if .Podcast.Links}}

<p align="center">
{{- range .Podcast.Links}}
  <a href="{{.URL}}">{{pictureTag $.AssetBaseURL .BaseName .Label 0}}</a>
{{- end}}
</p>
{{- end}}
{{- end}}
{{- if .PackageHosts}}

<p align="center">
  {{pictureTag .AssetBaseURL (index .SectionBases "packages") "Package Repositories" 840}}
</p>

<p align="center">
{{- range .PackageHosts}}
  <a href="{{.URL}}">{{pictureTag $.AssetBaseURL .BaseName (printf "%s (%s)" .Name .Host) 198}}</a>
{{- end}}
</p>
{{- end}}
{{- if and .GitHub .GitHub.Releases}}

<p align="center">
  {{pictureTag .AssetBaseURL (index .SectionBases "releases") "Recent Releases" 840}}
</p>
{{range .GitHub.Releases}}
- [**{{.RepoFullName}}**]({{.RepoURL}}) ([`{{.TagName}}`]({{.ReleaseURL}}), {{humanize .PublishedAt}}){{if .Description}} - {{.Description}}{{end}}
{{- end}}
{{- end}}
{{- if and .GitHub .GitHub.Contributions}}

<p align="center">
  {{pictureTag .AssetBaseURL (index .SectionBases "contributions") "Community Contributions" 840}}
</p>
{{range .GitHub.Contributions}}
- [**{{.RepoFullName}}**]({{.RepoURL}}) ({{humanize .OccurredAt}}){{if .Description}} - {{.Description}}{{end}}
{{- end}}
{{- end}}
{{- if and .GitHub .GitHub.Starred}}

<p align="center">
  {{pictureTag .AssetBaseURL (index .SectionBases "starred") "Starred" 840}}
</p>
{{range .GitHub.Starred}}
- [**{{.RepoFullName}}**]({{.RepoURL}}) · `★ {{formatInt .Stars}}`{{if .Description}} - {{.Description}}{{end}}
{{- end}}
{{- end}}
{{- if and .Blog .Blog.Enabled .Blog.Items}}

<p align="center">
  {{- if .Blog.SiteURL}}
  <a href="{{.Blog.SiteURL}}">{{pictureTag .AssetBaseURL (index .SectionBases "blog") "Blog" 840}}</a>
  {{- else}}
  {{pictureTag .AssetBaseURL (index .SectionBases "blog") "Blog" 840}}
  {{- end}}
</p>
{{range .Blog.Items}}
- **[{{.Title}}]({{.URL}})**{{if gt .ReadTimeMinutes 0}} · `{{.ReadTimeMinutes}} min read`{{end}} ({{humanize .PublishedAt}})
{{- if .Subtitle}}
  - {{.Subtitle}}
{{- end}}
{{- end}}
{{- end}}
{{- $hasTwitch := and .Twitch .Twitch.Enabled .Twitch.BaseName}}
{{- $hasYouTube := and .YouTube .YouTube.Enabled .YouTube.Items}}
{{- if or $hasTwitch $hasYouTube}}

<p align="center">
  {{pictureTag .AssetBaseURL (index .SectionBases "videos") (or .VideosHeading "Videos & Streams") 840}}
</p>
{{- if $hasYouTube}}
{{range .YouTube.Items}}
- [{{.Title}}]({{.URL}}) ({{humanize .PublishedAt}})
{{- end}}
{{- end}}
{{- if $hasTwitch}}

<p align="center">
  <a href="{{.Twitch.URL}}">{{pictureTag .AssetBaseURL .Twitch.BaseName .Twitch.Alt 0}}</a>
</p>
{{- end}}
{{- end}}

<p align="center"><sub>Crafted with <a href="https://github.com/flexiondotorg/flexiondotorg/tree/main/cmd/vitrine">Vitrine</a></sub></p>
