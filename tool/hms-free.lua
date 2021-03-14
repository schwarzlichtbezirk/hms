
-- only with free resources iconset
-- with allowed commercial usage

wpkconf = {
	-- result package name
	label = "hms-free",
	-- list of skins IDs, see 'id' tags of 'skinlist' in 'resmodel.json' file
	skinset = {
		"daylight", "blue", "dark", "neon",
		"cup-of-coffee", "coffee-beans", "old-monitor",
	},
	-- list of icons collections IDs, see 'id' tags of 'iconlist' in 'resmodel.json' file
	iconset = {
		"oxygen", "ubuntu", "tulliana",
	},
	-- default skin ID if nothing was selected
	defskinid = "neon",
	-- default icons collection ID if nothing was selected
	deficonid = "ubuntu",
}

-- enable/disable progress log
logrec = false
logdir = false

-- icons format
iconwebp = true
iconpng = false

dofile(path.join(scrdir, "pack.lua"))
