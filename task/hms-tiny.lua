
-- package with minimal resources size

cfg = {
	-- package info
	info = {
		label = "hms-tiny",
		link = "github.com/schwarzlichtbezirk/hms",
	},
	-- list of skins IDs, see 'id' tags of 'skinlist' in 'resmodel.json' file
	skinset = {
		"daylight", "blue", "dark",
	},
	-- list of icons collections IDs, see 'id' tags of 'iconlist' in 'resmodel.json' file
	iconset = {
		tulliana = {"webp"},
	},
	-- default skin ID if nothing was selected
	defskinid = "blue",
	-- default icons collection ID if nothing was selected
	deficonid = "tulliana",
}

-- enable/disable progress log
logrec = false
logdir = false

dofile(path.join(scrdir, "pack-res.lua"))
