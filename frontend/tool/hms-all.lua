
-- all icons collections in webp format

wpkconf = {
	-- result package name
	fname = "hms-all.wpk",
	-- default skin ID if nothing was selected
	defskinid = "neon",
	-- default icons collection ID if nothing was selected
	deficonid = "junior",
}

-- enable/disable progress log
logrec = false
logdir = false

-- icons format
iconwebp = true
iconpng = false

dofile(path.join(scrdir, "pack.lua"))
