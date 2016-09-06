#! /usr/bin/env python

import json
import time

print(json.dumps({
	"ts": time.time(),
	"acrylic_expires": -1,
}))
