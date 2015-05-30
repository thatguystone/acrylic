package toner

var (
	defaultLayouts = map[string]string{
		"_js":     `<script src="{{ src }}"></script>` + "\n",
		"_css":    `<link type="text/css" rel="stylesheet" href="{{ href }}">` + "\n",
		"_img":    `<img src="{{ src }}">` + "\n",
		"_single": "{{ Page.Content }}",
		"_list":   "<not yet implemented>",
	}

	layoutNames = []string{}
)

func init() {
	for k := range defaultLayouts {
		layoutNames = append(layoutNames, k)
	}
}
