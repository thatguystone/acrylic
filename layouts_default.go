package toner

var (
	defaultLayouts = map[string]string{
		"_js":      `<script src="{{ src }}"></script>` + "\n",
		"_css":     `<link type="text/css" rel="stylesheet" href="{{ href }}">` + "\n",
		"_img_tag": `<img src="{{ src }}">`,
		"_img":     `<not yet implemented>`,
		"_single":  "{{ Content }}",
		"_list":    "<not yet implemented>",
	}
)
