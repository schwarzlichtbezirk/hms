
-- all icons collections in webp & avif format

cfg = {
	-- package info
	info = {
		label = "hms-edge",
		link = "github.com/schwarzlichtbezirk/hms"
	},
	-- list of skins IDs, see 'id' tags of 'skinlist' in 'resmodel.json' file
	skinset = {
		"daylight", "light", "blue", "dark", "neon",
		"cup-of-coffee", "coffee-beans", "old-monitor",
	},
	-- list of icons collections IDs, see 'id' tags of 'iconlist' in 'resmodel.json' file
	iconset = {
		junior = {"avif", "webp"},
		oxygen = {"avif", "webp"},
		tulliana = {"avif", "webp"},
		ubuntu = {"avif", "webp"},
		papirus = {"svg"},
		paomedia = {"svg"},
		chakram = {"avif", "webp"},
		senary = {"avif", "webp"},
		senary2 = {"avif", "webp"},
		delta = {"avif", "webp"},
		whistlepuff = {"avif", "webp"},
		xrabbit = {"avif", "webp"},
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
