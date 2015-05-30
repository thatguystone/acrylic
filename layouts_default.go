package toner

var (
	defaultLayouts = map[string]string{
		"_js":     "<script src=\"{{ src }}\"></script>\n",
		"_css":    "<link type=\"text/css\" rel=\"stylesheet\" href=\"{{ href }}\" />\n",
		"_single": "",
		"_list":   "",
	}

	layoutNames = []string{}
)

func init() {
	for k := range defaultLayouts {
		layoutNames = append(layoutNames, k)
	}
}
