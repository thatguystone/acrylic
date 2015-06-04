# Toner

Directory layout:

* `data`: any extra data you want to be able to access in templates
* `content`: actual content that you want on your site
* `layouts`: templates used to display the content
* `themes`: overall site theming
* `public`: the generated site

Tags:

* `js <string>`: add a script to page
* `js_all`: print the combined js tag
* `css <string>`: add a css file to page
* `css_all`: print the combined css tag
* `img <src> [opts...]`: insert an image; opts follow
	# width=px - use 0 to scale automatically
	# height=px - use 0 to scale automatically
	# crop="<left,centered>" (quotes matter)

Notes:

* Content exists independent of extension: `test.html` and `test.png` are the same.
* meta `layoutName` to change which layout is used for the page
