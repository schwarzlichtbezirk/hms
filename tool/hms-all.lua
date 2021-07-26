
-- all icons collections in webp format

wpkconf = {
	-- result package name
	label = "hms-all",
	-- default skin ID if nothing was selected
	defskinid = "neon",
	-- default icons collection ID if nothing was selected
	deficonid = "junior",
}

-- enable/disable progress log
logrec = false
logdir = false

-- icons formats provided in package
packfmt = {
	webp = true,
	png = false,
}

dofile(path.join(scrdir, "pack.lua"))
