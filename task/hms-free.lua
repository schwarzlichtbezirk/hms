
-- only with free resources iconset
-- with allowed commercial usage

cfg = {
	-- package info
	info = {
		label = "hms-free",
		link = "github.com/schwarzlichtbezirk/hms",
	},
	-- list of skins IDs, see 'id' tags of 'skinlist' in 'resmodel.json' file
	skinset = {
		"daylight", "light", "blue", "dark", "neon",
		"cup-of-coffee", "coffee-beans", "old-monitor",
	},
	-- list of icons collections IDs, see 'id' tags of 'iconlist' in 'resmodel.json' file
	iconset = {
		oxygen = {"avif", "webp", "png"},
		ubuntu = {"avif", "webp", "png"},
		papirus = {"svg"},
		paomedia = {"svg"},
		tulliana = {"avif", "webp", "png"},
	},
	-- default skin ID if nothing was selected
	defskinid = "neon",
	-- default icons collection ID if nothing was selected
	deficonid = "ubuntu",
}

-- enable/disable progress log
logrec = false
logdir = false

dofile(path.join(scrdir, "pack-res.lua"))
