
-- minimal size resources
wpkconf = {
	fname = "hms-tiny.wpk",
	skinset = {
		"daylight", "blue",
	},
	iconset = {
		"ubuntu",
	},
	defskinid = "blue",
	deficonid = "ubuntu",
}

-- enable/disable progress log
logrec = false
logdir = false

-- icons format
iconwebp = true
iconpng = false

dofile(scrdir.."pack.lua")
