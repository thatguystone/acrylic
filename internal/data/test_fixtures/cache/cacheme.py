#! /usr/bin/env python

import json
import time

print(json.dumps({
	"ts": time.time(),
	"acrylic_expires": int(time.time()) + 864000,
}))
