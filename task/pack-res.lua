-- check up config data existence
assert(cfg, "config is absent")
assert(cfg.info, "package info is absent")
assert(cfg.info.label, "label in package info is absent")
assert(cfg.skinset, "skin set is absent")
assert(cfg.iconset, "icon set is absent")
assert(cfg.defskinid, "default skin ID is absent")
assert(cfg.deficonid, "default icon ID is absent")

-- get frontend data directory
local rootdir = path.join(scrdir, "..", "frontend").."/"

-- full map of skins identifiers to lists of files
local fullskinmap = {
	["daylight"] = {"daylight.css"},
	["blue"] = {"blue.css"},
	["dark"] = {"dark.css"},
	["neon"] = {"neon.css", "nightglow-rainbow.jpg"},
	["cup-of-coffee"] = {"cup-of-coffee.css", "cup-of-coffee.jpg"},
	["coffee-beans"] = {"coffee-beans.css", "coffee-beans.jpg"},
	["old-monitor"] = {"old-monitor.css"},
}
-- full list of skins identifiers
local fullskinset = {
	"daylight", "blue", "dark", "neon", "cup-of-coffee", "coffee-beans", "old-monitor",
}
-- full list of icons identifiers
local fulliconset = {
	junior = {"avif", "webp", "jp2"},
	oxygen = {"avif", "webp", "png"},
	tulliana = {"webp", "png"},
	ubuntu = {"webp", "png"},
	papirus = {"svg"},
	chakram = {"avif", "webp", "jp2"},
	senary = {"avif", "webp", "jp2"},
	senary2 = {"avif", "webp", "jp2"},
	delta = {"avif", "webp", "jp2"},
	whistlepuff = {"avif", "webp", "gif"},
	xrabbit = {"avif", "webp", "jp2"},
}
-- iconfmt json bodies
local iconfmtjson = {
	svg = [[

		{
			"ext": ".svg",
			"mime": "image/svg+xml"
		}]],
	avif = [[

		{
			"ext": ".avif",
			"mime": "image/avif"
		}]],
	webp = [[

		{
			"ext": ".webp",
			"mime": "image/webp"
		}]],
	jp2 = [[

		{
			"ext": ".jp2",
			"mime": "image/jp2"
		}]],
	png = [[

		{
			"ext": ".png",
			"mime": "image/png"
		}]],
	gif = [[

		{
			"ext": ".gif",
			"mime": "image/gif"
		}]],
}

-- write to log formatted string
local function logfmt(...)
	log(string.format(...))
end
-- format path with environment variables
local function envfmt(p)
	return path.toslash(string.gsub(p, "%${(%w+)}", function(varname)
		return os.getenv(varname)
	end))
end

-- inits new package
local pkg = wpk.new()
pkg.automime = true -- put MIME type for each file if it is not given explicit
pkg.secret = "hms-package" -- private key to sign cryptographic hashes for each file
pkg.crc32 = true -- generate CRC32 Castagnoli code for each file
pkg.sha256 = true -- generate SHA256 hash for each file
pkg:setinfo(cfg.info) -- set package info

-- open wpk-file for write
pkg:begin(envfmt("${GOPATH}/bin/"..cfg.info.label..".wpk"))

-- write record log
local function logfile(kpath)
	logfmt("#%d %s, %d bytes, %s",
		pkg:gettag(kpath, "fid").uint32, kpath,
		pkg:filesize(kpath), pkg:gettag(kpath, "mime").string)
end
-- patterns for ignored files
local skippattern = {
	"^resmodel%.json$", -- skip original script
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
local function commonput(kpath, fpath)
	pkg:putfile(kpath, fpath)
	if logrec then logfile(kpath) end
end
local function authput(kpath, fpath)
	pkg:putfile(kpath, fpath)
	pkg:settag(kpath, "author", "schwarzlichtbezirk")
	if logrec then logfile(kpath) end
end
local function packdir(prefix, dir, putfunc)
	for i, name in ipairs(path.enum(dir)) do
		local kpath = path.join(prefix, name)
		local fpath = path.join(dir, name)
		local access, isdir = checkfile(fpath)
		if access and checkname(name) then
			if isdir then
				packdir(kpath, fpath, putfunc)
			else
				putfunc(kpath, fpath)
			end
		end
	end
	if logdir then logfmt("packed dir %s", dir) end
end

if logdir then logfmt("writes %s package", pkg.pkgpath) end

-- put some directories as is
packdir("assets", rootdir.."assets", commonput)
-- put skins
for i, id in ipairs(cfg.skinset) do
	for j, fname in ipairs(fullskinmap[id]) do
		local kpath = path.join("skin", id, fname)
		authput(kpath, rootdir..kpath)
	end
end
-- put icons
for id, fmtlst in pairs(cfg.iconset) do
	local function iconput(kpath, fpath)
		local is = false
		for i, name in ipairs(fmtlst) do
			if string.lower(string.sub(kpath, -string.len(name)-1)) == "."..name then
				is = true
				break
			end
		end
		if is then
			pkg:putfile(kpath, fpath)
			if logrec then logfile(kpath) end
		end
	end
	-- put icons with specified formats
	local kpath = "icon/"..id
	packdir(kpath, rootdir..kpath, iconput)
	-- read iconset json file
	local kpath = "icon/"..id..".json"
	local f = assert(io.open(rootdir..kpath, "rb"))
	local content = f:read("*all")
	f:close()
	-- make iconset content
	local fmts = {}
	for i, name in ipairs(fmtlst) do
		table.insert(fmts, iconfmtjson[name])
	end
	-- modify file content to put available formats
	content = string.gsub(content, '\t"iconfmt": %[%]', '\t"iconfmt": ['..table.concat(fmts, ",")..'\n\t]')
	-- write modified iconset json file to package
	pkg:putdata(kpath, content)
	pkg:settag(kpath, "author", "schwarzlichtbezirk")
	if logrec then logfile(kpath) end
end

-- put modified resmodel.json
do
	local f = assert(io.open(rootdir.."assets/resmodel.json", "rb"))
	local content = f:read("*all")
	f:close()
	-- cut extra items from skinlist
	for i, id1 in ipairs(fullskinset) do
		local found = false
		for i, id2 in ipairs(cfg.skinset) do
			if id1 == id2 then
				found = true
				break
			end
		end
		if not found then
			content = string.gsub(content, ",?%s*{%s-\"id\": \""..string.gsub(id1, "%-", "%%-").."\".-}", "")
		end
	end
	-- cut extra items from iconlist
	for id1, fmtlst in pairs(fulliconset) do
		local found = false
		for id2 in pairs(cfg.iconset) do
			if id1 == id2 then
				found = true
				break
			end
		end
		if not found then
			content = string.gsub(content, ",?%s*{%s-\"id\": \""..string.gsub(id1, "%-", "%%-").."\".-}", "")
		end
	end
	-- replace defskinid
	content = string.gsub(content, "\"defskinid\": \"[%w%-]+\"", "\"defskinid\": \""..cfg.defskinid.."\"")
	-- replace deficonid
	content = string.gsub(content, "\"deficonid\": \"[%w%-]+\"", "\"deficonid\": \""..cfg.deficonid.."\"")
	-- correct arrays
	content = string.gsub(content, "%[,", "[")
	-- put modified file
	pkg:putdata("assets/resmodel.json", content)
end

-- replace wpk-name in settings
do
	local f = assert(io.open(envfmt("${GOPATH}/bin/config/settings.yaml"), "rb"))
	local content = f:read("*all")
	f:close()
	content = string.gsub(content, "\"wpk%-full%.wpk\"", "\""..cfg.info.label..".wpk\"")
	local f = assert(io.open(envfmt("${GOPATH}/bin/config/settings.yaml"), "wb+"))
	f:write(content)
	f:close()
end

if logdir then
	logfmt("packaged %d files to %d aliases on %d bytes", pkg.recnum, pkg.tagnum, pkg.datasize)
else
	logfmt("%s package: %d files, %d aliases, %d bytes", pkg.pkgpath, pkg.recnum, pkg.tagnum, pkg.datasize)
end

-- write records table, tags table and finalize wpk-file
pkg:finalize()