package toner

var (
	defaultLayouts = map[string]string{
		"_img":    `<img src="{% img_src src %}">`,
		"_single": "{{ Content }}",
		"_list":   "<not yet implemented>",
	}
)
