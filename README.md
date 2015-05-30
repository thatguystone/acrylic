# Toner

Directory layout:

* `data`: any extra data you want to be able to access in templates
* `content`: actual content that you want on your site
* `layouts`: templates used to display the content
* `themes`: overall site theming
* `public`: the generated site

Tags:

* `js <string>`: add a script to page
* `js_tags`: print out all js tags
* `css <string>`: add a css file to page
* `css_tags`: print out all css tags
* `img <src> [opts...]`: insert an image; opts follow
	# width=px - use 0 to scale automatically
	# height=px - use 0 to scale automatically
	# crop="<left,centered>" (quotes matter)
