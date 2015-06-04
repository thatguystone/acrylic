package toner

var (
	defaultLayouts = map[string]string{
		"_img": `<img src="{% img_src src width=w height=h crop=crop ext=ext %}" style="` +
			`width:{{ w }}px;` +
			`height:{{ h }}px;` +
			`">`,
		"_list":   "<not yet implemented>",
		"_single": "{% content %}",
		"_index":  "It works! Now go add a layout for _index.",
	}
)
