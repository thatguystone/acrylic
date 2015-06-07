# Acrylic [![Build Status](https://travis-ci.org/thatguystone/acrylic.svg)](https://travis-ci.org/thatguystone/acrylic)

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
	1. width=px - use 0 to scale automatically
	1. height=px - use 0 to scale automatically
	1. crop="<left,centered>" (quotes matter)

Notes:

* Content exists independent of extension: `test.html` and `test.png` are the same.
* metas:
	1. `layoutName: <string>` to change which layout is used for the page
	1. `publish: <bool>` to control content publishing
* summary
	1. not safe to use from content, can result in deadlock for circular summaries
	1. <!--more--> for summary cuts
* assets: may be fetched from a remote HTTP server
* future publishing disabled by default
	1. use meta `publish:true` to force to publish
	1. use config PublishFuture to publish all
