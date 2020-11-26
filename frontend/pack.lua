
-- full map of skins identifiers to lists of files
local fullskinmap = {
	["daylight"] = {"daylight.css"},
	["blue"] = {"blue.css"},
	["neon"] = {"neon.css", "nightglow-rainbow.jpg"},
	["cup-of-coffee"] = {"cup-of-coffee.css", "cup-of-coffee.jpg"},
	["coffee-beans"] = {"coffee-beans.css", "coffee-beans.jpg"},
	["old-monitor"] = {"old-monitor.css"},
}
-- full list of skins identifiers
local fullskinset = {
	"daylight", "blue", "neon", "cup-of-coffee", "coffee-beans", "old-monitor",
}
-- full list of icons identifiers
local fulliconset = {
	"junior", "oxygen", "delta", "ubuntu", "chakram", "whistlepuff", "xrabbit",
}

-- let's set package configuration to default if it was not provided
wpkconf = wpkconf or {
	fname = "hms-full.wpk",
	skinset = fullskinset,
	iconset = fulliconset,
	defskinid = "neon",
	deficonid = "junior",
}

-- enable/disable progress log
logrec = false
logdir = false

-- write to log formatted string
local function logfmt(...)
	log(string.format(...))
end
-- format path with environment variables
local function envfmt(p)
	return path.toslash(string.gsub(p, "%$%((%w+)%)", function(varname)
		return os.getenv(varname)
	end))
end

-- inits new package
local pkg = wpk.new()
pkg.automime = true -- put MIME type for each file if it is not given explicit
pkg.secret = "hms-package" -- private key to sign cryptographic hashes for each file
pkg.crc32 = true -- generate CRC32 Castagnoli code for each file
pkg.sha256 = true -- generate SHA256 hash for each file

-- open wpk-file for write
pkg:begin(envfmt("$(GOPATH)/bin/"..wpkconf.fname))

-- write record log
local function logfile(kpath)
	logfmt("#%d %s, %d bytes, %s",
		pkg:gettag(kpath, "fid").uint32, kpath,
		pkg:filesize(kpath), pkg:gettag(kpath, "mime").string)
end
-- patterns for ignored files
local skippattern = {
	"^resmodel%.json$", -- skip original script
	"^pack%.lua$", -- script that generate this package
	"^build%.%w+%.cmd$",
	"^thumb%.db$",
}
-- extensions of files that should not be included to package
local skipext = {
	wpk = true,
	sys = true,
	tmp = true,
	bak = true,
	-- compiler intermediate output
	log = true, tlog = true, lastbuildstate = true, unsuccessfulbuild = true,
	obj = true, lib = true, res = true,
	ilk = true, idb = true, ipdb = true, iobj = true, pdb = true, pgc = true, pgd = true,
	pch = true, ipch = true,
	cache = true,
}
-- check file names can be included to package
local function checkname(name)
	local fc = string.sub(name, 1, 1) -- first char
	if fc == "." or fc == "~" then return false end
	name = string.lower(name)
	for i, pattern in ipairs(skippattern) do
		if string.match(name, pattern) then return false end
	end
	local ext = string.match(name, "%.(%w+)$") -- file extension
	if ext and skipext[ext] then return false end
	return true
end
-- pack given directory and add to each file name given prefix
local function packdir(prefix, dir)
	for i, name in ipairs(path.enum(dir)) do
		local kpath = prefix..name
		local fpath = dir..name
		local access, isdir = checkfile(fpath)
		if access and checkname(name) then
			if isdir then
				packdir(kpath.."/", fpath.."/")
			else
				pkg:putfile(kpath, fpath)
				if prefix == "devmode/" then
					pkg:settag(kpath, "author", "schwarzlichtbezirk")
				end
				if logrec then logfile(kpath) end
			end
		end
	end
	if logdir then logfmt("packed dir %s", dir) end
end

if logdir then logfmt("writes %s package", pkg.path) end

-- put some directories as is
packdir("assets/", scrdir.."assets/")
packdir("build/", scrdir.."build/")
packdir("devmode/", scrdir.."devmode/")
packdir("plugin/", scrdir.."plugin/")
packdir("tmpl/", scrdir.."tmpl/")
-- put skins
for i, id in ipairs(wpkconf.skinset) do
	for j, fname in ipairs(fullskinmap[id]) do
		local kpath = "skin/"..fname
		pkg:putfile(kpath, scrdir..kpath)
		pkg:settag(kpath, "author", "schwarzlichtbezirk")
	end
end
-- put icons
for i, id in ipairs(wpkconf.iconset) do
	local kpath = "icon/"..id.."/"
	packdir(kpath, scrdir..kpath)
	local kpath = "icon/"..id..".json"
	pkg:putfile(kpath, scrdir..kpath)
	pkg:settag(kpath, "author", "schwarzlichtbezirk")
end
-- put sources
for i, fpath in ipairs{path.glob(scrdir.."../?*.?*")} do
	local fname = string.match(fpath, "/([%w_]+%.%w+)$")
	if fname then
		local kpath = "src/"..fname
		pkg:putfile(kpath, fpath)
		pkg:settag(kpath, "author", "schwarzlichtbezirk")
		if logrec then logfile(kpath) end
	end
end

-- put modified resmodel.json
do
	local f = assert(io.open(scrdir.."assets/resmodel.json", "rb"))
	local content = f:read("*all")
	f:close()
	local function cut(id1, list)
		local found = false
		for i, id2 in ipairs(list) do
			if id1 == id2 then
				found = true
				break
			end
		end
		if not found then
			content = string.gsub(content, ",?%s*{%s-\"id\": \""..string.gsub(id1, "%-", "%%-").."\".-}", "")
		end
	end
	-- cut extra items from skinlist
	for i, id in pairs(fullskinset) do
		cut(id, wpkconf.skinset)
	end
	-- cut extra items from iconlist
	for i, id in ipairs(fulliconset) do
		cut(id, wpkconf.iconset)
	end
	-- replace defskinid
	content = string.gsub(content, "\"defskinid\": \"[%w%-]+\"", "\"defskinid\": \""..wpkconf.defskinid.."\"")
	-- replace deficonid
	content = string.gsub(content, "\"deficonid\": \"[%w%-]+\"", "\"deficonid\": \""..wpkconf.deficonid.."\"")
	content = string.gsub(content, "%[,", "[")
	pkg:putdata("assets/resmodel.json", content)
end

-- replace wpk-name in settings
do
	local f = assert(io.open(envfmt("$(GOPATH)/bin/hms/settings.yaml"), "rb"))
	local content = f:read("*all")
	f:close()
	content = string.gsub(content, "wpk%-name:(%s+)[%w%-]+%.wpk", "wpk-name:%1"..wpkconf.fname)
	local f = assert(io.open(envfmt("$(GOPATH)/bin/hms/settings.yaml"), "wb+"))
	f:write(content)
	f:close()
end

if logdir then
	logfmt("packaged %d files to %d aliases on %d bytes", pkg.recnum, pkg.tagnum, pkg.datasize)
else
	logfmt("%s package: %d files, %d aliases, %d bytes", pkg.path, pkg.recnum, pkg.tagnum, pkg.datasize)
end

-- write records table, tags table and finalize wpk-file
pkg:complete()
