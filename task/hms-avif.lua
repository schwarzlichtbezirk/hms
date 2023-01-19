
-- all icons collections in avif format

cfg = {
	-- package info
	info = {
		label = "hms-avif",
		link = "github.com/schwarzlichtbezirk/hms",
	},
	-- list of skins IDs, see 'id' tags of 'skinlist' in 'resmodel.json' file
	skinset = {
		"daylight", "light", "blue", "dark", "neon",
		"cup-of-coffee", "coffee-beans", "old-monitor",
	},
	-- list of icons collections IDs, see 'id' tags of 'iconlist' in 'resmodel.json' file
	iconset = {
		junior = {"avif"},
		oxygen = {"avif"},
		tulliana = {"avif"},
		ubuntu = {"avif"},
		papirus = {"svg"},
		paomedia = {"svg"},
		chakram = {"avif"},
		senary = {"avif"},
		senary2 = {"avif"},
		delta = {"avif"},
		whistlepuff = {"avif"},
		xrabbit = {"avif"},
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
