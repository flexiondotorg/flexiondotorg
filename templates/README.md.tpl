<p align="center">
  <a href="https://wimpysworld.com" target="_blank"><img src="https://raw.githubusercontent.com/flexiondotorg/flexiondotorg/main/.github/github-header-image.png"></a>
</p>
<p align="center">
  &nbsp;<a href="https://fosstodon.org/@wimpy" target="_blank"><img alt="Mastodon" src="https://img.shields.io/badge/Mastodon-6468fa?style=for-the-badge&logo=mastodon&logoColor=%23ffffff"></a>&nbsp;
  &nbsp;<a href="https://www.instagram.com/wimpysworld/" target="_blank"><img alt="Instagram" src="https://img.shields.io/badge/instagram-d3175c?style=for-the-badge&logo=instagram&logoColor=%23ffffff"></a>&nbsp;
  &nbsp;<a href="https://twitter.com/m_wimpress" target="_blank"><img alt="Twitter" src="https://img.shields.io/badge/Twitter-303030?style=for-the-badge&logo=x&logoColor=%23ffffff"></a>&nbsp;
  &nbsp;<a href="https://www.reddit.com/user/flexiondotorg" target="_blank"><img alt="Reddit" src="https://img.shields.io/badge/Reddit-fc4a23?style=for-the-badge&logo=reddit&logoColor=%23ffffff"></a>&nbsp;
  &nbsp;<a href="https://www.linkedin.com/in/martinwimpress/" target="_blank"><img alt="LinkedIn" src="https://img.shields.io/badge/LinkedIn-1667be?style=for-the-badge&logo=linkedin&logoColor=%23ffffff"></a>&nbsp;
</p>
<a href="https://github.com/flexiondotorg" target="_blank"><img align="right" src="https://github-readme-stats.vercel.app/api?username=flexiondotorg&show_icons=true&show=reviews,discussions_started,discussions_answered,prs_merged&include_all_commits=true&bg_color=0E1117&title_color=fa66ed&icon_color=6bbbfa&text_color=c5c8c6&ring_color=98ed3f&border_radius=8" alt="GitHub Stats"></a>
<br />
<a href="https://github.com/flexiondotorg" target="_blank"><img align="right" src="https://streak-stats.demolab.com?user=flexiondotorg&theme=cobalt&border_radius=8&date_format=j%20M%5B%20Y%5D&mode=daily&card_width=465&hide_total_contributions=true" alt="GitHub Streak" /></a>

Hello 👋 I'm Martin (*also known as Wimpy*), an open-source enthusiast, frequent contributor and maker of things.

I lead [Ubuntu MATE](https://ubuntu-mate.org) 🧉 and I'm the author [Quickemu](https://github.com/quickemu-project)
and [deb-get](https://github.com/wimpysworld/deb-get) plus a [bunch of other projects](https://wimpysworld.com/projects/).

We have a modest [Discord server for Wimpy's World](https://wimpysworld.io/discord) 💬 which serves as a community hub my projects.
We're a friendly bunch, and all are welcome 🏳️‍🌈 Join us!

<div align="center"><a href="https://wimpysworld.io/discord" target="_blank"><img alt="Discord" src="https://img.shields.io/discord/712850672223125565?style=for-the-badge&logo=discord&logoColor=%23ffffff&label=Discord&labelColor=%234253e8&color=%23e4e2e2"></a></div>

I [steam live-coding on Twitch](https://twitch.tv/WimpysWorld) 📡 and those streams are [archived on YouTube](https://youtube.com/WimpysWorld) 📺️

These awesome people sponsor me: {{range sponsors 5}}[{{.User.Login}}]({{.User.URL}}) {{- end}} Thank you! ❤️
If you appreciate the work I do, then [please consider sponsoring me too](https://github.com/sponsors/flexiondotorg) 🤑 **Curious what I've been working on recently?**
<div align="center">
  <img align="center" alt="GitHub Contribution Snake" src="https://raw.githubusercontent.com/flexiondotorg/flexiondotorg/snake/github-contribution-grid-snake-dark.svg">
</div>

### Recent project contributions 🏗️

{{range recentContributions 5}}
- [{{.Repo.Name}}]({{.Repo.URL}}) ({{humanize .OccurredAt}})
  - {{.Repo.Description}}
{{- end}}

### Recent pull requests 🛠️

{{range recentPullRequests 5}}
- [{{.Title}}]({{.URL}}) on [{{.Repo.Name}}]({{.Repo.URL}}) ({{humanize .CreatedAt}})
{{- end}}

### Recent releases 📦️

{{range recentReleases 5}}
- [{{.Name}}]({{.URL}}) ([{{.LastRelease.TagName}}]({{.LastRelease.URL}}), {{humanize .LastRelease.PublishedAt}})
  - {{.Description}}
{{- end}}

### Recent stars ⭐️

{{range recentStars 5}}
- [{{.Repo.Name}}]({{.Repo.URL}}) ({{.Repo.Stargazers}})
  - {{.Repo.Description}}
{{end}}

### 🎙️ Podcast
<img align="right" src="https://raw.githubusercontent.com/flexiondotorg/flexiondotorg/main/.github/linuxmatters.png" alt="Linux Matters Podcast" width="200" height="200">

I co-present [Linux Matters podcast](https://linuxmatters.sh) with my friends [@popey](https://github.com/popey) and [@marxjohnson](https://github.com/marxjohnson).
Here's some recent episodes:
{{range rss "https://linuxmatters.sh/episode/index.xml" 5}}
- [{{.Title}}]({{.URL}}) ({{humanize .PublishedAt}})
{{- end}}

### 📺️ YouTube
<a href="https://twitch.tv/WimpysWorld" target="_blank"><img alt="Twitch Status" src="https://img.shields.io/twitch/status/WimpysWorld?style=for-the-badge&logo=twitch&logoColor=ffffff&label=Twitch&labelColor=%23904ef9&color=%23e4e2e2"></a>&nbsp;&nbsp;
<a href="https://youtube.com/WimpysWorld" target="_blank"><img alt="YouTube Channel Subscribers" src="https://img.shields.io/youtube/channel/subscribers/UChpYmMp7EFaxuogUX1eAqyw?style=for-the-badge&logo=youtube&logoColor=ffffff&label=YouTube&labelColor=%23fb1b20&color=%23e4e2e2"></a>
{{range rss "https://www.youtube.com/feeds/videos.xml?channel_id=UChpYmMp7EFaxuogUX1eAqyw" 5}}
- [{{.Title}}]({{.URL}}) ({{humanize .PublishedAt}})
{{- end}}

### ✍️ Blog
{{range rss "https://wimpysworld.com/posts/rss.xml" 5}}
- [{{.Title}}]({{.URL}}) ({{humanize .PublishedAt}})
{{- end}}

<p align="center">
  <a href="https://github.com/flexiondotorg/flexiondotorg" target="_blank"><img alt="Hit Counter" src="https://img.shields.io/endpoint?url=https%3A%2F%2Fhits.dwyl.com%2Fflexiondotorg%2Fflexiondotorg.json&style=flat-square&logo=github&logoColor=ffffff&label=Visitors&labelColor=%23f76ce9&color=%236fbbf6">
</p>
