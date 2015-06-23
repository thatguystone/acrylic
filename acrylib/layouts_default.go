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
 <lastBuildDate>{{ time.Now() }}</lastBuildDate>
 <pubDate>Sun, 06 Sep 2009 16:20:00 +0000</pubDate>
 <ttl>1800</ttl>

 <item>
  <title>Example entry</title>
  <description>Here is some text containing an interesting description.</description>
  <link>http://www.example.com/blog/post/1</link>
  <guid isPermaLink="true">7bd204c6-1655-4c27-aeee-53f933c5395f</guid>
  <pubDate>Sun, 06 Sep 2009 16:20:00 +0000</pubDate>
 </item>

</channel>
</rss>`,
	}
)
