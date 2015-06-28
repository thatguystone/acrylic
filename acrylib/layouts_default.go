package acrylib

var (
	defaultLayouts = map[string]string{
		"_img_tag": `<img src="{% img_src src width=w height=h crop=crop ext=ext %}" style="` +
			`width:{{ w }}px;` +
			`height:{{ h }}px;` +
			`">`,
		"_img_page": ``, // Content rendered inside a _single for images
		"_list":     "<not yet implemented>",
		"_single":   "{{ Page.Content }}",
		"_index":    "It works! Now go add a layout for _index.",
		"_rss": `<?xml version="1.0" encoding="UTF-8" ?>
<rss version="2.0">
<channel>
	<title>{{ Site.Title }} :: {{ Page.Title }}</title>
	<description>{{ Site.Description }}</description>
	<link>{{ Site.URL }}</link>
	<pubDate>{{ Site.Now.Time|date:"Mon, 02 Jan 2006 15:04:05 -0700" }}</pubDate>

	{% for c in Site.Find(Page.CPath|add:"/..").Childs %}
		<item>
			<title>{{ c.Title }}</title>
			<description>{{ c.Summary }}</description>
			<link>{{ c.AbsURL }}</link>
			<pubDate>{{ c.Date.Time|date:"Mon, 02 Jan 2006 15:04:05 -0700" }}</pubDate>
		</item>
	{% endfor %}
</channel>
</rss>`,
	}
)
