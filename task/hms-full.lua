
-- only with free resources iconset
-- with allowed commercial usage

cfg = {
	-- package info
	info = {
		label = "hms-full",
		link = "github.com/schwarzlichtbezirk/hms"
	},
	-- list of skins IDs, see 'id' tags of 'skinlist' in 'resmodel.json' file
	skinset = {
		"daylight", "blue", "dark", "neon",
		"cup-of-coffee", "coffee-beans", "old-monitor",
	},
	-- list of icons collections IDs, see 'id' tags of 'iconlist' in 'resmodel.json' file
	iconset = {
		junior = {"avif", "webp", "jp2"},
		oxygen = {"avif", "webp", "png"},
		tulliana = {"webp", "png"},
		ubuntu = {"webp", "png"},
		papirus = {"svg"},
		chakram = {"avif", "webp", "jp2"},
		senary = {"avif", "webp", "jp2"},
		senary2 = {"avif", "webp", "jp2"},
		delta = {"avif", "webp", "jp2"},
		whistlepuff = {"avif", "webp", "gif"},
		xrabbit = {"avif", "webp", "jp2"},
	},
	-- default skin ID if nothing was selected
	defskinid = "neon",
	-- default icons collection ID if nothing was selected
	deficonid = "junior",
}

-- enable/disable progress log
logrec = false
logdir = false

dofile(path.join(scrdir, "pack-res.lua"))
