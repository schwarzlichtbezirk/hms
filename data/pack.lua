
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
pkg:begin(envfmt"$(GOPATH)/bin/hms.wpk")

-- write record log
local function logfile(kpath)
	logfmt("#%d %s, %d bytes, %s",
		pkg:gettag(kpath, "fid").uint32, kpath,
		pkg:filesize(kpath), pkg:gettag(kpath, "mime").string)
end
-- patterns for ignored files
local skippattern = {
	"^pack%.lua$", -- script that generate this package
	"^build%.%w+%.cmd$",
	"^thumb%.db$",
	"^rca%w+$",
	"^%$recycle%.bin$",
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
local function packdir(dir, prefix)
	for i, name in ipairs(path.enum(dir)) do
		local kpath = prefix..name
		local fpath = dir..name
		local access, isdir = checkfile(fpath)
		if access and checkname(name) then
			if isdir then
				packdir(fpath.."/", kpath.."/")
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

packdir(scrdir, "")
for i, fpath in ipairs{path.glob(scrdir.."../?*.?*")} do
	local fname = string.match(fpath, "/([%w_]+%.%w+)$")
	if fname then
		local kpath = "src/"..fname
		pkg:putfile(kpath, fpath)
		pkg:settag(kpath, "author", "schwarzlichtbezirk")
		if logrec then logfile(kpath) end
	end
end

if logdir then
	logfmt("packaged %d files to %d aliases on %d bytes", pkg.recnum, pkg.tagnum, pkg.datasize)
else
	logfmt("%s package: %d files, %d aliases, %d bytes", pkg.path, pkg.recnum, pkg.tagnum, pkg.datasize)
end

-- write records table, tags table and finalize wpk-file
pkg:complete()
