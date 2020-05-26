
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
-- check file names can be included to package
local function checkname(name)
	local fc = string.sub(name, 1, 1) -- first char
	if fc == "." or fc == "~" then return false end
	name = string.lower(name)
	if name == "thumb.db" then return false end
	local ext = string.sub(name, -4, -1) -- file extension
	if ext == ".sys" or ext == ".tmp" or ext == ".bak" or ext == ".wpk" then return false end
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
				-- make aliases
				if string.sub(kpath, 1, 8) == "devmode/" then
					pkg:putalias(kpath, "devm"..string.sub(kpath, 8))
				elseif string.sub(kpath, 1, 6) == "build/" then
					pkg:putalias(kpath, "relm"..string.sub(kpath, 6))
				elseif string.sub(kpath, 1, 7) == "plugin/" then
					pkg:putalias(kpath, "plug"..string.sub(kpath, 7))
				elseif string.sub(kpath, 1, 7) == "assets/" then
					pkg:putalias(kpath, "asst"..string.sub(kpath, 7))
				elseif string.sub(kpath, 1, 9) == "template/" then
					pkg:putalias(kpath, "tmpl"..string.sub(kpath, 9))
				end
			end
		end
	end
	if logdir then logfmt("packed dir %s", dir) end
end

if logdir then logfmt("writes %s package", pkg.path) end

packdir(scrdir, "")
for i, fpath in ipairs{path.glob(scrdir.."../?*.?*")} do
	local kpath = "src/"..string.match(fpath, "/([%w_]+%.%w+)$")
	pkg:putfile(kpath, fpath)
	pkg:settag(kpath, "author", "schwarzlichtbezirk")
	if logrec then logfile(kpath) end
end

if logdir then
	logfmt("packaged %d files to %d aliases on %d bytes", pkg.recnum, pkg.tagnum, pkg.datasize)
else
	logfmt("%s package: %d files, %d aliases, %d bytes", pkg.path, pkg.recnum, pkg.tagnum, pkg.datasize)
end

-- write records table, tags table and finalize wpk-file
pkg:complete()
