
-- all icons collections in webp format

cfg = {
	-- package info
	info = {
		label = "hms-webp",
		link = "github.com/schwarzlichtbezirk/hms",
	},
	-- list of skins IDs, see 'id' tags of 'skinlist' in 'resmodel.json' file
	skinset = {
		"daylight", "light", "blue", "dark", "neon",
		"cup-of-coffee", "coffee-beans", "old-monitor",
	},
	-- list of icons collections IDs, see 'id' tags of 'iconlist' in 'resmodel.json' file
	iconset = {
		junior = {"webp"},
		oxygen = {"webp"},
		tulliana = {"webp"},
		ubuntu = {"webp"},
		papirus = {"svg"},
		paomedia = {"svg"},
		chakram = {"webp"},
		senary = {"webp"},
		senary2 = {"webp"},
		delta = {"webp"},
		whistlepuff = {"webp"},
		xrabbit = {"webp"},
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
