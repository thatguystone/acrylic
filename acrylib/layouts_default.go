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
	}
)
