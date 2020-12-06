
-- only with free resources iconset
-- with allowed commercial usage
wpkconf = {
	fname = "hms-free.wpk",
	skinset = {
		"daylight", "blue", "neon",
		"cup-of-coffee", "coffee-beans", "old-monitor",
	},
	iconset = {
		"oxygen", "ubuntu",
	},
	defskinid = "neon",
	deficonid = "ubuntu",
}

-- enable/disable progress log
logrec = false
logdir = false

-- icons format
iconwebp = true
iconpng = false

dofile(scrdir.."pack.lua")
