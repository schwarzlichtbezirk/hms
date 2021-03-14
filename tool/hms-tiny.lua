
-- package with minimal resources size

wpkconf = {
	-- result package name
	label = "hms-tiny",
	-- list of skins IDs, see 'id' tags of 'skinlist' in 'resmodel.json' file
	skinset = {
		"daylight", "blue", "dark",
	},
	-- list of icons collections IDs, see 'id' tags of 'iconlist' in 'resmodel.json' file
	iconset = {
		"tulliana",
	},
	-- default skin ID if nothing was selected
	defskinid = "blue",
	-- default icons collection ID if nothing was selected
	deficonid = "tulliana",
}

-- enable/disable progress log
logrec = false
logdir = false

-- icons format
iconwebp = true
iconpng = false

dofile(path.join(scrdir, "pack.lua"))
